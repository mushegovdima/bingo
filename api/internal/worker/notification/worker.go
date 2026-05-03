// Package notificationworker drains the notification_jobs outbox: claims a batch of
// jobs, paginates the audience defined by RecipientFilter, fans out the shared
// message body to each batch of recipients via one unary notifier.SendBulk call, and
// checkpoints progress so a crash mid-fan-out resumes from the last successful batch.
//
// Concurrency is safe across multiple instances: Claim does a single atomic
// UPDATE ... RETURNING, with the visibility predicate repeated in the outer WHERE so
// the EvalPlanQual re-check awards each row to exactly one caller. A visibility
// timeout (claimStaleAfter in the repo) reclaims jobs whose worker died holding the lease.
package notificationworker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	notificationcontract "go.mod/internal/contracts/notification"
	dbmodels "go.mod/internal/db"
	"go.mod/internal/db/repository"
	"go.mod/internal/notifications"
	"go.mod/internal/notifier"
)

const (
	defaultPollInterval = 5 * time.Second
	defaultBatchSize    = 500
	defaultMaxAttempts  = 5
)

type jobRepo interface {
	Claim(ctx context.Context, limit int) ([]*dbmodels.NotificationJob, error)
	UpdateCursor(ctx context.Context, id, cursor int64) error
	MarkDone(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64, attempts int, errMsg string, terminal bool) error
}

type userRepo interface {
	ListByFilter(ctx context.Context, filter repository.UserFilter, afterID int64, limit int) ([]*dbmodels.User, error)
}

type Config struct {
	PollInterval time.Duration
	// BatchSize is the number of recipients per SendBulk call.
	BatchSize int
	// JobBatch is the number of jobs claimed per tick.
	JobBatch int
	// MaxAttempts is the number of failed attempts after which a job is marked terminally failed.
	MaxAttempts int
}

type Worker struct {
	jobs   jobRepo
	users  userRepo
	sender notifier.Sender
	cfg    Config
	logger *slog.Logger
}

func New(jobs jobRepo, users userRepo, sender notifier.Sender, cfg Config, logger *slog.Logger) *Worker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.BatchSize <= 0 || cfg.BatchSize > notifier.MaxBulkRecipients {
		cfg.BatchSize = defaultBatchSize
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = defaultMaxAttempts
	}
	return &Worker{jobs: jobs, users: users, sender: sender, cfg: cfg, logger: logger}
}

// Run loops until ctx is cancelled. On every tick (or immediately after finishing a job)
// it tries to claim and process one job. The "drain until empty, then sleep" pattern keeps
// latency low when the queue is busy without spinning when it's empty.
func (w *Worker) Run(ctx context.Context) {
	w.logger.InfoContext(ctx, "notification worker started",
		slog.Duration("poll_interval", w.cfg.PollInterval),
		slog.Int("batch_size", w.cfg.BatchSize),
	)
	timer := time.NewTimer(0) // fire immediately
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			w.logger.InfoContext(ctx, "notification worker stopped")
			return
		case <-timer.C:
		}

		processed, err := w.tick(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			w.logger.ErrorContext(ctx, "notification worker tick failed", slog.Any("error", err))
		}

		// Reset timer: drain immediately if we just processed a job, otherwise back off.
		next := w.cfg.PollInterval
		if processed {
			next = 0
		}
		timer.Reset(next)
	}
}

// tick claims a batch of jobs and processes them in order. Returns processed=true if
// at least one job was claimed.
func (w *Worker) tick(ctx context.Context) (processed bool, err error) {
	jobs, err := w.jobs.Claim(ctx, w.cfg.JobBatch)
	if err != nil {
		return false, err
	}
	if len(jobs) == 0 {
		return false, nil
	}

	for _, job := range jobs {
		if ctx.Err() != nil {
			return true, nil
		}
		w.processOne(ctx, job)
	}
	return true, nil
}

// processOne fans out the job from job.Cursor onwards. Each batch is checkpointed before
// moving to the next, so a crash resumes from the last persisted cursor rather than
// re-sending earlier batches (notifier de-duplicates by id but we'd waste work).
func (w *Worker) processOne(ctx context.Context, job *dbmodels.NotificationJob) {
	log := w.logger.With(slog.Int64("job_id", job.ID), slog.String("type", job.Type))
	cursor := job.Cursor

	var audience notificationcontract.UserFilter
	if len(job.Filter) > 0 {
		if err := json.Unmarshal(job.Filter, &audience); err != nil {
			// Bad JSON in the outbox is unrecoverable for this job — fail it terminally.
			w.jobs.MarkFailed(ctx, job.ID, job.Attempts+1, "decode filter: "+err.Error(), true) //nolint:errcheck // best-effort
			log.ErrorContext(ctx, "failed to decode job filter", slog.Any("error", err))
			return
		}
	}
	filter := toUserFilter(audience)

	for {
		users, err := w.users.ListByFilter(ctx, filter, cursor, w.cfg.BatchSize)
		if err != nil {
			w.recordFailure(ctx, job, "list recipients: "+err.Error())
			return
		}
		if len(users) == 0 {
			break
		}

		if err := w.sendBatch(ctx, job, users); err != nil {
			w.recordFailure(ctx, job, "send: "+err.Error())
			return
		}

		cursor = users[len(users)-1].ID
		if err := w.jobs.UpdateCursor(ctx, job.ID, cursor); err != nil {
			// Cursor write failure is non-fatal for delivery (notifier dedups), but if we
			// can't checkpoint we'll re-deliver this batch on next claim. Log and bail so
			// the visibility timeout reclaims the job rather than burning attempts.
			log.ErrorContext(ctx, "failed to checkpoint cursor", slog.Any("error", err))
			return
		}

		// Stop the loop early if the context is cancelled (graceful shutdown).
		if err := ctx.Err(); err != nil {
			log.InfoContext(ctx, "shutdown during fan-out, will resume", slog.Int64("cursor", cursor))
			return
		}
	}

	if err := w.jobs.MarkDone(ctx, job.ID); err != nil {
		log.ErrorContext(ctx, "failed to mark job done", slog.Any("error", err))
		return
	}
	log.InfoContext(ctx, "notification job completed")
}

// sendBatch merges the job's notification args with per-recipient user fields,
// then calls notifications.Substitute once per recipient — the single substitution point.
func (w *Worker) sendBatch(ctx context.Context, job *dbmodels.NotificationJob, users []*dbmodels.User) error {
	recipients := make([]notifier.BulkRecipient, len(users))
	for i, u := range users {
		nu := notifications.NotificationUser{Name: u.Name, Username: u.Username}
		args := mergeArgs(job.Args, nu.UserArgs())
		recipients[i] = notifier.BulkRecipient{
			UserID:     u.ID,
			TelegramID: u.TelegramID,
			Text:       notifications.Substitute(job.Text, args),
		}
	}
	return w.sender.SendBulk(ctx, job.Type, recipients)
}

// mergeArgs returns a new map with all entries from base overwritten/extended by overlay.
func mergeArgs(base, overlay map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overlay {
		merged[k] = v
	}
	return merged
}

// recordFailure increments attempts and either re-queues the job (status=pending) or
// marks it terminally failed when the cap is reached.
func (w *Worker) recordFailure(ctx context.Context, job *dbmodels.NotificationJob, errMsg string) {
	attempts := job.Attempts + 1
	terminal := attempts >= w.cfg.MaxAttempts
	if err := w.jobs.MarkFailed(ctx, job.ID, attempts, errMsg, terminal); err != nil {
		w.logger.ErrorContext(ctx, "failed to mark job failed",
			slog.Int64("job_id", job.ID), slog.Any("error", err))
	}
}

// toUserFilter translates the notification-context audience description into the
// repository-level user query shape. Keeping this mapping at the worker boundary
// is the reason the user repository does not depend on the notification contracts package.
func toUserFilter(f notificationcontract.UserFilter) repository.UserFilter {
	return repository.UserFilter{
		IDs:              f.UserIDs,
		Roles:            f.Roles,
		ExcludeBlocked:   f.ExcludeBlocked,
		OnlyWithTelegram: f.OnlyTelegram,
	}
}

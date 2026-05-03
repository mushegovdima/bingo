package repository

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/uptrace/bun"
	"go.mod/internal/contracts/notification"
	"go.mod/internal/db"
)

type NotificationJobRepository struct {
	db              *bun.DB
	logger          *slog.Logger
	claimStaleAfter time.Duration
}

// NewNotificationJobRepository builds the outbox repository. claimStaleAfter is the
// visibility timeout: a job stuck in 'running' with locked_at older than this is
// treated as crashed and re-claimable.
func NewNotificationJobRepository(db *bun.DB, logger *slog.Logger, claimStaleAfter time.Duration) *NotificationJobRepository {
	return &NotificationJobRepository{db: db, logger: logger, claimStaleAfter: claimStaleAfter}
}

// Insert writes a job inside the caller's transaction. Pass r.db for non-transactional inserts;
// pass a *bun.Tx (which also satisfies bun.IDB) to atomically enqueue alongside business writes.
func (r *NotificationJobRepository) Insert(ctx context.Context, idb bun.IDB, job *db.NotificationJob) error {
	op := "db.notificationjobrepository.Insert"
	log := r.logger.With(slog.String("op", op), slog.String("type", job.Type))

	_, err := idb.NewInsert().Model(job).Returning("id").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	log.InfoContext(ctx, "notification job enqueued", slog.Int64("job_id", job.ID))
	return nil
}

// Enqueue persists a notification job built from the producer-facing contract request.
// It hides the underlying row shape from callers so producers (e.g. seasonservice) need not
// import dbmodels. The write participates in the caller's transaction via idb.
func (r *NotificationJobRepository) Enqueue(ctx context.Context, idb bun.IDB, req notification.EnqueueRequest) (int64, error) {
	op := "db.notificationjobrepository.Enqueue"
	log := r.logger.With(slog.String("op", op), slog.String("type", req.Type))

	filterJSON, err := json.Marshal(req.Filter)
	if err != nil {
		return 0, err
	}
	job := &db.NotificationJob{
		Type:   req.Type,
		Text:   req.Text,
		Args:   req.Args,
		Filter: filterJSON,
		Status: string(notification.JobStatusPending),
	}
	if _, err := idb.NewInsert().Model(job).Returning("id").Exec(ctx); err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return 0, err
	}
	log.InfoContext(ctx, "notification job enqueued", slog.Int64("job_id", job.ID))
	return job.ID, nil
}

// Claim atomically picks up to `limit` claimable jobs (pending, or running with an
// expired lease), marks them running, and returns them. The whole operation is a
// single UPDATE ... RETURNING — no explicit transaction, no row locks.
//
// Concurrency safety: the inner SELECT is evaluated once at planning time, so two
// workers racing on the same statement could read the same id set. To prevent both
// of them from acquiring the same row, the OUTER WHERE repeats the visibility
// predicate. Under READ COMMITTED, when the second UPDATE waits on a row lock taken
// by the first, Postgres re-evaluates the WHERE on the post-update tuple
// (EvalPlanQual). At that point status is already 'running' with a fresh locked_at,
// so the predicate fails and the row is skipped — exactly one worker wins per row.
// See: https://www.postgresql.org/docs/current/transaction-iso.html#XACT-READ-COMMITTED
//
// An empty slice means the queue is empty.
func (r *NotificationJobRepository) Claim(ctx context.Context, limit int) ([]*db.NotificationJob, error) {
	op := "db.notificationjobrepository.Claim"
	log := r.logger.With(slog.String("op", op))

	if limit <= 0 {
		limit = 1
	}

	staleBefore := time.Now().Add(-r.claimStaleAfter)
	var jobs []*db.NotificationJob
	err := r.db.NewRaw(`
		UPDATE notification_jobs
		SET status = 'running',
		    locked_at = now(),
		    updated_at = now()
		WHERE id IN (
			SELECT id FROM notification_jobs
			WHERE status = 'pending'
			   OR (status = 'running' AND locked_at < ?)
			ORDER BY created_at ASC
			LIMIT ?
		)
		  AND (status = 'pending' OR locked_at < ?)
		RETURNING *
	`, staleBefore, limit, staleBefore).Scan(ctx, &jobs)
	if err != nil {
		log.ErrorContext(ctx, "claim failed", slog.Any("error", err))
		return nil, err
	}
	return jobs, nil
}

// UpdateCursor checkpoints progress through the recipient list. Called after each successful batch.
func (r *NotificationJobRepository) UpdateCursor(ctx context.Context, id, cursor int64) error {
	op := "db.notificationjobrepository.UpdateCursor"
	log := r.logger.With(slog.String("op", op), slog.Int64("job_id", id))

	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*db.NotificationJob)(nil)).
		Set("cursor = ?", cursor).
		Set("locked_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

// MarkDone finalises a successful job.
func (r *NotificationJobRepository) MarkDone(ctx context.Context, id int64) error {
	op := "db.notificationjobrepository.MarkDone"
	log := r.logger.With(slog.String("op", op), slog.Int64("job_id", id))

	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*db.NotificationJob)(nil)).
		Set("status = ?", "done").
		Set("locked_at = NULL").
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	log.InfoContext(ctx, "notification job done")
	return nil
}

// MarkFailed records a processing failure. The caller decides whether to retry (status='pending')
// or give up (status='failed') based on attempt count.
func (r *NotificationJobRepository) MarkFailed(ctx context.Context, id int64, attempts int, errMsg string, terminal bool) error {
	op := "db.notificationjobrepository.MarkFailed"
	log := r.logger.With(slog.String("op", op), slog.Int64("job_id", id), slog.Bool("terminal", terminal))

	status := "pending"
	if terminal {
		status = "failed"
	}
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*db.NotificationJob)(nil)).
		Set("status = ?", status).
		Set("attempts = ?", attempts).
		Set("last_error = ?", errMsg).
		Set("locked_at = NULL").
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	log.WarnContext(ctx, "notification job failed", slog.String("error", errMsg))
	return nil
}

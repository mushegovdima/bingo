package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"notifier/internal/db/repository"
	"notifier/internal/sender"
)

type Config struct {
	BatchSize      int
	ReservationTTL time.Duration
}

type notificationRepo interface {
	FetchAndReserve(ctx context.Context, limit int, reservationTTL time.Duration) ([]repository.Notification, error)
	MarkProcessed(ctx context.Context, id string) error
}

type messageSender interface {
	Send(ctx context.Context, chatID int64, text string) error
}

type Worker struct {
	repo     notificationRepo
	sender   messageSender
	interval time.Duration
	cfg      Config
	logger   *slog.Logger
}

func New(repo notificationRepo, sender messageSender, interval time.Duration, cfg Config, logger *slog.Logger) *Worker {
	return &Worker{
		repo:     repo,
		sender:   sender,
		interval: interval,
		cfg:      cfg,
		logger:   logger,
	}
}

// Run starts the worker loop. Blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		for w.tick(ctx) {
			if ctx.Err() != nil {
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// tick processes one batch. Returns true if there may be more work to do.
func (w *Worker) tick(ctx context.Context) bool {
	ns, err := w.repo.FetchAndReserve(ctx, w.cfg.BatchSize, w.cfg.ReservationTTL)
	if err != nil {
		w.logger.ErrorContext(ctx, "worker: fetch failed", slog.Any("error", err))
		return false
	}
	if len(ns) == 0 {
		return false
	}

	w.logger.InfoContext(ctx, "worker: processing batch", slog.Int("count", len(ns)))

	for _, n := range ns {
		// Deadline = reservation TTL so sender can sleep up to RetryAfter within the window.
		// If we can't send before the reservation expires, another worker will pick it up.
		entryCtx, cancel := context.WithTimeout(ctx, w.cfg.ReservationTTL)
		err := w.sender.Send(entryCtx, n.TelegramID, n.Text)
		cancel()

		if err != nil {
			if errors.Is(err, sender.ErrPermanent) {
				// Permanent Telegram error (bot blocked, chat not found, etc.).
				// Mark as processed so it never gets retried.
				w.logger.WarnContext(ctx, "worker: permanent send error, skipping",
					slog.String("id", n.ID),
					slog.Int64("telegram_id", n.TelegramID),
					slog.Any("error", err),
				)
				if err := w.repo.MarkProcessed(ctx, n.ID); err != nil {
					w.logger.ErrorContext(ctx, "worker: mark processed (permanent) failed",
						slog.String("id", n.ID),
						slog.Any("error", err),
					)
				}
				continue
			}
			w.logger.ErrorContext(ctx, "worker: send failed",
				slog.String("id", n.ID),
				slog.Int64("telegram_id", n.TelegramID),
				slog.Any("error", err),
			)
			// reservation will expire → another worker will retry
			continue
		}

		if err := w.repo.MarkProcessed(ctx, n.ID); err != nil {
			w.logger.ErrorContext(ctx, "worker: mark processed failed",
				slog.String("id", n.ID),
				slog.Any("error", err),
			)
		}
	}

	// if we got a full batch, there's likely more
	return len(ns) == w.cfg.BatchSize
}

package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

type Notification struct {
	bun.BaseModel `bun:"table:notifications"`

	ID          string     `bun:"id,pk"`
	Type        string     `bun:"type,notnull"`
	UserID      int64      `bun:"user_id,notnull"`
	TelegramID  int64      `bun:"telegram_id,notnull"`
	Text        string     `bun:"text,notnull"`
	SendAt      time.Time  `bun:"send_at,notnull"`
	CreatedAt   time.Time  `bun:"created_at,notnull"`
	ReservedAt  *time.Time `bun:"reserved_at"`
	ProcessedAt *time.Time `bun:"processed_at"`
}

type NotificationRepository struct {
	db *bun.DB
}

func NewNotificationRepository(db *bun.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Insert saves a new notification. Silently ignores duplicates (idempotent).
func (r *NotificationRepository) Insert(ctx context.Context, n *Notification) error {
	_, err := r.db.NewInsert().
		Model(n).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("notifications.Insert: %w", err)
	}
	return nil
}

// BulkInsert saves multiple notifications in a single statement.
// Silently ignores duplicates (idempotent). No-op for empty slice.
func (r *NotificationRepository) BulkInsert(ctx context.Context, ns []Notification) error {
	if len(ns) == 0 {
		return nil
	}
	_, err := r.db.NewInsert().
		Model(&ns).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("notifications.BulkInsert: %w", err)
	}
	return nil
}

// FetchAndReserve atomically claims up to limit claimable notifications by setting
// reserved_at = now() + reservationTTL. Notifications whose reservation has expired
// (reserved_at < now()) are also eligible, enabling recovery from crashed workers.
func (r *NotificationRepository) FetchAndReserve(ctx context.Context, limit int, reservationTTL time.Duration) ([]Notification, error) {
	var ns []Notification
	err := r.db.NewRaw(`
		UPDATE notifications
		SET reserved_at = now() + ?::interval
		WHERE id IN (
			SELECT id FROM notifications
			WHERE processed_at IS NULL
			  AND send_at <= now()
			  AND (reserved_at IS NULL OR reserved_at < now())
			ORDER BY send_at
			LIMIT ?
		)
		RETURNING *
	`, reservationTTL.String(), limit).Scan(ctx, &ns)
	if err != nil {
		return nil, fmt.Errorf("notifications.FetchAndReserve: %w", err)
	}
	return ns, nil
}

// MarkProcessed sets processed_at for the given notification ID.
// reserved_at is intentionally kept: processed_at - reserved_at gives processing duration.
func (r *NotificationRepository) MarkProcessed(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		TableExpr("notifications").
		Set("processed_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("notifications.MarkProcessed: %w", err)
	}
	return nil
}

// DeleteOld removes processed notifications older than the given duration.
func (r *NotificationRepository) DeleteOld(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	_, err := r.db.NewDelete().
		TableExpr("notifications").
		Where("processed_at IS NOT NULL AND processed_at < ?", cutoff).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("notifications.DeleteOld: %w", err)
	}
	return nil
}

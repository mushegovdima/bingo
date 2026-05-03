// Package notificationservice exposes the business-side facade for the notification outbox.
// Producers (e.g. seasonservice on activation) call Enqueue inside their own transaction so
// the job is committed atomically with the originating business write. The worker that
// drains the outbox lives in internal/worker/notification.
package notificationservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/uptrace/bun"
	notificationcontract "go.mod/internal/contracts/notification"
	"go.mod/internal/notifications"
)

type jobRepo interface {
	Enqueue(ctx context.Context, idb bun.IDB, req notificationcontract.EnqueueRequest) (int64, error)
}

// templateLoader loads the raw (unrendered) body of a named message template.
// Implemented by templateservice.Service.
type templateLoader interface {
	Body(ctx context.Context, codename string) (string, error)
}

type Service struct {
	repo   jobRepo
	db     bun.IDB
	tmpl   templateLoader
	logger *slog.Logger
}

func NewService(repo jobRepo, db bun.IDB, tmpl templateLoader, logger *slog.Logger) *Service {
	return &Service{repo: repo, db: db, tmpl: tmpl, logger: logger}
}

// Enqueue inserts a notification job into the outbox using the provided IDB so the write
// participates in the caller's transaction. Pass *bun.Tx to atomically enqueue alongside
// business writes; pass *bun.DB for fire-and-forget enqueue outside a transaction.
func (s *Service) Enqueue(ctx context.Context, idb bun.IDB, req notificationcontract.EnqueueRequest) error {
	if req.Type == "" {
		return fmt.Errorf("notificationservice.Enqueue: type is required")
	}
	if req.Text == "" {
		return fmt.Errorf("notificationservice.Enqueue: text is required")
	}

	jobID, err := s.repo.Enqueue(ctx, idb, req)
	if err != nil {
		return fmt.Errorf("notificationservice.Enqueue: %w", err)
	}
	s.logger.InfoContext(ctx, "notification job enqueued",
		slog.Int64("job_id", jobID), slog.String("type", req.Type))
	return nil
}

// EnqueueDirect enqueues a notification job outside any transaction, using the service's
// own DB connection. Use this for fire-and-forget notifications in services that do not
// manage their own transactions.
func (s *Service) EnqueueDirect(ctx context.Context, req notificationcontract.EnqueueRequest) error {
	return s.Enqueue(ctx, s.db, req)
}

// EnqueueTemplate loads the raw template body and stores it together with the
// notification-specific placeholder values (ArgsOf) as a job in the outbox.
// {{User.*}} placeholders are left unresolved — the worker substitutes them
// per-recipient at processing time via notifications.Substitute.
func (s *Service) EnqueueTemplate(ctx context.Context, idb bun.IDB, n notifications.Notification, filter notificationcontract.UserFilter) error {
	body, err := s.tmpl.Body(ctx, n.Codename())
	if err != nil {
		return fmt.Errorf("notificationservice.EnqueueTemplate: load %q: %w", n.Codename(), err)
	}
	return s.Enqueue(ctx, idb, notificationcontract.EnqueueRequest{
		Type:   n.Codename(),
		Text:   body,
		Args:   n.Args(),
		Filter: filter,
	})
}

// EnqueueTemplateDirect renders the named template with data and enqueues fire-and-forget
// (outside any transaction). Returns an error if the template is not found or render fails.
func (s *Service) EnqueueTemplateDirect(ctx context.Context, n notifications.Notification, filter notificationcontract.UserFilter) error {
	return s.EnqueueTemplate(ctx, s.db, n, filter)
}

// Notify renders the notification's template and enqueues it fire-and-forget (outside any
// transaction). ExcludeBlocked and OnlyTelegram are always applied automatically.
// filter.UserIDs empty = broadcast to all active Telegram users.
func (s *Service) Notify(ctx context.Context, n notifications.Notification, filter notificationcontract.UserFilter) error {
	filter.ExcludeBlocked = true
	filter.OnlyTelegram = true
	return s.EnqueueTemplateDirect(ctx, n, filter)
}

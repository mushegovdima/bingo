// Package templateservice manages notification message templates.
// Templates are stored in the database and cached locally by codename.
// The Render method substitutes {{argKey}} placeholders in the body with
// caller-supplied data to produce the final message text.
package templateservice

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mod/internal/cache"
	templatedomain "go.mod/internal/domain/template"
	"go.mod/internal/notifications"
)

// ErrNotFound aliases the domain sentinel so callers can match via errors.Is
// without importing templatedomain.
var ErrNotFound = templatedomain.ErrNotFound

type templateRepo interface {
	GetByCodename(ctx context.Context, codename string) (*templatedomain.Template, error)
	List(ctx context.Context) ([]*templatedomain.Template, error)
	UpdateBody(ctx context.Context, codename, body string, changedBy int64) (*templatedomain.Template, error)
	ListHistory(ctx context.Context, codename string) ([]*templatedomain.TemplateHistory, error)
}

// Service manages message templates with an in-process cache keyed by codename.
type Service struct {
	repo   templateRepo
	cache  *cache.LocalCache[string, *templatedomain.Template]
	logger *slog.Logger
}

func NewService(repo templateRepo, logger *slog.Logger) *Service {
	c := cache.NewLocalCache[string, *templatedomain.Template](context.Background(), 5*time.Minute)
	return &Service{repo: repo, cache: c, logger: logger}
}

// Render fetches the template body by codename (from cache if available) and substitutes
// all {{FieldName}} placeholders with the corresponding field values of n, and
// all {{user.FieldName}} placeholders with fields from u (if u is not nil).
// Returns ErrNotFound if no template with that codename exists.
func (s *Service) Render(ctx context.Context, n notifications.Notification, u *notifications.NotificationUser) (string, error) {
	tpl, err := s.load(ctx, n.Codename())
	if err != nil {
		return "", err
	}

	args := n.Args()
	if u != nil {
		for k, v := range u.UserArgs() {
			args[k] = v
		}
	}
	return notifications.Substitute(tpl.Body, args), nil
}

// Body returns the raw (unrendered) template body for the given codename.
// It is used by notificationservice.EnqueueTemplate to store the template body
// alongside notification args in the outbox job.
func (s *Service) Body(ctx context.Context, codename string) (string, error) {
	tpl, err := s.load(ctx, codename)
	if err != nil {
		return "", err
	}
	return tpl.Body, nil
}

// GetByCodename returns the template metadata. Result is served from cache when available.
func (s *Service) GetByCodename(ctx context.Context, codename string) (*templatedomain.Template, error) {
	return s.load(ctx, codename)
}

// List returns all templates. Results are NOT cached (management-plane call).
func (s *Service) List(ctx context.Context) ([]*templatedomain.Template, error) {
	return s.repo.List(ctx)
}

// UpdateBody persists a new body for the template, appends a history record, and
// invalidates the cache entry.
func (s *Service) UpdateBody(ctx context.Context, codename string, changedBy int64, body string) (*templatedomain.Template, error) {
	op := "templateservice.UpdateBody"

	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("%s: body must not be empty", op)
	}

	result, err := s.repo.UpdateBody(ctx, codename, body, changedBy)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update template body",
			slog.String("codename", codename), slog.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	s.cache.Delete(codename)

	s.logger.InfoContext(ctx, "template body updated",
		slog.String("codename", codename), slog.Int64("changed_by", changedBy))
	return result, nil
}

// ListHistory returns the body-change history for a template, newest first.
func (s *Service) ListHistory(ctx context.Context, codename string) ([]*templatedomain.TemplateHistory, error) {
	return s.repo.ListHistory(ctx, codename)
}

// load returns the cached template for codename, loading from the DB on cache miss.
func (s *Service) load(ctx context.Context, codename string) (*templatedomain.Template, error) {
	if tpl, ok := s.cache.Get(codename); ok {
		return tpl, nil
	}

	tpl, err := s.repo.GetByCodename(ctx, codename)
	if err != nil {
		return nil, fmt.Errorf("templateservice.load: %w", err)
	}
	if tpl == nil {
		return nil, ErrNotFound
	}

	s.cache.Set(codename, tpl)
	return tpl, nil
}

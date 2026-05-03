package seasonservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"
	notification "go.mod/internal/contracts/notification"
	seasondomain "go.mod/internal/domain/season"
	"go.mod/internal/notifications"
)

// Sentinels alias the domain values so callers can match via errors.Is
// without importing seasondomain.
var (
	ErrNotFound     = seasondomain.ErrNotFound
	ErrHasRelations = seasondomain.ErrHasRelations
)

type seasonRepo interface {
	Insert(ctx context.Context, idb bun.IDB, c *seasondomain.Season) error
	Update(ctx context.Context, idb bun.IDB, c *seasondomain.Season, columns ...string) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*seasondomain.Season, error)
	GetActive(ctx context.Context) (*seasondomain.Season, error)
	List(ctx context.Context) ([]*seasondomain.Season, error)
	ListActive(ctx context.Context) ([]*seasondomain.Season, error)
}

// notificationEnqueuer enqueues a fan-out notification job inside the caller's transaction.
// SeasonService uses this to atomically commit a "season available" job alongside the
// season write — the worker handles delivery asynchronously.
type notificationEnqueuer interface {
	Notify(ctx context.Context, n notifications.Notification, filter notification.UserFilter) error
}

type txRunner interface {
	RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context, tx bun.Tx) error) error
}

type SeasonService struct {
	repo          seasonRepo
	tx            txRunner
	notifications notificationEnqueuer
	logger        *slog.Logger
}

func NewService(repo seasonRepo, tx txRunner, notifications notificationEnqueuer, logger *slog.Logger) *SeasonService {
	return &SeasonService{repo: repo, tx: tx, notifications: notifications, logger: logger}
}

// CreateRequest — Data to create a new season.
type CreateRequest struct {
	Title     string    `json:"title"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsActive  bool      `json:"is_active"`
}

// UpdateRequest — Partial update of a season (nil fields are ignored).
type UpdateRequest struct {
	Title     *string    `json:"title"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	IsActive  *bool      `json:"is_active"`
}

func (s *SeasonService) Create(ctx context.Context, req *CreateRequest) (*seasondomain.Season, error) {
	op := "seasonservice.Create"
	log := s.logger.With(slog.String("op", op))

	c := &seasondomain.Season{
		Title:     req.Title,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		IsActive:  req.IsActive,
	}

	if err := s.tx.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := s.repo.Insert(ctx, tx, c); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.ErrorContext(ctx, "failed to create season", slog.Any("error", err))
		return nil, err
	}

	if c.IsActive {
		if err := s.enqueueSeasonAvailable(ctx, c); err != nil {
			log.WarnContext(ctx, "season_available notification skipped", slog.Any("error", err))
		}
	}

	return c, nil
}

func (s *SeasonService) Update(ctx context.Context, id int64, req *UpdateRequest) (*seasondomain.Season, error) {
	op := "seasonservice.Update"
	log := s.logger.With(slog.String("op", op), slog.Int64("season_id", id))

	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get season", slog.Any("error", err))
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}

	var columns []string
	wasActive := c.IsActive

	if req.Title != nil {
		c.Title = *req.Title
		columns = append(columns, "title")
	}
	if req.StartDate != nil {
		c.StartDate = *req.StartDate
		columns = append(columns, "start_date")
	}
	if req.EndDate != nil {
		c.EndDate = *req.EndDate
		columns = append(columns, "end_date")
	}
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
		columns = append(columns, "is_active")
	}
	activated := !wasActive && c.IsActive

	if len(columns) == 0 {
		return c, nil
	}

	if err := s.tx.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := s.repo.Update(ctx, tx, c, columns...); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.ErrorContext(ctx, "failed to update season", slog.Any("error", err))
		return nil, err
	}

	if activated {
		if err := s.enqueueSeasonAvailable(ctx, c); err != nil {
			log.WarnContext(ctx, "season_available notification skipped", slog.Any("error", err))
		}
	}

	return c, nil
}

func (s *SeasonService) Delete(ctx context.Context, id int64) error {
	op := "seasonservice.Delete"
	log := s.logger.With(slog.String("op", op), slog.Int64("season_id", id))

	err := s.repo.Delete(ctx, id)
	if err == nil {
		return nil
	}

	// Postgres FK violation — code 23503
	var pgErr pgdriver.Error
	if errors.As(err, &pgErr) && pgErr.Field('C') == "23503" {
		return ErrHasRelations
	}

	log.ErrorContext(ctx, "failed to delete season", slog.Any("error", err))
	return fmt.Errorf("seasonservice.Delete: %w", err)
}

func (s *SeasonService) GetActive(ctx context.Context) (*seasondomain.Season, error) {
	op := "seasonservice.GetActive"
	log := s.logger.With(slog.String("op", op))

	c, err := s.repo.GetActive(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to get active season", slog.Any("error", err))
		return nil, err
	}
	return c, nil
}

func (s *SeasonService) List(ctx context.Context) ([]*seasondomain.Season, error) {
	op := "seasonservice.List"
	log := s.logger.With(slog.String("op", op))

	list, err := s.repo.List(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to list seasons", slog.Any("error", err))
		return nil, err
	}
	return list, nil
}

func (s *SeasonService) GetByID(ctx context.Context, id int64) (*seasondomain.Season, error) {
	op := "seasonservice.GetByID"
	log := s.logger.With(slog.String("op", op), slog.Int64("season_id", id))

	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get season", slog.Any("error", err))
		return nil, err
	}
	return c, nil
}

func (s *SeasonService) ListActive(ctx context.Context) ([]*seasondomain.Season, error) {
	op := "seasonservice.ListActive"
	log := s.logger.With(slog.String("op", op))

	list, err := s.repo.ListActive(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to list active seasons", slog.Any("error", err))
		return nil, err
	}
	return list, nil
}

// enqueueSeasonAvailable renders the season_available template and broadcasts it to all active Telegram users.
func (s *SeasonService) enqueueSeasonAvailable(ctx context.Context, c *seasondomain.Season) error {
	return s.notifications.Notify(ctx,
		notifications.SeasonAvailable{
			Title:     c.Title,
			StartDate: c.StartDate.Format("02.01.2006"),
			EndDate:   c.EndDate.Format("02.01.2006"),
		},
		notification.UserFilter{},
	)
}

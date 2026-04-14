package seasonservice

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/uptrace/bun/driver/pgdriver"
	dbmodels "go.mod/internal/db"
	"go.mod/internal/domain"
)

// ErrHasRelations возвращается когда кампания не может быть удалена из-за связанных записей.
var ErrHasRelations = errors.New("season has related records and cannot be deleted")

// ErrNotFound возвращается когда кампания не найдена.
var ErrNotFound = errors.New("season not found")

type seasonRepo interface {
	Insert(ctx context.Context, c *dbmodels.Season) error
	Update(ctx context.Context, c *dbmodels.Season, columns ...string) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*dbmodels.Season, error)
	GetActive(ctx context.Context) (*dbmodels.Season, error)
	List(ctx context.Context) ([]*dbmodels.Season, error)
	ListActive(ctx context.Context) ([]*dbmodels.Season, error)
}

type SeasonService struct {
	repo   seasonRepo
	logger *slog.Logger
}

func NewService(repo seasonRepo, logger *slog.Logger) *SeasonService {
	return &SeasonService{repo: repo, logger: logger}
}

// CreateRequest — данные для создания кампании.
type CreateRequest struct {
	Title     string    `json:"title"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsActive  bool      `json:"is_active"`
}

// UpdateRequest — частичное обновление кампании (nil-поля игнорируются).
type UpdateRequest struct {
	Title     *string    `json:"title"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	IsActive  *bool      `json:"is_active"`
}

func (s *SeasonService) Create(ctx context.Context, req *CreateRequest) (*domain.Season, error) {
	op := "seasonservice.Create"
	log := s.logger.With(slog.String("op", op))

	c := &dbmodels.Season{
		Title:     req.Title,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		IsActive:  req.IsActive,
	}
	if err := s.repo.Insert(ctx, c); err != nil {
		log.ErrorContext(ctx, "failed to insert season", slog.Any("error", err))
		return nil, err
	}
	return toDomain(c), nil
}

func (s *SeasonService) Update(ctx context.Context, id int64, req *UpdateRequest) (*domain.Season, error) {
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

	if len(columns) == 0 {
		return toDomain(c), nil
	}

	if err := s.repo.Update(ctx, c, columns...); err != nil {
		log.ErrorContext(ctx, "failed to update season", slog.Any("error", err))
		return nil, err
	}
	return toDomain(c), nil
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
	return err
}

func (s *SeasonService) GetActive(ctx context.Context) (*domain.Season, error) {
	op := "seasonservice.GetActive"
	log := s.logger.With(slog.String("op", op))

	c, err := s.repo.GetActive(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to get active season", slog.Any("error", err))
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return toDomain(c), nil
}

func (s *SeasonService) List(ctx context.Context) ([]*domain.Season, error) {
	op := "seasonservice.List"
	log := s.logger.With(slog.String("op", op))

	list, err := s.repo.List(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to list seasons", slog.Any("error", err))
		return nil, err
	}

	result := make([]*domain.Season, len(list))
	for i, c := range list {
		result[i] = toDomain(c)
	}
	return result, nil
}

func (s *SeasonService) GetByID(ctx context.Context, id int64) (*domain.Season, error) {
	op := "seasonservice.GetByID"
	log := s.logger.With(slog.String("op", op), slog.Int64("season_id", id))

	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get season", slog.Any("error", err))
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return toDomain(c), nil
}

func (s *SeasonService) ListActive(ctx context.Context) ([]*domain.Season, error) {
	op := "seasonservice.ListActive"
	log := s.logger.With(slog.String("op", op))

	list, err := s.repo.ListActive(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to list active seasons", slog.Any("error", err))
		return nil, err
	}
	result := make([]*domain.Season, len(list))
	for i, c := range list {
		result[i] = toDomain(c)
	}
	return result, nil
}

func toDomain(c *dbmodels.Season) *domain.Season {
	return &domain.Season{
		ID:        c.ID,
		Title:     c.Title,
		StartDate: c.StartDate,
		EndDate:   c.EndDate,
		IsActive:  c.IsActive,
	}
}

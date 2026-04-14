package taskservice

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/uptrace/bun/driver/pgdriver"
	dbmodels "go.mod/internal/db"
	"go.mod/internal/domain"
)

var (
	ErrNotFound     = errors.New("task not found")
	ErrHasRelations = errors.New("task has related records")
)

type taskRepo interface {
	Insert(ctx context.Context, t *dbmodels.Task) error
	Update(ctx context.Context, t *dbmodels.Task, columns ...string) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*dbmodels.Task, error)
	ListBySeason(ctx context.Context, seasonID int64) ([]dbmodels.Task, error)
}

type TaskService struct {
	repo   taskRepo
	logger *slog.Logger
}

func NewService(repo taskRepo, logger *slog.Logger) *TaskService {
	return &TaskService{repo: repo, logger: logger}
}

type CreateRequest struct {
	SeasonID  int64           `json:"season_id"`
	Category    string          `json:"category"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	RewardCoins int             `json:"reward_coins"`
	SortOrder   int             `json:"sort_order"`
	Metadata    json.RawMessage `json:"metadata"`
	IsActive    bool            `json:"is_active"`
}

type UpdateRequest struct {
	Category    *string          `json:"category"`
	Title       *string          `json:"title"`
	Description *string          `json:"description"`
	RewardCoins *int             `json:"reward_coins"`
	SortOrder   *int             `json:"sort_order"`
	Metadata    *json.RawMessage `json:"metadata"`
	IsActive    *bool            `json:"is_active"`
}

func (s *TaskService) Create(ctx context.Context, req *CreateRequest) (*domain.Task, error) {
	op := "taskservice.Create"
	log := s.logger.With(slog.String("op", op))

	metadata := req.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}

	t := &dbmodels.Task{
		SeasonID:  req.SeasonID,
		Category:    req.Category,
		Title:       req.Title,
		Description: req.Description,
		RewardCoins: req.RewardCoins,
		SortOrder:   req.SortOrder,
		Metadata:    metadata,
		IsActive:    req.IsActive,
	}

	if err := s.repo.Insert(ctx, t); err != nil {
		log.ErrorContext(ctx, "failed to insert task", slog.Any("error", err))
		return nil, err
	}
	return toDomain(t), nil
}

func (s *TaskService) Update(ctx context.Context, id int64, req *UpdateRequest) (*domain.Task, error) {
	op := "taskservice.Update"
	log := s.logger.With(slog.String("op", op), slog.Int64("task_id", id))

	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get task", slog.Any("error", err))
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}

	var columns []string
	if req.Category != nil {
		t.Category = *req.Category
		columns = append(columns, "category")
	}
	if req.Title != nil {
		t.Title = *req.Title
		columns = append(columns, "title")
	}
	if req.Description != nil {
		t.Description = *req.Description
		columns = append(columns, "description")
	}
	if req.RewardCoins != nil {
		t.RewardCoins = *req.RewardCoins
		columns = append(columns, "reward_coins")
	}
	if req.SortOrder != nil {
		t.SortOrder = *req.SortOrder
		columns = append(columns, "sort_order")
	}
	if req.Metadata != nil {
		t.Metadata = *req.Metadata
		columns = append(columns, "metadata")
	}
	if req.IsActive != nil {
		t.IsActive = *req.IsActive
		columns = append(columns, "is_active")
	}

	if len(columns) > 0 {
		if err := s.repo.Update(ctx, t, columns...); err != nil {
			log.ErrorContext(ctx, "failed to update task", slog.Any("error", err))
			return nil, err
		}
	}
	return toDomain(t), nil
}

func (s *TaskService) Delete(ctx context.Context, id int64) error {
	op := "taskservice.Delete"
	log := s.logger.With(slog.String("op", op), slog.Int64("task_id", id))

	if err := s.repo.Delete(ctx, id); err != nil {
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "23503" {
			return ErrHasRelations
		}
		log.ErrorContext(ctx, "failed to delete task", slog.Any("error", err))
		return err
	}
	return nil
}

func (s *TaskService) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	op := "taskservice.GetByID"
	log := s.logger.With(slog.String("op", op), slog.Int64("task_id", id))

	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get task", slog.Any("error", err))
		return nil, err
	}
	if t == nil {
		return nil, nil
	}
	return toDomain(t), nil
}

func (s *TaskService) ListBySeason(ctx context.Context, seasonID int64) ([]domain.Task, error) {
	op := "taskservice.ListBySeason"
	log := s.logger.With(slog.String("op", op), slog.Int64("season_id", seasonID))

	tasks, err := s.repo.ListBySeason(ctx, seasonID)
	if err != nil {
		log.ErrorContext(ctx, "failed to list tasks", slog.Any("error", err))
		return nil, err
	}
	return toDomainSlice(tasks), nil
}

func toDomain(t *dbmodels.Task) *domain.Task {
	return &domain.Task{
		ID:          t.ID,
		SeasonID:  t.SeasonID,
		Category:    t.Category,
		Title:       t.Title,
		Description: t.Description,
		RewardCoins: t.RewardCoins,
		SortOrder:   t.SortOrder,
		Metadata:    t.Metadata,
		IsActive:    t.IsActive,
	}
}

func toDomainSlice(tasks []dbmodels.Task) []domain.Task {
	result := make([]domain.Task, len(tasks))
	for i := range tasks {
		result[i] = *toDomain(&tasks[i])
	}
	return result
}

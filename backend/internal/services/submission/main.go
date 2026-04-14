package submissionservice

import (
	"context"
	"errors"
	"log/slog"
	"time"

	dbmodels "go.mod/internal/db"
	"go.mod/internal/domain"
	balanceservice "go.mod/internal/services/balance"
)

var ErrNotFound = errors.New("submission not found")

type submissionRepo interface {
	Insert(ctx context.Context, s *dbmodels.TaskSubmission) error
	Update(ctx context.Context, s *dbmodels.TaskSubmission, columns ...string) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*dbmodels.TaskSubmission, error)
	ListByUser(ctx context.Context, userID int64) ([]dbmodels.TaskSubmission, error)
	ListAll(ctx context.Context) ([]dbmodels.TaskSubmission, error)
}

type taskFinder interface {
	GetByID(ctx context.Context, id int64) (*domain.Task, error)
}

type coinsAccruer interface {
	AddCoins(ctx context.Context, req balanceservice.AddCoinsRequest) (*domain.Transaction, error)
}

type SubmissionService struct {
	repo        submissionRepo
	taskService taskFinder
	balanceSvc  coinsAccruer
	logger      *slog.Logger
}

func NewService(repo submissionRepo, taskService taskFinder, balanceSvc coinsAccruer, logger *slog.Logger) *SubmissionService {
	return &SubmissionService{
		repo:        repo,
		taskService: taskService,
		balanceSvc:  balanceSvc,
		logger:      logger,
	}
}

// CreateRequest for manager-created (pre-approved) task submission.
type CreateRequest struct {
	UserID     int64 `json:"user_id"`
	TaskID     int64 `json:"task_id"`
	SeasonID int64 `json:"season_id"`
}

// Create creates a task submission with status=approved and credits the task reward to the user.
// Only managers should call this endpoint.
func (s *SubmissionService) Create(ctx context.Context, reviewerID int64, req *CreateRequest) (*domain.TaskSubmission, error) {
	op := "submissionservice.Create"
	log := s.logger.With(slog.String("op", op), slog.Int64("task_id", req.TaskID), slog.Int64("user_id", req.UserID))

	task, err := s.taskService.GetByID(ctx, req.TaskID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get task", slog.Any("error", err))
		return nil, err
	}
	if task == nil {
		return nil, errors.New("task not found")
	}

	now := time.Now()
	sub := &dbmodels.TaskSubmission{
		UserID:     req.UserID,
		TaskID:     req.TaskID,
		Status:     domain.SubmissionApproved,
		ReviewerID: &reviewerID,
		ReviewedAt: &now,
	}
	if err := s.repo.Insert(ctx, sub); err != nil {
		log.ErrorContext(ctx, "failed to insert submission", slog.Any("error", err))
		return nil, err
	}

	// Credit coins to the user for the approved task.
	if task.RewardCoins > 0 {
		taskID := task.ID
		if _, err := s.balanceSvc.AddCoins(ctx, balanceservice.AddCoinsRequest{
			UserID:     req.UserID,
			SeasonID: req.SeasonID,
			Amount:     task.RewardCoins,
			Reason:     domain.TransactionReasonTask,
			RefID:      &taskID,
			RefTitle:   task.Title,
		}); err != nil {
			log.ErrorContext(ctx, "failed to credit task coins", slog.Any("error", err))
		}
	}

	log.InfoContext(ctx, "submission created and approved", slog.Int64("submission_id", sub.ID))
	return toDomain(sub), nil
}

func (s *SubmissionService) GetByID(ctx context.Context, id int64) (*domain.TaskSubmission, error) {
	op := "submissionservice.GetByID"
	log := s.logger.With(slog.String("op", op), slog.Int64("submission_id", id))

	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get submission", slog.Any("error", err))
		return nil, err
	}
	if sub == nil {
		return nil, nil
	}
	return toDomain(sub), nil
}

func (s *SubmissionService) ListByUser(ctx context.Context, userID int64) ([]domain.TaskSubmission, error) {
	op := "submissionservice.ListByUser"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	items, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		log.ErrorContext(ctx, "failed to list submissions", slog.Any("error", err))
		return nil, err
	}
	return toDomainSlice(items), nil
}

func (s *SubmissionService) ListAll(ctx context.Context) ([]domain.TaskSubmission, error) {
	op := "submissionservice.ListAll"
	log := s.logger.With(slog.String("op", op))

	items, err := s.repo.ListAll(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to list all submissions", slog.Any("error", err))
		return nil, err
	}
	return toDomainSlice(items), nil
}

func (s *SubmissionService) Delete(ctx context.Context, id int64) error {
	op := "submissionservice.Delete"
	log := s.logger.With(slog.String("op", op), slog.Int64("submission_id", id))

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get submission", slog.Any("error", err))
		return err
	}
	if existing == nil {
		return ErrNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		log.ErrorContext(ctx, "failed to delete submission", slog.Any("error", err))
		return err
	}
	return nil
}

func toDomain(s *dbmodels.TaskSubmission) *domain.TaskSubmission {
	return &domain.TaskSubmission{
		ID:            s.ID,
		UserID:        s.UserID,
		TaskID:        s.TaskID,
		Status:        s.Status,
		ReviewComment: s.ReviewComment,
		ReviewerID:    s.ReviewerID,
		SubmittedAt:   s.SubmittedAt,
		ReviewedAt:    s.ReviewedAt,
	}
}

func toDomainSlice(items []dbmodels.TaskSubmission) []domain.TaskSubmission {
	result := make([]domain.TaskSubmission, len(items))
	for i := range items {
		result[i] = *toDomain(&items[i])
	}
	return result
}

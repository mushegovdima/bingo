package submissionservice

import (
	"context"
	"log/slog"
	"time"

	notification "go.mod/internal/contracts/notification"
	wallet "go.mod/internal/contracts/wallet"
	submissiondomain "go.mod/internal/domain/submission"
	taskdomain "go.mod/internal/domain/task"
	walletdomain "go.mod/internal/domain/wallet"
	"go.mod/internal/notifications"
)

// ErrNotFound aliases the domain sentinel so callers match via errors.Is
// without importing submissiondomain.
var ErrNotFound = submissiondomain.ErrNotFound

type submissionRepo interface {
	Insert(ctx context.Context, s *submissiondomain.TaskSubmission) error
	Update(ctx context.Context, s *submissiondomain.TaskSubmission, columns ...string) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*submissiondomain.TaskSubmission, error)
	GetByUserAndTask(ctx context.Context, userID, taskID int64) (*submissiondomain.TaskSubmission, error)
	ListByUser(ctx context.Context, userID int64) ([]submissiondomain.TaskSubmission, error)
	ListAll(ctx context.Context) ([]submissiondomain.TaskSubmission, error)
}

type taskFinder interface {
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
}

type coinsAccruer interface {
	AddCoins(ctx context.Context, req wallet.CreditRequest) (*walletdomain.Transaction, error)
}

type notifier interface {
	Notify(ctx context.Context, n notifications.Notification, filter notification.UserFilter) error
}

type SubmissionService struct {
	repo          submissionRepo
	taskService   taskFinder
	balanceSvc    coinsAccruer
	notifications notifier
	logger        *slog.Logger
}

func NewService(repo submissionRepo, taskService taskFinder, balanceSvc coinsAccruer, notifications notifier, logger *slog.Logger) *SubmissionService {
	return &SubmissionService{
		repo:          repo,
		taskService:   taskService,
		balanceSvc:    balanceSvc,
		notifications: notifications,
		logger:        logger,
	}
}

// CreateRequest for manager-created (pre-approved) task submission.
type CreateRequest struct {
	UserID   int64 `json:"user_id"`
	TaskID   int64 `json:"task_id"`
	SeasonID int64 `json:"season_id"`
}

// Create creates a task submission with status=approved and credits the task reward to the user.
// Only managers should call this endpoint.
func (s *SubmissionService) Create(ctx context.Context, reviewerID int64, req *CreateRequest) (*submissiondomain.TaskSubmission, error) {
	op := "submissionservice.Create"
	log := s.logger.With(slog.String("op", op), slog.Int64("task_id", req.TaskID), slog.Int64("user_id", req.UserID))

	task, err := s.taskService.GetByID(ctx, req.TaskID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get task", slog.Any("error", err))
		return nil, err
	}
	if task == nil {
		return nil, taskdomain.ErrNotFound
	}

	now := time.Now()
	sub := &submissiondomain.TaskSubmission{
		UserID:     req.UserID,
		TaskID:     req.TaskID,
		Status:     submissiondomain.SubmissionApproved,
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
		if _, err := s.balanceSvc.AddCoins(ctx, wallet.CreditRequest{
			UserID:   req.UserID,
			SeasonID: req.SeasonID,
			Amount:   task.RewardCoins,
			Reason:   walletdomain.TransactionReasonTask,
			RefID:    &taskID,
			RefTitle: task.Title,
		}); err != nil {
			log.ErrorContext(ctx, "failed to credit task coins", slog.Any("error", err))
		}
	}

	if err := s.notifications.Notify(ctx,
		notifications.TaskApproved{TaskTitle: task.Title, Coins: task.RewardCoins},
		notification.UserFilter{UserIDs: []int64{req.UserID}},
	); err != nil {
		log.WarnContext(ctx, "failed to enqueue task_approved notification", slog.Any("error", err))
	}

	log.InfoContext(ctx, "submission created and approved", slog.Int64("submission_id", sub.ID))
	return sub, nil
}

func (s *SubmissionService) GetByID(ctx context.Context, id int64) (*submissiondomain.TaskSubmission, error) {
	op := "submissionservice.GetByID"
	log := s.logger.With(slog.String("op", op), slog.Int64("submission_id", id))

	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get submission", slog.Any("error", err))
		return nil, err
	}
	return sub, nil
}

func (s *SubmissionService) ListByUser(ctx context.Context, userID int64) ([]submissiondomain.TaskSubmission, error) {
	op := "submissionservice.ListByUser"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	items, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		log.ErrorContext(ctx, "failed to list submissions", slog.Any("error", err))
		return nil, err
	}
	return items, nil
}

func (s *SubmissionService) ListAll(ctx context.Context) ([]submissiondomain.TaskSubmission, error) {
	op := "submissionservice.ListAll"
	log := s.logger.With(slog.String("op", op))

	items, err := s.repo.ListAll(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to list all submissions", slog.Any("error", err))
		return nil, err
	}
	return items, nil
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

// (no notification builders — rendering is handled by notificationservice)

// ErrAlreadySubmitted means the resident already has a submission for this task.
var ErrAlreadySubmitted = submissiondomain.ErrAlreadySubmitted

// SubmitRequest for resident-created (pending) task submission.
type SubmitRequest struct {
	TaskID  int64  `json:"task_id"`
	Comment string `json:"comment"`
}

// Submit creates a TaskSubmission with status=pending on behalf of a resident.
// Returns ErrAlreadySubmitted if the resident already has a non-rejected submission for this task.
func (s *SubmissionService) Submit(ctx context.Context, userID int64, req *SubmitRequest) (*submissiondomain.TaskSubmission, error) {
	op := "submissionservice.Submit"
	log := s.logger.With(slog.String("op", op), slog.Int64("task_id", req.TaskID), slog.Int64("user_id", userID))

	if req.Comment == "" {
		return nil, submissiondomain.ErrEmptyComment
	}

	existing, err := s.repo.GetByUserAndTask(ctx, userID, req.TaskID)
	if err != nil {
		log.ErrorContext(ctx, "failed to check existing submission", slog.Any("error", err))
		return nil, err
	}
	if existing != nil && existing.Status != submissiondomain.SubmissionRejected {
		return nil, ErrAlreadySubmitted
	}

	sub := submissiondomain.NewPending(userID, req.TaskID, req.Comment, time.Now())
	if err := s.repo.Insert(ctx, sub); err != nil {
		log.ErrorContext(ctx, "failed to insert submission", slog.Any("error", err))
		return nil, err
	}

	log.InfoContext(ctx, "submission created (pending)", slog.Int64("submission_id", sub.ID))
	return sub, nil
}

// Approve transitions a pending submission to approved, credits coins, and notifies the user.
func (s *SubmissionService) Approve(ctx context.Context, reviewerID, submissionID int64) (*submissiondomain.TaskSubmission, error) {
	op := "submissionservice.Approve"
	log := s.logger.With(slog.String("op", op), slog.Int64("submission_id", submissionID))

	sub, err := s.repo.GetByID(ctx, submissionID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get submission", slog.Any("error", err))
		return nil, err
	}
	if sub == nil {
		return nil, ErrNotFound
	}

	if err := sub.Approve(reviewerID, time.Now()); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, sub, "status", "reviewer_id", "reviewed_at", "review_comment"); err != nil {
		log.ErrorContext(ctx, "failed to update submission", slog.Any("error", err))
		return nil, err
	}

	// Credit coins — look up task to get season_id and reward amount.
	task, taskErr := s.taskService.GetByID(ctx, sub.TaskID)
	if taskErr != nil {
		log.WarnContext(ctx, "failed to get task for coin credit", slog.Any("error", taskErr))
	}

	taskTitle := ""
	coins := 0
	if task != nil {
		taskTitle = task.Title
		coins = task.RewardCoins
		if task.RewardCoins > 0 {
			taskID := task.ID
			if _, err := s.balanceSvc.AddCoins(ctx, wallet.CreditRequest{
				UserID:   sub.UserID,
				SeasonID: task.SeasonID,
				Amount:   task.RewardCoins,
				Reason:   walletdomain.TransactionReasonTask,
				RefID:    &taskID,
				RefTitle: task.Title,
			}); err != nil {
				log.WarnContext(ctx, "failed to credit task coins", slog.Any("error", err))
			}
		}
	}

	if err := s.notifications.Notify(ctx,
		notifications.TaskApproved{TaskTitle: taskTitle, Coins: coins},
		notification.UserFilter{UserIDs: []int64{sub.UserID}},
	); err != nil {
		log.WarnContext(ctx, "failed to enqueue task_approved notification", slog.Any("error", err))
	}

	log.InfoContext(ctx, "submission approved", slog.Int64("submission_id", sub.ID))
	return sub, nil
}

// Reject transitions a pending submission to rejected and notifies the user.
func (s *SubmissionService) Reject(ctx context.Context, reviewerID, submissionID int64, comment string) (*submissiondomain.TaskSubmission, error) {
	op := "submissionservice.Reject"
	log := s.logger.With(slog.String("op", op), slog.Int64("submission_id", submissionID))

	sub, err := s.repo.GetByID(ctx, submissionID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get submission", slog.Any("error", err))
		return nil, err
	}
	if sub == nil {
		return nil, ErrNotFound
	}

	if err := sub.Reject(reviewerID, comment, time.Now()); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, sub, "status", "reviewer_id", "reviewed_at", "review_comment"); err != nil {
		log.ErrorContext(ctx, "failed to update submission", slog.Any("error", err))
		return nil, err
	}

	// Get task title for notification.
	taskTitle := ""
	if task, _ := s.taskService.GetByID(ctx, sub.TaskID); task != nil {
		taskTitle = task.Title
	}

	if err := s.notifications.Notify(ctx,
		notifications.TaskRejected{TaskTitle: taskTitle, ReviewComment: comment},
		notification.UserFilter{UserIDs: []int64{sub.UserID}},
	); err != nil {
		log.WarnContext(ctx, "failed to enqueue task_rejected notification", slog.Any("error", err))
	}

	log.InfoContext(ctx, "submission rejected", slog.Int64("submission_id", sub.ID))
	return sub, nil
}

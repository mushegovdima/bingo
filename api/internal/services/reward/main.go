package rewardservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/uptrace/bun/driver/pgdriver"
	notification "go.mod/internal/contracts/notification"
	wallet "go.mod/internal/contracts/wallet"
	rewarddomain "go.mod/internal/domain/reward"
	walletdomain "go.mod/internal/domain/wallet"
	"go.mod/internal/notifications"
)

// Sentinel errors. Domain invariants live in rewarddomain; this package
// re-exports them as aliases so callers can match via errors.Is without
// importing the domain package transitively.
var (
	ErrNotFound            = rewarddomain.ErrNotFound
	ErrClaimNotFound       = rewarddomain.ErrClaimNotFound
	ErrRewardUnavailable   = rewarddomain.ErrRewardUnavailable
	ErrLimitExceeded       = rewarddomain.ErrLimitExceeded
	ErrInsufficientBalance = wallet.ErrInsufficientBalance
	// ErrHasRelations is a persistence-translation error (FK violation on Delete);
	// it is not a domain invariant and stays here.
	ErrHasRelations = errors.New("reward has related records and cannot be deleted")
)

type rewardRepo interface {
	Insert(ctx context.Context, rw *rewarddomain.Reward) error
	Update(ctx context.Context, rw *rewarddomain.Reward, columns ...string) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*rewarddomain.Reward, error)
	ListBySeason(ctx context.Context, seasonID int64) ([]rewarddomain.Reward, error)
	InsertClaim(ctx context.Context, c *rewarddomain.RewardClaim) error
	UpdateClaim(ctx context.Context, c *rewarddomain.RewardClaim, columns ...string) error
	GetClaimByID(ctx context.Context, id int64) (*rewarddomain.RewardClaim, error)
	ListClaimsByUser(ctx context.Context, userID int64) ([]rewarddomain.RewardClaim, error)
	ListAllClaims(ctx context.Context) ([]rewarddomain.RewardClaim, error)
	CountActiveClaims(ctx context.Context, rewardID int64) (int, error)
}

type coinsSpender interface {
	SpendCoins(ctx context.Context, req wallet.DebitRequest) (*walletdomain.Transaction, error)
	RefundCoins(ctx context.Context, req wallet.DebitRequest) (*walletdomain.Transaction, error)
}

type notifier interface {
	Notify(ctx context.Context, n notifications.Notification, filter notification.UserFilter) error
}

type RewardService struct {
	repo          rewardRepo
	balanceSvc    coinsSpender
	notifications notifier
	logger        *slog.Logger
}

func NewService(repo rewardRepo, balanceSvc coinsSpender, notifications notifier, logger *slog.Logger) *RewardService {
	return &RewardService{repo: repo, balanceSvc: balanceSvc, notifications: notifications, logger: logger}
}

// --- Reward CRUD ---

type CreateRewardRequest struct {
	SeasonID    int64                     `json:"season_id"`
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	CostCoins   int                       `json:"cost_coins"`
	Limit       *int                      `json:"limit"`
	Status      rewarddomain.RewardStatus `json:"status"`
}

type UpdateRewardRequest struct {
	Title       *string                    `json:"title"`
	Description *string                    `json:"description"`
	CostCoins   *int                       `json:"cost_coins"`
	Limit       **int                      `json:"limit"`
	Status      *rewarddomain.RewardStatus `json:"status"`
}

func (s *RewardService) Create(ctx context.Context, req *CreateRewardRequest) (*rewarddomain.Reward, error) {
	op := "rewardservice.Create"
	log := s.logger.With(slog.String("op", op))

	status := req.Status
	if status == "" {
		status = rewarddomain.RewardAvailable
	}

	rw := &rewarddomain.Reward{
		SeasonID:    req.SeasonID,
		Title:       req.Title,
		Description: req.Description,
		CostCoins:   req.CostCoins,
		Limit:       req.Limit,
		Status:      status,
	}
	if err := s.repo.Insert(ctx, rw); err != nil {
		log.ErrorContext(ctx, "failed to insert reward", slog.Any("error", err))
		return nil, err
	}
	return rw, nil
}

func (s *RewardService) Update(ctx context.Context, id int64, req *UpdateRewardRequest) (*rewarddomain.Reward, error) {
	op := "rewardservice.Update"
	log := s.logger.With(slog.String("op", op), slog.Int64("reward_id", id))

	rw, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get reward", slog.Any("error", err))
		return nil, err
	}
	if rw == nil {
		return nil, ErrNotFound
	}

	var columns []string
	if req.Title != nil {
		rw.Title = *req.Title
		columns = append(columns, "title")
	}
	if req.Description != nil {
		rw.Description = *req.Description
		columns = append(columns, "description")
	}
	if req.CostCoins != nil {
		rw.CostCoins = *req.CostCoins
		columns = append(columns, "cost_coins")
	}
	if req.Limit != nil {
		rw.Limit = *req.Limit
		columns = append(columns, "limit")
	}
	if req.Status != nil {
		rw.Status = *req.Status
		columns = append(columns, "status")
	}

	if len(columns) == 0 {
		return rw, nil
	}

	if err := s.repo.Update(ctx, rw, columns...); err != nil {
		log.ErrorContext(ctx, "failed to update reward", slog.Any("error", err))
		return nil, err
	}
	return rw, nil
}

func (s *RewardService) Delete(ctx context.Context, id int64) error {
	op := "rewardservice.Delete"
	log := s.logger.With(slog.String("op", op), slog.Int64("reward_id", id))

	err := s.repo.Delete(ctx, id)
	if err == nil {
		return nil
	}
	var pgErr pgdriver.Error
	if errors.As(err, &pgErr) && pgErr.Field('C') == "23503" {
		return ErrHasRelations
	}
	log.ErrorContext(ctx, "failed to delete reward", slog.Any("error", err))
	return fmt.Errorf("rewardservice.Delete: %w", err)
}

func (s *RewardService) GetByID(ctx context.Context, id int64) (*rewarddomain.Reward, error) {
	op := "rewardservice.GetByID"
	log := s.logger.With(slog.String("op", op), slog.Int64("reward_id", id))

	rw, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get reward", slog.Any("error", err))
		return nil, err
	}
	return rw, nil
}

func (s *RewardService) ListBySeason(ctx context.Context, seasonID int64) ([]rewarddomain.Reward, error) {
	op := "rewardservice.ListBySeason"
	log := s.logger.With(slog.String("op", op), slog.Int64("season_id", seasonID))

	items, err := s.repo.ListBySeason(ctx, seasonID)
	if err != nil {
		log.ErrorContext(ctx, "failed to list rewards", slog.Any("error", err))
		return nil, err
	}
	return items, nil
}

// --- Claim management ---

// SubmitClaim creates a new prize claim for a user, deducting coins immediately.
func (s *RewardService) SubmitClaim(ctx context.Context, userID, seasonID, rewardID int64) (*rewarddomain.RewardClaim, error) {
	op := "rewardservice.SubmitClaim"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("reward_id", rewardID))

	rw, err := s.repo.GetByID(ctx, rewardID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get reward", slog.Any("error", err))
		return nil, err
	}
	if rw == nil || rw.Status != rewarddomain.RewardAvailable {
		return nil, ErrRewardUnavailable
	}

	if rw.Limit != nil {
		count, err := s.repo.CountActiveClaims(ctx, rewardID)
		if err != nil {
			log.ErrorContext(ctx, "failed to count claims", slog.Any("error", err))
			return nil, err
		}
		if count >= *rw.Limit {
			return nil, ErrLimitExceeded
		}
	}

	if _, err := s.balanceSvc.SpendCoins(ctx, wallet.DebitRequest{
		UserID:   userID,
		SeasonID: seasonID,
		Amount:   rw.CostCoins,
		RefID:    &rw.ID,
		RefTitle: rw.Title,
	}); err != nil {
		if errors.Is(err, wallet.ErrInsufficientBalance) {
			return nil, ErrInsufficientBalance
		}
		log.ErrorContext(ctx, "failed to spend coins", slog.Any("error", err))
		return nil, err
	}

	claim := &rewarddomain.RewardClaim{
		UserID:     userID,
		RewardID:   rewardID,
		Status:     rewarddomain.ClaimPending,
		SpentCoins: rw.CostCoins,
	}
	if err := s.repo.InsertClaim(ctx, claim); err != nil {
		log.ErrorContext(ctx, "failed to insert claim", slog.Any("error", err))
		return nil, err
	}

	if err := s.notifications.Notify(ctx,
		notifications.ClaimSubmitted{RewardTitle: rw.Title, SpentCoins: rw.CostCoins},
		notification.UserFilter{UserIDs: []int64{userID}},
	); err != nil {
		log.WarnContext(ctx, "failed to enqueue claim_submitted notification", slog.Any("error", err))
	}

	log.InfoContext(ctx, "reward claim submitted", slog.Int64("claim_id", claim.ID))
	return claim, nil
}

type UpdateClaimRequest struct {
	Status rewarddomain.RewardClaimStatus `json:"status"`
}

// UpdateClaimStatus updates the claim status. Refunds coins when the claim is cancelled.
func (s *RewardService) UpdateClaimStatus(ctx context.Context, claimID int64, req *UpdateClaimRequest, seasonID int64) (*rewarddomain.RewardClaim, error) {
	op := "rewardservice.UpdateClaimStatus"
	log := s.logger.With(slog.String("op", op), slog.Int64("claim_id", claimID))

	claim, err := s.repo.GetClaimByID(ctx, claimID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get claim", slog.Any("error", err))
		return nil, err
	}
	if claim == nil {
		return nil, ErrClaimNotFound
	}

	oldStatus := claim.Status
	claim.Status = req.Status

	if err := s.repo.UpdateClaim(ctx, claim, "status"); err != nil {
		log.ErrorContext(ctx, "failed to update claim", slog.Any("error", err))
		return nil, err
	}

	// Fetch reward once — needed for both refund and notification.
	rw, rwErr := s.repo.GetByID(ctx, claim.RewardID)
	if rwErr != nil {
		log.ErrorContext(ctx, "failed to get reward", slog.Any("error", rwErr))
	}

	rewardTitle := ""
	if rw != nil {
		rewardTitle = rw.Title
	}

	// Refund coins if the claim transitions to cancelled from a non-cancelled state.
	if req.Status == rewarddomain.ClaimCancelled && oldStatus != rewarddomain.ClaimCancelled {
		if rw != nil {
			refTitle := fmt.Sprintf("Refund: %s", rw.Title)
			if _, err := s.balanceSvc.RefundCoins(ctx, wallet.DebitRequest{
				UserID:   claim.UserID,
				SeasonID: seasonID,
				Amount:   claim.SpentCoins,
				RefID:    &rw.ID,
				RefTitle: refTitle,
			}); err != nil {
				log.ErrorContext(ctx, "failed to refund coins", slog.Any("error", err))
			}
		}
	}

	// Notify user about status change.
	switch {
	case req.Status == rewarddomain.ClaimCompleted && oldStatus != rewarddomain.ClaimCompleted:
		if err := s.notifications.Notify(ctx,
			notifications.ClaimCompleted{RewardTitle: rewardTitle},
			notification.UserFilter{UserIDs: []int64{claim.UserID}},
		); err != nil {
			log.WarnContext(ctx, "failed to enqueue claim_completed notification", slog.Any("error", err))
		}
	case req.Status == rewarddomain.ClaimCancelled && oldStatus != rewarddomain.ClaimCancelled:
		if err := s.notifications.Notify(ctx,
			notifications.ClaimCancelled{RewardTitle: rewardTitle, RefundedCoins: claim.SpentCoins},
			notification.UserFilter{UserIDs: []int64{claim.UserID}},
		); err != nil {
			log.WarnContext(ctx, "failed to enqueue claim_cancelled notification", slog.Any("error", err))
		}
	}

	return claim, nil
}

func (s *RewardService) GetClaimByID(ctx context.Context, id int64) (*rewarddomain.RewardClaim, error) {
	op := "rewardservice.GetClaimByID"
	log := s.logger.With(slog.String("op", op), slog.Int64("claim_id", id))

	c, err := s.repo.GetClaimByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get claim", slog.Any("error", err))
		return nil, err
	}
	return c, nil
}

func (s *RewardService) ListClaimsByUser(ctx context.Context, userID int64) ([]rewarddomain.RewardClaim, error) {
	op := "rewardservice.ListClaimsByUser"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	items, err := s.repo.ListClaimsByUser(ctx, userID)
	if err != nil {
		log.ErrorContext(ctx, "failed to list claims", slog.Any("error", err))
		return nil, err
	}
	return items, nil
}

func (s *RewardService) ListAllClaims(ctx context.Context) ([]rewarddomain.RewardClaim, error) {
	op := "rewardservice.ListAllClaims"
	log := s.logger.With(slog.String("op", op))

	items, err := s.repo.ListAllClaims(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to list all claims", slog.Any("error", err))
		return nil, err
	}
	return items, nil
}

// (no notification builders — rendering is handled by notificationservice)

package rewardservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/uptrace/bun/driver/pgdriver"
	dbmodels "go.mod/internal/db"
	"go.mod/internal/domain"
	balanceservice "go.mod/internal/services/balance"
)

var ErrNotFound = errors.New("reward not found")
var ErrClaimNotFound = errors.New("reward claim not found")
var ErrHasRelations = errors.New("reward has related records and cannot be deleted")
var ErrRewardUnavailable = errors.New("reward is not available")
var ErrLimitExceeded = errors.New("reward limit exceeded")
var ErrInsufficientBalance = errors.New("insufficient balance")

type rewardRepo interface {
	Insert(ctx context.Context, rw *dbmodels.Reward) error
	Update(ctx context.Context, rw *dbmodels.Reward, columns ...string) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*dbmodels.Reward, error)
	ListBySeason(ctx context.Context, seasonID int64) ([]dbmodels.Reward, error)
	InsertClaim(ctx context.Context, c *dbmodels.RewardClaim) error
	UpdateClaim(ctx context.Context, c *dbmodels.RewardClaim, columns ...string) error
	GetClaimByID(ctx context.Context, id int64) (*dbmodels.RewardClaim, error)
	ListClaimsByUser(ctx context.Context, userID int64) ([]dbmodels.RewardClaim, error)
	ListAllClaims(ctx context.Context) ([]dbmodels.RewardClaim, error)
	CountActiveClaims(ctx context.Context, rewardID int64) (int, error)
}

type coinsSpender interface {
	SpendCoins(ctx context.Context, req balanceservice.SpendCoinsRequest) (*domain.Transaction, error)
	RefundCoins(ctx context.Context, req balanceservice.SpendCoinsRequest) (*domain.Transaction, error)
}

type RewardService struct {
	repo       rewardRepo
	balanceSvc coinsSpender
	logger     *slog.Logger
}

func NewService(repo rewardRepo, balanceSvc coinsSpender, logger *slog.Logger) *RewardService {
	return &RewardService{repo: repo, balanceSvc: balanceSvc, logger: logger}
}

// --- Reward CRUD ---

type CreateRewardRequest struct {
	SeasonID  int64               `json:"season_id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	CostCoins   int                 `json:"cost_coins"`
	Limit       *int                `json:"limit"`
	Status      domain.RewardStatus `json:"status"`
}

type UpdateRewardRequest struct {
	Title       *string              `json:"title"`
	Description *string              `json:"description"`
	CostCoins   *int                 `json:"cost_coins"`
	Limit       **int                `json:"limit"`
	Status      *domain.RewardStatus `json:"status"`
}

func (s *RewardService) Create(ctx context.Context, req *CreateRewardRequest) (*domain.Reward, error) {
	op := "rewardservice.Create"
	log := s.logger.With(slog.String("op", op))

	status := req.Status
	if status == "" {
		status = domain.RewardAvailable
	}

	rw := &dbmodels.Reward{
		SeasonID:  req.SeasonID,
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
	return toDomainReward(rw), nil
}

func (s *RewardService) Update(ctx context.Context, id int64, req *UpdateRewardRequest) (*domain.Reward, error) {
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
		return toDomainReward(rw), nil
	}

	if err := s.repo.Update(ctx, rw, columns...); err != nil {
		log.ErrorContext(ctx, "failed to update reward", slog.Any("error", err))
		return nil, err
	}
	return toDomainReward(rw), nil
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
	return err
}

func (s *RewardService) GetByID(ctx context.Context, id int64) (*domain.Reward, error) {
	op := "rewardservice.GetByID"
	log := s.logger.With(slog.String("op", op), slog.Int64("reward_id", id))

	rw, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get reward", slog.Any("error", err))
		return nil, err
	}
	if rw == nil {
		return nil, nil
	}
	return toDomainReward(rw), nil
}

func (s *RewardService) ListBySeason(ctx context.Context, seasonID int64) ([]domain.Reward, error) {
	op := "rewardservice.ListBySeason"
	log := s.logger.With(slog.String("op", op), slog.Int64("season_id", seasonID))

	items, err := s.repo.ListBySeason(ctx, seasonID)
	if err != nil {
		log.ErrorContext(ctx, "failed to list rewards", slog.Any("error", err))
		return nil, err
	}
	return toDomainRewardSlice(items), nil
}

// --- Claim management ---

// SubmitClaim creates a new prize claim for a user, deducting coins immediately.
func (s *RewardService) SubmitClaim(ctx context.Context, userID, seasonID, rewardID int64) (*domain.RewardClaim, error) {
	op := "rewardservice.SubmitClaim"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("reward_id", rewardID))

	rw, err := s.repo.GetByID(ctx, rewardID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get reward", slog.Any("error", err))
		return nil, err
	}
	if rw == nil || rw.Status != domain.RewardAvailable {
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

	if _, err := s.balanceSvc.SpendCoins(ctx, balanceservice.SpendCoinsRequest{
		UserID:     userID,
		SeasonID: seasonID,
		Amount:     rw.CostCoins,
		RefID:      &rw.ID,
		RefTitle:   rw.Title,
	}); err != nil {
		if errors.Is(err, balanceservice.ErrInsufficientBalance) {
			return nil, ErrInsufficientBalance
		}
		log.ErrorContext(ctx, "failed to spend coins", slog.Any("error", err))
		return nil, err
	}

	claim := &dbmodels.RewardClaim{
		UserID:     userID,
		RewardID:   rewardID,
		Status:     domain.ClaimPending,
		SpentCoins: rw.CostCoins,
	}
	if err := s.repo.InsertClaim(ctx, claim); err != nil {
		log.ErrorContext(ctx, "failed to insert claim", slog.Any("error", err))
		return nil, err
	}

	log.InfoContext(ctx, "reward claim submitted", slog.Int64("claim_id", claim.ID))
	return toDomainClaim(claim), nil
}

type UpdateClaimRequest struct {
	Status domain.RewardClaimStatus `json:"status"`
}

// UpdateClaimStatus updates the claim status. Refunds coins when the claim is cancelled.
func (s *RewardService) UpdateClaimStatus(ctx context.Context, claimID int64, req *UpdateClaimRequest, seasonID int64) (*domain.RewardClaim, error) {
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

	// Refund coins if the claim transitions to cancelled from a non-cancelled state.
	if req.Status == domain.ClaimCancelled && oldStatus != domain.ClaimCancelled {
		rw, err := s.repo.GetByID(ctx, claim.RewardID)
		if err != nil {
			log.ErrorContext(ctx, "failed to get reward for refund", slog.Any("error", err))
		} else if rw != nil {
			refTitle := fmt.Sprintf("Refund: %s", rw.Title)
			if _, err := s.balanceSvc.RefundCoins(ctx, balanceservice.SpendCoinsRequest{
				UserID:     claim.UserID,
				SeasonID: seasonID,
				Amount:     claim.SpentCoins,
				RefID:      &rw.ID,
				RefTitle:   refTitle,
			}); err != nil {
				log.ErrorContext(ctx, "failed to refund coins", slog.Any("error", err))
			}
		}
	}

	return toDomainClaim(claim), nil
}

func (s *RewardService) GetClaimByID(ctx context.Context, id int64) (*domain.RewardClaim, error) {
	op := "rewardservice.GetClaimByID"
	log := s.logger.With(slog.String("op", op), slog.Int64("claim_id", id))

	c, err := s.repo.GetClaimByID(ctx, id)
	if err != nil {
		log.ErrorContext(ctx, "failed to get claim", slog.Any("error", err))
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return toDomainClaim(c), nil
}

func (s *RewardService) ListClaimsByUser(ctx context.Context, userID int64) ([]domain.RewardClaim, error) {
	op := "rewardservice.ListClaimsByUser"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	items, err := s.repo.ListClaimsByUser(ctx, userID)
	if err != nil {
		log.ErrorContext(ctx, "failed to list claims", slog.Any("error", err))
		return nil, err
	}
	return toDomainClaimSlice(items), nil
}

func (s *RewardService) ListAllClaims(ctx context.Context) ([]domain.RewardClaim, error) {
	op := "rewardservice.ListAllClaims"
	log := s.logger.With(slog.String("op", op))

	items, err := s.repo.ListAllClaims(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to list all claims", slog.Any("error", err))
		return nil, err
	}
	return toDomainClaimSlice(items), nil
}

func toDomainReward(rw *dbmodels.Reward) *domain.Reward {
	return &domain.Reward{
		ID:          rw.ID,
		SeasonID:  rw.SeasonID,
		Title:       rw.Title,
		Description: rw.Description,
		CostCoins:   rw.CostCoins,
		Limit:       rw.Limit,
		Status:      rw.Status,
	}
}

func toDomainRewardSlice(items []dbmodels.Reward) []domain.Reward {
	result := make([]domain.Reward, len(items))
	for i := range items {
		result[i] = *toDomainReward(&items[i])
	}
	return result
}

func toDomainClaim(c *dbmodels.RewardClaim) *domain.RewardClaim {
	return &domain.RewardClaim{
		ID:         c.ID,
		UserID:     c.UserID,
		RewardID:   c.RewardID,
		Status:     c.Status,
		SpentCoins: c.SpentCoins,
		CreatedAt:  c.CreatedAt,
	}
}

func toDomainClaimSlice(items []dbmodels.RewardClaim) []domain.RewardClaim {
	result := make([]domain.RewardClaim, len(items))
	for i := range items {
		result[i] = *toDomainClaim(&items[i])
	}
	return result
}

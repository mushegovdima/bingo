package balanceservice

import (
	"context"
	"errors"
	"log/slog"

	dbmodels "go.mod/internal/db"
	"go.mod/internal/db/repository"
	"go.mod/internal/domain"
)

var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrBalanceNotFound = errors.New("balance not found")

// ChangeBalanceRequest is used by managers to directly credit or debit a user's balance.
// A positive Amount tops up the balance; a negative Amount deducts from it.
type ChangeBalanceRequest struct {
	UserID   int64
	SeasonID int64
	Amount   int
	Note     string
}

// AddCoinsRequest credits coins to a user's balance for completing a task or attending an event.
type AddCoinsRequest struct {
	UserID   int64
	SeasonID int64
	Amount   int
	Reason   domain.TransactionReason
	RefID    *int64
	RefTitle string
}

// SpendCoinsRequest deducts or refunds coins from a user's balance (reward purchase / cancellation).
type SpendCoinsRequest struct {
	UserID   int64
	SeasonID int64
	Amount   int
	RefID    *int64
	RefTitle string
}

type balanceRepo interface {
	GetByUserAndSeason(ctx context.Context, userID, seasonID int64) (*dbmodels.SeasonMember, error)
	ListTransactions(ctx context.Context, balanceID int64) ([]dbmodels.Transaction, error)
	AdjustAndRecord(ctx context.Context, userID, seasonID int64, amount int, reason domain.TransactionReason, refID *int64, refTitle string) (*dbmodels.Transaction, error)
	SpendAndRecord(ctx context.Context, userID, seasonID int64, amount int, refID *int64, refTitle string) (*dbmodels.Transaction, error)
	ListByUser(ctx context.Context, userID int64) ([]*dbmodels.SeasonMember, error)
	EnsureBalance(ctx context.Context, userID, seasonID int64) (*dbmodels.SeasonMember, error)
}

type BalanceService struct {
	repo   balanceRepo
	logger *slog.Logger
}

func NewService(repo balanceRepo, logger *slog.Logger) *BalanceService {
	return &BalanceService{repo: repo, logger: logger}
}

func (s *BalanceService) GetBalance(ctx context.Context, userID, seasonID int64) (*domain.SeasonMember, error) {
	op := "balanceservice.GetBalance"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	b, err := s.repo.GetByUserAndSeason(ctx, userID, seasonID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get balance", slog.Any("error", err))
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	return toDomainBalance(b), nil
}

func (s *BalanceService) GetTransactions(ctx context.Context, userID, seasonID int64) ([]domain.Transaction, error) {
	op := "balanceservice.GetTransactions"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	b, err := s.repo.GetByUserAndSeason(ctx, userID, seasonID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get balance", slog.Any("error", err))
		return nil, err
	}
	if b == nil {
		return nil, ErrBalanceNotFound
	}

	txs, err := s.repo.ListTransactions(ctx, b.ID)
	if err != nil {
		log.ErrorContext(ctx, "failed to list transactions", slog.Any("error", err))
		return nil, err
	}
	return toDomainTransactions(txs), nil
}

// ChangeBalance performs a manual balance change (manager operation).
// req.Amount can be positive (credit) or negative (debit).
func (s *BalanceService) ChangeBalance(ctx context.Context, req ChangeBalanceRequest) (*domain.Transaction, error) {
	op := "balanceservice.ChangeBalance"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", req.UserID), slog.Int("amount", req.Amount))

	tx, err := s.repo.AdjustAndRecord(ctx, req.UserID, req.SeasonID, req.Amount, domain.TransactionReasonManual, nil, req.Note)
	if err != nil {
		log.ErrorContext(ctx, "failed to change balance", slog.Any("error", err))
		return nil, err
	}
	log.InfoContext(ctx, "manual balance change", slog.Int("amount", req.Amount))
	return toDomainTransaction(tx), nil
}

// AddCoins credits coins to the user's balance with a given reason (task/event).
func (s *BalanceService) AddCoins(ctx context.Context, req AddCoinsRequest) (*domain.Transaction, error) {
	op := "balanceservice.AddCoins"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", req.UserID))

	tx, err := s.repo.AdjustAndRecord(ctx, req.UserID, req.SeasonID, req.Amount, req.Reason, req.RefID, req.RefTitle)
	if err != nil {
		log.ErrorContext(ctx, "failed to add coins", slog.Any("error", err))
		return nil, err
	}
	return toDomainTransaction(tx), nil
}

// SpendCoins deducts coins for a reward purchase.
// Returns ErrInsufficientBalance when the user cannot afford the amount.
func (s *BalanceService) SpendCoins(ctx context.Context, req SpendCoinsRequest) (*domain.Transaction, error) {
	op := "balanceservice.SpendCoins"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", req.UserID))

	tx, err := s.repo.SpendAndRecord(ctx, req.UserID, req.SeasonID, req.Amount, req.RefID, req.RefTitle)
	if errors.Is(err, repository.AdjustErrInsufficientBalance) {
		return nil, ErrInsufficientBalance
	}
	if err != nil {
		log.ErrorContext(ctx, "failed to spend coins", slog.Any("error", err))
		return nil, err
	}
	return toDomainTransaction(tx), nil
}

// RefundCoins credits back coins for a cancelled reward claim.
func (s *BalanceService) RefundCoins(ctx context.Context, req SpendCoinsRequest) (*domain.Transaction, error) {
	op := "balanceservice.RefundCoins"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", req.UserID))

	tx, err := s.repo.AdjustAndRecord(ctx, req.UserID, req.SeasonID, req.Amount, domain.TransactionReasonReward, req.RefID, req.RefTitle)
	if err != nil {
		log.ErrorContext(ctx, "failed to refund coins", slog.Any("error", err))
		return nil, err
	}
	return toDomainTransaction(tx), nil
}

// ListUserBalances returns all season balances for a user enriched with season details.
func (s *BalanceService) ListUserBalances(ctx context.Context, userID int64) ([]domain.SeasonMemberWithSeason, error) {
	op := "balanceservice.ListUserBalances"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	balances, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		log.ErrorContext(ctx, "failed to list balances", slog.Any("error", err))
		return nil, err
	}
	result := make([]domain.SeasonMemberWithSeason, 0, len(balances))
	for _, b := range balances {
		item := domain.SeasonMemberWithSeason{
			SeasonMember: *toDomainBalance(b),
		}
		if b.Season != nil {
			item.Season = toDomainSeason(b.Season)
		}
		result = append(result, item)
	}
	return result, nil
}

// JoinSeason creates a zero-balance wallet for the user in the given season (idempotent).
func (s *BalanceService) JoinSeason(ctx context.Context, userID, seasonID int64) (*domain.SeasonMember, error) {
	op := "balanceservice.JoinSeason"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("season_id", seasonID))

	b, err := s.repo.EnsureBalance(ctx, userID, seasonID)
	if err != nil {
		log.ErrorContext(ctx, "failed to ensure balance", slog.Any("error", err))
		return nil, err
	}
	return toDomainBalance(b), nil
}

func toDomainSeason(c *dbmodels.Season) domain.Season {
	return domain.Season{
		ID:        c.ID,
		Title:     c.Title,
		StartDate: c.StartDate,
		EndDate:   c.EndDate,
		IsActive:  c.IsActive,
	}
}

func toDomainBalance(b *dbmodels.SeasonMember) *domain.SeasonMember {
	return &domain.SeasonMember{
		ID:          b.ID,
		UserID:      b.UserID,
		SeasonID:    b.SeasonID,
		Balance:     b.Balance,
		TotalEarned: b.TotalEarned,
		UpdatedAt:   b.UpdatedAt,
	}
}

func toDomainTransaction(tx *dbmodels.Transaction) *domain.Transaction {
	return &domain.Transaction{
		ID:        tx.ID,
		MemberID:  tx.MemberID,
		Amount:    tx.Amount,
		Reason:    tx.Reason,
		RefID:     tx.RefID,
		RefTitle:  tx.RefTitle,
		CreatedAt: tx.CreatedAt,
	}
}

func toDomainTransactions(txs []dbmodels.Transaction) []domain.Transaction {
	result := make([]domain.Transaction, len(txs))
	for i := range txs {
		result[i] = *toDomainTransaction(&txs[i])
	}
	return result
}

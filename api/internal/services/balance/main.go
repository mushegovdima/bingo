package balanceservice

import (
	"context"
	"errors"
	"log/slog"

	wallet "go.mod/internal/contracts/wallet"
	walletdomain "go.mod/internal/domain/wallet"
)

// ErrInsufficientBalance is re-exported from the wallet contracts package so callers
// that still import balanceservice continue to compile during the transition.
var ErrInsufficientBalance = wallet.ErrInsufficientBalance
var ErrBalanceNotFound = errors.New("balance not found")

type balanceRepo interface {
	GetByUserAndSeason(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error)
	ListTransactions(ctx context.Context, balanceID int64) ([]walletdomain.Transaction, error)
	AdjustAndRecord(ctx context.Context, userID, seasonID int64, amount int, reason walletdomain.TransactionReason, refID *int64, refTitle string) (*walletdomain.Transaction, error)
	SpendAndRecord(ctx context.Context, userID, seasonID int64, amount int, refID *int64, refTitle string) (*walletdomain.Transaction, error)
	ListByUser(ctx context.Context, userID int64) ([]walletdomain.SeasonMemberWithSeason, error)
	EnsureBalance(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error)
	GetLeaderboardNeighbors(ctx context.Context, userID, seasonID int64) ([]walletdomain.LeaderboardEntry, error)
	GetFullLeaderboard(ctx context.Context, userID, seasonID int64) ([]walletdomain.LeaderboardEntry, error)
}

type BalanceService struct {
	repo   balanceRepo
	logger *slog.Logger
}

func NewService(repo balanceRepo, logger *slog.Logger) *BalanceService {
	return &BalanceService{repo: repo, logger: logger}
}

func (s *BalanceService) GetBalance(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error) {
	op := "balanceservice.GetBalance"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	b, err := s.repo.GetByUserAndSeason(ctx, userID, seasonID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get balance", slog.Any("error", err))
		return nil, err
	}
	return b, nil
}

func (s *BalanceService) GetTransactions(ctx context.Context, userID, seasonID int64) ([]walletdomain.Transaction, error) {
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
	return txs, nil
}

// ChangeBalance performs a manual balance change (manager operation).
// req.Amount can be positive (credit) or negative (debit).
func (s *BalanceService) ChangeBalance(ctx context.Context, req wallet.ChangeRequest) (*walletdomain.Transaction, error) {
	op := "balanceservice.ChangeBalance"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", req.UserID), slog.Int("amount", req.Amount))

	tx, err := s.repo.AdjustAndRecord(ctx, req.UserID, req.SeasonID, req.Amount, walletdomain.TransactionReasonManual, nil, req.Note)
	if err != nil {
		log.ErrorContext(ctx, "failed to change balance", slog.Any("error", err))
		return nil, err
	}
	log.InfoContext(ctx, "manual balance change", slog.Int("amount", req.Amount))
	return tx, nil
}

// AddCoins credits coins to the user's balance with a given reason (task/event).
func (s *BalanceService) AddCoins(ctx context.Context, req wallet.CreditRequest) (*walletdomain.Transaction, error) {
	op := "balanceservice.AddCoins"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", req.UserID))

	tx, err := s.repo.AdjustAndRecord(ctx, req.UserID, req.SeasonID, req.Amount, req.Reason, req.RefID, req.RefTitle)
	if err != nil {
		log.ErrorContext(ctx, "failed to add coins", slog.Any("error", err))
		return nil, err
	}
	return tx, nil
}

// SpendCoins deducts coins for a reward purchase.
// Returns wallet.ErrInsufficientBalance when the user cannot afford the amount.
func (s *BalanceService) SpendCoins(ctx context.Context, req wallet.DebitRequest) (*walletdomain.Transaction, error) {
	op := "balanceservice.SpendCoins"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", req.UserID))

	tx, err := s.repo.SpendAndRecord(ctx, req.UserID, req.SeasonID, req.Amount, req.RefID, req.RefTitle)
	if err != nil {
		if !errors.Is(err, walletdomain.ErrInsufficientBalance) {
			log.ErrorContext(ctx, "failed to spend coins", slog.Any("error", err))
		}
		return nil, err
	}
	return tx, nil
}

// RefundCoins credits back coins for a cancelled reward claim.
func (s *BalanceService) RefundCoins(ctx context.Context, req wallet.DebitRequest) (*walletdomain.Transaction, error) {
	op := "balanceservice.RefundCoins"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", req.UserID))

	tx, err := s.repo.AdjustAndRecord(ctx, req.UserID, req.SeasonID, req.Amount, walletdomain.TransactionReasonReward, req.RefID, req.RefTitle)
	if err != nil {
		log.ErrorContext(ctx, "failed to refund coins", slog.Any("error", err))
		return nil, err
	}
	return tx, nil
}

// ListUserBalances returns all season balances for a user enriched with season details.
func (s *BalanceService) ListUserBalances(ctx context.Context, userID int64) ([]walletdomain.SeasonMemberWithSeason, error) {
	op := "balanceservice.ListUserBalances"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	balances, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		log.ErrorContext(ctx, "failed to list balances", slog.Any("error", err))
		return nil, err
	}
	return balances, nil
}

// GetLeaderboard returns the current user's position and the neighbors above and below.
func (s *BalanceService) GetLeaderboard(ctx context.Context, userID, seasonID int64) ([]walletdomain.LeaderboardEntry, error) {
	op := "balanceservice.GetLeaderboard"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("season_id", seasonID))

	entries, err := s.repo.GetLeaderboardNeighbors(ctx, userID, seasonID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get leaderboard", slog.Any("error", err))
		return nil, err
	}
	return entries, nil
}

// GetFullLeaderboard returns all participants in the season ranked by balance DESC, id ASC on ties.
func (s *BalanceService) GetFullLeaderboard(ctx context.Context, userID, seasonID int64) ([]walletdomain.LeaderboardEntry, error) {
	op := "balanceservice.GetFullLeaderboard"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("season_id", seasonID))

	entries, err := s.repo.GetFullLeaderboard(ctx, userID, seasonID)
	if err != nil {
		log.ErrorContext(ctx, "failed to get full leaderboard", slog.Any("error", err))
		return nil, err
	}
	return entries, nil
}

// JoinSeason creates a zero-balance wallet for the user in the given season (idempotent).
func (s *BalanceService) JoinSeason(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error) {
	op := "balanceservice.JoinSeason"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("season_id", seasonID))

	b, err := s.repo.EnsureBalance(ctx, userID, seasonID)
	if err != nil {
		log.ErrorContext(ctx, "failed to ensure balance", slog.Any("error", err))
		return nil, err
	}
	return b, nil
}

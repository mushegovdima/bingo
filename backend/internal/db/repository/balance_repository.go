package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/uptrace/bun"
	"go.mod/internal/db"
	"go.mod/internal/domain"
)

type BalanceRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewBalanceRepository(bunDB *bun.DB, logger *slog.Logger) *BalanceRepository {
	return &BalanceRepository{db: bunDB, logger: logger}
}

// AdjustErrInsufficientBalance is returned by SpendAndRecord when the balance is too low.
var AdjustErrInsufficientBalance = errors.New("insufficient balance")

// GetByUserAndSeason returns the balance for a user in a given season, or nil if none exists.
func (r *BalanceRepository) GetByUserAndSeason(ctx context.Context, userID, seasonID int64) (*db.SeasonMember, error) {
	op := "db.balancerepository.GetByUserAndSeason"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("season_id", seasonID))

	b := &db.SeasonMember{}
	err := r.db.NewSelect().Model(b).
		Where("user_id = ? AND season_id = ?", userID, seasonID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return b, nil
}

// ListTransactions returns all transactions for a balance ordered from newest to oldest.
func (r *BalanceRepository) ListTransactions(ctx context.Context, memberID int64) ([]db.Transaction, error) {
	op := "db.balancerepository.ListTransactions"
	log := r.logger.With(slog.String("op", op), slog.Int64("member_id", memberID))

	var txs []db.Transaction
	err := r.db.NewSelect().Model(&txs).
		Where("member_id = ?", memberID).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return txs, nil
}

// AdjustAndRecord atomically upserts the season balance and records the transaction.
// For positive amounts, total_earned is incremented as well.
func (r *BalanceRepository) AdjustAndRecord(
	ctx context.Context,
	userID, seasonID int64,
	amount int,
	reason domain.TransactionReason,
	refID *int64,
	refTitle string,
) (*db.Transaction, error) {
	op := "db.balancerepository.AdjustAndRecord"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	var tx db.Transaction
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, bunTx bun.Tx) error {
		var balanceID int64
		err := bunTx.NewRaw(`
			INSERT INTO season_members (user_id, season_id, balance, total_earned, updated_at)
			VALUES (?, ?, ?, GREATEST(?, 0), NOW())
			ON CONFLICT (user_id, season_id) DO UPDATE
			SET balance      = season_members.balance      + EXCLUDED.balance,
			    total_earned = season_members.total_earned + GREATEST(EXCLUDED.balance, 0),
			    updated_at   = EXCLUDED.updated_at
			RETURNING id
		`, userID, seasonID, amount, amount).Scan(ctx, &balanceID)
		if err != nil {
			return err
		}

		tx = db.Transaction{
			MemberID: balanceID,
			Amount:   amount,
			Reason:   reason,
			RefID:    refID,
			RefTitle: refTitle,
		}
		_, err = bunTx.NewInsert().Model(&tx).Returning("id, created_at").Exec(ctx)
		return err
	})
	if err != nil {
		log.ErrorContext(ctx, "transaction failed", slog.Any("error", err))
		return nil, err
	}
	return &tx, nil
}

// SpendAndRecord atomically deducts coins and records the transaction.
// Returns AdjustErrInsufficientBalance if the user does not have enough coins.
func (r *BalanceRepository) SpendAndRecord(
	ctx context.Context,
	userID, seasonID int64,
	amount int,
	refID *int64,
	refTitle string,
) (*db.Transaction, error) {
	op := "db.balancerepository.SpendAndRecord"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	var tx db.Transaction
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, bunTx bun.Tx) error {
		var balanceID int64
		err := bunTx.NewRaw(`
			UPDATE season_members
			SET balance    = balance - ?,
			    updated_at = NOW()
			WHERE user_id = ? AND season_id = ? AND balance >= ?
			RETURNING id
		`, amount, userID, seasonID, amount).Scan(ctx, &balanceID)
		if errors.Is(err, sql.ErrNoRows) {
			return AdjustErrInsufficientBalance
		}
		if err != nil {
			return err
		}

		tx = db.Transaction{
			MemberID: balanceID,
			Amount:   -amount,
			Reason:   domain.TransactionReasonReward,
			RefID:    refID,
			RefTitle: refTitle,
		}
		_, err = bunTx.NewInsert().Model(&tx).Returning("id, created_at").Exec(ctx)
		return err
	})
	if err != nil {
		log.ErrorContext(ctx, "transaction failed", slog.Any("error", err))
		return nil, err
	}
	return &tx, nil
}

// ListByUser returns all season balances for a user, newest first, with Season populated.
func (r *BalanceRepository) ListByUser(ctx context.Context, userID int64) ([]*db.SeasonMember, error) {
	op := "db.balancerepository.ListByUser"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	var balances []*db.SeasonMember
	err := r.db.NewSelect().
		Model(&balances).
		Relation("Season").
		Where("season_member.user_id = ?", userID).
		OrderExpr("season_member.id DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return balances, nil
}

// EnsureBalance creates a zero-balance row for userID+seasonID if it does not exist yet.
// Idempotent: if the balance already exists, it is returned unchanged.
func (r *BalanceRepository) EnsureBalance(ctx context.Context, userID, seasonID int64) (*db.SeasonMember, error) {
	op := "db.balancerepository.EnsureBalance"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("season_id", seasonID))

	_, err := r.db.NewRaw(`
		INSERT INTO season_members (user_id, season_id, balance, total_earned, updated_at)
		VALUES (?, ?, 0, 0, NOW())
		ON CONFLICT (user_id, season_id) DO NOTHING
	`, userID, seasonID).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "insert failed", slog.Any("error", err))
		return nil, err
	}
	return r.GetByUserAndSeason(ctx, userID, seasonID)
}

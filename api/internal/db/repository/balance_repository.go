package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/uptrace/bun"
	"go.mod/internal/db"
	seasondomain "go.mod/internal/domain/season"
	walletdomain "go.mod/internal/domain/wallet"
)

type BalanceRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewBalanceRepository(bunDB *bun.DB, logger *slog.Logger) *BalanceRepository {
	return &BalanceRepository{db: bunDB, logger: logger}
}

// Re-export the domain sentinel so tests/callers can match on it via errors.Is
// without importing the domain package transitively. Same value, no wrapping.
var ErrInsufficientBalance = walletdomain.ErrInsufficientBalance

// GetByUserAndSeason returns the wallet for a user in a given season, or nil if none exists.
func (r *BalanceRepository) GetByUserAndSeason(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error) {
	op := "db.balancerepository.GetByUserAndSeason"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("season_id", seasonID))

	row := &db.SeasonMember{}
	err := r.db.NewSelect().Model(row).
		Where("user_id = ? AND season_id = ?", userID, seasonID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	m := toDomainSeasonMember(row)
	return &m, nil
}

// ListTransactions returns all transactions for a wallet ordered from newest to oldest.
func (r *BalanceRepository) ListTransactions(ctx context.Context, memberID int64) ([]walletdomain.Transaction, error) {
	op := "db.balancerepository.ListTransactions"
	log := r.logger.With(slog.String("op", op), slog.Int64("member_id", memberID))

	var rows []db.Transaction
	err := r.db.NewSelect().Model(&rows).
		Where("member_id = ?", memberID).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := make([]walletdomain.Transaction, len(rows))
	for i := range rows {
		out[i] = toDomainTransaction(&rows[i])
	}
	return out, nil
}

// AdjustAndRecord atomically upserts the season balance and records the transaction.
// For positive amounts, total_earned is incremented as well.
func (r *BalanceRepository) AdjustAndRecord(
	ctx context.Context,
	userID, seasonID int64,
	amount int,
	reason walletdomain.TransactionReason,
	refID *int64,
	refTitle string,
) (*walletdomain.Transaction, error) {
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
	domainTx := toDomainTransaction(&tx)
	return &domainTx, nil
}

// SpendAndRecord atomically deducts coins and records the transaction.
// Returns walletdomain.ErrInsufficientBalance if the user does not have enough coins.
func (r *BalanceRepository) SpendAndRecord(
	ctx context.Context,
	userID, seasonID int64,
	amount int,
	refID *int64,
	refTitle string,
) (*walletdomain.Transaction, error) {
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
			return walletdomain.ErrInsufficientBalance
		}
		if err != nil {
			return err
		}

		tx = db.Transaction{
			MemberID: balanceID,
			Amount:   -amount,
			Reason:   walletdomain.TransactionReasonReward,
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
	domainTx := toDomainTransaction(&tx)
	return &domainTx, nil
}

// ListByUser returns all season wallets for a user, newest first, enriched with Season.
func (r *BalanceRepository) ListByUser(ctx context.Context, userID int64) ([]walletdomain.SeasonMemberWithSeason, error) {
	op := "db.balancerepository.ListByUser"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	var rows []*db.SeasonMember
	err := r.db.NewSelect().
		Model(&rows).
		Relation("Season").
		Where("season_member.user_id = ?", userID).
		OrderExpr("season_member.id DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := make([]walletdomain.SeasonMemberWithSeason, 0, len(rows))
	for _, row := range rows {
		item := walletdomain.SeasonMemberWithSeason{
			SeasonMember: toDomainSeasonMember(row),
		}
		if row.Season != nil {
			item.Season = toDomainSeason(row.Season)
		}
		out = append(out, item)
	}
	return out, nil
}

// EnsureBalance creates a zero-balance row for userID+seasonID if it does not exist yet.
// Idempotent: if the wallet already exists, it is returned unchanged.
func (r *BalanceRepository) EnsureBalance(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error) {
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

// GetLeaderboardNeighbors returns up to 3 entries around the current user:
// the participant above, the current user, and the participant below — ranked by balance DESC, id ASC on ties.
// If the user has no wallet in the season, returns nil.
func (r *BalanceRepository) GetLeaderboardNeighbors(ctx context.Context, userID, seasonID int64) ([]walletdomain.LeaderboardEntry, error) {
	op := "db.balancerepository.GetLeaderboardNeighbors"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("season_id", seasonID))

	type row struct {
		Position int    `bun:"position"`
		UserID   int64  `bun:"user_id"`
		Name     string `bun:"name"`
		Username string `bun:"username"`
		PhotoURL string `bun:"photo_url"`
		Balance  int    `bun:"balance"`
	}

	var rows []row
	err := r.db.NewRaw(`
		WITH ranked AS (
			SELECT
				sm.user_id,
				u.name,
				u.username,
				u.photo_url,
				sm.balance,
				CAST(ROW_NUMBER() OVER (ORDER BY sm.balance DESC, sm.id ASC) AS INTEGER) AS position
			FROM season_members sm
			JOIN users u ON u.id = sm.user_id
			WHERE sm.season_id = ?
		),
		my_rank AS (
			SELECT position FROM ranked WHERE user_id = ?
		)
		SELECT r.position, r.user_id, r.name, r.username, r.photo_url, r.balance
		FROM ranked r, my_rank m
		WHERE r.position BETWEEN m.position - 1 AND m.position + 1
		ORDER BY r.position
	`, seasonID, userID).Scan(ctx, &rows)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}

	out := make([]walletdomain.LeaderboardEntry, len(rows))
	for i, r := range rows {
		out[i] = walletdomain.LeaderboardEntry{
			Position:  r.Position,
			UserID:    r.UserID,
			Name:      r.Name,
			Username:  r.Username,
			PhotoURL:  r.PhotoURL,
			Balance:   r.Balance,
			IsCurrent: r.UserID == userID,
		}
	}
	return out, nil
}

// GetFullLeaderboard returns all participants in the season ranked by balance DESC, id ASC on ties.
func (r *BalanceRepository) GetFullLeaderboard(ctx context.Context, userID, seasonID int64) ([]walletdomain.LeaderboardEntry, error) {
	op := "db.balancerepository.GetFullLeaderboard"
	log := r.logger.With(slog.String("op", op), slog.Int64("season_id", seasonID))

	type row struct {
		Position int    `bun:"position"`
		UserID   int64  `bun:"user_id"`
		Name     string `bun:"name"`
		Username string `bun:"username"`
		PhotoURL string `bun:"photo_url"`
		Balance  int    `bun:"balance"`
	}

	var rows []row
	err := r.db.NewRaw(`
		SELECT
			CAST(ROW_NUMBER() OVER (ORDER BY sm.balance DESC, sm.id ASC) AS INTEGER) AS position,
			sm.user_id,
			u.name,
			u.username,
			u.photo_url,
			sm.balance
		FROM season_members sm
		JOIN users u ON u.id = sm.user_id
		WHERE sm.season_id = ?
		ORDER BY position
	`, seasonID).Scan(ctx, &rows)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}

	out := make([]walletdomain.LeaderboardEntry, len(rows))
	for i, r := range rows {
		out[i] = walletdomain.LeaderboardEntry{
			Position:  r.Position,
			UserID:    r.UserID,
			Name:      r.Name,
			Username:  r.Username,
			PhotoURL:  r.PhotoURL,
			Balance:   r.Balance,
			IsCurrent: r.UserID == userID,
		}
	}
	return out, nil
}

func toDomainSeasonMember(row *db.SeasonMember) walletdomain.SeasonMember {
	return walletdomain.SeasonMember{
		ID:          row.ID,
		UserID:      row.UserID,
		SeasonID:    row.SeasonID,
		Balance:     row.Balance,
		TotalEarned: row.TotalEarned,
		UpdatedAt:   row.UpdatedAt,
	}
}

func toDomainTransaction(row *db.Transaction) walletdomain.Transaction {
	return walletdomain.Transaction{
		ID:        row.ID,
		MemberID:  row.MemberID,
		Amount:    row.Amount,
		Reason:    row.Reason,
		RefID:     row.RefID,
		RefTitle:  row.RefTitle,
		CreatedAt: row.CreatedAt,
	}
}

func toDomainSeason(row *db.Season) seasondomain.Season {
	return seasondomain.Season{
		ID:        row.ID,
		Title:     row.Title,
		StartDate: row.StartDate,
		EndDate:   row.EndDate,
		IsActive:  row.IsActive,
	}
}

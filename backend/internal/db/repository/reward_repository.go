package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/uptrace/bun"
	"go.mod/internal/db"
)

type RewardRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewRewardRepository(bunDB *bun.DB, logger *slog.Logger) *RewardRepository {
	return &RewardRepository{db: bunDB, logger: logger}
}

// --- Reward CRUD ---

func (r *RewardRepository) Insert(ctx context.Context, rw *db.Reward) error {
	op := "db.rewardrepository.Insert"
	log := r.logger.With(slog.String("op", op))

	_, err := r.db.NewInsert().Model(rw).Returning("id").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *RewardRepository) Update(ctx context.Context, rw *db.Reward, columns ...string) error {
	op := "db.rewardrepository.Update"
	log := r.logger.With(slog.String("op", op), slog.Int64("reward_id", rw.ID))

	_, err := r.db.NewUpdate().Model(rw).Column(columns...).Where("id = ?", rw.ID).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *RewardRepository) Delete(ctx context.Context, id int64) error {
	op := "db.rewardrepository.Delete"
	log := r.logger.With(slog.String("op", op), slog.Int64("reward_id", id))

	_, err := r.db.NewDelete().Model((*db.Reward)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *RewardRepository) GetByID(ctx context.Context, id int64) (*db.Reward, error) {
	op := "db.rewardrepository.GetByID"
	log := r.logger.With(slog.String("op", op), slog.Int64("reward_id", id))

	rw := &db.Reward{}
	err := r.db.NewSelect().Model(rw).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return rw, nil
}

func (r *RewardRepository) ListBySeason(ctx context.Context, seasonID int64) ([]db.Reward, error) {
	op := "db.rewardrepository.ListBySeason"
	log := r.logger.With(slog.String("op", op), slog.Int64("season_id", seasonID))

	var items []db.Reward
	err := r.db.NewSelect().Model(&items).
		Where("season_id = ?", seasonID).
		OrderExpr("id ASC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return items, nil
}

// --- RewardClaim CRUD ---

func (r *RewardRepository) InsertClaim(ctx context.Context, c *db.RewardClaim) error {
	op := "db.rewardrepository.InsertClaim"
	log := r.logger.With(slog.String("op", op))

	_, err := r.db.NewInsert().Model(c).Returning("id, created_at").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *RewardRepository) UpdateClaim(ctx context.Context, c *db.RewardClaim, columns ...string) error {
	op := "db.rewardrepository.UpdateClaim"
	log := r.logger.With(slog.String("op", op), slog.Int64("claim_id", c.ID))

	_, err := r.db.NewUpdate().Model(c).Column(columns...).Where("id = ?", c.ID).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *RewardRepository) GetClaimByID(ctx context.Context, id int64) (*db.RewardClaim, error) {
	op := "db.rewardrepository.GetClaimByID"
	log := r.logger.With(slog.String("op", op), slog.Int64("claim_id", id))

	c := &db.RewardClaim{}
	err := r.db.NewSelect().Model(c).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return c, nil
}

func (r *RewardRepository) ListClaimsByUser(ctx context.Context, userID int64) ([]db.RewardClaim, error) {
	op := "db.rewardrepository.ListClaimsByUser"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	var items []db.RewardClaim
	err := r.db.NewSelect().Model(&items).
		Where("user_id = ?", userID).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return items, nil
}

func (r *RewardRepository) ListAllClaims(ctx context.Context) ([]db.RewardClaim, error) {
	op := "db.rewardrepository.ListAllClaims"
	log := r.logger.With(slog.String("op", op))

	var items []db.RewardClaim
	err := r.db.NewSelect().Model(&items).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return items, nil
}

func (r *RewardRepository) CountActiveClaims(ctx context.Context, rewardID int64) (int, error) {
	op := "db.rewardrepository.CountActiveClaims"
	log := r.logger.With(slog.String("op", op), slog.Int64("reward_id", rewardID))

	count, err := r.db.NewSelect().Model((*db.RewardClaim)(nil)).
		Where("reward_id = ? AND status != 'cancelled'", rewardID).
		Count(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return 0, err
	}
	return count, nil
}

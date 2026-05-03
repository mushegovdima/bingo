package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/uptrace/bun"
	"go.mod/internal/db"
	rewarddomain "go.mod/internal/domain/reward"
	"log/slog"
)

type RewardRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewRewardRepository(bunDB *bun.DB, logger *slog.Logger) *RewardRepository {
	return &RewardRepository{db: bunDB, logger: logger}
}

// --- Reward CRUD ---

func (r *RewardRepository) Insert(ctx context.Context, rw *rewarddomain.Reward) error {
	op := "db.rewardrepository.Insert"
	log := r.logger.With(slog.String("op", op))

	row := toDBReward(rw)
	_, err := r.db.NewInsert().Model(row).Returning("id").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	rw.ID = row.ID
	return nil
}

func (r *RewardRepository) Update(ctx context.Context, rw *rewarddomain.Reward, columns ...string) error {
	op := "db.rewardrepository.Update"
	log := r.logger.With(slog.String("op", op), slog.Int64("reward_id", rw.ID))

	row := toDBReward(rw)
	_, err := r.db.NewUpdate().Model(row).Column(columns...).Where("id = ?", rw.ID).Exec(ctx)
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

func (r *RewardRepository) GetByID(ctx context.Context, id int64) (*rewarddomain.Reward, error) {
	op := "db.rewardrepository.GetByID"
	log := r.logger.With(slog.String("op", op), slog.Int64("reward_id", id))

	row := &db.Reward{}
	err := r.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := toDomainReward(row)
	return &out, nil
}

func (r *RewardRepository) ListBySeason(ctx context.Context, seasonID int64) ([]rewarddomain.Reward, error) {
	op := "db.rewardrepository.ListBySeason"
	log := r.logger.With(slog.String("op", op), slog.Int64("season_id", seasonID))

	var rows []db.Reward
	err := r.db.NewSelect().Model(&rows).
		Where("season_id = ?", seasonID).
		OrderExpr("id ASC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := make([]rewarddomain.Reward, len(rows))
	for i := range rows {
		out[i] = toDomainReward(&rows[i])
	}
	return out, nil
}

// --- RewardClaim CRUD ---

func (r *RewardRepository) InsertClaim(ctx context.Context, c *rewarddomain.RewardClaim) error {
	op := "db.rewardrepository.InsertClaim"
	log := r.logger.With(slog.String("op", op))

	row := toDBClaim(c)
	_, err := r.db.NewInsert().Model(row).Returning("id, created_at").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	c.ID = row.ID
	c.CreatedAt = row.CreatedAt
	return nil
}

func (r *RewardRepository) UpdateClaim(ctx context.Context, c *rewarddomain.RewardClaim, columns ...string) error {
	op := "db.rewardrepository.UpdateClaim"
	log := r.logger.With(slog.String("op", op), slog.Int64("claim_id", c.ID))

	row := toDBClaim(c)
	_, err := r.db.NewUpdate().Model(row).Column(columns...).Where("id = ?", c.ID).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *RewardRepository) GetClaimByID(ctx context.Context, id int64) (*rewarddomain.RewardClaim, error) {
	op := "db.rewardrepository.GetClaimByID"
	log := r.logger.With(slog.String("op", op), slog.Int64("claim_id", id))

	row := &db.RewardClaim{}
	err := r.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := toDomainClaim(row)
	return &out, nil
}

func (r *RewardRepository) ListClaimsByUser(ctx context.Context, userID int64) ([]rewarddomain.RewardClaim, error) {
	op := "db.rewardrepository.ListClaimsByUser"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	var rows []db.RewardClaim
	err := r.db.NewSelect().Model(&rows).
		Where("user_id = ?", userID).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := make([]rewarddomain.RewardClaim, len(rows))
	for i := range rows {
		out[i] = toDomainClaim(&rows[i])
	}
	return out, nil
}

func (r *RewardRepository) ListAllClaims(ctx context.Context) ([]rewarddomain.RewardClaim, error) {
	op := "db.rewardrepository.ListAllClaims"
	log := r.logger.With(slog.String("op", op))

	var rows []db.RewardClaim
	err := r.db.NewSelect().Model(&rows).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := make([]rewarddomain.RewardClaim, len(rows))
	for i := range rows {
		out[i] = toDomainClaim(&rows[i])
	}
	return out, nil
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

// --- mappers ---

func toDomainReward(row *db.Reward) rewarddomain.Reward {
	return rewarddomain.Reward{
		ID:          row.ID,
		SeasonID:    row.SeasonID,
		Title:       row.Title,
		Description: row.Description,
		CostCoins:   row.CostCoins,
		Limit:       row.Limit,
		Status:      row.Status,
	}
}

func toDBReward(rw *rewarddomain.Reward) *db.Reward {
	return &db.Reward{
		Entity:      db.Entity{ID: rw.ID},
		SeasonID:    rw.SeasonID,
		Title:       rw.Title,
		Description: rw.Description,
		CostCoins:   rw.CostCoins,
		Limit:       rw.Limit,
		Status:      rw.Status,
	}
}

func toDomainClaim(row *db.RewardClaim) rewarddomain.RewardClaim {
	return rewarddomain.RewardClaim{
		ID:         row.ID,
		UserID:     row.UserID,
		RewardID:   row.RewardID,
		Status:     row.Status,
		SpentCoins: row.SpentCoins,
		CreatedAt:  row.CreatedAt,
	}
}

func toDBClaim(c *rewarddomain.RewardClaim) *db.RewardClaim {
	return &db.RewardClaim{
		Entity:     db.Entity{ID: c.ID},
		UserID:     c.UserID,
		RewardID:   c.RewardID,
		Status:     c.Status,
		SpentCoins: c.SpentCoins,
		CreatedAt:  c.CreatedAt,
	}
}

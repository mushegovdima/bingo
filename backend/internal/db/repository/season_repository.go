package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/uptrace/bun"
	"go.mod/internal/db"
)

type SeasonRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewSeasonRepository(db *bun.DB, logger *slog.Logger) *SeasonRepository {
	return &SeasonRepository{db: db, logger: logger}
}

func (r *SeasonRepository) Insert(ctx context.Context, c *db.Season) error {
	op := "db.seasonrepository.Insert"
	log := r.logger.With(slog.String("op", op))

	_, err := r.db.NewInsert().Model(c).Returning("id").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	log.InfoContext(ctx, "season inserted", slog.Int64("season_id", c.ID))
	return nil
}

func (r *SeasonRepository) Update(ctx context.Context, c *db.Season, columns ...string) error {
	op := "db.seasonrepository.Update"
	log := r.logger.With(slog.String("op", op), slog.Int64("season_id", c.ID))

	_, err := r.db.NewUpdate().Model(c).Column(columns...).Where("id = ?", c.ID).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	log.InfoContext(ctx, "season updated", slog.Any("columns", columns))
	return nil
}

func (r *SeasonRepository) Delete(ctx context.Context, id int64) error {
	op := "db.seasonrepository.Delete"
	log := r.logger.With(slog.String("op", op), slog.Int64("season_id", id))

	_, err := r.db.NewDelete().Model((*db.Season)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	log.InfoContext(ctx, "season deleted")
	return nil
}

func (r *SeasonRepository) GetByID(ctx context.Context, id int64) (*db.Season, error) {
	op := "db.seasonrepository.GetByID"
	log := r.logger.With(slog.String("op", op), slog.Int64("season_id", id))

	c := &db.Season{}
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

func (r *SeasonRepository) List(ctx context.Context) ([]*db.Season, error) {
	op := "db.seasonrepository.List"
	log := r.logger.With(slog.String("op", op))

	var seasons []*db.Season
	err := r.db.NewSelect().Model(&seasons).OrderExpr("id DESC").Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return seasons, nil
}

func (r *SeasonRepository) ListActive(ctx context.Context) ([]*db.Season, error) {
	op := "db.seasonrepository.ListActive"
	log := r.logger.With(slog.String("op", op))

	var seasons []*db.Season
	err := r.db.NewSelect().Model(&seasons).Where("is_active = true").OrderExpr("id DESC").Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return seasons, nil
}

func (r *SeasonRepository) GetActive(ctx context.Context) (*db.Season, error) {
	op := "db.seasonrepository.GetActive"
	log := r.logger.With(slog.String("op", op))

	c := &db.Season{}
	err := r.db.NewSelect().Model(c).Where("is_active = true").Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return c, nil
}

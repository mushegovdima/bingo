package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/uptrace/bun"
	"go.mod/internal/db"
	seasondomain "go.mod/internal/domain/season"
	"log/slog"
)

type SeasonRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewSeasonRepository(db *bun.DB, logger *slog.Logger) *SeasonRepository {
	return &SeasonRepository{db: db, logger: logger}
}

func (r *SeasonRepository) Insert(ctx context.Context, idb bun.IDB, c *seasondomain.Season) error {
	op := "db.seasonrepository.Insert"
	log := r.logger.With(slog.String("op", op))

	row := toDBSeason(c)
	_, err := idb.NewInsert().Model(row).Returning("id").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	c.ID = row.ID
	log.InfoContext(ctx, "season inserted", slog.Int64("season_id", c.ID))
	return nil
}

func (r *SeasonRepository) Update(ctx context.Context, idb bun.IDB, c *seasondomain.Season, columns ...string) error {
	op := "db.seasonrepository.Update"
	log := r.logger.With(slog.String("op", op), slog.Int64("season_id", c.ID))

	row := toDBSeason(c)
	_, err := idb.NewUpdate().Model(row).Column(columns...).Where("id = ?", c.ID).Exec(ctx)
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

func (r *SeasonRepository) GetByID(ctx context.Context, id int64) (*seasondomain.Season, error) {
	op := "db.seasonrepository.GetByID"
	log := r.logger.With(slog.String("op", op), slog.Int64("season_id", id))

	row := &db.Season{}
	err := r.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := toDomainSeason(row)
	return &out, nil
}

func (r *SeasonRepository) List(ctx context.Context) ([]*seasondomain.Season, error) {
	op := "db.seasonrepository.List"
	log := r.logger.With(slog.String("op", op))

	var rows []*db.Season
	err := r.db.NewSelect().Model(&rows).OrderExpr("id DESC").Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := make([]*seasondomain.Season, len(rows))
	for i, row := range rows {
		s := toDomainSeason(row)
		out[i] = &s
	}
	return out, nil
}

func (r *SeasonRepository) ListActive(ctx context.Context) ([]*seasondomain.Season, error) {
	op := "db.seasonrepository.ListActive"
	log := r.logger.With(slog.String("op", op))

	var rows []*db.Season
	err := r.db.NewSelect().Model(&rows).Where("is_active = true").OrderExpr("id DESC").Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := make([]*seasondomain.Season, len(rows))
	for i, row := range rows {
		s := toDomainSeason(row)
		out[i] = &s
	}
	return out, nil
}

func (r *SeasonRepository) GetActive(ctx context.Context) (*seasondomain.Season, error) {
	op := "db.seasonrepository.GetActive"
	log := r.logger.With(slog.String("op", op))

	row := &db.Season{}
	err := r.db.NewSelect().Model(row).Where("is_active = true").Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := toDomainSeason(row)
	return &out, nil
}

func toDBSeason(c *seasondomain.Season) *db.Season {
	return &db.Season{
		Entity:    db.Entity{ID: c.ID},
		Title:     c.Title,
		StartDate: c.StartDate,
		EndDate:   c.EndDate,
		IsActive:  c.IsActive,
	}
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/uptrace/bun"
	"go.mod/internal/db"
)

type TaskRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewTaskRepository(bunDB *bun.DB, logger *slog.Logger) *TaskRepository {
	return &TaskRepository{db: bunDB, logger: logger}
}

func (r *TaskRepository) Insert(ctx context.Context, t *db.Task) error {
	op := "db.taskrepository.Insert"
	log := r.logger.With(slog.String("op", op))

	_, err := r.db.NewInsert().Model(t).Returning("id").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *TaskRepository) Update(ctx context.Context, t *db.Task, columns ...string) error {
	op := "db.taskrepository.Update"
	log := r.logger.With(slog.String("op", op), slog.Int64("task_id", t.ID))

	_, err := r.db.NewUpdate().Model(t).Column(columns...).Where("id = ?", t.ID).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *TaskRepository) Delete(ctx context.Context, id int64) error {
	op := "db.taskrepository.Delete"
	log := r.logger.With(slog.String("op", op), slog.Int64("task_id", id))

	_, err := r.db.NewDelete().Model((*db.Task)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*db.Task, error) {
	op := "db.taskrepository.GetByID"
	log := r.logger.With(slog.String("op", op), slog.Int64("task_id", id))

	t := &db.Task{}
	err := r.db.NewSelect().Model(t).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return t, nil
}

func (r *TaskRepository) ListBySeason(ctx context.Context, seasonID int64) ([]db.Task, error) {
	op := "db.taskrepository.ListBySeason"
	log := r.logger.With(slog.String("op", op), slog.Int64("season_id", seasonID))

	var tasks []db.Task
	err := r.db.NewSelect().Model(&tasks).
		Where("season_id = ?", seasonID).
		OrderExpr("sort_order ASC, id ASC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return tasks, nil
}

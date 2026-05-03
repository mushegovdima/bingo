package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/uptrace/bun"
	"go.mod/internal/db"
	taskdomain "go.mod/internal/domain/task"
	"log/slog"
)

type TaskRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewTaskRepository(bunDB *bun.DB, logger *slog.Logger) *TaskRepository {
	return &TaskRepository{db: bunDB, logger: logger}
}

func (r *TaskRepository) Insert(ctx context.Context, t *taskdomain.Task) error {
	op := "db.taskrepository.Insert"
	log := r.logger.With(slog.String("op", op))

	row := toDBTask(t)
	_, err := r.db.NewInsert().Model(row).Returning("id").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	t.ID = row.ID
	return nil
}

func (r *TaskRepository) Update(ctx context.Context, t *taskdomain.Task, columns ...string) error {
	op := "db.taskrepository.Update"
	log := r.logger.With(slog.String("op", op), slog.Int64("task_id", t.ID))

	row := toDBTask(t)
	_, err := r.db.NewUpdate().Model(row).Column(columns...).Where("id = ?", t.ID).Exec(ctx)
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

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	op := "db.taskrepository.GetByID"
	log := r.logger.With(slog.String("op", op), slog.Int64("task_id", id))

	row := &db.Task{}
	err := r.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := toDomainTask(row)
	return &out, nil
}

func (r *TaskRepository) ListBySeason(ctx context.Context, seasonID int64) ([]taskdomain.Task, error) {
	op := "db.taskrepository.ListBySeason"
	log := r.logger.With(slog.String("op", op), slog.Int64("season_id", seasonID))

	var rows []db.Task
	err := r.db.NewSelect().Model(&rows).
		Where("season_id = ?", seasonID).
		OrderExpr("sort_order ASC, id ASC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := make([]taskdomain.Task, len(rows))
	for i := range rows {
		out[i] = toDomainTask(&rows[i])
	}
	return out, nil
}

func toDomainTask(row *db.Task) taskdomain.Task {
	return taskdomain.Task{
		ID:          row.ID,
		SeasonID:    row.SeasonID,
		Category:    row.Category,
		Title:       row.Title,
		Description: row.Description,
		RewardCoins: row.RewardCoins,
		SortOrder:   row.SortOrder,
		Metadata:    row.Metadata,
		IsActive:    row.IsActive,
	}
}

func toDBTask(t *taskdomain.Task) *db.Task {
	return &db.Task{
		Entity:      db.Entity{ID: t.ID},
		SeasonID:    t.SeasonID,
		Category:    t.Category,
		Title:       t.Title,
		Description: t.Description,
		RewardCoins: t.RewardCoins,
		SortOrder:   t.SortOrder,
		Metadata:    t.Metadata,
		IsActive:    t.IsActive,
	}
}

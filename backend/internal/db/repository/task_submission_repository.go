package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/uptrace/bun"
	"go.mod/internal/db"
)

type TaskSubmissionRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewTaskSubmissionRepository(bunDB *bun.DB, logger *slog.Logger) *TaskSubmissionRepository {
	return &TaskSubmissionRepository{db: bunDB, logger: logger}
}

func (r *TaskSubmissionRepository) Insert(ctx context.Context, s *db.TaskSubmission) error {
	op := "db.submissionrepository.Insert"
	log := r.logger.With(slog.String("op", op))

	_, err := r.db.NewInsert().Model(s).Returning("id, submitted_at").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *TaskSubmissionRepository) Update(ctx context.Context, s *db.TaskSubmission, columns ...string) error {
	op := "db.submissionrepository.Update"
	log := r.logger.With(slog.String("op", op), slog.Int64("submission_id", s.ID))

	_, err := r.db.NewUpdate().Model(s).Column(columns...).Where("id = ?", s.ID).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *TaskSubmissionRepository) Delete(ctx context.Context, id int64) error {
	op := "db.submissionrepository.Delete"
	log := r.logger.With(slog.String("op", op), slog.Int64("submission_id", id))

	_, err := r.db.NewDelete().Model((*db.TaskSubmission)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	return nil
}

func (r *TaskSubmissionRepository) GetByID(ctx context.Context, id int64) (*db.TaskSubmission, error) {
	op := "db.submissionrepository.GetByID"
	log := r.logger.With(slog.String("op", op), slog.Int64("submission_id", id))

	s := &db.TaskSubmission{}
	err := r.db.NewSelect().Model(s).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return s, nil
}

func (r *TaskSubmissionRepository) ListByUser(ctx context.Context, userID int64) ([]db.TaskSubmission, error) {
	op := "db.submissionrepository.ListByUser"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	var items []db.TaskSubmission
	err := r.db.NewSelect().Model(&items).
		Where("user_id = ?", userID).
		OrderExpr("submitted_at DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return items, nil
}

func (r *TaskSubmissionRepository) ListAll(ctx context.Context) ([]db.TaskSubmission, error) {
	op := "db.submissionrepository.ListAll"
	log := r.logger.With(slog.String("op", op))

	var items []db.TaskSubmission
	err := r.db.NewSelect().Model(&items).
		OrderExpr("submitted_at DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return items, nil
}

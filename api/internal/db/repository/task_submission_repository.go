package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/uptrace/bun"
	"go.mod/internal/db"
	submissiondomain "go.mod/internal/domain/submission"
)

type TaskSubmissionRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewTaskSubmissionRepository(bunDB *bun.DB, logger *slog.Logger) *TaskSubmissionRepository {
	return &TaskSubmissionRepository{db: bunDB, logger: logger}
}

func (r *TaskSubmissionRepository) Insert(ctx context.Context, s *submissiondomain.TaskSubmission) error {
	op := "db.submissionrepository.Insert"
	log := r.logger.With(slog.String("op", op))

	row := toDBSubmission(s)
	_, err := r.db.NewInsert().Model(row).Returning("id, submitted_at").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	s.ID = row.ID
	s.SubmittedAt = row.SubmittedAt
	return nil
}

func (r *TaskSubmissionRepository) Update(ctx context.Context, s *submissiondomain.TaskSubmission, columns ...string) error {
	op := "db.submissionrepository.Update"
	log := r.logger.With(slog.String("op", op), slog.Int64("submission_id", s.ID))

	row := toDBSubmission(s)
	_, err := r.db.NewUpdate().Model(row).Column(columns...).Where("id = ?", s.ID).Exec(ctx)
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

func (r *TaskSubmissionRepository) GetByID(ctx context.Context, id int64) (*submissiondomain.TaskSubmission, error) {
	op := "db.submissionrepository.GetByID"
	log := r.logger.With(slog.String("op", op), slog.Int64("submission_id", id))

	row := &db.TaskSubmission{}
	err := r.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := toDomainSubmission(row)
	return &out, nil
}

func (r *TaskSubmissionRepository) ListByUser(ctx context.Context, userID int64) ([]submissiondomain.TaskSubmission, error) {
	op := "db.submissionrepository.ListByUser"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", userID))

	var rows []db.TaskSubmission
	err := r.db.NewSelect().Model(&rows).
		Where("user_id = ?", userID).
		OrderExpr("submitted_at DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := make([]submissiondomain.TaskSubmission, len(rows))
	for i := range rows {
		out[i] = toDomainSubmission(&rows[i])
	}
	return out, nil
}

func (r *TaskSubmissionRepository) GetByUserAndTask(ctx context.Context, userID, taskID int64) (*submissiondomain.TaskSubmission, error) {
	op := "db.submissionrepository.GetByUserAndTask"
	log := r.logger.With(slog.String("op", op))

	row := &db.TaskSubmission{}
	err := r.db.NewSelect().Model(row).
		Where("user_id = ? AND task_id = ?", userID, taskID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := toDomainSubmission(row)
	return &out, nil
}

func (r *TaskSubmissionRepository) ListAll(ctx context.Context) ([]submissiondomain.TaskSubmission, error) {
	op := "db.submissionrepository.ListAll"
	log := r.logger.With(slog.String("op", op))

	var rows []db.TaskSubmission
	err := r.db.NewSelect().Model(&rows).
		OrderExpr("submitted_at DESC").
		Scan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	out := make([]submissiondomain.TaskSubmission, len(rows))
	for i := range rows {
		out[i] = toDomainSubmission(&rows[i])
	}
	return out, nil
}

// --- mappers ---

func toDomainSubmission(row *db.TaskSubmission) submissiondomain.TaskSubmission {
	return submissiondomain.TaskSubmission{
		ID:            row.ID,
		UserID:        row.UserID,
		TaskID:        row.TaskID,
		Status:        row.Status,
		Comment:       row.Comment,
		ReviewComment: row.ReviewComment,
		ReviewerID:    row.ReviewerID,
		SubmittedAt:   row.SubmittedAt,
		ReviewedAt:    row.ReviewedAt,
	}
}

func toDBSubmission(s *submissiondomain.TaskSubmission) *db.TaskSubmission {
	return &db.TaskSubmission{
		Entity:        db.Entity{ID: s.ID},
		UserID:        s.UserID,
		TaskID:        s.TaskID,
		Status:        s.Status,
		Comment:       s.Comment,
		ReviewComment: s.ReviewComment,
		ReviewerID:    s.ReviewerID,
		SubmittedAt:   s.SubmittedAt,
		ReviewedAt:    s.ReviewedAt,
	}
}

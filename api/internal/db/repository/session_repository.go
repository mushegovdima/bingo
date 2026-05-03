package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/uptrace/bun"
	"go.mod/internal/db"
	sessiondomain "go.mod/internal/domain/session"
)

type SessionRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewSessionRepository(db *bun.DB, logger *slog.Logger) *SessionRepository {
	return &SessionRepository{db: db, logger: logger}
}

func (r *SessionRepository) Insert(ctx context.Context, session *db.Session) error {
	op := "db.sessionrepository.Insert"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", session.UserID))

	_, err := r.db.NewInsert().Model(session).Returning("id").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	log.InfoContext(ctx, "session inserted", slog.Int64("session_id", session.ID))
	return nil
}

func (r *SessionRepository) UpdateExpiresAt(ctx context.Context, id int64, expiresAt *time.Time) (*db.Session, error) {
	op := "db.sessionrepository.UpdateExpiresAt"
	log := r.logger.With(slog.String("op", op), slog.Int64("session_id", id))

	session := &db.Session{}
	_, err := r.db.NewUpdate().Model(session).
		Set("expires_at = ?", expiresAt).
		Where("id = ?", id).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return session, nil
}

func (r *SessionRepository) GetByID(ctx context.Context, id int64) (*db.Session, error) {
	op := "db.sessionrepository.GetByID"
	log := r.logger.With(slog.String("op", op), slog.Int64("session_id", id))

	session := &db.Session{}
	err := r.db.NewSelect().Model(session).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		log.DebugContext(ctx, "session not found")
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	return session, nil
}

func (r *SessionRepository) Deactivate(ctx context.Context, id int64) error {
	op := "db.sessionrepository.Deactivate"
	log := r.logger.With(slog.String("op", op), slog.Int64("session_id", id))

	_, err := r.db.NewUpdate().TableExpr("sessions").
		Set("status = ?", sessiondomain.SessionInactive).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	log.InfoContext(ctx, "session deactivated")
	return nil
}

func (r *SessionRepository) Slide(ctx context.Context, id int64, expiresAt time.Time) error {
	op := "db.sessionrepository.Slide"
	log := r.logger.With(slog.String("op", op), slog.Int64("session_id", id))

	_, err := r.db.NewUpdate().TableExpr("sessions").
		Set("expires_at = ?", expiresAt).
		Set("last_activity = current_timestamp").
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	log.DebugContext(ctx, "session slid")
	return nil
}

package sessionservice

import (
	"context"
	"log/slog"
	"time"

	"go.mod/internal/db"
	"go.mod/internal/db/repository"
)

type SessionService struct {
	repo   *repository.SessionRepository
	logger *slog.Logger
}

func NewService(repo *repository.SessionRepository, logger *slog.Logger) *SessionService {
	return &SessionService{
		repo:   repo,
		logger: logger,
	}
}

func (s *SessionService) CreateSession(ctx context.Context, session *db.Session) (*int64, error) {
	op := "sessionservice.CreateSession"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", session.UserID))

	if err := s.repo.Insert(ctx, session); err != nil {
		log.Error("failed to insert session", slog.Any("error", err))
		return nil, err
	}

	return &session.ID, nil
}

func (s *SessionService) SetNewExpiresAt(ctx context.Context, sessionID int64, expiresAt *time.Time) (*db.Session, error) {
	op := "sessionservice.SetNewExpiresAt"
	log := s.logger.With(slog.String("op", op), slog.Int64("session_id", sessionID))

	session := &db.Session{}
	_, err := s.repo.DB().NewUpdate().Model(session).
		Set("expires_at = ?", expiresAt).
		Where("id = ?", sessionID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update session", slog.Any("error", err))
		return nil, err
	}

	return session, nil
}

func (s *SessionService) GetByID(ctx context.Context, id int64) (*db.Session, error) {
	op := "sessionservice.GetByID"
	log := s.logger.With(slog.String("op", op), slog.Int64("session_id", id))

	session, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("failed to get session", slog.Any("error", err))
		return nil, err
	}
	return session, nil
}

func (s *SessionService) Deactivate(ctx context.Context, id int64) error {
	op := "sessionservice.Deactivate"
	log := s.logger.With(slog.String("op", op), slog.Int64("session_id", id))

	if err := s.repo.Deactivate(ctx, id); err != nil {
		log.Error("failed to deactivate session", slog.Any("error", err))
		return err
	}
	log.Info("session deactivated")
	return nil
}

func (s *SessionService) Slide(ctx context.Context, sessionID int64, ttlMinutes int) (time.Time, error) {
	op := "sessionservice.Slide"
	log := s.logger.With(slog.String("op", op), slog.Int64("session_id", sessionID))

	newExpiry := time.Now().Add(time.Duration(ttlMinutes) * time.Minute)
	if err := s.repo.Slide(ctx, sessionID, newExpiry); err != nil {
		log.Error("failed to slide session", slog.Any("error", err))
		return time.Time{}, err
	}
	return newExpiry, nil
}

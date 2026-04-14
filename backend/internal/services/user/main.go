package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.mod/internal/config"
	"go.mod/internal/db"
	"go.mod/internal/db/repository"
	"go.mod/internal/domain"
)

type UserService struct {
	repo   *repository.UserRepository
	logger *slog.Logger
	config *config.Config
}

func NewService(repo *repository.UserRepository, logger *slog.Logger, cfg *config.Config) *UserService {
	return &UserService{
		repo:   repo,
		logger: logger,
		config: cfg,
	}
}

// CreateOrUpdate ищет пользователя по telegramID. Если не найден — создаёт.
// Если найден — обновляет изменившиеся поля (name, username, photo_url).
// Возвращает domain.User.
func (s *UserService) CreateOrUpdate(ctx context.Context, req *UserCreateRequest) (*domain.User, error) {
	op := "userservice.CreateOrUpdate"
	log := s.logger.With(slog.String("op", op), slog.Int64("telegram_id", req.TelegramID))

	existing, err := s.repo.GetByTelegramID(ctx, req.TelegramID)
	if err != nil {
		log.Error("failed to get user by telegram_id", slog.Any("error", err))
		return nil, err
	}

	if existing == nil {
		user := &db.User{
			TelegramID: req.TelegramID,
			Name:       req.Name,
			Username:   req.Username,
			PhotoURL:   req.PhotoURL,
			Roles:      []domain.UserRole{domain.Resident},
		}
		if err := s.repo.Insert(ctx, user); err != nil {
			log.Error("failed to insert user", slog.Any("error", err))
			return nil, err
		}
		log.Info("user created", slog.Int64("user_id", user.ID))
		return toDomain(user), nil
	}

	var changed []string
	if existing.Username != req.Username {
		existing.Username = req.Username
		changed = append(changed, "username")
	}
	if existing.PhotoURL != req.PhotoURL {
		existing.PhotoURL = req.PhotoURL
		changed = append(changed, "photo_url")
	}

	if len(changed) > 0 {
		if err := s.repo.Update(ctx, existing, changed...); err != nil {
			log.Error("failed to update user", slog.Any("error", err))
			return nil, err
		}
		log.Info("user updated", slog.Int64("user_id", existing.ID), slog.Any("fields", changed))
	}

	return toDomain(existing), nil
}

func toDomain(u *db.User) *domain.User {
	return &domain.User{
		ID:         u.ID,
		TelegramID: u.TelegramID,
		Name:       u.Name,
		Username:   u.Username,
		PhotoURL:   u.PhotoURL,
		Roles:      u.Roles,
		IsBlocked:  u.IsBlocked,
		CreatedAt:  u.CreatedAt,
	}
}

func (s *UserService) GetById(ctx context.Context, id int64) (*domain.User, error) {
	op := "userservice.GetById"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", id))

	usr, err := s.repo.GetById(ctx, id)
	if err != nil {
		log.Error("failed to get user by id", slog.Any("error", err))
		return nil, err
	}
	if usr == nil {
		log.Debug("user not found")
		return nil, nil
	}

	log.Debug("user found", slog.Int64("telegram_id", usr.TelegramID))
	return &domain.User{
		ID:         usr.ID,
		TelegramID: usr.TelegramID,
		Name:       usr.Name,
		Username:   usr.Username,
		PhotoURL:   usr.PhotoURL,
		Roles:      usr.Roles,
		IsBlocked:  usr.IsBlocked,
		CreatedAt:  usr.CreatedAt,
	}, nil
}

func (s *UserService) BlockUser(ctx context.Context, id int64) error {
	op := "userservice.BlockUser"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", id))

	user, err := s.repo.GetById(ctx, id)
	if err != nil {
		log.Error("failed to get user", slog.Any("error", err))
		return err
	}
	if user == nil {
		log.Warn("user not found")
		return errors.New("user not found")
	}
	if user.IsBlocked {
		log.Debug("user already blocked")
		return nil
	}

	if err := s.repo.SetIsBlocked(ctx, id, true); err != nil {
		log.Error("failed to block user", slog.Any("error", err))
		return err
	}

	log.Info("user blocked")
	return nil
}

// List returns all users ordered by id ascending.
func (s *UserService) List(ctx context.Context) ([]*domain.User, error) {
	op := "userservice.List"
	log := s.logger.With(slog.String("op", op))

	users, err := s.repo.List(ctx)
	if err != nil {
		log.Error("failed to list users", slog.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result := make([]*domain.User, len(users))
	for i, u := range users {
		result[i] = toDomain(u)
	}
	return result, nil
}

// UpdateUser updates the editable fields (name, roles, is_blocked) of a user.
func (s *UserService) UpdateUser(ctx context.Context, id int64, req UpdateUserRequest) (*domain.User, error) {
	op := "userservice.UpdateUser"
	log := s.logger.With(slog.String("op", op), slog.Int64("user_id", id))

	user, err := s.repo.GetById(ctx, id)
	if err != nil {
		log.Error("failed to get user", slog.Any("error", err))
		return nil, fmt.Errorf("%s: get: %w", op, err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	var changed []string
	if req.Name != "" && user.Name != req.Name {
		user.Name = req.Name
		changed = append(changed, "name")
	}
	if !rolesEqual(user.Roles, req.Roles) {
		user.Roles = req.Roles
		changed = append(changed, "roles")
	}
	if user.IsBlocked != req.IsBlocked {
		user.IsBlocked = req.IsBlocked
		changed = append(changed, "is_blocked")
	}

	if len(changed) > 0 {
		if err := s.repo.Update(ctx, user, changed...); err != nil {
			log.Error("failed to update user", slog.Any("error", err))
			return nil, fmt.Errorf("%s: update: %w", op, err)
		}
		log.Info("user updated", slog.Any("fields", changed))
	}

	return toDomain(user), nil
}

func rolesEqual(a, b []domain.UserRole) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[domain.UserRole]struct{}, len(a))
	for _, r := range a {
		m[r] = struct{}{}
	}
	for _, r := range b {
		if _, ok := m[r]; !ok {
			return false
		}
	}
	return true
}

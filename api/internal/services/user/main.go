package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	usercontract "go.mod/internal/contracts/user"
	"go.mod/internal/db"
	userdomain "go.mod/internal/domain/user"
	"go.mod/internal/notifications"
)

// userRepo is the persistence contract userservice depends on.
// Defined here (not in the repository package) so the service can be unit-tested
// with a fake — Go satisfies it structurally.
type userRepo interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*db.User, error)
	GetById(ctx context.Context, id int64) (*db.User, error)
	Insert(ctx context.Context, user *db.User) error
	Update(ctx context.Context, user *db.User, columns ...string) error
	List(ctx context.Context) ([]*db.User, error)
	SetIsBlocked(ctx context.Context, id int64, isBlocked bool) error
}

type UserService struct {
	repo   userRepo
	logger *slog.Logger
}

func NewService(repo userRepo, logger *slog.Logger) *UserService {
	return &UserService{
		repo:   repo,
		logger: logger,
	}
}

// CreateOrUpdate ищет пользователя по telegramID. Если не найден — создаёт.
// Если найден — обновляет изменившиеся поля (name, username, photo_url).
// Возвращает userdomain.User.
func (s *UserService) CreateOrUpdate(ctx context.Context, req *usercontract.CreateRequest) (*userdomain.User, error) {
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
			Roles:      []userdomain.UserRole{userdomain.Resident},
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

func toDomain(u *db.User) *userdomain.User {
	return &userdomain.User{
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

// GetNotificationUser returns the minimal user profile needed for template rendering.
// Returns nil (without error) when the user is not found.
func (s *UserService) GetNotificationUser(ctx context.Context, id int64) (*notifications.NotificationUser, error) {
	usr, err := s.repo.GetById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("userservice.GetNotificationUser: %w", err)
	}
	if usr == nil {
		return nil, nil
	}
	return &notifications.NotificationUser{
		Name:     usr.Name,
		Username: usr.Username,
	}, nil
}

func (s *UserService) GetById(ctx context.Context, id int64) (*userdomain.User, error) {
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
	return &userdomain.User{
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
func (s *UserService) List(ctx context.Context) ([]*userdomain.User, error) {
	op := "userservice.List"
	log := s.logger.With(slog.String("op", op))

	users, err := s.repo.List(ctx)
	if err != nil {
		log.Error("failed to list users", slog.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result := make([]*userdomain.User, len(users))
	for i, u := range users {
		result[i] = toDomain(u)
	}
	return result, nil
}

// UpdateUser updates the editable fields (name, roles, is_blocked) of a user.
func (s *UserService) UpdateUser(ctx context.Context, id int64, req usercontract.UpdateRequest) (*userdomain.User, error) {
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

func rolesEqual(a, b []userdomain.UserRole) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[userdomain.UserRole]struct{}, len(a))
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

package authservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mod/internal/config"
	"go.mod/internal/db"
	"go.mod/internal/domain"
	userservice "go.mod/internal/services/user"
)

//go:generate mockery --name=userCreator --filename=mock_user_creator_test.go --inpackage
type userCreator interface {
	CreateOrUpdate(ctx context.Context, req *userservice.UserCreateRequest) (*domain.User, error)
}

//go:generate mockery --name=sessionCreator --filename=mock_session_creator_test.go --inpackage
type sessionCreator interface {
	CreateSession(ctx context.Context, session *db.Session) (*int64, error)
}

type AuthService struct {
	logger         *slog.Logger
	cfg            *config.Config
	userService    userCreator
	sessionService sessionCreator
}

func NewService(
	userService userCreator,
	sessionService sessionCreator,
	logger *slog.Logger,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		logger:         logger,
		cfg:            cfg,
		userService:    userService,
		sessionService: sessionService,
	}
}

// AuthenticateUser проверяет подпись Telegram Login Widget, делает upsert пользователя
// и создаёт сессию. Возвращает AuthResult с session_id и user_id.
func (s *AuthService) AuthenticateUser(ctx context.Context, req *UserLoginByTelegramRequest) (*AuthResult, error) {
	op := "authservice.AuthenticateUser"
	log := s.logger.With(slog.String("op", op), slog.Int64("telegram_id", req.ID))

	if err := s.verifyTelegramHash(req); err != nil {
		log.Warn("invalid telegram hash", slog.Any("error", err))
		return nil, errors.New("unauthorized")
	}

	// Данные не должны быть старше 24 часов
	if time.Now().Unix()-req.AuthDate > 86400 {
		log.Warn("auth_date is too old", slog.Int64("auth_date", req.AuthDate))
		return nil, errors.New("telegram auth data is outdated")
	}

	name := req.FirstName
	if req.LastName != "" {
		name += " " + req.LastName
	}

	user, err := s.userService.CreateOrUpdate(ctx, &userservice.UserCreateRequest{
		TelegramID: req.ID,
		Name:       name,
		Username:   req.Username,
		PhotoURL:   req.PhotoURL,
	})

	if err != nil {
		log.Error("failed to upsert user", slog.Any("error", err))
		return nil, err
	}

	if user.IsBlocked {
		log.Warn("user is blocked", slog.Int64("user_id", user.ID))
		return nil, errors.New("user is blocked")
	}

	expiresAt := time.Now().Add(time.Duration(s.cfg.SessionTTLMinutes) * time.Minute)
	sessionID, err := s.sessionService.CreateSession(ctx, &db.Session{
		UserID:    user.ID,
		UserAgent: req.UserAgent,
		IP:        req.IP,
		ExpiresAt: &expiresAt,
		Status:    domain.SessionActive,
	})
	if err != nil {
		log.Error("failed to create session", slog.Any("error", err))
		return nil, err
	}

	log.Info("user authenticated", slog.Int64("user_id", user.ID), slog.Int64("session_id", *sessionID))
	return &AuthResult{SessionID: *sessionID, UserID: user.ID, User: user}, nil
}

// verifyTelegramHash проверяет подпись данных по алгоритму Telegram Login Widget:
// secret_key = SHA256(bot_token)
// hash = hex(HMAC_SHA256(data_check_string, secret_key))
func (s *AuthService) verifyTelegramHash(req *UserLoginByTelegramRequest) error {
	fields := map[string]string{
		"id":         strconv.FormatInt(req.ID, 10),
		"first_name": req.FirstName,
		"auth_date":  strconv.FormatInt(req.AuthDate, 10),
	}
	if req.LastName != "" {
		fields["last_name"] = req.LastName
	}
	if req.Username != "" {
		fields["username"] = req.Username
	}
	if req.PhotoURL != "" {
		fields["photo_url"] = req.PhotoURL
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+fields[k])
	}
	dataCheckString := strings.Join(parts, "\n")

	secretKeyHash := sha256.Sum256([]byte(s.cfg.TGBotToken))

	mac := hmac.New(sha256.New, secretKeyHash[:])
	mac.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedHash), []byte(req.Hash)) {
		return errors.New("hash mismatch")
	}
	return nil
}

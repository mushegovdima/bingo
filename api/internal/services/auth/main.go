package authservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mod/internal/config"
	sessioncontract "go.mod/internal/contracts/session"
	usercontract "go.mod/internal/contracts/user"
	sessiondomain "go.mod/internal/domain/session"
	userdomain "go.mod/internal/domain/user"
)

// Sentinel errors returned by AuthService.
var (
	ErrUnauthorized         = errors.New("unauthorized")
	ErrAuthDataOutdated     = errors.New("telegram auth data is outdated")
	ErrUserBlocked          = errors.New("user is blocked")
	ErrTelegramHashMismatch = errors.New("telegram hash mismatch")
)

//go:generate mockery --name=userCreator --filename=mock_user_creator_test.go --inpackage
type userCreator interface {
	CreateOrUpdate(ctx context.Context, req *usercontract.CreateRequest) (*userdomain.User, error)
}

//go:generate mockery --name=sessionCreator --filename=mock_session_creator_test.go --inpackage
type sessionCreator interface {
	CreateSession(ctx context.Context, in sessioncontract.CreateInput) (int64, error)
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
		return nil, ErrUnauthorized
	}

	// Данные не должны быть старше 24 часов
	if time.Now().Unix()-req.AuthDate > 86400 {
		log.Warn("auth_date is too old", slog.Int64("auth_date", req.AuthDate))
		return nil, ErrAuthDataOutdated
	}

	name := req.FirstName
	if req.LastName != "" {
		name += " " + req.LastName
	}

	user, err := s.userService.CreateOrUpdate(ctx, &usercontract.CreateRequest{
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
		return nil, ErrUserBlocked
	}

	expiresAt := time.Now().Add(time.Duration(s.cfg.SessionTTLMinutes) * time.Minute)
	sessionID, err := s.sessionService.CreateSession(ctx, sessioncontract.CreateInput{
		UserID:    user.ID,
		UserAgent: req.UserAgent,
		IP:        req.IP,
		ExpiresAt: &expiresAt,
		Status:    sessiondomain.SessionActive,
	})
	if err != nil {
		log.Error("failed to create session", slog.Any("error", err))
		return nil, err
	}

	log.Info("user authenticated", slog.Int64("user_id", user.ID), slog.Int64("session_id", sessionID))
	return &AuthResult{SessionID: sessionID, UserID: user.ID, User: user}, nil
}

// verifyTelegramHash проверяет подпись данных по алгоритму Telegram Login Widget:// secret_key = SHA256(bot_token)
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
		return ErrTelegramHashMismatch
	}
	return nil
}

// AuthenticateWebApp validates Telegram Mini App initData and creates a session.
// The HMAC algorithm for Mini Apps differs from Login Widget:
//
//	secret_key = HMAC_SHA256(key="WebAppData", data=bot_token)
//	hash       = HMAC_SHA256(key=secret_key,   data=data_check_string)
func (s *AuthService) AuthenticateWebApp(ctx context.Context, req *WebAppLoginRequest) (*AuthResult, error) {
	op := "authservice.AuthenticateWebApp"
	log := s.logger.With(slog.String("op", op))

	parsed, err := url.ParseQuery(req.InitData)
	if err != nil {
		log.Warn("failed to parse initData", slog.Any("error", err))
		return nil, ErrUnauthorized
	}

	hash := parsed.Get("hash")
	if hash == "" {
		log.Warn("missing hash in initData")
		return nil, ErrUnauthorized
	}

	// Build data_check_string: all fields except hash, sorted, joined with \n
	var parts []string
	for k, vs := range parsed {
		if k == "hash" {
			continue
		}
		parts = append(parts, k+"="+vs[0])
	}
	sort.Strings(parts)
	dataCheckString := strings.Join(parts, "\n")

	// secret_key = HMAC_SHA256(key="WebAppData", data=bot_token)
	mac1 := hmac.New(sha256.New, []byte("WebAppData"))
	mac1.Write([]byte(s.cfg.TGBotToken))
	secretKey := mac1.Sum(nil)

	mac2 := hmac.New(sha256.New, secretKey)
	mac2.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac2.Sum(nil))

	if !hmac.Equal([]byte(expectedHash), []byte(hash)) {
		log.Warn("initData hash mismatch")
		return nil, ErrUnauthorized
	}

	// auth_date must be fresh (24h)
	authDateStr := parsed.Get("auth_date")
	authDate, _ := strconv.ParseInt(authDateStr, 10, 64)
	if time.Now().Unix()-authDate > 86400 {
		log.Warn("initData auth_date is too old")
		return nil, ErrAuthDataOutdated
	}

	// Parse user JSON from initData
	userJSON := parsed.Get("user")
	if userJSON == "" {
		log.Warn("missing user field in initData")
		return nil, ErrUnauthorized
	}
	var tgUser struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
		PhotoURL  string `json:"photo_url"`
	}
	if err := json.Unmarshal([]byte(userJSON), &tgUser); err != nil {
		log.Warn("failed to parse user JSON", slog.Any("error", err))
		return nil, ErrUnauthorized
	}

	name := tgUser.FirstName
	if tgUser.LastName != "" {
		name += " " + tgUser.LastName
	}

	user, err := s.userService.CreateOrUpdate(ctx, &usercontract.CreateRequest{
		TelegramID: tgUser.ID,
		Name:       name,
		Username:   tgUser.Username,
		PhotoURL:   tgUser.PhotoURL,
	})
	if err != nil {
		log.Error("failed to upsert user", slog.Any("error", err))
		return nil, err
	}
	if user.IsBlocked {
		log.Warn("user is blocked", slog.Int64("user_id", user.ID))
		return nil, ErrUserBlocked
	}

	expiresAt := time.Now().Add(time.Duration(s.cfg.SessionTTLMinutes) * time.Minute)
	sessionID, err := s.sessionService.CreateSession(ctx, sessioncontract.CreateInput{
		UserID:    user.ID,
		UserAgent: req.UserAgent,
		IP:        req.IP,
		ExpiresAt: &expiresAt,
		Status:    sessiondomain.SessionActive,
	})
	if err != nil {
		log.Error("failed to create session", slog.Any("error", err))
		return nil, err
	}

	log.Info("webapp user authenticated", slog.Int64("user_id", user.ID), slog.Int64("session_id", sessionID))
	return &AuthResult{SessionID: sessionID, UserID: user.ID, User: user}, nil
}

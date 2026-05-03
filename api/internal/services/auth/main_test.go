package authservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mod/internal/config"
	sessioncontract "go.mod/internal/contracts/session"
	usercontract "go.mod/internal/contracts/user"
	sessiondomain "go.mod/internal/domain/session"
	userdomain "go.mod/internal/domain/user"
	"log/slog"
	"os"
	"sort"
	"testing"
	"time"
)

// --- helpers ---

const testBotToken = "test_bot_token_123"

func buildHash(botToken string, fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, fields[k]))
	}
	dataCheckString := ""
	for i, p := range parts {
		if i > 0 {
			dataCheckString += "\n"
		}
		dataCheckString += p
	}

	secretKey := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(dataCheckString))
	return hex.EncodeToString(mac.Sum(nil))
}

func newTestService(u userCreator, s sessionCreator) *AuthService {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.Config{
		TGBotToken:        testBotToken,
		SessionTTLMinutes: 60,
	}
	return NewService(u, s, logger, cfg)
}

func validRequest() *UserLoginByTelegramRequest {
	authDate := time.Now().Unix()
	fields := map[string]string{
		"id":         "123456",
		"first_name": "Ivan",
		"auth_date":  fmt.Sprintf("%d", authDate),
		"username":   "ivan_test",
	}
	return &UserLoginByTelegramRequest{
		ID:        123456,
		FirstName: "Ivan",
		Username:  "ivan_test",
		AuthDate:  authDate,
		Hash:      buildHash(testBotToken, fields),
		UserAgent: "Mozilla/5.0",
		IP:        "127.0.0.1",
	}
}

// --- mocks ---

type mockUserCreator struct {
	fn func(ctx context.Context, req *usercontract.CreateRequest) (*userdomain.User, error)
}

func (m *mockUserCreator) CreateOrUpdate(ctx context.Context, req *usercontract.CreateRequest) (*userdomain.User, error) {
	return m.fn(ctx, req)
}

type mockSessionCreator struct {
	fn func(ctx context.Context, in sessioncontract.CreateInput) (int64, error)
}

func (m *mockSessionCreator) CreateSession(ctx context.Context, in sessioncontract.CreateInput) (int64, error) {
	return m.fn(ctx, in)
}

// --- tests ---

func TestAuthenticateUser_Success(t *testing.T) {
	req := validRequest()

	userMock := &mockUserCreator{fn: func(_ context.Context, r *usercontract.CreateRequest) (*userdomain.User, error) {
		assert.Equal(t, req.ID, r.TelegramID)
		assert.Equal(t, req.FirstName, r.Name)
		assert.Equal(t, req.Username, r.Username)
		return &userdomain.User{ID: 1, TelegramID: req.ID, Name: req.FirstName}, nil
	}}
	sessionMock := &mockSessionCreator{fn: func(_ context.Context, s sessioncontract.CreateInput) (int64, error) {
		assert.Equal(t, int64(1), s.UserID)
		assert.Equal(t, req.UserAgent, s.UserAgent)
		assert.Equal(t, req.IP, s.IP)
		assert.Equal(t, sessiondomain.SessionActive, s.Status)
		assert.NotNil(t, s.ExpiresAt)
		return 42, nil
	}}

	svc := newTestService(userMock, sessionMock)
	result, err := svc.AuthenticateUser(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(42), result.SessionID)
	assert.Equal(t, int64(1), result.UserID)
	require.NotNil(t, result.User)
	assert.Equal(t, int64(1), result.User.ID)
}

func TestAuthenticateUser_InvalidHash(t *testing.T) {
	req := validRequest()
	req.Hash = "deadbeefdeadbeefdeadbeef"

	svc := newTestService(nil, nil)
	result, err := svc.AuthenticateUser(context.Background(), req)

	assert.Nil(t, result)
	assert.EqualError(t, err, "unauthorized")
}

func TestAuthenticateUser_OutdatedAuthDate(t *testing.T) {
	authDate := time.Now().Add(-25 * time.Hour).Unix()
	fields := map[string]string{
		"id":         "123456",
		"first_name": "Ivan",
		"auth_date":  fmt.Sprintf("%d", authDate),
	}
	req := &UserLoginByTelegramRequest{
		ID:        123456,
		FirstName: "Ivan",
		AuthDate:  authDate,
		Hash:      buildHash(testBotToken, fields),
	}

	svc := newTestService(nil, nil)
	result, err := svc.AuthenticateUser(context.Background(), req)

	assert.Nil(t, result)
	assert.EqualError(t, err, "telegram auth data is outdated")
}

func TestAuthenticateUser_WithLastName(t *testing.T) {
	authDate := time.Now().Unix()
	fields := map[string]string{
		"id":         "123456",
		"first_name": "Ivan",
		"last_name":  "Petrov",
		"auth_date":  fmt.Sprintf("%d", authDate),
	}
	req := &UserLoginByTelegramRequest{
		ID:        123456,
		FirstName: "Ivan",
		LastName:  "Petrov",
		AuthDate:  authDate,
		Hash:      buildHash(testBotToken, fields),
	}

	userMock := &mockUserCreator{fn: func(_ context.Context, r *usercontract.CreateRequest) (*userdomain.User, error) {
		assert.Equal(t, "Ivan Petrov", r.Name)
		return &userdomain.User{ID: 2, Name: r.Name}, nil
	}}
	sessionMock := &mockSessionCreator{fn: func(_ context.Context, _ sessioncontract.CreateInput) (int64, error) {
		return 10, nil
	}}

	svc := newTestService(userMock, sessionMock)
	result, err := svc.AuthenticateUser(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int64(10), result.SessionID)
}

func TestAuthenticateUser_UserServiceError(t *testing.T) {
	req := validRequest()

	userMock := &mockUserCreator{fn: func(_ context.Context, _ *usercontract.CreateRequest) (*userdomain.User, error) {
		return nil, errors.New("db error")
	}}

	svc := newTestService(userMock, nil)
	result, err := svc.AuthenticateUser(context.Background(), req)

	assert.Nil(t, result)
	assert.EqualError(t, err, "db error")
}

func TestAuthenticateUser_SessionServiceError(t *testing.T) {
	req := validRequest()

	userMock := &mockUserCreator{fn: func(_ context.Context, _ *usercontract.CreateRequest) (*userdomain.User, error) {
		return &userdomain.User{ID: 1}, nil
	}}
	sessionMock := &mockSessionCreator{fn: func(_ context.Context, _ sessioncontract.CreateInput) (int64, error) {
		return 0, errors.New("session db error")
	}}

	svc := newTestService(userMock, sessionMock)
	result, err := svc.AuthenticateUser(context.Background(), req)

	assert.Nil(t, result)
	assert.EqualError(t, err, "session db error")
}

func TestAuthenticateUser_OnlyRequiredFields(t *testing.T) {
	authDate := time.Now().Unix()
	fields := map[string]string{
		"id":         "999",
		"first_name": "Test",
		"auth_date":  fmt.Sprintf("%d", authDate),
	}
	req := &UserLoginByTelegramRequest{
		ID:        999,
		FirstName: "Test",
		AuthDate:  authDate,
		Hash:      buildHash(testBotToken, fields),
	}

	userMock := &mockUserCreator{fn: func(_ context.Context, _ *usercontract.CreateRequest) (*userdomain.User, error) {
		return &userdomain.User{ID: 5}, nil
	}}
	sessionMock := &mockSessionCreator{fn: func(_ context.Context, _ sessioncontract.CreateInput) (int64, error) {
		return 55, nil
	}}

	svc := newTestService(userMock, sessionMock)
	_, err := svc.AuthenticateUser(context.Background(), req)
	require.NoError(t, err)
}

package authapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	authapi "go.mod/internal/api/auth"
	"go.mod/internal/config"
	dbmodels "go.mod/internal/db"
	"go.mod/internal/domain"
	"go.mod/internal/middleware"
	authservice "go.mod/internal/services/auth"
)

type fakeAuthenticator struct {
	authenticateFn func(ctx context.Context, req *authservice.UserLoginByTelegramRequest) (*authservice.AuthResult, error)
}

func (f *fakeAuthenticator) AuthenticateUser(ctx context.Context, req *authservice.UserLoginByTelegramRequest) (*authservice.AuthResult, error) {
	if f.authenticateFn != nil {
		return f.authenticateFn(ctx, req)
	}
	return &authservice.AuthResult{
		SessionID: 1,
		UserID:    42,
		User:      &domain.User{ID: 42, Name: "Test User"},
	}, nil
}

type fakeUserSvc struct {
	getByIDFn func(ctx context.Context, id int64) (*domain.User, error)
}

func (f *fakeUserSvc) GetById(ctx context.Context, id int64) (*domain.User, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return &domain.User{ID: id, Name: "Test User"}, nil
}

type fakeSessionCreator struct{}

func (f *fakeSessionCreator) CreateSession(_ context.Context, _ *dbmodels.Session) (*int64, error) {
	id := int64(1)
	return &id, nil
}

func newAuthHandler(auth *fakeAuthenticator, users *fakeUserSvc) http.Handler {
	store := sessions.NewCookieStore([]byte("test-secret-key-for-tests-only!!"))
	cfg := &config.Config{}
	h := authapi.NewHandler(auth, users, &fakeSessionCreator{}, store, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r := chi.NewRouter()
	r.Mount("/", h.Routes(middleware.RequireAuth, middleware.RequireAuth))
	return r
}

func authDo(t *testing.T, handler http.Handler, method, path string, body any, sess *middleware.SessionCtx) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reqBody)
	if sess != nil {
		req = req.WithContext(middleware.WithSession(req.Context(), sess))
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func TestAuthHandler_Login(t *testing.T) {
	t.Run("returns 200 with user on success", func(t *testing.T) {
		handler := newAuthHandler(
			&fakeAuthenticator{
				authenticateFn: func(_ context.Context, _ *authservice.UserLoginByTelegramRequest) (*authservice.AuthResult, error) {
					return &authservice.AuthResult{
						SessionID: 1,
						UserID:    42,
						User:      &domain.User{ID: 42, Name: "Alice"},
					}, nil
				},
			},
			&fakeUserSvc{},
		)
		w := authDo(t, handler, http.MethodPost, "/login", map[string]any{
			"id":         42,
			"first_name": "Alice",
			"auth_date":  1712345678,
			"hash":       "abc123",
		}, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var got domain.User
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if got.Name != "Alice" {
			t.Errorf("expected name=Alice, got %q", got.Name)
		}
	})

	t.Run("returns 401 when authentication fails", func(t *testing.T) {
		handler := newAuthHandler(
			&fakeAuthenticator{
				authenticateFn: func(_ context.Context, _ *authservice.UserLoginByTelegramRequest) (*authservice.AuthResult, error) {
					return nil, errors.New("invalid hash")
				},
			},
			&fakeUserSvc{},
		)
		w := authDo(t, handler, http.MethodPost, "/login", map[string]any{
			"id":        1,
			"auth_date": 123,
			"hash":      "bad",
		}, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns 400 on malformed JSON", func(t *testing.T) {
		handler := newAuthHandler(&fakeAuthenticator{}, &fakeUserSvc{})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("{bad"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("returns 200 always", func(t *testing.T) {
		handler := newAuthHandler(&fakeAuthenticator{}, &fakeUserSvc{})
		w := authDo(t, handler, http.MethodPost, "/logout", nil, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}

func TestAuthHandler_GetMe(t *testing.T) {
	t.Run("returns 200 with current user", func(t *testing.T) {
		handler := newAuthHandler(
			&fakeAuthenticator{},
			&fakeUserSvc{
				getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
					return &domain.User{ID: id, Name: "Bob"}, nil
				},
			},
		)
		w := authDo(t, handler, http.MethodGet, "/me", nil, &middleware.SessionCtx{SessionID: 1, UserID: 7})
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var got domain.User
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if got.Name != "Bob" {
			t.Errorf("expected name=Bob, got %q", got.Name)
		}
	})

	t.Run("returns 404 when user not found", func(t *testing.T) {
		handler := newAuthHandler(
			&fakeAuthenticator{},
			&fakeUserSvc{
				getByIDFn: func(_ context.Context, _ int64) (*domain.User, error) { return nil, nil },
			},
		)
		w := authDo(t, handler, http.MethodGet, "/me", nil, &middleware.SessionCtx{SessionID: 1, UserID: 99})
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 401 when no session", func(t *testing.T) {
		handler := newAuthHandler(&fakeAuthenticator{}, &fakeUserSvc{})
		w := authDo(t, handler, http.MethodGet, "/me", nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns 500 when user service fails", func(t *testing.T) {
		handler := newAuthHandler(
			&fakeAuthenticator{},
			&fakeUserSvc{
				getByIDFn: func(_ context.Context, _ int64) (*domain.User, error) {
					return nil, errors.New("db error")
				},
			},
		)
		w := authDo(t, handler, http.MethodGet, "/me", nil, &middleware.SessionCtx{SessionID: 1, UserID: 1})
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

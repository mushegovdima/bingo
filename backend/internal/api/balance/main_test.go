package balanceapi_test

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
	balanceapi "go.mod/internal/api/balance"
	"go.mod/internal/domain"
	"go.mod/internal/middleware"
	balanceservice "go.mod/internal/services/balance"
)

type fakeBalanceSvc struct {
	getBalanceFn      func(ctx context.Context, userID, seasonID int64) (*domain.SeasonMember, error)
	getTransactionsFn func(ctx context.Context, userID, seasonID int64) ([]domain.Transaction, error)
	changeBalanceFn   func(ctx context.Context, req balanceservice.ChangeBalanceRequest) (*domain.Transaction, error)
}

func (f *fakeBalanceSvc) GetBalance(ctx context.Context, userID, seasonID int64) (*domain.SeasonMember, error) {
	if f.getBalanceFn != nil {
		return f.getBalanceFn(ctx, userID, seasonID)
	}
	return &domain.SeasonMember{UserID: userID, SeasonID: seasonID, Balance: 100}, nil
}

func (f *fakeBalanceSvc) GetTransactions(ctx context.Context, userID, seasonID int64) ([]domain.Transaction, error) {
	if f.getTransactionsFn != nil {
		return f.getTransactionsFn(ctx, userID, seasonID)
	}
	return []domain.Transaction{}, nil
}

func (f *fakeBalanceSvc) ChangeBalance(ctx context.Context, req balanceservice.ChangeBalanceRequest) (*domain.Transaction, error) {
	if f.changeBalanceFn != nil {
		return f.changeBalanceFn(ctx, req)
	}
	return &domain.Transaction{}, nil
}

func (f *fakeBalanceSvc) ListUserBalances(ctx context.Context, userID int64) ([]domain.SeasonMemberWithSeason, error) {
	return nil, nil
}

func (f *fakeBalanceSvc) JoinSeason(ctx context.Context, userID, seasonID int64) (*domain.SeasonMember, error) {
	return nil, nil
}

func passThroughMW(next http.Handler) http.Handler { return next }

func newBalanceHandler(svc *fakeBalanceSvc) http.Handler {
	h := balanceapi.NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r := chi.NewRouter()
	r.Mount("/", h.Routes(passThroughMW))
	return r
}

func balanceDo(t *testing.T, handler http.Handler, method, path string, body any, sess *middleware.SessionCtx) *httptest.ResponseRecorder {
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

var defaultSess = &middleware.SessionCtx{SessionID: 1, UserID: 42}

func TestBalanceHandler_GetBalance(t *testing.T) {
	t.Run("returns 200 with balance", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{
			getBalanceFn: func(_ context.Context, userID, seasonID int64) (*domain.SeasonMember, error) {
				return &domain.SeasonMember{UserID: userID, SeasonID: seasonID, Balance: 50}, nil
			},
		})
		w := balanceDo(t, handler, http.MethodGet, "/1", nil, defaultSess)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var got domain.SeasonMember
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if got.Balance != 50 {
			t.Errorf("expected balance=50, got %d", got.Balance)
		}
	})

	t.Run("returns 200 with zero balance when nil", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{
			getBalanceFn: func(_ context.Context, _, _ int64) (*domain.SeasonMember, error) { return nil, nil },
		})
		w := balanceDo(t, handler, http.MethodGet, "/1", nil, defaultSess)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 401 when no session", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{})
		w := balanceDo(t, handler, http.MethodGet, "/1", nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid season id", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{})
		w := balanceDo(t, handler, http.MethodGet, "/abc", nil, defaultSess)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{
			getBalanceFn: func(_ context.Context, _, _ int64) (*domain.SeasonMember, error) {
				return nil, errors.New("db error")
			},
		})
		w := balanceDo(t, handler, http.MethodGet, "/1", nil, defaultSess)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestBalanceHandler_GetTransactions(t *testing.T) {
	t.Run("returns 200 with transactions", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{
			getTransactionsFn: func(_ context.Context, _, _ int64) ([]domain.Transaction, error) {
				return []domain.Transaction{{ID: 1, Amount: 10}}, nil
			},
		})
		w := balanceDo(t, handler, http.MethodGet, "/1/transactions", nil, defaultSess)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 200 empty when balance not found", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{
			getTransactionsFn: func(_ context.Context, _, _ int64) ([]domain.Transaction, error) {
				return nil, balanceservice.ErrBalanceNotFound
			},
		})
		w := balanceDo(t, handler, http.MethodGet, "/1/transactions", nil, defaultSess)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 401 when no session", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{})
		w := balanceDo(t, handler, http.MethodGet, "/1/transactions", nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{
			getTransactionsFn: func(_ context.Context, _, _ int64) ([]domain.Transaction, error) {
				return nil, errors.New("db error")
			},
		})
		w := balanceDo(t, handler, http.MethodGet, "/1/transactions", nil, defaultSess)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestBalanceHandler_ChangeBalance(t *testing.T) {
	t.Run("returns 200 with transaction", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{
			changeBalanceFn: func(_ context.Context, req balanceservice.ChangeBalanceRequest) (*domain.Transaction, error) {
				return &domain.Transaction{ID: 1, Amount: req.Amount}, nil
			},
		})
		w := balanceDo(t, handler, http.MethodPost, "/1/adjust", map[string]any{
			"user_id": 5,
			"amount":  10,
		}, defaultSess)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 400 when user_id is zero", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{})
		w := balanceDo(t, handler, http.MethodPost, "/1/adjust", map[string]any{
			"user_id": 0,
			"amount":  10,
		}, defaultSess)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 400 when amount is zero", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{})
		w := balanceDo(t, handler, http.MethodPost, "/1/adjust", map[string]any{
			"user_id": 5,
			"amount":  0,
		}, defaultSess)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 400 on malformed JSON", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{})
		req := httptest.NewRequest(http.MethodPost, "/1/adjust", bytes.NewBufferString("{bad"))
		req = req.WithContext(middleware.WithSession(req.Context(), defaultSess))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newBalanceHandler(&fakeBalanceSvc{
			changeBalanceFn: func(_ context.Context, _ balanceservice.ChangeBalanceRequest) (*domain.Transaction, error) {
				return nil, errors.New("db error")
			},
		})
		w := balanceDo(t, handler, http.MethodPost, "/1/adjust", map[string]any{
			"user_id": 5,
			"amount":  10,
		}, defaultSess)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

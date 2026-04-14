package claimapi_test

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
	claimapi "go.mod/internal/api/claim"
	"go.mod/internal/domain"
	"go.mod/internal/middleware"
	rewardservice "go.mod/internal/services/reward"
)

type fakeClaimSvc struct {
	submitClaimFn       func(ctx context.Context, userID, seasonID, rewardID int64) (*domain.RewardClaim, error)
	updateClaimStatusFn func(ctx context.Context, claimID int64, req *rewardservice.UpdateClaimRequest, seasonID int64) (*domain.RewardClaim, error)
	getClaimByIDFn      func(ctx context.Context, id int64) (*domain.RewardClaim, error)
	listClaimsByUserFn  func(ctx context.Context, userID int64) ([]domain.RewardClaim, error)
}

func (f *fakeClaimSvc) SubmitClaim(ctx context.Context, userID, seasonID, rewardID int64) (*domain.RewardClaim, error) {
	if f.submitClaimFn != nil {
		return f.submitClaimFn(ctx, userID, seasonID, rewardID)
	}
	return &domain.RewardClaim{ID: 1}, nil
}

func (f *fakeClaimSvc) UpdateClaimStatus(ctx context.Context, claimID int64, req *rewardservice.UpdateClaimRequest, seasonID int64) (*domain.RewardClaim, error) {
	if f.updateClaimStatusFn != nil {
		return f.updateClaimStatusFn(ctx, claimID, req, seasonID)
	}
	return &domain.RewardClaim{ID: claimID}, nil
}

func (f *fakeClaimSvc) GetClaimByID(ctx context.Context, id int64) (*domain.RewardClaim, error) {
	if f.getClaimByIDFn != nil {
		return f.getClaimByIDFn(ctx, id)
	}
	return &domain.RewardClaim{ID: id}, nil
}

func (f *fakeClaimSvc) ListClaimsByUser(ctx context.Context, userID int64) ([]domain.RewardClaim, error) {
	if f.listClaimsByUserFn != nil {
		return f.listClaimsByUserFn(ctx, userID)
	}
	return []domain.RewardClaim{}, nil
}

func (f *fakeClaimSvc) ListAllClaims(_ context.Context) ([]domain.RewardClaim, error) {
	return nil, nil
}

func passThroughMW(next http.Handler) http.Handler { return next }

func newClaimHandler(svc *fakeClaimSvc) http.Handler {
	h := claimapi.NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r := chi.NewRouter()
	r.Mount("/", h.Routes(passThroughMW))
	return r
}

func claimDo(t *testing.T, handler http.Handler, method, path string, body any, sess *middleware.SessionCtx) *httptest.ResponseRecorder {
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

var defaultSess = &middleware.SessionCtx{SessionID: 1, UserID: 10}

func TestClaimHandler_Submit(t *testing.T) {
	t.Run("returns 201 on success", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			submitClaimFn: func(_ context.Context, _, _, _ int64) (*domain.RewardClaim, error) {
				return &domain.RewardClaim{ID: 5}, nil
			},
		})
		w := claimDo(t, handler, http.MethodPost, "/", map[string]any{"reward_id": 1, "season_id": 2}, defaultSess)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
	})

	t.Run("returns 401 when no session", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{})
		w := claimDo(t, handler, http.MethodPost, "/", map[string]any{"reward_id": 1, "season_id": 2}, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns 400 on malformed JSON", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{})
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad"))
		req = req.WithContext(middleware.WithSession(req.Context(), defaultSess))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 400 when reward unavailable", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			submitClaimFn: func(_ context.Context, _, _, _ int64) (*domain.RewardClaim, error) {
				return nil, rewardservice.ErrRewardUnavailable
			},
		})
		w := claimDo(t, handler, http.MethodPost, "/", map[string]any{"reward_id": 1, "season_id": 2}, defaultSess)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 409 when limit exceeded", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			submitClaimFn: func(_ context.Context, _, _, _ int64) (*domain.RewardClaim, error) {
				return nil, rewardservice.ErrLimitExceeded
			},
		})
		w := claimDo(t, handler, http.MethodPost, "/", map[string]any{"reward_id": 1, "season_id": 2}, defaultSess)
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
	})

	t.Run("returns 422 when insufficient balance", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			submitClaimFn: func(_ context.Context, _, _, _ int64) (*domain.RewardClaim, error) {
				return nil, rewardservice.ErrInsufficientBalance
			},
		})
		w := claimDo(t, handler, http.MethodPost, "/", map[string]any{"reward_id": 1, "season_id": 2}, defaultSess)
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			submitClaimFn: func(_ context.Context, _, _, _ int64) (*domain.RewardClaim, error) {
				return nil, errors.New("db error")
			},
		})
		w := claimDo(t, handler, http.MethodPost, "/", map[string]any{"reward_id": 1, "season_id": 2}, defaultSess)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestClaimHandler_List(t *testing.T) {
	t.Run("returns 200 with own claims", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			listClaimsByUserFn: func(_ context.Context, userID int64) ([]domain.RewardClaim, error) {
				return []domain.RewardClaim{{ID: 1, UserID: userID}}, nil
			},
		})
		w := claimDo(t, handler, http.MethodGet, "/", nil, defaultSess)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 200 filtered by user_id param", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			listClaimsByUserFn: func(_ context.Context, userID int64) ([]domain.RewardClaim, error) {
				return []domain.RewardClaim{{ID: 2, UserID: userID}}, nil
			},
		})
		w := claimDo(t, handler, http.MethodGet, "/?user_id=99", nil, defaultSess)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid user_id param", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{})
		w := claimDo(t, handler, http.MethodGet, "/?user_id=abc", nil, defaultSess)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 401 when no session", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{})
		w := claimDo(t, handler, http.MethodGet, "/", nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			listClaimsByUserFn: func(_ context.Context, _ int64) ([]domain.RewardClaim, error) {
				return nil, errors.New("db error")
			},
		})
		w := claimDo(t, handler, http.MethodGet, "/", nil, defaultSess)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestClaimHandler_Get(t *testing.T) {
	t.Run("returns 200 with claim", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			getClaimByIDFn: func(_ context.Context, id int64) (*domain.RewardClaim, error) {
				return &domain.RewardClaim{ID: id}, nil
			},
		})
		w := claimDo(t, handler, http.MethodGet, "/4", nil, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			getClaimByIDFn: func(_ context.Context, _ int64) (*domain.RewardClaim, error) { return nil, nil },
		})
		w := claimDo(t, handler, http.MethodGet, "/99", nil, nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid id", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{})
		w := claimDo(t, handler, http.MethodGet, "/abc", nil, nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			getClaimByIDFn: func(_ context.Context, _ int64) (*domain.RewardClaim, error) {
				return nil, errors.New("db error")
			},
		})
		w := claimDo(t, handler, http.MethodGet, "/1", nil, nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestClaimHandler_UpdateStatus(t *testing.T) {
	t.Run("returns 200 with updated claim", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			updateClaimStatusFn: func(_ context.Context, claimID int64, _ *rewardservice.UpdateClaimRequest, _ int64) (*domain.RewardClaim, error) {
				return &domain.RewardClaim{ID: claimID}, nil
			},
		})
		w := claimDo(t, handler, http.MethodPatch, "/3/status", map[string]any{"status": "approved", "season_id": 1}, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			updateClaimStatusFn: func(_ context.Context, _ int64, _ *rewardservice.UpdateClaimRequest, _ int64) (*domain.RewardClaim, error) {
				return nil, rewardservice.ErrClaimNotFound
			},
		})
		w := claimDo(t, handler, http.MethodPatch, "/99/status", map[string]any{"status": "approved", "season_id": 1}, nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid id", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{})
		w := claimDo(t, handler, http.MethodPatch, "/abc/status", map[string]any{}, nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newClaimHandler(&fakeClaimSvc{
			updateClaimStatusFn: func(_ context.Context, _ int64, _ *rewardservice.UpdateClaimRequest, _ int64) (*domain.RewardClaim, error) {
				return nil, errors.New("db error")
			},
		})
		w := claimDo(t, handler, http.MethodPatch, "/1/status", map[string]any{"status": "approved", "season_id": 1}, nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

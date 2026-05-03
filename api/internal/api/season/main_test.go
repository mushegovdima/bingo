package seasonapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	seasonapi "go.mod/internal/api/season"
	seasondomain "go.mod/internal/domain/season"
	seasonservice "go.mod/internal/services/season"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- fake service ---

type fakeSeasonSvc struct {
	createFn    func(ctx context.Context, req *seasonservice.CreateRequest) (*seasondomain.Season, error)
	updateFn    func(ctx context.Context, id int64, req *seasonservice.UpdateRequest) (*seasondomain.Season, error)
	deleteFn    func(ctx context.Context, id int64) error
	getActiveFn func(ctx context.Context) (*seasondomain.Season, error)
}

func (f *fakeSeasonSvc) Create(ctx context.Context, req *seasonservice.CreateRequest) (*seasondomain.Season, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return &seasondomain.Season{ID: 1, Title: req.Title}, nil
}

func (f *fakeSeasonSvc) Update(ctx context.Context, id int64, req *seasonservice.UpdateRequest) (*seasondomain.Season, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, id, req)
	}
	return &seasondomain.Season{ID: id}, nil
}

func (f *fakeSeasonSvc) Delete(ctx context.Context, id int64) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func (f *fakeSeasonSvc) GetActive(ctx context.Context) (*seasondomain.Season, error) {
	if f.getActiveFn != nil {
		return f.getActiveFn(ctx)
	}
	return nil, nil
}

func (f *fakeSeasonSvc) GetByID(ctx context.Context, id int64) (*seasondomain.Season, error) {
	return nil, nil
}

func (f *fakeSeasonSvc) List(ctx context.Context) ([]*seasondomain.Season, error) {
	return nil, nil
}

func (f *fakeSeasonSvc) ListActive(ctx context.Context) ([]*seasondomain.Season, error) {
	return nil, nil
}

// --- helpers ---

func newSeasonHandler(svc *fakeSeasonSvc) http.Handler {
	h := seasonapi.NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	return r
}

func do(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	r := httptest.NewRequest(method, path, reqBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

// --- POST / (create) ---

func TestSeasonHandler_Create(t *testing.T) {
	t.Run("returns 201 with created season", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{
			createFn: func(_ context.Context, req *seasonservice.CreateRequest) (*seasondomain.Season, error) {
				return &seasondomain.Season{ID: 42, Title: req.Title, IsActive: req.IsActive}, nil
			},
		})
		w := do(t, handler, http.MethodPost, "/", map[string]any{
			"title":     "Spring 2026",
			"is_active": true,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
		var got seasondomain.Season
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if got.ID != 42 || got.Title != "Spring 2026" {
			t.Errorf("unexpected body: %+v", got)
		}
	})

	t.Run("returns 400 on malformed JSON", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad json"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{
			createFn: func(_ context.Context, _ *seasonservice.CreateRequest) (*seasondomain.Season, error) {
				return nil, errors.New("db error")
			},
		})
		w := do(t, handler, http.MethodPost, "/", map[string]any{"title": "x"})
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

// --- GET /active ---

func TestSeasonHandler_GetActive(t *testing.T) {
	t.Run("returns 200 with active season", func(t *testing.T) {
		now := time.Now()
		handler := newSeasonHandler(&fakeSeasonSvc{
			getActiveFn: func(_ context.Context) (*seasondomain.Season, error) {
				return &seasondomain.Season{ID: 5, Title: "Active", StartDate: now}, nil
			},
		})
		w := do(t, handler, http.MethodGet, "/active", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var got seasondomain.Season
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if got.ID != 5 {
			t.Errorf("expected id=5, got %d", got.ID)
		}
	})

	t.Run("returns 404 when no active season", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{
			getActiveFn: func(_ context.Context) (*seasondomain.Season, error) { return nil, nil },
		})
		w := do(t, handler, http.MethodGet, "/active", nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{
			getActiveFn: func(_ context.Context) (*seasondomain.Season, error) {
				return nil, errors.New("db error")
			},
		})
		w := do(t, handler, http.MethodGet, "/active", nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

// --- PATCH /{id} ---

func TestSeasonHandler_Update(t *testing.T) {
	t.Run("returns 200 with updated season", func(t *testing.T) {
		newTitle := "Updated"
		handler := newSeasonHandler(&fakeSeasonSvc{
			updateFn: func(_ context.Context, id int64, req *seasonservice.UpdateRequest) (*seasondomain.Season, error) {
				return &seasondomain.Season{ID: id, Title: *req.Title}, nil
			},
		})
		w := do(t, handler, http.MethodPatch, "/7", map[string]any{"title": newTitle})
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var got seasondomain.Season
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if got.Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, got.Title)
		}
	})

	t.Run("returns 400 on invalid id", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{})
		w := do(t, handler, http.MethodPatch, "/abc", map[string]any{})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 404 when season not found", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{
			updateFn: func(_ context.Context, _ int64, _ *seasonservice.UpdateRequest) (*seasondomain.Season, error) {
				return nil, seasonservice.ErrNotFound
			},
		})
		w := do(t, handler, http.MethodPatch, "/99", map[string]any{})
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{
			updateFn: func(_ context.Context, _ int64, _ *seasonservice.UpdateRequest) (*seasondomain.Season, error) {
				return nil, errors.New("db error")
			},
		})
		w := do(t, handler, http.MethodPatch, "/1", map[string]any{})
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

// --- DELETE /{id} ---

func TestSeasonHandler_Delete(t *testing.T) {
	t.Run("returns 204 on success", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{})
		w := do(t, handler, http.MethodDelete, "/1", nil)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid id", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{})
		w := do(t, handler, http.MethodDelete, "/abc", nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 409 when season has relations", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{
			deleteFn: func(_ context.Context, _ int64) error {
				return seasonservice.ErrHasRelations
			},
		})
		w := do(t, handler, http.MethodDelete, "/1", nil)
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newSeasonHandler(&fakeSeasonSvc{
			deleteFn: func(_ context.Context, _ int64) error {
				return errors.New("db error")
			},
		})
		w := do(t, handler, http.MethodDelete, "/1", nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

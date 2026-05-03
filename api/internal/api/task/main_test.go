package taskapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	taskapi "go.mod/internal/api/task"
	taskdomain "go.mod/internal/domain/task"
	taskservice "go.mod/internal/services/task"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- fake service ---

type fakeTaskSvc struct {
	createFn       func(ctx context.Context, req *taskservice.CreateRequest) (*taskdomain.Task, error)
	updateFn       func(ctx context.Context, id int64, req *taskservice.UpdateRequest) (*taskdomain.Task, error)
	deleteFn       func(ctx context.Context, id int64) error
	getByIDFn      func(ctx context.Context, id int64) (*taskdomain.Task, error)
	listBySeasonFn func(ctx context.Context, seasonID int64) ([]taskdomain.Task, error)
}

func (f *fakeTaskSvc) Create(ctx context.Context, req *taskservice.CreateRequest) (*taskdomain.Task, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return &taskdomain.Task{ID: 1}, nil
}

func (f *fakeTaskSvc) Update(ctx context.Context, id int64, req *taskservice.UpdateRequest) (*taskdomain.Task, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, id, req)
	}
	return &taskdomain.Task{ID: id}, nil
}

func (f *fakeTaskSvc) Delete(ctx context.Context, id int64) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func (f *fakeTaskSvc) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return &taskdomain.Task{ID: id}, nil
}

func (f *fakeTaskSvc) ListBySeason(ctx context.Context, seasonID int64) ([]taskdomain.Task, error) {
	if f.listBySeasonFn != nil {
		return f.listBySeasonFn(ctx, seasonID)
	}
	return []taskdomain.Task{}, nil
}

// --- helpers ---

func passThroughMW(next http.Handler) http.Handler { return next }

func newTaskHandler(svc *fakeTaskSvc) http.Handler {
	h := taskapi.NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r := chi.NewRouter()
	r.Mount("/", h.Routes(passThroughMW))
	return r
}

func taskDo(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
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

// --- GET /?season_id=X ---

func TestTaskHandler_List(t *testing.T) {
	t.Run("returns 200 with tasks", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			listBySeasonFn: func(_ context.Context, seasonID int64) ([]taskdomain.Task, error) {
				return []taskdomain.Task{{ID: 1, SeasonID: seasonID}}, nil
			},
		})
		w := taskDo(t, handler, http.MethodGet, "/?season_id=5", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var got []taskdomain.Task
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("expected 1 task, got %d", len(got))
		}
	})

	t.Run("returns 400 when season_id missing", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{})
		w := taskDo(t, handler, http.MethodGet, "/", nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 400 when season_id invalid", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{})
		w := taskDo(t, handler, http.MethodGet, "/?season_id=abc", nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			listBySeasonFn: func(_ context.Context, _ int64) ([]taskdomain.Task, error) {
				return nil, errors.New("db error")
			},
		})
		w := taskDo(t, handler, http.MethodGet, "/?season_id=1", nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

// --- GET /{id} ---

func TestTaskHandler_Get(t *testing.T) {
	t.Run("returns 200 with task", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			getByIDFn: func(_ context.Context, id int64) (*taskdomain.Task, error) {
				return &taskdomain.Task{ID: id, Title: "My Task"}, nil
			},
		})
		w := taskDo(t, handler, http.MethodGet, "/3", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var got taskdomain.Task
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if got.ID != 3 {
			t.Errorf("expected id=3, got %d", got.ID)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			getByIDFn: func(_ context.Context, _ int64) (*taskdomain.Task, error) { return nil, nil },
		})
		w := taskDo(t, handler, http.MethodGet, "/99", nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid id", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{})
		w := taskDo(t, handler, http.MethodGet, "/abc", nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			getByIDFn: func(_ context.Context, _ int64) (*taskdomain.Task, error) {
				return nil, errors.New("db error")
			},
		})
		w := taskDo(t, handler, http.MethodGet, "/1", nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

// --- POST / ---

func TestTaskHandler_Create(t *testing.T) {
	t.Run("returns 201 with created task", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			createFn: func(_ context.Context, req *taskservice.CreateRequest) (*taskdomain.Task, error) {
				return &taskdomain.Task{ID: 10, Title: req.Title}, nil
			},
		})
		w := taskDo(t, handler, http.MethodPost, "/", map[string]any{"title": "Do it", "season_id": 1})
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
		var got taskdomain.Task
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if got.ID != 10 {
			t.Errorf("expected id=10, got %d", got.ID)
		}
	})

	t.Run("returns 400 on malformed JSON", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{})
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			createFn: func(_ context.Context, _ *taskservice.CreateRequest) (*taskdomain.Task, error) {
				return nil, errors.New("db error")
			},
		})
		w := taskDo(t, handler, http.MethodPost, "/", map[string]any{"title": "x"})
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

// --- PATCH /{id} ---

func TestTaskHandler_Update(t *testing.T) {
	t.Run("returns 200 with updated task", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			updateFn: func(_ context.Context, id int64, req *taskservice.UpdateRequest) (*taskdomain.Task, error) {
				return &taskdomain.Task{ID: id, Title: *req.Title}, nil
			},
		})
		w := taskDo(t, handler, http.MethodPatch, "/5", map[string]any{"title": "New Title"})
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid id", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{})
		w := taskDo(t, handler, http.MethodPatch, "/abc", map[string]any{})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			updateFn: func(_ context.Context, _ int64, _ *taskservice.UpdateRequest) (*taskdomain.Task, error) {
				return nil, taskservice.ErrNotFound
			},
		})
		w := taskDo(t, handler, http.MethodPatch, "/1", map[string]any{})
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			updateFn: func(_ context.Context, _ int64, _ *taskservice.UpdateRequest) (*taskdomain.Task, error) {
				return nil, errors.New("db error")
			},
		})
		w := taskDo(t, handler, http.MethodPatch, "/1", map[string]any{})
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

// --- DELETE /{id} ---

func TestTaskHandler_Delete(t *testing.T) {
	t.Run("returns 204 on success", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{})
		w := taskDo(t, handler, http.MethodDelete, "/1", nil)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid id", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{})
		w := taskDo(t, handler, http.MethodDelete, "/abc", nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 409 when has relations", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			deleteFn: func(_ context.Context, _ int64) error {
				return taskservice.ErrHasRelations
			},
		})
		w := taskDo(t, handler, http.MethodDelete, "/1", nil)
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newTaskHandler(&fakeTaskSvc{
			deleteFn: func(_ context.Context, _ int64) error {
				return errors.New("db error")
			},
		})
		w := taskDo(t, handler, http.MethodDelete, "/1", nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

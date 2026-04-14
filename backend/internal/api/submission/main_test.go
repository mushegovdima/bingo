package submissionapi_test

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
	submissionapi "go.mod/internal/api/submission"
	"go.mod/internal/domain"
	"go.mod/internal/middleware"
	submissionservice "go.mod/internal/services/submission"
)

type fakeSubmissionSvc struct {
	createFn     func(ctx context.Context, reviewerID int64, req *submissionservice.CreateRequest) (*domain.TaskSubmission, error)
	getByIDFn    func(ctx context.Context, id int64) (*domain.TaskSubmission, error)
	listByUserFn func(ctx context.Context, userID int64) ([]domain.TaskSubmission, error)
	deleteFn     func(ctx context.Context, id int64) error
}

func (f *fakeSubmissionSvc) Create(ctx context.Context, reviewerID int64, req *submissionservice.CreateRequest) (*domain.TaskSubmission, error) {
	if f.createFn != nil {
		return f.createFn(ctx, reviewerID, req)
	}
	return &domain.TaskSubmission{ID: 1}, nil
}

func (f *fakeSubmissionSvc) GetByID(ctx context.Context, id int64) (*domain.TaskSubmission, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return &domain.TaskSubmission{ID: id}, nil
}

func (f *fakeSubmissionSvc) ListByUser(ctx context.Context, userID int64) ([]domain.TaskSubmission, error) {
	if f.listByUserFn != nil {
		return f.listByUserFn(ctx, userID)
	}
	return []domain.TaskSubmission{}, nil
}

func (f *fakeSubmissionSvc) ListAll(_ context.Context) ([]domain.TaskSubmission, error) {
	return nil, nil
}

func (f *fakeSubmissionSvc) Delete(ctx context.Context, id int64) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func passThroughMW(next http.Handler) http.Handler { return next }

func newSubmissionHandler(svc *fakeSubmissionSvc) http.Handler {
	h := submissionapi.NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r := chi.NewRouter()
	r.Mount("/", h.Routes(passThroughMW))
	return r
}

func subDo(t *testing.T, handler http.Handler, method, path string, body any, sess *middleware.SessionCtx) *httptest.ResponseRecorder {
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

var defaultSess = &middleware.SessionCtx{SessionID: 1, UserID: 7}

func TestSubmissionHandler_List(t *testing.T) {
	t.Run("returns 200 with own submissions", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{
			listByUserFn: func(_ context.Context, userID int64) ([]domain.TaskSubmission, error) {
				return []domain.TaskSubmission{{ID: 1, UserID: userID}}, nil
			},
		})
		w := subDo(t, handler, http.MethodGet, "/", nil, defaultSess)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 200 filtered by user_id param", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{
			listByUserFn: func(_ context.Context, _ int64) ([]domain.TaskSubmission, error) {
				return []domain.TaskSubmission{{ID: 2}}, nil
			},
		})
		w := subDo(t, handler, http.MethodGet, "/?user_id=55", nil, defaultSess)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid user_id param", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{})
		w := subDo(t, handler, http.MethodGet, "/?user_id=abc", nil, defaultSess)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 401 when no session", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{})
		w := subDo(t, handler, http.MethodGet, "/", nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{
			listByUserFn: func(_ context.Context, _ int64) ([]domain.TaskSubmission, error) {
				return nil, errors.New("db error")
			},
		})
		w := subDo(t, handler, http.MethodGet, "/", nil, defaultSess)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestSubmissionHandler_Get(t *testing.T) {
	t.Run("returns 200 with submission", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{
			getByIDFn: func(_ context.Context, id int64) (*domain.TaskSubmission, error) {
				return &domain.TaskSubmission{ID: id}, nil
			},
		})
		w := subDo(t, handler, http.MethodGet, "/2", nil, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{
			getByIDFn: func(_ context.Context, _ int64) (*domain.TaskSubmission, error) { return nil, nil },
		})
		w := subDo(t, handler, http.MethodGet, "/99", nil, nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid id", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{})
		w := subDo(t, handler, http.MethodGet, "/abc", nil, nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{
			getByIDFn: func(_ context.Context, _ int64) (*domain.TaskSubmission, error) {
				return nil, errors.New("db error")
			},
		})
		w := subDo(t, handler, http.MethodGet, "/1", nil, nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestSubmissionHandler_Create(t *testing.T) {
	t.Run("returns 201 with reviewer id from session", func(t *testing.T) {
		var gotReviewerID int64
		handler := newSubmissionHandler(&fakeSubmissionSvc{
			createFn: func(_ context.Context, reviewerID int64, _ *submissionservice.CreateRequest) (*domain.TaskSubmission, error) {
				gotReviewerID = reviewerID
				return &domain.TaskSubmission{ID: 1}, nil
			},
		})
		w := subDo(t, handler, http.MethodPost, "/", map[string]any{"task_id": 1, "user_id": 3}, defaultSess)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
		if gotReviewerID != defaultSess.UserID {
			t.Errorf("expected reviewerID=%d, got %d", defaultSess.UserID, gotReviewerID)
		}
	})

	t.Run("returns 401 when no session", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{})
		w := subDo(t, handler, http.MethodPost, "/", map[string]any{"task_id": 1}, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns 400 on malformed JSON", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{})
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{bad"))
		req = req.WithContext(middleware.WithSession(req.Context(), defaultSess))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{
			createFn: func(_ context.Context, _ int64, _ *submissionservice.CreateRequest) (*domain.TaskSubmission, error) {
				return nil, errors.New("db error")
			},
		})
		w := subDo(t, handler, http.MethodPost, "/", map[string]any{"task_id": 1}, defaultSess)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestSubmissionHandler_Delete(t *testing.T) {
	t.Run("returns 204 on success", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{})
		w := subDo(t, handler, http.MethodDelete, "/1", nil, nil)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid id", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{})
		w := subDo(t, handler, http.MethodDelete, "/abc", nil, nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{
			deleteFn: func(_ context.Context, _ int64) error {
				return submissionservice.ErrNotFound
			},
		})
		w := subDo(t, handler, http.MethodDelete, "/99", nil, nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		handler := newSubmissionHandler(&fakeSubmissionSvc{
			deleteFn: func(_ context.Context, _ int64) error {
				return errors.New("db error")
			},
		})
		w := subDo(t, handler, http.MethodDelete, "/1", nil, nil)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

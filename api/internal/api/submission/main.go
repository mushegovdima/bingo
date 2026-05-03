package submissionapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	submissiondomain "go.mod/internal/domain/submission"
	userdomain "go.mod/internal/domain/user"
	"go.mod/internal/httpresp"
	"go.mod/internal/middleware"
	submissionservice "go.mod/internal/services/submission"
)

type service interface {
	Create(ctx context.Context, reviewerID int64, req *submissionservice.CreateRequest) (*submissiondomain.TaskSubmission, error)
	Submit(ctx context.Context, userID int64, req *submissionservice.SubmitRequest) (*submissiondomain.TaskSubmission, error)
	GetByID(ctx context.Context, id int64) (*submissiondomain.TaskSubmission, error)
	ListByUser(ctx context.Context, userID int64) ([]submissiondomain.TaskSubmission, error)
	ListAll(ctx context.Context) ([]submissiondomain.TaskSubmission, error)
	Delete(ctx context.Context, id int64) error
	Approve(ctx context.Context, reviewerID, submissionID int64) (*submissiondomain.TaskSubmission, error)
	Reject(ctx context.Context, reviewerID, submissionID int64, comment string) (*submissiondomain.TaskSubmission, error)
}

type userFinder interface {
	GetById(ctx context.Context, id int64) (*userdomain.User, error)
}

// SubmissionResponse extends TaskSubmission with reviewer display info.
type SubmissionResponse struct {
	ID            int64                                 `json:"id"`
	UserID        int64                                 `json:"user_id"`
	UserName      *string                               `json:"user_name"`
	TaskID        int64                                 `json:"task_id"`
	Status        submissiondomain.TaskSubmissionStatus `json:"status"`
	Comment       string                                `json:"comment"`
	ReviewComment string                                `json:"review_comment"`
	ReviewerID    *int64                                `json:"reviewer_id"`
	ReviewerName  *string                               `json:"reviewer_name"`
	SubmittedAt   time.Time                             `json:"submitted_at"`
	ReviewedAt    *time.Time                            `json:"reviewed_at"`
}

type Handler struct {
	svc     service
	userSvc userFinder
	logger  *slog.Logger
}

func NewHandler(svc service, userSvc userFinder, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, userSvc: userSvc, logger: logger}
}

func (h *Handler) toResponse(ctx context.Context, s submissiondomain.TaskSubmission) SubmissionResponse {
	resp := SubmissionResponse{
		ID:            s.ID,
		UserID:        s.UserID,
		TaskID:        s.TaskID,
		Status:        s.Status,
		Comment:       s.Comment,
		ReviewComment: s.ReviewComment,
		ReviewerID:    s.ReviewerID,
		SubmittedAt:   s.SubmittedAt,
		ReviewedAt:    s.ReviewedAt,
	}
	if u, err := h.userSvc.GetById(ctx, s.UserID); err == nil && u != nil {
		name := u.Name
		resp.UserName = &name
	}
	if s.ReviewerID != nil {
		if u, err := h.userSvc.GetById(ctx, *s.ReviewerID); err == nil && u != nil {
			name := u.Name
			resp.ReviewerName = &name
		}
	}
	return resp
}

func (h *Handler) toResponses(ctx context.Context, items []submissiondomain.TaskSubmission) []SubmissionResponse {
	out := make([]SubmissionResponse, len(items))
	for i := range items {
		out[i] = h.toResponse(ctx, items[i])
	}
	return out
}

func (h *Handler) Routes(managerOnly func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Resident-accessible routes (outer group provides RequireAuth).
	r.Post("/submit", h.submit)
	r.Get("/{id}", h.get)
	r.Delete("/{id}", h.delete)

	// Manager-only routes.
	r.Group(func(r chi.Router) {
		r.Use(managerOnly)
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/all", h.listAll)
		r.Post("/{id}/approve", h.approve)
		r.Post("/{id}/reject", h.reject)
	})

	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.list"
	log := h.logger.With(slog.String("op", op))

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userIDParam := r.URL.Query().Get("user_id")
	if userIDParam != "" {
		targetID, err := strconv.ParseInt(userIDParam, 10, 64)
		if err != nil {
			httpresp.Err(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		items, err := h.svc.ListByUser(r.Context(), targetID)
		if err != nil {
			log.Error("failed to list submissions", slog.Any("error", err))
			httpresp.Err(w, http.StatusInternalServerError, "internal server error")
			return
		}
		httpresp.OK(w, h.toResponses(r.Context(), items))
		return
	}

	items, err := h.svc.ListByUser(r.Context(), sess.UserID)
	if err != nil {
		log.Error("failed to list own submissions", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, h.toResponses(r.Context(), items))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.get"
	log := h.logger.With(slog.String("op", op))

	id, ok := httpresp.PathInt64(w, r, "id")
	if !ok {
		return
	}

	sub, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		log.Error("failed to get submission", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if sub == nil {
		httpresp.Err(w, http.StatusNotFound, "submission not found")
		return
	}
	httpresp.OK(w, sub)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.create"
	log := h.logger.With(slog.String("op", op))

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	req, err := httpresp.DecodeJSON[submissionservice.CreateRequest](r)
	if err != nil {
		httpresp.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	sub, err := h.svc.Create(r.Context(), sess.UserID, &req)
	if err != nil {
		log.Error("failed to create submission", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.Created(w, sub)
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.submit"
	log := h.logger.With(slog.String("op", op))

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	req, err := httpresp.DecodeJSON[submissionservice.SubmitRequest](r)
	if err != nil {
		httpresp.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	sub, err := h.svc.Submit(r.Context(), sess.UserID, &req)
	if err != nil {
		if errors.Is(err, submissiondomain.ErrEmptyComment) {
			httpresp.Err(w, http.StatusBadRequest, "comment is required")
			return
		}
		if errors.Is(err, submissiondomain.ErrAlreadySubmitted) {
			httpresp.Err(w, http.StatusConflict, "task already submitted")
			return
		}
		log.Error("failed to submit task", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.Created(w, sub)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.delete"
	log := h.logger.With(slog.String("op", op))

	id, ok := httpresp.PathInt64(w, r, "id")
	if !ok {
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, submissionservice.ErrNotFound) {
			httpresp.Err(w, http.StatusNotFound, "submission not found")
			return
		}
		log.Error("failed to delete submission", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listAll(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.listAll"
	log := h.logger.With(slog.String("op", op))

	items, err := h.svc.ListAll(r.Context())
	if err != nil {
		log.Error("failed to list all submissions", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, h.toResponses(r.Context(), items))
}

type rejectRequest struct {
	Comment string `json:"comment"`
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.approve"
	log := h.logger.With(slog.String("op", op))

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := httpresp.PathInt64(w, r, "id")
	if !ok {
		return
	}

	sub, err := h.svc.Approve(r.Context(), sess.UserID, id)
	if err != nil {
		if errors.Is(err, submissionservice.ErrNotFound) {
			httpresp.Err(w, http.StatusNotFound, "submission not found")
			return
		}
		if errors.Is(err, submissiondomain.ErrNotPending) {
			httpresp.Err(w, http.StatusConflict, "submission is not pending")
			return
		}
		log.Error("failed to approve submission", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, h.toResponse(r.Context(), *sub))
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.reject"
	log := h.logger.With(slog.String("op", op))

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := httpresp.PathInt64(w, r, "id")
	if !ok {
		return
	}

	req, err := httpresp.DecodeJSON[rejectRequest](r)
	if err != nil {
		httpresp.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	sub, err := h.svc.Reject(r.Context(), sess.UserID, id, req.Comment)
	if err != nil {
		if errors.Is(err, submissionservice.ErrNotFound) {
			httpresp.Err(w, http.StatusNotFound, "submission not found")
			return
		}
		if errors.Is(err, submissiondomain.ErrNotPending) {
			httpresp.Err(w, http.StatusConflict, "submission is not pending")
			return
		}
		if errors.Is(err, submissiondomain.ErrEmptyComment) {
			httpresp.Err(w, http.StatusBadRequest, "comment is required for rejection")
			return
		}
		log.Error("failed to reject submission", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, h.toResponse(r.Context(), *sub))
}

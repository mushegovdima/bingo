package submissionapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mod/internal/api/response"
	"go.mod/internal/domain"
	"go.mod/internal/middleware"
	submissionservice "go.mod/internal/services/submission"
)

type service interface {
	Create(ctx context.Context, reviewerID int64, req *submissionservice.CreateRequest) (*domain.TaskSubmission, error)
	GetByID(ctx context.Context, id int64) (*domain.TaskSubmission, error)
	ListByUser(ctx context.Context, userID int64) ([]domain.TaskSubmission, error)
	ListAll(ctx context.Context) ([]domain.TaskSubmission, error)
	Delete(ctx context.Context, id int64) error
}

type Handler struct {
	svc    service
	logger *slog.Logger
}

func NewHandler(svc service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) Routes(requireAuth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/", h.list)
		r.Post("/", h.create)
	})
	r.Get("/{id}", h.get)
	r.Delete("/{id}", h.delete)
	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.list"
	log := h.logger.With(slog.String("op", op))

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		response.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userIDParam := r.URL.Query().Get("user_id")
	if userIDParam != "" {
		targetID, err := strconv.ParseInt(userIDParam, 10, 64)
		if err != nil {
			response.Err(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		items, err := h.svc.ListByUser(r.Context(), targetID)
		if err != nil {
			log.Error("failed to list submissions", slog.Any("error", err))
			response.Err(w, http.StatusInternalServerError, "internal server error")
			return
		}
		response.OK(w, items)
		return
	}

	items, err := h.svc.ListByUser(r.Context(), sess.UserID)
	if err != nil {
		log.Error("failed to list own submissions", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, items)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.get"
	log := h.logger.With(slog.String("op", op))

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	sub, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		log.Error("failed to get submission", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if sub == nil {
		response.Err(w, http.StatusNotFound, "submission not found")
		return
	}
	response.OK(w, sub)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.create"
	log := h.logger.With(slog.String("op", op))

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		response.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	req, err := response.DecodeJSON[submissionservice.CreateRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	sub, err := h.svc.Create(r.Context(), sess.UserID, &req)
	if err != nil {
		log.Error("failed to create submission", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.Created(w, sub)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	op := "submissionapi.delete"
	log := h.logger.With(slog.String("op", op))

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, submissionservice.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "submission not found")
			return
		}
		log.Error("failed to delete submission", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

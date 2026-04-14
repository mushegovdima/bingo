package taskapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mod/internal/api/response"
	"go.mod/internal/domain"
	taskservice "go.mod/internal/services/task"
)

type service interface {
	Create(ctx context.Context, req *taskservice.CreateRequest) (*domain.Task, error)
	Update(ctx context.Context, id int64, req *taskservice.UpdateRequest) (*domain.Task, error)
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.Task, error)
	ListBySeason(ctx context.Context, seasonID int64) ([]domain.Task, error)
}

type Handler struct {
	svc    service
	logger *slog.Logger
}

func NewHandler(svc service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) Routes(managerMW func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.With(managerMW).Post("/", h.create)
	r.With(managerMW).Patch("/{id}", h.update)
	r.With(managerMW).Delete("/{id}", h.delete)
	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	op := "taskapi.list"
	log := h.logger.With(slog.String("op", op))

	seasonIDStr := r.URL.Query().Get("season_id")
	if seasonIDStr == "" {
		response.Err(w, http.StatusBadRequest, "season_id query param is required")
		return
	}
	seasonID, err := strconv.ParseInt(seasonIDStr, 10, 64)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid season_id")
		return
	}

	tasks, err := h.svc.ListBySeason(r.Context(), seasonID)
	if err != nil {
		log.Error("failed to list tasks", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, tasks)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	op := "taskapi.get"
	log := h.logger.With(slog.String("op", op))

	id, err := parseID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	task, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		log.Error("failed to get task", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if task == nil {
		response.Err(w, http.StatusNotFound, "task not found")
		return
	}
	response.OK(w, task)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	op := "taskapi.create"
	log := h.logger.With(slog.String("op", op))

	req, err := response.DecodeJSON[taskservice.CreateRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	task, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		log.Error("failed to create task", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.Created(w, task)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	op := "taskapi.update"
	log := h.logger.With(slog.String("op", op))

	id, err := parseID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	req, err := response.DecodeJSON[taskservice.UpdateRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	task, err := h.svc.Update(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, taskservice.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error("failed to update task", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, task)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	op := "taskapi.delete"
	log := h.logger.With(slog.String("op", op))

	id, err := parseID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, taskservice.ErrHasRelations) {
			response.Err(w, http.StatusConflict, "task has related records and cannot be deleted")
			return
		}
		log.Error("failed to delete task", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

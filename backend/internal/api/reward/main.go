package rewardapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mod/internal/api/response"
	"go.mod/internal/domain"
	rewardservice "go.mod/internal/services/reward"
)

type service interface {
	Create(ctx context.Context, req *rewardservice.CreateRewardRequest) (*domain.Reward, error)
	Update(ctx context.Context, id int64, req *rewardservice.UpdateRewardRequest) (*domain.Reward, error)
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*domain.Reward, error)
	ListBySeason(ctx context.Context, seasonID int64) ([]domain.Reward, error)
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
	op := "rewardapi.list"
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

	items, err := h.svc.ListBySeason(r.Context(), seasonID)
	if err != nil {
		log.Error("failed to list rewards", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, items)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	op := "rewardapi.get"
	log := h.logger.With(slog.String("op", op))

	id, err := parseID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	rw, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		log.Error("failed to get reward", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if rw == nil {
		response.Err(w, http.StatusNotFound, "reward not found")
		return
	}
	response.OK(w, rw)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	op := "rewardapi.create"
	log := h.logger.With(slog.String("op", op))

	req, err := response.DecodeJSON[rewardservice.CreateRewardRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	rw, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		log.Error("failed to create reward", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.Created(w, rw)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	op := "rewardapi.update"
	log := h.logger.With(slog.String("op", op))

	id, err := parseID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	req, err := response.DecodeJSON[rewardservice.UpdateRewardRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	rw, err := h.svc.Update(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, rewardservice.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "reward not found")
			return
		}
		log.Error("failed to update reward", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, rw)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	op := "rewardapi.delete"
	log := h.logger.With(slog.String("op", op))

	id, err := parseID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, rewardservice.ErrHasRelations) {
			response.Err(w, http.StatusConflict, "reward has related records and cannot be deleted")
			return
		}
		log.Error("failed to delete reward", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

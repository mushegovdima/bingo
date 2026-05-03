package rewardapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	rewarddomain "go.mod/internal/domain/reward"
	"go.mod/internal/httpresp"
	rewardservice "go.mod/internal/services/reward"
)

type service interface {
	Create(ctx context.Context, req *rewardservice.CreateRewardRequest) (*rewarddomain.Reward, error)
	Update(ctx context.Context, id int64, req *rewardservice.UpdateRewardRequest) (*rewarddomain.Reward, error)
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*rewarddomain.Reward, error)
	ListBySeason(ctx context.Context, seasonID int64) ([]rewarddomain.Reward, error)
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
		httpresp.Err(w, http.StatusBadRequest, "season_id query param is required")
		return
	}
	seasonID, err := strconv.ParseInt(seasonIDStr, 10, 64)
	if err != nil {
		httpresp.Err(w, http.StatusBadRequest, "invalid season_id")
		return
	}

	items, err := h.svc.ListBySeason(r.Context(), seasonID)
	if err != nil {
		log.Error("failed to list rewards", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, items)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	op := "rewardapi.get"
	log := h.logger.With(slog.String("op", op))

	id, ok := httpresp.PathInt64(w, r, "id")
	if !ok {
		return
	}

	rw, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		log.Error("failed to get reward", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if rw == nil {
		httpresp.Err(w, http.StatusNotFound, "reward not found")
		return
	}
	httpresp.OK(w, rw)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	op := "rewardapi.create"
	log := h.logger.With(slog.String("op", op))

	req, err := httpresp.DecodeJSON[rewardservice.CreateRewardRequest](r)
	if err != nil {
		httpresp.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	rw, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		log.Error("failed to create reward", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.Created(w, rw)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	op := "rewardapi.update"
	log := h.logger.With(slog.String("op", op))

	id, ok := httpresp.PathInt64(w, r, "id")
	if !ok {
		return
	}

	req, err := httpresp.DecodeJSON[rewardservice.UpdateRewardRequest](r)
	if err != nil {
		httpresp.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	rw, err := h.svc.Update(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, rewardservice.ErrNotFound) {
			httpresp.Err(w, http.StatusNotFound, "reward not found")
			return
		}
		log.Error("failed to update reward", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, rw)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	op := "rewardapi.delete"
	log := h.logger.With(slog.String("op", op))

	id, ok := httpresp.PathInt64(w, r, "id")
	if !ok {
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, rewardservice.ErrHasRelations) {
			httpresp.Err(w, http.StatusConflict, "reward has related records and cannot be deleted")
			return
		}
		log.Error("failed to delete reward", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

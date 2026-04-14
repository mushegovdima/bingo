package seasonapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mod/internal/api/response"
	"go.mod/internal/domain"
	seasonservice "go.mod/internal/services/season"
)

type service interface {
	Create(ctx context.Context, req *seasonservice.CreateRequest) (*domain.Season, error)
	Update(ctx context.Context, id int64, req *seasonservice.UpdateRequest) (*domain.Season, error)
	Delete(ctx context.Context, id int64) error
	GetActive(ctx context.Context) (*domain.Season, error)
	GetByID(ctx context.Context, id int64) (*domain.Season, error)
	List(ctx context.Context) ([]*domain.Season, error)
	ListActive(ctx context.Context) ([]*domain.Season, error)
}

type Handler struct {
	svc    service
	logger *slog.Logger
}

func NewHandler(svc service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/active", h.getActive)
	r.Get("/{id}", h.getByID)
	r.Patch("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	return r
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	op := "seasonapi.create"
	log := h.logger.With(slog.String("op", op))

	req, err := response.DecodeJSON[seasonservice.CreateRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	season, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		log.Error("failed to create season", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.Created(w, season)
}

func (h *Handler) getActive(w http.ResponseWriter, r *http.Request) {
	op := "seasonapi.getActive"
	log := h.logger.With(slog.String("op", op))

	season, err := h.svc.GetActive(r.Context())
	if err != nil {
		log.Error("failed to get active season", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if season == nil {
		response.Err(w, http.StatusNotFound, "no active season")
		return
	}

	response.OK(w, season)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	op := "seasonapi.update"
	log := h.logger.With(slog.String("op", op))

	id, err := parseID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	req, err := response.DecodeJSON[seasonservice.UpdateRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	season, err := h.svc.Update(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, seasonservice.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "season not found")
			return
		}
		log.Error("failed to update season", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.OK(w, season)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	op := "seasonapi.delete"
	log := h.logger.With(slog.String("op", op))

	id, err := parseID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, seasonservice.ErrHasRelations) {
			response.Err(w, http.StatusConflict, "season has related records and cannot be deleted")
			return
		}
		log.Error("failed to delete season", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	op := "seasonapi.list"
	log := h.logger.With(slog.String("op", op))

	if r.URL.Query().Get("active") == "true" {
		seasons, err := h.svc.ListActive(r.Context())
		if err != nil {
			log.Error("failed to list active seasons", slog.Any("error", err))
			response.Err(w, http.StatusInternalServerError, "internal server error")
			return
		}
		response.OK(w, seasons)
		return
	}

	seasons, err := h.svc.List(r.Context())
	if err != nil {
		log.Error("failed to list seasons", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, seasons)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	op := "seasonapi.getByID"
	log := h.logger.With(slog.String("op", op))

	id, err := parseID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	season, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		log.Error("failed to get season", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if season == nil {
		response.Err(w, http.StatusNotFound, "season not found")
		return
	}
	response.OK(w, season)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

package claimapi

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
	rewardservice "go.mod/internal/services/reward"
)

type claimService interface {
	SubmitClaim(ctx context.Context, userID, seasonID, rewardID int64) (*domain.RewardClaim, error)
	UpdateClaimStatus(ctx context.Context, claimID int64, req *rewardservice.UpdateClaimRequest, seasonID int64) (*domain.RewardClaim, error)
	GetClaimByID(ctx context.Context, id int64) (*domain.RewardClaim, error)
	ListClaimsByUser(ctx context.Context, userID int64) ([]domain.RewardClaim, error)
	ListAllClaims(ctx context.Context) ([]domain.RewardClaim, error)
}

type Handler struct {
	svc    claimService
	logger *slog.Logger
}

func NewHandler(svc claimService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) Routes(requireAuth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Post("/", h.submit)
		r.Get("/", h.list)
	})
	r.Get("/{id}", h.get)
	r.Patch("/{id}/status", h.updateStatus)
	return r
}

type submitClaimRequest struct {
	RewardID   int64 `json:"reward_id"`
	SeasonID int64 `json:"season_id"`
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	op := "claimapi.submit"
	log := h.logger.With(slog.String("op", op))

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		response.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	req, err := response.DecodeJSON[submitClaimRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	claim, err := h.svc.SubmitClaim(r.Context(), sess.UserID, req.SeasonID, req.RewardID)
	if err != nil {
		switch {
		case errors.Is(err, rewardservice.ErrRewardUnavailable):
			response.Err(w, http.StatusBadRequest, "reward is not available")
		case errors.Is(err, rewardservice.ErrLimitExceeded):
			response.Err(w, http.StatusConflict, "reward limit exceeded")
		case errors.Is(err, rewardservice.ErrInsufficientBalance):
			response.Err(w, http.StatusUnprocessableEntity, "insufficient balance")
		default:
			log.Error("failed to submit claim", slog.Any("error", err))
			response.Err(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	response.Created(w, claim)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	op := "claimapi.list"
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
		items, err := h.svc.ListClaimsByUser(r.Context(), targetID)
		if err != nil {
			log.Error("failed to list claims by user", slog.Any("error", err))
			response.Err(w, http.StatusInternalServerError, "internal server error")
			return
		}
		response.OK(w, items)
		return
	}

	items, err := h.svc.ListClaimsByUser(r.Context(), sess.UserID)
	if err != nil {
		log.Error("failed to list own claims", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, items)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	op := "claimapi.get"
	log := h.logger.With(slog.String("op", op))

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	claim, err := h.svc.GetClaimByID(r.Context(), id)
	if err != nil {
		log.Error("failed to get claim", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if claim == nil {
		response.Err(w, http.StatusNotFound, "claim not found")
		return
	}
	response.OK(w, claim)
}

type updateStatusRequest struct {
	Status     domain.RewardClaimStatus `json:"status"`
	SeasonID int64                    `json:"season_id"`
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	op := "claimapi.updateStatus"
	log := h.logger.With(slog.String("op", op))

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid id")
		return
	}

	req, err := response.DecodeJSON[updateStatusRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	claim, err := h.svc.UpdateClaimStatus(r.Context(), id, &rewardservice.UpdateClaimRequest{Status: req.Status}, req.SeasonID)
	if err != nil {
		if errors.Is(err, rewardservice.ErrClaimNotFound) {
			response.Err(w, http.StatusNotFound, "claim not found")
			return
		}
		log.Error("failed to update claim status", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, claim)
}

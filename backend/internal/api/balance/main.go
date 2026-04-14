package balanceapi

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
	balanceservice "go.mod/internal/services/balance"
)

type service interface {
	GetBalance(ctx context.Context, userID, seasonID int64) (*domain.SeasonMember, error)
	GetTransactions(ctx context.Context, userID, seasonID int64) ([]domain.Transaction, error)
	ChangeBalance(ctx context.Context, req balanceservice.ChangeBalanceRequest) (*domain.Transaction, error)
	ListUserBalances(ctx context.Context, userID int64) ([]domain.SeasonMemberWithSeason, error)
	JoinSeason(ctx context.Context, userID, seasonID int64) (*domain.SeasonMember, error)
}

type Handler struct {
	svc    service
	logger *slog.Logger
}

func NewHandler(svc service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// Routes returns the subrouter. managerMW must be the RequireRole(Manager) middleware.
func (h *Handler) Routes(managerMW func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Get("/my", h.listMyBalances)
	r.Get("/{seasonID}", h.getBalance)
	r.Get("/{seasonID}/transactions", h.getTransactions)
	r.With(managerMW).Post("/{seasonID}/adjust", h.changeBalance)
	r.Post("/{seasonID}/join", h.joinSeason)
	return r
}

func (h *Handler) getBalance(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.getBalance"
	log := h.logger.With(slog.String("op", op))

	seasonID, err := parseSeasonID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid season_id")
		return
	}

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		response.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bal, err := h.svc.GetBalance(r.Context(), sess.UserID, seasonID)
	if err != nil {
		log.Error("failed to get balance", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if bal == nil {
		// No balance yet — return a zero balance instead of 404.
		response.OK(w, &domain.SeasonMember{UserID: sess.UserID, SeasonID: seasonID})
		return
	}
	response.OK(w, bal)
}

func (h *Handler) getTransactions(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.getTransactions"
	log := h.logger.With(slog.String("op", op))

	seasonID, err := parseSeasonID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid season_id")
		return
	}

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		response.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	txs, err := h.svc.GetTransactions(r.Context(), sess.UserID, seasonID)
	if errors.Is(err, balanceservice.ErrBalanceNotFound) {
		response.OK(w, []domain.Transaction{})
		return
	}
	if err != nil {
		log.Error("failed to get transactions", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, txs)
}

type adjustRequest struct {
	UserID int64  `json:"user_id"`
	Amount int    `json:"amount"`
	Note   string `json:"note"`
}

func (h *Handler) changeBalance(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.changeBalance"
	log := h.logger.With(slog.String("op", op))

	seasonID, err := parseSeasonID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid season_id")
		return
	}

	req, err := response.DecodeJSON[adjustRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.UserID == 0 {
		response.Err(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.Amount == 0 {
		response.Err(w, http.StatusBadRequest, "amount must not be zero")
		return
	}

	tx, err := h.svc.ChangeBalance(r.Context(), balanceservice.ChangeBalanceRequest{
		UserID:     req.UserID,
		SeasonID: seasonID,
		Amount:     req.Amount,
		Note:       req.Note,
	})
	if err != nil {
		log.Error("failed to change balance", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, tx)
}

func (h *Handler) listMyBalances(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.listMyBalances"
	log := h.logger.With(slog.String("op", op))

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		response.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	balances, err := h.svc.ListUserBalances(r.Context(), sess.UserID)
	if err != nil {
		log.Error("failed to list balances", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, balances)
}

func (h *Handler) joinSeason(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.joinSeason"
	log := h.logger.With(slog.String("op", op))

	seasonID, err := parseSeasonID(r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid season_id")
		return
	}

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		response.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bal, err := h.svc.JoinSeason(r.Context(), sess.UserID, seasonID)
	if err != nil {
		log.Error("failed to join season", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, bal)
}

func parseSeasonID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "seasonID"), 10, 64)
}

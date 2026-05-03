package balanceapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	wallet "go.mod/internal/contracts/wallet"
	walletdomain "go.mod/internal/domain/wallet"
	"go.mod/internal/httpresp"
	"go.mod/internal/middleware"
	balanceservice "go.mod/internal/services/balance"
)

type service interface {
	GetBalance(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error)
	GetTransactions(ctx context.Context, userID, seasonID int64) ([]walletdomain.Transaction, error)
	ChangeBalance(ctx context.Context, req wallet.ChangeRequest) (*walletdomain.Transaction, error)
	ListUserBalances(ctx context.Context, userID int64) ([]walletdomain.SeasonMemberWithSeason, error)
	JoinSeason(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error)
	GetLeaderboard(ctx context.Context, userID, seasonID int64) ([]walletdomain.LeaderboardEntry, error)
	GetFullLeaderboard(ctx context.Context, userID, seasonID int64) ([]walletdomain.LeaderboardEntry, error)
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
	r.Get("/leaderboard/{seasonID}", h.getLeaderboard)
	r.With(managerMW).Get("/leaderboard/{seasonID}/full", h.getFullLeaderboard)
	r.Get("/{seasonID}", h.getBalance)
	r.Get("/{seasonID}/transactions", h.getTransactions)
	r.With(managerMW).Post("/{seasonID}/adjust", h.changeBalance)
	r.Post("/{seasonID}/join", h.joinSeason)
	return r
}

func (h *Handler) getBalance(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.getBalance"
	log := h.logger.With(slog.String("op", op))

	seasonID, ok := httpresp.PathInt64(w, r, "seasonID")
	if !ok {
		return
	}

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bal, err := h.svc.GetBalance(r.Context(), sess.UserID, seasonID)
	if err != nil {
		log.Error("failed to get balance", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if bal == nil {
		// No balance yet — return a zero balance instead of 404.
		httpresp.OK(w, &walletdomain.SeasonMember{UserID: sess.UserID, SeasonID: seasonID})
		return
	}
	httpresp.OK(w, bal)
}

func (h *Handler) getTransactions(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.getTransactions"
	log := h.logger.With(slog.String("op", op))

	seasonID, ok := httpresp.PathInt64(w, r, "seasonID")
	if !ok {
		return
	}

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	txs, err := h.svc.GetTransactions(r.Context(), sess.UserID, seasonID)
	if errors.Is(err, balanceservice.ErrBalanceNotFound) {
		httpresp.OK(w, []walletdomain.Transaction{})
		return
	}
	if err != nil {
		log.Error("failed to get transactions", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, txs)
}

type adjustRequest struct {
	UserID int64  `json:"user_id"`
	Amount int    `json:"amount"`
	Note   string `json:"note"`
}

func (h *Handler) changeBalance(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.changeBalance"
	log := h.logger.With(slog.String("op", op))

	seasonID, ok := httpresp.PathInt64(w, r, "seasonID")
	if !ok {
		return
	}

	req, err := httpresp.DecodeJSON[adjustRequest](r)
	if err != nil {
		httpresp.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.UserID == 0 {
		httpresp.Err(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.Amount == 0 {
		httpresp.Err(w, http.StatusBadRequest, "amount must not be zero")
		return
	}

	tx, err := h.svc.ChangeBalance(r.Context(), wallet.ChangeRequest{
		UserID:   req.UserID,
		SeasonID: seasonID,
		Amount:   req.Amount,
		Note:     req.Note,
	})
	if err != nil {
		log.Error("failed to change balance", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, tx)
}

func (h *Handler) listMyBalances(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.listMyBalances"
	log := h.logger.With(slog.String("op", op))

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	balances, err := h.svc.ListUserBalances(r.Context(), sess.UserID)
	if err != nil {
		log.Error("failed to list balances", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, balances)
}

func (h *Handler) joinSeason(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.joinSeason"
	log := h.logger.With(slog.String("op", op))

	seasonID, ok := httpresp.PathInt64(w, r, "seasonID")
	if !ok {
		return
	}

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bal, err := h.svc.JoinSeason(r.Context(), sess.UserID, seasonID)
	if err != nil {
		log.Error("failed to join season", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, bal)
}

func (h *Handler) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.getLeaderboard"
	log := h.logger.With(slog.String("op", op))

	seasonID, ok := httpresp.PathInt64(w, r, "seasonID")
	if !ok {
		return
	}

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	entries, err := h.svc.GetLeaderboard(r.Context(), sess.UserID, seasonID)
	if err != nil {
		log.Error("failed to get leaderboard", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, entries)
}

func (h *Handler) getFullLeaderboard(w http.ResponseWriter, r *http.Request) {
	op := "balanceapi.getFullLeaderboard"
	log := h.logger.With(slog.String("op", op))

	seasonID, ok := httpresp.PathInt64(w, r, "seasonID")
	if !ok {
		return
	}

	sess := middleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	entries, err := h.svc.GetFullLeaderboard(r.Context(), sess.UserID, seasonID)
	if err != nil {
		log.Error("failed to get full leaderboard", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, entries)
}

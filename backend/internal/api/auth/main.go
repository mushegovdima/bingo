package authapi

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"go.mod/internal/api/response"
	"go.mod/internal/config"
	dbmodels "go.mod/internal/db"
	"go.mod/internal/domain"
	apimiddleware "go.mod/internal/middleware"
	authservice "go.mod/internal/services/auth"
)

type authenticator interface {
	AuthenticateUser(ctx context.Context, req *authservice.UserLoginByTelegramRequest) (*authservice.AuthResult, error)
}

type userService interface {
	GetById(ctx context.Context, id int64) (*domain.User, error)
}

type sessionCreator interface {
	CreateSession(ctx context.Context, session *dbmodels.Session) (*int64, error)
}

type Handler struct {
	auth     authenticator
	users    userService
	sessions sessionCreator
	store    *sessions.CookieStore
	cfg      *config.Config
	logger   *slog.Logger
}

func NewHandler(auth authenticator, users userService, sessions sessionCreator, store *sessions.CookieStore, cfg *config.Config, logger *slog.Logger) *Handler {
	return &Handler{auth: auth, users: users, sessions: sessions, store: store, cfg: cfg, logger: logger}
}

// Routes returns a single router combining public and protected auth endpoints.
// requireAuth middleware is applied only to protected routes.
// managerOnly middleware guards the impersonate endpoint.
func (h *Handler) Routes(requireAuth, managerOnly func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/me", h.getMe)
	})
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Use(managerOnly)
		r.Post("/impersonate/{userID}", h.impersonate)
	})
	return r
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	op := "authapi.getMe"
	log := h.logger.With(slog.String("op", op))

	sess := apimiddleware.SessionFromContext(r.Context())

	usr, err := h.users.GetById(r.Context(), sess.UserID)
	if err != nil {
		log.Error("failed to get user", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if usr == nil {
		response.Err(w, http.StatusNotFound, "user not found")
		return
	}
	if usr.IsBlocked {
		response.Err(w, http.StatusForbidden, "user is blocked")
		return
	}

	response.OK(w, usr)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	op := "authapi.login"
	log := h.logger.With(slog.String("op", op))

	req, err := response.DecodeJSON[authservice.UserLoginByTelegramRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	req.UserAgent = r.UserAgent()
	req.IP = r.RemoteAddr

	result, err := h.auth.AuthenticateUser(r.Context(), &req)
	if err != nil {
		log.Warn("authentication failed", slog.Any("error", err))
		response.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := apimiddleware.SaveSession(w, r, h.store, result.SessionID, result.UserID); err != nil {
		log.Error("failed to save session", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.OK(w, result.User)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	op := "authapi.logout"
	log := h.logger.With(slog.String("op", op))

	if err := apimiddleware.ClearSession(w, r, h.store); err != nil {
		log.Warn("failed to clear session", slog.Any("error", err))
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) impersonate(w http.ResponseWriter, r *http.Request) {
	op := "authapi.impersonate"
	log := h.logger.With(slog.String("op", op))

	callerSess := apimiddleware.SessionFromContext(r.Context())

	targetID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if targetID == callerSess.UserID {
		response.Err(w, http.StatusBadRequest, "cannot impersonate yourself")
		return
	}

	target, err := h.users.GetById(r.Context(), targetID)
	if err != nil {
		log.Error("failed to get target user", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if target == nil {
		response.Err(w, http.StatusNotFound, "user not found")
		return
	}
	if target.IsBlocked {
		response.Err(w, http.StatusForbidden, "target user is blocked")
		return
	}

	expiresAt := time.Now().Add(time.Duration(h.cfg.SessionTTLMinutes) * time.Minute)
	sessionID, err := h.sessions.CreateSession(r.Context(), &dbmodels.Session{
		UserID:    target.ID,
		UserAgent: r.UserAgent(),
		IP:        r.RemoteAddr,
		ExpiresAt: &expiresAt,
		Status:    domain.SessionActive,
	})
	if err != nil {
		log.Error("failed to create impersonation session", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := apimiddleware.SaveSession(w, r, h.store, *sessionID, target.ID); err != nil {
		log.Error("failed to save session cookie", slog.Any("error", err))
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}

	log.Info("manager impersonated user",
		slog.Int64("caller_id", callerSess.UserID),
		slog.Int64("target_user_id", target.ID),
	)
	response.OK(w, target)
}

package authapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"go.mod/internal/config"
	sessioncontract "go.mod/internal/contracts/session"
	sessiondomain "go.mod/internal/domain/session"
	userdomain "go.mod/internal/domain/user"
	"go.mod/internal/httpresp"
	apimiddleware "go.mod/internal/middleware"
	authservice "go.mod/internal/services/auth"
)

type authenticator interface {
	AuthenticateUser(ctx context.Context, req *authservice.UserLoginByTelegramRequest) (*authservice.AuthResult, error)
	AuthenticateWebApp(ctx context.Context, req *authservice.WebAppLoginRequest) (*authservice.AuthResult, error)
}

type userService interface {
	GetById(ctx context.Context, id int64) (*userdomain.User, error)
}

type sessionCreator interface {
	CreateSession(ctx context.Context, in sessioncontract.CreateInput) (int64, error)
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
	r.Post("/login/webapp", h.loginWebApp)
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
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if usr == nil {
		httpresp.Err(w, http.StatusNotFound, "user not found")
		return
	}
	if usr.IsBlocked {
		httpresp.Err(w, http.StatusForbidden, "user is blocked")
		return
	}

	httpresp.OK(w, usr)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	op := "authapi.login"
	log := h.logger.With(slog.String("op", op))

	req, err := httpresp.DecodeJSON[authservice.UserLoginByTelegramRequest](r)
	if err != nil {
		httpresp.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	req.UserAgent = r.UserAgent()
	req.IP = r.RemoteAddr

	result, err := h.auth.AuthenticateUser(r.Context(), &req)
	if err != nil {
		log.Warn("authentication failed", slog.Any("error", err))
		switch {
		case errors.Is(err, authservice.ErrUserBlocked):
			httpresp.Err(w, http.StatusForbidden, "user is blocked")
		case errors.Is(err, authservice.ErrAuthDataOutdated):
			httpresp.Err(w, http.StatusUnauthorized, "telegram auth data is outdated")
		default:
			httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		}
		return
	}

	if err := apimiddleware.SaveSession(w, r, h.store, result.SessionID, result.UserID); err != nil {
		log.Error("failed to save session", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}

	httpresp.OK(w, result.User)
}

func (h *Handler) loginWebApp(w http.ResponseWriter, r *http.Request) {
	op := "authapi.loginWebApp"
	log := h.logger.With(slog.String("op", op))

	req, err := httpresp.DecodeJSON[authservice.WebAppLoginRequest](r)
	if err != nil {
		httpresp.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	req.UserAgent = r.UserAgent()
	req.IP = r.RemoteAddr

	result, err := h.auth.AuthenticateWebApp(r.Context(), &req)
	if err != nil {
		log.Warn("webapp authentication failed", slog.Any("error", err))
		switch {
		case errors.Is(err, authservice.ErrUserBlocked):
			httpresp.Err(w, http.StatusForbidden, "user is blocked")
		case errors.Is(err, authservice.ErrAuthDataOutdated):
			httpresp.Err(w, http.StatusUnauthorized, "telegram auth data is outdated")
		default:
			httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		}
		return
	}

	if err := apimiddleware.SaveSession(w, r, h.store, result.SessionID, result.UserID); err != nil {
		log.Error("failed to save session", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}

	httpresp.OK(w, result.User)
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

	targetID, ok := httpresp.PathInt64(w, r, "userID")
	if !ok {
		return
	}
	if targetID == callerSess.UserID {
		httpresp.Err(w, http.StatusBadRequest, "cannot impersonate yourself")
		return
	}

	target, err := h.users.GetById(r.Context(), targetID)
	if err != nil {
		log.Error("failed to get target user", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if target == nil {
		httpresp.Err(w, http.StatusNotFound, "user not found")
		return
	}
	if target.IsBlocked {
		httpresp.Err(w, http.StatusForbidden, "target user is blocked")
		return
	}

	expiresAt := time.Now().Add(time.Duration(h.cfg.SessionTTLMinutes) * time.Minute)
	sessionID, err := h.sessions.CreateSession(r.Context(), sessioncontract.CreateInput{
		UserID:    target.ID,
		UserAgent: r.UserAgent(),
		IP:        r.RemoteAddr,
		ExpiresAt: &expiresAt,
		Status:    sessiondomain.SessionActive,
	})
	if err != nil {
		log.Error("failed to create impersonation session", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := apimiddleware.SaveSession(w, r, h.store, sessionID, target.ID); err != nil {
		log.Error("failed to save session cookie", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}

	log.Info("manager impersonated user",
		slog.Int64("caller_id", callerSess.UserID),
		slog.Int64("target_user_id", target.ID),
	)
	httpresp.OK(w, target)
}

package templateapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	templatedomain "go.mod/internal/domain/template"
	"go.mod/internal/httpresp"
	apimiddleware "go.mod/internal/middleware"
	"go.mod/internal/notifications"
)

type service interface {
	GetByCodename(ctx context.Context, codename string) (*templatedomain.Template, error)
	List(ctx context.Context) ([]*templatedomain.Template, error)
	UpdateBody(ctx context.Context, codename string, changedBy int64, body string) (*templatedomain.Template, error)
	ListHistory(ctx context.Context, codename string) ([]*templatedomain.TemplateHistory, error)
}

// notificationByCodename is built once from notifications.All so that adding a new
// notification type only requires updating notifications.go — not this file.
var notificationByCodename = func() map[string]notifications.Notification {
	m := make(map[string]notifications.Notification, len(notifications.All))
	for _, n := range notifications.All {
		m[n.Codename()] = n
	}
	return m
}()

type templateResponse struct {
	*templatedomain.Template
	Vars []notifications.TemplateVar `json:"vars"`
}

func (h *Handler) withVars(t *templatedomain.Template) templateResponse {
	return templateResponse{Template: t, Vars: notifications.VarsOf(notificationByCodename[t.Codename])}
}

type Handler struct {
	svc    service
	logger *slog.Logger
}

func NewHandler(svc service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// Routes wires the handler into a chi.Router.
//
//	GET    /templates            — list all (auth required)
//	GET    /templates/{codename} — get one (auth required)
//	PATCH  /templates/{codename} — update body (manager only)
//	GET    /templates/{codename}/history — change history (manager only)
func (h *Handler) Routes(managerMW func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Get("/{codename}", h.get)
	r.With(managerMW).Patch("/{codename}", h.update)
	r.With(managerMW).Get("/{codename}/history", h.history)
	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	op := "templateapi.list"
	log := h.logger.With(slog.String("op", op))

	items, err := h.svc.List(r.Context())
	if err != nil {
		log.ErrorContext(r.Context(), "failed to list templates", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	resp := make([]templateResponse, len(items))
	for i, t := range items {
		resp[i] = h.withVars(t)
	}
	httpresp.OK(w, resp)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	op := "templateapi.get"
	codename := chi.URLParam(r, "codename")
	log := h.logger.With(slog.String("op", op), slog.String("codename", codename))

	tpl, err := h.svc.GetByCodename(r.Context(), codename)
	if errors.Is(err, templatedomain.ErrNotFound) {
		httpresp.Err(w, http.StatusNotFound, "template not found")
		return
	}
	if err != nil {
		log.ErrorContext(r.Context(), "failed to get template", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, h.withVars(tpl))
}

// updateBody is the PATCH request — only body is mutable.
type updateBody struct {
	Body string `json:"body"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	op := "templateapi.update"
	codename := chi.URLParam(r, "codename")
	log := h.logger.With(slog.String("op", op), slog.String("codename", codename))

	sess := apimiddleware.SessionFromContext(r.Context())
	if sess == nil {
		httpresp.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresp.Err(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Body == "" {
		httpresp.Err(w, http.StatusBadRequest, "body is required")
		return
	}

	result, err := h.svc.UpdateBody(r.Context(), codename, sess.UserID, req.Body)
	if errors.Is(err, templatedomain.ErrNotFound) {
		httpresp.Err(w, http.StatusNotFound, "template not found")
		return
	}
	if err != nil {
		log.ErrorContext(r.Context(), "failed to update template", slog.Any("error", err))
		httpresp.Err(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	httpresp.OK(w, h.withVars(result))
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	op := "templateapi.history"
	codename := chi.URLParam(r, "codename")
	log := h.logger.With(slog.String("op", op), slog.String("codename", codename))

	items, err := h.svc.ListHistory(r.Context(), codename)
	if errors.Is(err, templatedomain.ErrNotFound) {
		httpresp.Err(w, http.StatusNotFound, "template not found")
		return
	}
	if err != nil {
		log.ErrorContext(r.Context(), "failed to list history", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, items)
}

package adminapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	usercontract "go.mod/internal/contracts/user"
	userdomain "go.mod/internal/domain/user"
	"go.mod/internal/httpresp"
)

type userManager interface {
	List(ctx context.Context) ([]*userdomain.User, error)
	UpdateUser(ctx context.Context, id int64, req usercontract.UpdateRequest) (*userdomain.User, error)
}

type Handler struct {
	users  userManager
	logger *slog.Logger
}

func NewHandler(users userManager, logger *slog.Logger) *Handler {
	return &Handler{users: users, logger: logger}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/users", h.listUsers)
	r.Patch("/users/{id}", h.updateUser)
	return r
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	op := "adminapi.listUsers"
	log := h.logger.With(slog.String("op", op))

	users, err := h.users.List(r.Context())
	if err != nil {
		log.Error("failed to list users", slog.Any("error", err))
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, users)
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	op := "adminapi.updateUser"
	log := h.logger.With(slog.String("op", op))

	id, ok := httpresp.PathInt64(w, r, "id")
	if !ok {
		return
	}

	req, err := httpresp.DecodeJSON[usercontract.UpdateRequest](r)
	if err != nil {
		httpresp.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.users.UpdateUser(r.Context(), id, req)
	if err != nil {
		log.Error("failed to update user", slog.Int64("user_id", id), slog.Any("error", err))
		if err.Error() == "user not found" {
			httpresp.Err(w, http.StatusNotFound, "user not found")
			return
		}
		httpresp.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	httpresp.OK(w, user)
}

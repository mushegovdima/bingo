package adminapi

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mod/internal/api/response"
	"go.mod/internal/domain"
	userservice "go.mod/internal/services/user"
)

type userManager interface {
	List(ctx context.Context) ([]*domain.User, error)
	UpdateUser(ctx context.Context, id int64, req userservice.UpdateUserRequest) (*domain.User, error)
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
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, users)
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	op := "adminapi.updateUser"
	log := h.logger.With(slog.String("op", op))

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "invalid user id")
		return
	}

	req, err := response.DecodeJSON[userservice.UpdateUserRequest](r)
	if err != nil {
		response.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.users.UpdateUser(r.Context(), id, req)
	if err != nil {
		log.Error("failed to update user", slog.Int64("user_id", id), slog.Any("error", err))
		if err.Error() == "user not found" {
			response.Err(w, http.StatusNotFound, "user not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.OK(w, user)
}

package userservice

import "go.mod/internal/domain"

type UserCreateRequest struct {
	TelegramID int64  `json:"telegram_id" validate:"required"`
	Name       string `json:"name" validate:"required"`
	Username   string `json:"username"`
	PhotoURL   string `json:"photo_url"`
}

type UpdateUserRequest struct {
	Name      string            `json:"name"`
	Roles     []domain.UserRole `json:"roles"`
	IsBlocked bool              `json:"is_blocked"`
}

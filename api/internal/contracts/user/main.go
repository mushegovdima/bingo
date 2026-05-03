// Package user defines cross-bounded-context DTOs for the user domain.
// Other services import these to avoid taking a hard dependency on userservice
// when they only need to describe a request/response shape.
package user

import (
	userdomain "go.mod/internal/domain/user"
)

// CreateRequest describes a user-create-or-update intent originating outside userservice
// (e.g. from authservice during Telegram authentication). Validation tags are kept on the
// struct because the type also flows through HTTP handlers.
type CreateRequest struct {
	TelegramID int64  `json:"telegram_id" validate:"required"`
	Name       string `json:"name" validate:"required"`
	Username   string `json:"username"`
	PhotoURL   string `json:"photo_url"`
}

// UpdateRequest describes mutable user fields that admin-facing flows are allowed to change.
type UpdateRequest struct {
	Name      string                `json:"name"`
	Roles     []userdomain.UserRole `json:"roles"`
	IsBlocked bool                  `json:"is_blocked"`
}

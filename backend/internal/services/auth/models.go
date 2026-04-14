package authservice

import "go.mod/internal/domain"

type UserLoginByTelegramRequest struct {
	ID        int64  `json:"id"         validate:"required"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  int64  `json:"auth_date"  validate:"required"`
	Hash      string `json:"hash"       validate:"required"`

	// Заполняется на HTTP-слое, не из JSON
	UserAgent string `json:"-"`
	IP        string `json:"-"`
}

type AuthResult struct {
	SessionID int64
	UserID    int64
	User      *domain.User
}

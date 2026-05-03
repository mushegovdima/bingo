package user

import (
	"errors"
	"time"
)

// Domain invariant errors.
var (
	// ErrInvalidRole means an empty/unknown role was provided.
	ErrInvalidRole = errors.New("invalid role")
)

// UserRole enumerates roles a User may carry.
type UserRole string

const (
	Manager  UserRole = "manager"
	Resident UserRole = "resident"
)

// IsKnown reports whether r is a recognised role.
func (r UserRole) IsKnown() bool {
	return r == Manager || r == Resident
}

// User is the identity aggregate root: a Telegram-authenticated account
// with one or more roles.
type User struct {
	ID         int64      `json:"id"`
	TelegramID int64      `json:"telegram_id"`
	Name       string     `json:"name"`
	Username   string     `json:"username"`
	PhotoURL   string     `json:"photo_url,omitempty"`
	Roles      []UserRole `json:"roles"`
	IsBlocked  bool       `json:"is_blocked"`
	CreatedAt  time.Time  `json:"created_at"`
}

// HasRole reports whether u is granted the given role.
func (u *User) HasRole(r UserRole) bool {
	for _, role := range u.Roles {
		if role == r {
			return true
		}
	}
	return false
}

// IsManager is a convenience helper.
func (u *User) IsManager() bool { return u.HasRole(Manager) }

// AddRole grants r to the user (idempotent). Returns ErrInvalidRole when r
// is unknown.
func (u *User) AddRole(r UserRole) error {
	if !r.IsKnown() {
		return ErrInvalidRole
	}
	if u.HasRole(r) {
		return nil
	}
	u.Roles = append(u.Roles, r)
	return nil
}

// RemoveRole revokes r from the user (idempotent).
func (u *User) RemoveRole(r UserRole) {
	out := u.Roles[:0]
	for _, role := range u.Roles {
		if role != r {
			out = append(out, role)
		}
	}
	u.Roles = out
}

// Block marks the user as blocked.
func (u *User) Block() { u.IsBlocked = true }

// Unblock clears the blocked flag.
func (u *User) Unblock() { u.IsBlocked = false }

// Package session defines cross-bounded-context DTOs for the session subsystem.
// Producers (e.g. authservice during login or impersonation) depend on this contract
// rather than on sessionservice directly, keeping the import graph one-directional.
package session

import (
	sessiondomain "go.mod/internal/domain/session"
	"time"
)

// CreateInput describes the session attributes captured at login/impersonation time.
// The persistence shape stays internal to sessionservice.
type CreateInput struct {
	UserID    int64
	UserAgent string
	IP        string
	ExpiresAt *time.Time
	Status    sessiondomain.SessionStatus
}

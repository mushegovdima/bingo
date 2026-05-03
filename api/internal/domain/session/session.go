package session

import (
	"time"
)

// SessionStatus enumerates lifecycle states of a user Session.
type SessionStatus string

const (
	SessionActive   SessionStatus = "active"
	SessionExpired  SessionStatus = "expired"
	SessionInactive SessionStatus = "inactive"
)

// Session is an authenticated browser/device tied to a User.
type Session struct {
	ID           int64         `json:"id"`
	UserID       int64         `json:"user_id"`
	CreatedAt    time.Time     `json:"created_at"`
	ExpiresAt    *time.Time    `json:"expires_at"`
	UserAgent    string        `json:"user_agent"`
	IP           string        `json:"ip"`
	LastActivity time.Time     `json:"last_activity"`
	Status       SessionStatus `json:"status"`
}

// IsActive reports whether the session is in active state and not expired.
func (s *Session) IsActive(now time.Time) bool {
	if s.Status != SessionActive {
		return false
	}
	if s.ExpiresAt != nil && !now.Before(*s.ExpiresAt) {
		return false
	}
	return true
}

// IsExpiredAt reports whether the session has passed its ExpiresAt.
func (s *Session) IsExpiredAt(now time.Time) bool {
	return s.ExpiresAt != nil && !now.Before(*s.ExpiresAt)
}

// Touch records activity at now (e.g. on each authenticated request).
func (s *Session) Touch(now time.Time) { s.LastActivity = now }

// Expire marks the session as expired (e.g. on TTL elapse).
func (s *Session) Expire() { s.Status = SessionExpired }

// Deactivate marks the session as inactive (e.g. on explicit logout).
func (s *Session) Deactivate() { s.Status = SessionInactive }

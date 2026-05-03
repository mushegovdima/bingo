package session_test

import (
	"testing"
	"time"

	sessiondomain "go.mod/internal/domain/session"
)

func TestSession_IsActive(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	cases := []struct {
		name    string
		status  sessiondomain.SessionStatus
		expires *time.Time
		want    bool
	}{
		{"active and unexpired", sessiondomain.SessionActive, &future, true},
		{"active without expiry", sessiondomain.SessionActive, nil, true},
		{"active but expired", sessiondomain.SessionActive, &past, false},
		{"inactive", sessiondomain.SessionInactive, &future, false},
		{"expired status", sessiondomain.SessionExpired, &future, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &sessiondomain.Session{Status: tc.status, ExpiresAt: tc.expires}
			if got := s.IsActive(now); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSession_TouchExpireDeactivate(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	s := &sessiondomain.Session{Status: sessiondomain.SessionActive}
	s.Touch(now)
	if !s.LastActivity.Equal(now) {
		t.Errorf("Touch did not update LastActivity")
	}

	s.Expire()
	if s.Status != sessiondomain.SessionExpired {
		t.Errorf("Expire: got %q", s.Status)
	}

	s2 := &sessiondomain.Session{Status: sessiondomain.SessionActive}
	s2.Deactivate()
	if s2.Status != sessiondomain.SessionInactive {
		t.Errorf("Deactivate: got %q", s2.Status)
	}
}

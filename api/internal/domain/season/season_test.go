package season_test

import (
	"errors"
	"testing"
	"time"

	seasondomain "go.mod/internal/domain/season"
)

func TestSeason_Validate(t *testing.T) {
	t1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.AddDate(0, 1, 0)

	cases := []struct {
		name    string
		req     seasondomain.CreateRequest
		wantErr error
	}{
		{"ok", seasondomain.CreateRequest{Title: "Spring", StartDate: t1, EndDate: t2}, nil},
		{"empty title", seasondomain.CreateRequest{StartDate: t1, EndDate: t2}, seasondomain.ErrEmptyTitle},
		{"end before start", seasondomain.CreateRequest{Title: "x", StartDate: t2, EndDate: t1}, seasondomain.ErrInvalidDateRange},
		{"end equals start", seasondomain.CreateRequest{Title: "x", StartDate: t1, EndDate: t1}, seasondomain.ErrInvalidDateRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := seasondomain.NewValidated(tc.req)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got err=%v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestSeason_IsRunning(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	t.Run("inside range and active", func(t *testing.T) {
		s := &seasondomain.Season{StartDate: start, EndDate: end, IsActive: true}
		if !s.IsRunning(start.Add(24 * time.Hour)) {
			t.Errorf("expected running")
		}
	})
	t.Run("outside range", func(t *testing.T) {
		s := &seasondomain.Season{StartDate: start, EndDate: end, IsActive: true}
		if s.IsRunning(end.Add(24 * time.Hour)) {
			t.Errorf("expected not running after end")
		}
	})
	t.Run("not active", func(t *testing.T) {
		s := &seasondomain.Season{StartDate: start, EndDate: end, IsActive: false}
		if s.IsRunning(start.Add(24 * time.Hour)) {
			t.Errorf("inactive season must not be running")
		}
	})
}

func TestSeason_ActivateDeactivate(t *testing.T) {
	s := &seasondomain.Season{}
	s.Activate()
	if !s.IsActive {
		t.Errorf("Activate did not set IsActive")
	}
	s.Deactivate()
	if s.IsActive {
		t.Errorf("Deactivate did not clear IsActive")
	}
}

func TestSeasonEvent_Validate(t *testing.T) {
	t1 := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)

	cases := []struct {
		name    string
		event   seasondomain.SeasonEvent
		wantErr error
	}{
		{"ok", seasondomain.SeasonEvent{Title: "Town hall", EventStart: t1, EventEnd: t2}, nil},
		{"empty title", seasondomain.SeasonEvent{EventStart: t1, EventEnd: t2}, seasondomain.ErrEventEmptyTitle},
		{"end before start", seasondomain.SeasonEvent{Title: "x", EventStart: t2, EventEnd: t1}, seasondomain.ErrEventInvalidRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.event.Validate()
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got err=%v, want %v", err, tc.wantErr)
			}
		})
	}
}

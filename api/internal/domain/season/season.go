package season

import (
	"errors"
	"time"
)

// Domain invariant errors.
var (
	// ErrNotFound means a season lookup found no record.
	ErrNotFound = errors.New("season not found")
	// ErrHasRelations means a season cannot be deleted due to FK references.
	ErrHasRelations = errors.New("season has related records and cannot be deleted")
	// ErrEmptyTitle means a season was created/updated with no title.
	ErrEmptyTitle = errors.New("season title must not be empty")
	// ErrInvalidDateRange means EndDate is not strictly after StartDate.
	ErrInvalidDateRange = errors.New("season end date must be after start date")
)

// Season for Bingo game. For example, "Spring 2026" with specific start/end dates and rules.
type Season struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"` // "Марафон: Весна 2026"
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsActive  bool      `json:"is_active"` // Активен ли сезон для новых отчетов
}

// CreateRequest — Data to create a new season.
type CreateRequest struct {
	Title     string    `json:"title"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsActive  bool      `json:"is_active"`
}

// New constructs a Season from input. Use NewValidated when callers need to
// surface invariant violations.
func New(input CreateRequest) *Season {
	return &Season{
		Title:     input.Title,
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
		IsActive:  input.IsActive,
	}
}

// NewValidated constructs a Season after enforcing the basic invariants.
func NewValidated(input CreateRequest) (*Season, error) {
	s := New(input)
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

// Validate enforces structural invariants on the season.
func (s *Season) Validate() error {
	if s.Title == "" {
		return ErrEmptyTitle
	}
	if !s.EndDate.After(s.StartDate) {
		return ErrInvalidDateRange
	}
	return nil
}

func (s *Season) Activate() {
	s.IsActive = true
}

func (s *Season) Deactivate() {
	s.IsActive = false
}

// IsRunning reports whether now falls within [StartDate, EndDate] and the
// season is active.
func (s *Season) IsRunning(now time.Time) bool {
	if !s.IsActive {
		return false
	}
	return !now.Before(s.StartDate) && !now.After(s.EndDate)
}

package season

import (
	"errors"
	"time"
)

// Domain invariant errors for SeasonEvent.
var (
	// ErrEventInvalidRange means EventEnd is not strictly after EventStart.
	ErrEventInvalidRange = errors.New("event end must be after event start")
	// ErrEventEmptyTitle means the event was created without a title.
	ErrEventEmptyTitle = errors.New("event title must not be empty")
)

// SeasonEvent is a scheduled in-season event participants can attend
// (e.g. an online meeting). Coins reward is optional and may be adjusted at
// settlement time.
type SeasonEvent struct {
	ID          int64     `json:"id"`
	SeasonID    int64     `json:"season_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	EventStart  time.Time `json:"event_start"`
	EventEnd    time.Time `json:"event_end"`
	CoinsReward *int      `json:"coins_reward"`
}

// Validate enforces structural invariants on the event.
func (e *SeasonEvent) Validate() error {
	if e.Title == "" {
		return ErrEventEmptyTitle
	}
	if !e.EventEnd.After(e.EventStart) {
		return ErrEventInvalidRange
	}
	return nil
}

// IsLive reports whether now falls within [EventStart, EventEnd].
func (e *SeasonEvent) IsLive(now time.Time) bool {
	return !now.Before(e.EventStart) && !now.After(e.EventEnd)
}

package task

import (
	"encoding/json"
	"errors"
)

// Domain invariant errors.
var (
	// ErrNotFound means a task lookup found no record.
	ErrNotFound = errors.New("task not found")
	// ErrEmptyTitle means a task was constructed without a title.
	ErrEmptyTitle = errors.New("task title must not be empty")
	// ErrNegativeReward means RewardCoins was set to a negative value.
	ErrNegativeReward = errors.New("task reward must not be negative")
	// ErrHasRelations means the task cannot be deleted due to FK references.
	ErrHasRelations = errors.New("task has related records")
)

// Task is a single bingo-grid task that a Resident can submit a report for
// to earn RewardCoins.
type Task struct {
	ID          int64           `json:"id"`
	SeasonID    int64           `json:"season_id"`
	Category    string          `json:"category"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	RewardCoins int             `json:"reward_coins"`
	SortOrder   int             `json:"sort_order"`
	Metadata    json.RawMessage `json:"metadata"`
	IsActive    bool            `json:"is_active"`
}

// Validate enforces structural invariants on the task.
func (t *Task) Validate() error {
	if t.Title == "" {
		return ErrEmptyTitle
	}
	if t.RewardCoins < 0 {
		return ErrNegativeReward
	}
	return nil
}

// Activate makes the task visible/submittable.
func (t *Task) Activate() { t.IsActive = true }

// Deactivate hides the task from new submissions.
func (t *Task) Deactivate() { t.IsActive = false }

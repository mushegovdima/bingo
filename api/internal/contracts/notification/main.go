// Package notification defines cross-bounded-context DTOs for the notification subsystem:
// recipient filters used to express audience selection declaratively, and job-status enums.
package notification

// JobStatus enumerates the lifecycle states of a notification job.
type JobStatus string

const (
	JobStatusPending JobStatus = "pending"
	JobStatusRunning JobStatus = "running"
	JobStatusDone    JobStatus = "done"
	JobStatusFailed  JobStatus = "failed"
)

// UserFilter is a declarative description of which users a notification job targets.
// It is stored as JSONB in notification_jobs.filter and resolved by the worker against
// the users table at processing time. All non-zero fields combine with logical AND.
//
// Keep this struct flat and AND-only — anything fancier (joins, OR-groups, computed
// audiences) belongs behind a named resolver, not here.
type UserFilter struct {
	// UserIDs restricts delivery to a fixed set of users. When non-empty it short-circuits
	// the rest of the filter (intersected with role/flag predicates if also provided).
	UserIDs []int64 `json:"user_ids,omitempty"`
	// Roles restricts delivery to users having any of the listed roles (e.g. "manager", "resident").
	Roles []string `json:"roles,omitempty"`
	// ExcludeBlocked removes users with is_blocked = true.
	ExcludeBlocked bool `json:"exclude_blocked,omitempty"`
	// OnlyTelegram restricts delivery to users with a non-zero telegram_id (i.e. reachable via the bot).
	OnlyTelegram bool `json:"only_telegram,omitempty"`
}

// EnqueueRequest is produced by notificationservice.EnqueueTemplate and consumed by
// the outbox repository. Text is the raw (unrendered) template body; Args holds the
// notification-specific placeholder values that the worker will merge with per-recipient
// user fields before calling notifications.Substitute.
type EnqueueRequest struct {
	Type   string
	Text   string
	Args   map[string]string
	Filter UserFilter
}

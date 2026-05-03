package templatedomain

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a template lookup finds no record.
var ErrNotFound = errors.New("message template not found")

// Template is a named push notification body with {{argKey}} placeholder syntax.
// Codename is immutable after creation; only Body may change.
// Accepted placeholders are defined in the notifications package.
type Template struct {
	ID        int64     `json:"id"`
	Codename  string    `json:"codename"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TemplateHistory records each body change of a Template.
type TemplateHistory struct {
	ID         int64     `json:"id"`
	TemplateID int64     `json:"template_id"`
	Body       string    `json:"body"`
	ChangedBy  int64     `json:"changed_by"`
	ChangedAt  time.Time `json:"changed_at"`
}

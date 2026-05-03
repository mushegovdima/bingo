package db

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
	rewarddomain "go.mod/internal/domain/reward"
	sessiondomain "go.mod/internal/domain/session"
	submissiondomain "go.mod/internal/domain/submission"
	userdomain "go.mod/internal/domain/user"
	walletdomain "go.mod/internal/domain/wallet"
)

type Entity struct {
	ID int64 `bun:"id,pk,autoincrement"`
}

// --- Users ---

type User struct {
	Entity
	bun.BaseModel `bun:"table:users"`
	TelegramID    int64                 `bun:"telegram_id,notnull"`
	Name          string                `bun:"name,notnull"`
	Username      string                `bun:"username,notnull"`
	PhotoURL      string                `bun:"photo_url"`
	Roles         []userdomain.UserRole `bun:"roles,array"`
	IsBlocked     bool                  `bun:"is_blocked,notnull,default:false"`
	CreatedAt     time.Time             `bun:"created_at,notnull,default:current_timestamp"`
}

type Session struct {
	Entity
	bun.BaseModel `bun:"table:sessions"`

	UserID       int64                       `bun:"user_id,notnull"`
	CreatedAt    time.Time                   `bun:"created_at,notnull,default:current_timestamp"`
	ExpiresAt    *time.Time                  `bun:"expires_at"` // expire after created_at + TTL
	UserAgent    string                      `bun:"user_agent,notnull"`
	IP           string                      `bun:"ip,notnull"`
	LastActivity time.Time                   `bun:"last_activity,notnull,default:current_timestamp"`
	Status       sessiondomain.SessionStatus `bun:"status,notnull,default:'active'"`
}

// --- SEASON ---

type Season struct {
	Entity
	bun.BaseModel `bun:"table:seasons"`

	Title     string    `bun:"title,notnull"`
	StartDate time.Time `bun:"start_date,notnull"`
	EndDate   time.Time `bun:"end_date,notnull"`
	IsActive  bool      `bun:"is_active,notnull,default:false"`
}

type SeasonEvent struct {
	Entity
	bun.BaseModel `bun:"table:season_events"`

	SeasonID    int64     `bun:"season_id,notnull"`
	Title       string    `bun:"title,notnull"`
	Description *string   `bun:"description"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:current_timestamp"`
	EventStart  time.Time `bun:"event_start,notnull"`
	EventEnd    time.Time `bun:"event_end,notnull"`
	CoinsReward *int      `bun:"coins_reward"`
}

// --- SEASONAL MEMBERS (WALLETS) ---

type SeasonMember struct {
	Entity
	bun.BaseModel `bun:"table:season_members"`

	UserID      int64     `bun:"user_id,notnull"`
	SeasonID    int64     `bun:"season_id,notnull"`
	Balance     int       `bun:"balance,notnull,default:0"`
	TotalEarned int       `bun:"total_earned,notnull,default:0"`
	UpdatedAt   time.Time `bun:"updated_at,notnull"`

	User   *User   `bun:"rel:belongs-to,join:user_id=id"`
	Season *Season `bun:"rel:belongs-to,join:season_id=id"`
}

type Transaction struct {
	Entity
	bun.BaseModel `bun:"table:transactions"`

	MemberID  int64                          `bun:"member_id,notnull"`
	Amount    int                            `bun:"amount,notnull"`
	Reason    walletdomain.TransactionReason `bun:"reason,notnull"`
	RefID     *int64                         `bun:"ref_id"`
	RefTitle  string                         `bun:"ref_title"`
	CreatedAt time.Time                      `bun:"created_at,notnull,default:current_timestamp"`

	Member *SeasonMember `bun:"rel:belongs-to,join:member_id=id"`
}

// --- ИГРОВАЯ ЛОГИКА (БИНГО) ---

type Task struct {
	Entity
	bun.BaseModel `bun:"table:tasks"`

	SeasonID    int64           `bun:"season_id,notnull"`
	Category    string          `bun:"category,notnull"`
	Title       string          `bun:"title,notnull"`
	Description string          `bun:"description"`
	RewardCoins int             `bun:"reward_coins,notnull,default:0"`
	SortOrder   int             `bun:"sort_order,notnull,default:0"`
	Metadata    json.RawMessage `bun:"metadata,type:jsonb"`
	IsActive    bool            `bun:"is_active,notnull,default:true"`

	Season *Season `bun:"rel:belongs-to,join:season_id=id"`
}

type TaskSubmission struct {
	Entity
	bun.BaseModel `bun:"table:task_submissions"`

	UserID        int64                                 `bun:"user_id,notnull"`
	TaskID        int64                                 `bun:"task_id,notnull"`
	Status        submissiondomain.TaskSubmissionStatus `bun:"status,notnull,default:'pending'"`
	Comment       string                                `bun:"comment"`
	ReviewComment string                                `bun:"review_comment"`
	ReviewerID    *int64                                `bun:"reviewer_id"`
	SubmittedAt   time.Time                             `bun:"submitted_at,notnull,default:current_timestamp"`
	ReviewedAt    *time.Time                            `bun:"reviewed_at"`

	User     *User `bun:"rel:belongs-to,join:user_id=id"`
	Task     *Task `bun:"rel:belongs-to,join:task_id=id"`
	Reviewer *User `bun:"rel:belongs-to,join:reviewer_id=id"`
}

// --- МАГАЗИН (REWARDS) ---

type Reward struct {
	Entity
	bun.BaseModel `bun:"table:rewards"`

	SeasonID    int64                     `bun:"season_id,notnull"`
	Title       string                    `bun:"title,notnull"`
	Description string                    `bun:"description"`
	CostCoins   int                       `bun:"cost_coins,notnull,default:0"`
	Limit       *int                      `bun:"limit"`
	Status      rewarddomain.RewardStatus `bun:"status,notnull,default:'available'"`

	Season *Season `bun:"rel:belongs-to,join:season_id=id"`
}

type RewardClaim struct {
	Entity
	bun.BaseModel `bun:"table:reward_claims"`

	UserID     int64                          `bun:"user_id,notnull"`
	RewardID   int64                          `bun:"reward_id,notnull"`
	Status     rewarddomain.RewardClaimStatus `bun:"status,notnull,default:'pending'"`
	SpentCoins int                            `bun:"spent_coins,notnull"`
	CreatedAt  time.Time                      `bun:"created_at,notnull,default:current_timestamp"`

	User   *User   `bun:"rel:belongs-to,join:user_id=id"`
	Reward *Reward `bun:"rel:belongs-to,join:reward_id=id"`
}

// --- NOTIFICATION JOBS ---

// NotificationJob is a transactional-outbox record describing a fan-out notification.
// It is enqueued atomically with the originating business write (e.g. season activation),
// then picked up by the notification worker which paginates recipients and streams
// per-user messages to the notifier service. Cursor enables resume after crash.
//
// Filter is stored as opaque JSONB; the notification bounded context owns its schema
// (see internal/contracts/notification.RecipientFilter). Status is stored as a plain
// string for the same reason — the persistence layer doesn't model business enums.
type NotificationJob struct {
	Entity
	bun.BaseModel `bun:"table:notification_jobs"`

	Type      string            `bun:"type,notnull"`
	Text      string            `bun:"text,notnull"`
	Args      map[string]string `bun:"args,type:jsonb,notnull"`
	Filter    json.RawMessage   `bun:"filter,type:jsonb,notnull"`
	Status    string            `bun:"status,notnull,default:'pending'"`
	Cursor    int64             `bun:"cursor,notnull,default:0"`
	Attempts  int               `bun:"attempts,notnull,default:0"`
	LastError *string           `bun:"last_error"`
	LockedAt  *time.Time        `bun:"locked_at"`
	CreatedAt time.Time         `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt time.Time         `bun:"updated_at,notnull,default:current_timestamp"`
}

// --- MESSAGE TEMPLATES ---

// Template stores a named push notification body with {{argKey}} placeholder syntax.
// Codename is immutable after creation; only Body is mutable.
type Template struct {
	Entity
	bun.BaseModel `bun:"table:message_templates"`

	Codename  string    `bun:"codename,notnull"`
	Body      string    `bun:"body,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}

// TemplateHistory records each body change of a Template.
type TemplateHistory struct {
	Entity
	bun.BaseModel `bun:"table:message_template_history"`

	TemplateID int64     `bun:"template_id,notnull"`
	Body       string    `bun:"body,notnull"`
	ChangedBy  int64     `bun:"changed_by,notnull"`
	ChangedAt  time.Time `bun:"changed_at,notnull,default:current_timestamp"`
}

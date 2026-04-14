package db

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
	"go.mod/internal/domain"
)

type Entity struct {
	ID int64 `bun:"id,pk,autoincrement"`
}

// --- Users ---

type User struct {
	Entity
	bun.BaseModel `bun:"table:users"`
	TelegramID    int64             `bun:"telegram_id,notnull"`
	Name          string            `bun:"name,notnull"`
	Username      string            `bun:"username,notnull"`
	PhotoURL      string            `bun:"photo_url"`
	Roles         []domain.UserRole `bun:"roles,array"`
	IsBlocked     bool              `bun:"is_blocked,notnull,default:false"`
	CreatedAt     time.Time         `bun:"created_at,notnull,default:current_timestamp"`
}

type Session struct {
	Entity
	bun.BaseModel `bun:"table:sessions"`

	UserID       int64                `bun:"user_id,notnull"`
	CreatedAt    time.Time            `bun:"created_at,notnull,default:current_timestamp"`
	ExpiresAt    *time.Time           `bun:"expires_at"` // expire after created_at + TTL
	UserAgent    string               `bun:"user_agent,notnull"`
	IP           string               `bun:"ip,notnull"`
	LastActivity time.Time            `bun:"last_activity,notnull,default:current_timestamp"`
	Status       domain.SessionStatus `bun:"status,notnull,default:'active'"`
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

	MemberID  int64                    `bun:"member_id,notnull"`
	Amount    int                      `bun:"amount,notnull"`
	Reason    domain.TransactionReason `bun:"reason,notnull"`
	RefID     *int64                   `bun:"ref_id"`
	RefTitle  string                   `bun:"ref_title"`
	CreatedAt time.Time                `bun:"created_at,notnull,default:current_timestamp"`

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

	UserID        int64                       `bun:"user_id,notnull"`
	TaskID        int64                       `bun:"task_id,notnull"`
	Status        domain.TaskSubmissionStatus `bun:"status,notnull,default:'pending'"`
	ReviewComment string                      `bun:"review_comment"`
	ReviewerID    *int64                      `bun:"reviewer_id"`
	SubmittedAt   time.Time                   `bun:"submitted_at,notnull,default:current_timestamp"`
	ReviewedAt    *time.Time                  `bun:"reviewed_at"`

	User     *User `bun:"rel:belongs-to,join:user_id=id"`
	Task     *Task `bun:"rel:belongs-to,join:task_id=id"`
	Reviewer *User `bun:"rel:belongs-to,join:reviewer_id=id"`
}

// --- МАГАЗИН (REWARDS) ---

type Reward struct {
	Entity
	bun.BaseModel `bun:"table:rewards"`

	SeasonID    int64               `bun:"season_id,notnull"`
	Title       string              `bun:"title,notnull"`
	Description string              `bun:"description"`
	CostCoins   int                 `bun:"cost_coins,notnull,default:0"`
	Limit       *int                `bun:"limit"`
	Status      domain.RewardStatus `bun:"status,notnull,default:'available'"`

	Season *Season `bun:"rel:belongs-to,join:season_id=id"`
}

type RewardClaim struct {
	Entity
	bun.BaseModel `bun:"table:reward_claims"`

	UserID     int64                    `bun:"user_id,notnull"`
	RewardID   int64                    `bun:"reward_id,notnull"`
	Status     domain.RewardClaimStatus `bun:"status,notnull,default:'pending'"`
	SpentCoins int                      `bun:"spent_coins,notnull"`
	CreatedAt  time.Time                `bun:"created_at,notnull,default:current_timestamp"`

	User   *User   `bun:"rel:belongs-to,join:user_id=id"`
	Reward *Reward `bun:"rel:belongs-to,join:reward_id=id"`
}

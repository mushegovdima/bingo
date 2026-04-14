package domain

import (
	"encoding/json"
	"time"
)

// Enums

type UserRole string

const (
	Manager  UserRole = "manager"
	Resident UserRole = "resident"
)

type RewardClaimStatus string

const (
	ClaimPending   RewardClaimStatus = "pending"
	ClaimCompleted RewardClaimStatus = "completed"
	ClaimCancelled RewardClaimStatus = "cancelled"
)

type RewardStatus string

const (
	RewardAvailable RewardStatus = "available"
	RewardHidden    RewardStatus = "hidden"
)

type TaskSubmissionStatus string

const (
	SubmissionPending  TaskSubmissionStatus = "pending"
	SubmissionApproved TaskSubmissionStatus = "approved"
	SubmissionRejected TaskSubmissionStatus = "rejected"
)

type SessionStatus string

const (
	SessionActive   SessionStatus = "active"
	SessionExpired  SessionStatus = "expired"
	SessionInactive SessionStatus = "inactive"
)

type TransactionReason string

const (
	TransactionReasonEvent  TransactionReason = "event"
	TransactionReasonTask   TransactionReason = "task"
	TransactionReasonManual TransactionReason = "manual"
	TransactionReasonReward TransactionReason = "reward"
)

// --- Users ---

// User
type User struct {
	ID         int64      `json:"id"`
	TelegramID int64      `json:"telegram_id"`
	Name       string     `json:"name"`
	Username   string     `json:"username"`
	PhotoURL   string     `json:"photo_url,omitempty"`
	Roles      []UserRole `json:"roles"` // "resident", "manager"
	IsBlocked  bool       `json:"is_blocked"`
	CreatedAt  time.Time  `json:"created_at"`
}

// User session
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

// --- SEASONS ---

// Season for Bingo game. For example, "Spring 2026" with specific start/end dates and rules.
type Season struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"` // "Марафон: Весна 2026"
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsActive  bool      `json:"is_active"` // Активен ли сезон для новых отчетов
}

// Event in season
type SeasonEvent struct {
	ID          int64     `json:"id"`
	SeasonID    int64     `json:"season_id"`
	Title       string    `json:"title"` // "Online meeting with experts"
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	EventStart  time.Time `json:"event_start"`
	EventEnd    time.Time `json:"event_end"`
	CoinsReward *int      `json:"coins_reward"` // Бонусные монеты за участие в событии. Могут быть изменены при начислении
}

// --- SEASONAL BALANCES (WALLETS) ---

// SeasonMember — это и есть "результат сезона" для конкретного юзера.
type SeasonMember struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	SeasonID    int64     `json:"season_id"`    // Каждый сезон — свой баланс
	Balance     int       `json:"balance"`      // Сколько коинов осталось (или было на конец сезона)
	TotalEarned int       `json:"total_earned"` // Вся сумма за сезон (для лидерборда)
	UpdatedAt   time.Time `json:"updated_at"`
}

// SeasonMemberWithSeason combines a user's seasonal balance with season details.
// Returned by GET /balance/my to avoid a second round-trip for season info.
type SeasonMemberWithSeason struct {
	SeasonMember
	Season Season `json:"season"`
}

// Transaction — детальный лог, почему изменился баланс в этом сезоне.
type Transaction struct {
	ID        int64             `json:"id"`
	MemberID  int64             `json:"member_id"` // Привязка к записи участника сезона
	Amount    int               `json:"amount"`    // Напр. +200 или -500
	Reason    TransactionReason `json:"reason"`    // "event", "task", "reward", "manual"
	RefID     *int64            `json:"ref_id"`    // ID задачи или награды (напр. "task:12")
	RefTitle  string            `json:"ref_title"` // Название задачи или награды (напр. "Написать пост о том, как я провел выходные")
	CreatedAt time.Time         `json:"created_at"`
}

// --- ИГРОВАЯ ЛОГИКА (БИНГО) ---

// Task — это задание в сетке БИНГО. Например, "Написать пост о том, как я провел выходные".
type Task struct {
	ID          int64           `json:"id"`
	SeasonID    int64           `json:"season_id"`
	Category    string          `json:"category"` // "family", "self_care", etc.
	Title       string          `json:"title"`
	Description string          `json:"description"`
	RewardCoins int             `json:"reward_coins"`
	SortOrder   int             `json:"sort_order"` // Позиция в сетке БИНГО
	Metadata    json.RawMessage `json:"metadata"`   // Доп. условия (напр. {"min_words": 100})
	IsActive    bool            `json:"is_active"`
}

// TaskSubmission — Отчет пользователя по задаче.
type TaskSubmission struct {
	ID            int64                `json:"id"`
	UserID        int64                `json:"user_id"`
	TaskID        int64                `json:"task_id"`
	Status        TaskSubmissionStatus `json:"status"`         // "pending", "approved", "rejected"
	ReviewComment string               `json:"review_comment"` // Почему отклонили
	ReviewerID    *int64               `json:"reviewer_id"`
	SubmittedAt   time.Time            `json:"submitted_at"`
	ReviewedAt    *time.Time           `json:"reviewed_at"`
}

// --- МАГАЗИН (REWARDS) ---

// Reward — приз, который можно купить за заработанные монеты.
type Reward struct {
	ID          int64        `json:"id"`
	SeasonID    int64        `json:"season_id"` // Приз доступен только в этом сезоне
	Title       string       `json:"title"`
	Description string       `json:"description"`
	CostCoins   int          `json:"cost_coins"`
	Limit       *int         `json:"limit"`  // Лимит (напр. 5 штук), если nil — бесконечно
	Status      RewardStatus `json:"status"` // "available", "sold_out", "hidden"
}

// RewardClaim — заявка пользователя на покупку приза.
type RewardClaim struct {
	ID         int64             `json:"id"`
	UserID     int64             `json:"user_id"`
	RewardID   int64             `json:"reward_id"`
	Status     RewardClaimStatus `json:"status"`
	SpentCoins int               `json:"spent_coins"` // Фиксируем цену на момент покупки
	CreatedAt  time.Time         `json:"created_at"`
}

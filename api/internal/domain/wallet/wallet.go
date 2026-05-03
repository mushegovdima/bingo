package wallet

import (
	"errors"
	"time"

	seasondomain "go.mod/internal/domain/season"
)

// Domain invariant errors.
var (
	// ErrNonPositiveAmount means a credit/debit was called with amount <= 0.
	ErrNonPositiveAmount = errors.New("amount must be positive")
	// ErrInsufficientBalance means a debit would drive balance below zero.
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// TransactionReason describes why a wallet's balance changed.
type TransactionReason string

const (
	TransactionReasonEvent  TransactionReason = "event"
	TransactionReasonTask   TransactionReason = "task"
	TransactionReasonManual TransactionReason = "manual"
	TransactionReasonReward TransactionReason = "reward"
)

// SeasonMember is the per-season wallet aggregate. Balance is guarded by the
// Credit/Debit/Adjust methods so it can never go negative through domain
// operations (the SQL repo enforces the same invariant atomically for
// concurrent writers).
type SeasonMember struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	SeasonID    int64     `json:"season_id"`
	Balance     int       `json:"balance"`
	TotalEarned int       `json:"total_earned"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SeasonMemberWithSeason combines a user's seasonal wallet with the parent
// season — used by GET /balance/my so the client gets both in one call.
type SeasonMemberWithSeason struct {
	SeasonMember
	Season seasondomain.Season `json:"season"`
}

// LeaderboardEntry is one row in the leaderboard: a participant ranked by balance.
type LeaderboardEntry struct {
	Position  int    `json:"position"`
	UserID    int64  `json:"user_id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url,omitempty"`
	Balance   int    `json:"balance"`
	IsCurrent bool   `json:"is_current"`
}

// Transaction is the immutable ledger entry explaining a wallet balance
// change. RefID/RefTitle reference the originating entity (task, reward, ...).
type Transaction struct {
	ID        int64             `json:"id"`
	MemberID  int64             `json:"member_id"`
	Amount    int               `json:"amount"`
	Reason    TransactionReason `json:"reason"`
	RefID     *int64            `json:"ref_id"`
	RefTitle  string            `json:"ref_title"`
	CreatedAt time.Time         `json:"created_at"`
}

// Credit increases the wallet balance by amount and records a positive
// Transaction. Returns ErrNonPositiveAmount when amount <= 0.
func (m *SeasonMember) Credit(amount int, reason TransactionReason, refID *int64, refTitle string, now time.Time) (Transaction, error) {
	if amount <= 0 {
		return Transaction{}, ErrNonPositiveAmount
	}
	m.Balance += amount
	m.TotalEarned += amount
	m.UpdatedAt = now
	return Transaction{
		MemberID:  m.ID,
		Amount:    amount,
		Reason:    reason,
		RefID:     refID,
		RefTitle:  refTitle,
		CreatedAt: now,
	}, nil
}

// Debit decreases the wallet balance by amount and records a negative
// Transaction. Returns ErrNonPositiveAmount when amount <= 0 and
// ErrInsufficientBalance when balance < amount.
func (m *SeasonMember) Debit(amount int, reason TransactionReason, refID *int64, refTitle string, now time.Time) (Transaction, error) {
	if amount <= 0 {
		return Transaction{}, ErrNonPositiveAmount
	}
	if m.Balance < amount {
		return Transaction{}, ErrInsufficientBalance
	}
	m.Balance -= amount
	m.UpdatedAt = now
	return Transaction{
		MemberID:  m.ID,
		Amount:    -amount,
		Reason:    reason,
		RefID:     refID,
		RefTitle:  refTitle,
		CreatedAt: now,
	}, nil
}

// Adjust applies a signed delta (e.g. manual manager change). A negative
// delta that would overdraw the balance returns ErrInsufficientBalance; a
// zero delta returns ErrNonPositiveAmount.
func (m *SeasonMember) Adjust(delta int, reason TransactionReason, refID *int64, refTitle string, now time.Time) (Transaction, error) {
	switch {
	case delta == 0:
		return Transaction{}, ErrNonPositiveAmount
	case delta > 0:
		return m.Credit(delta, reason, refID, refTitle, now)
	default:
		return m.Debit(-delta, reason, refID, refTitle, now)
	}
}

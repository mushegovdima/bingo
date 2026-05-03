// Package wallet defines cross-bounded-context DTOs for wallet operations.
// Other services (submission, reward) depend on these types instead of importing balanceservice directly,
// so the wallet bounded context can be split into a separate service later without API churn.
package wallet

import (
	walletdomain "go.mod/internal/domain/wallet"
)

// ErrInsufficientBalance is returned when a debit operation cannot be completed
// because the user's balance is below the requested amount.
//
// This is an alias of the canonical domain sentinel; both refer to the same
// error value, so callers can match on either via errors.Is.
var ErrInsufficientBalance = walletdomain.ErrInsufficientBalance

// CreditRequest credits coins to a user's balance for a given reason (task, event, refund, manual).
type CreditRequest struct {
	UserID   int64
	SeasonID int64
	Amount   int
	Reason   walletdomain.TransactionReason
	RefID    *int64
	RefTitle string
}

// DebitRequest deducts coins from a user's balance (reward purchase) or refunds them
// (cancelled claim — RefundCoins reuses the same shape).
// SpendCoins returns ErrInsufficientBalance when the user cannot afford the amount.
type DebitRequest struct {
	UserID   int64
	SeasonID int64
	Amount   int
	RefID    *int64
	RefTitle string
}

// ChangeRequest is a manager-facing manual balance change. A positive Amount
// tops the balance up; a negative Amount deducts from it. Note is stored as the
// transaction's RefTitle (RefID is always nil — there is no external entity).
type ChangeRequest struct {
	UserID   int64
	SeasonID int64
	Amount   int
	Note     string
}

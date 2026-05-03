package reward

import (
	"errors"
	"time"
)

// Domain invariant errors for RewardClaim.
var (
	// ErrClaimNotFound means a reward claim lookup found no record.
	ErrClaimNotFound = errors.New("reward claim not found")
	// ErrClaimNotPending means a status transition was requested on a claim
	// that is no longer pending.
	ErrClaimNotPending = errors.New("reward claim is not pending")
)

// RewardClaimStatus enumerates lifecycle states of a RewardClaim.
type RewardClaimStatus string

const (
	ClaimPending   RewardClaimStatus = "pending"
	ClaimCompleted RewardClaimStatus = "completed"
	ClaimCancelled RewardClaimStatus = "cancelled"
)

// RewardClaim is a user's purchase request for a Reward. SpentCoins captures
// the price at purchase time so later reward edits don't change history.
// State transitions: pending → completed | cancelled.
type RewardClaim struct {
	ID         int64             `json:"id"`
	UserID     int64             `json:"user_id"`
	RewardID   int64             `json:"reward_id"`
	Status     RewardClaimStatus `json:"status"`
	SpentCoins int               `json:"spent_coins"`
	CreatedAt  time.Time         `json:"created_at"`
}

// IsPending reports whether the claim is awaiting handling.
func (c *RewardClaim) IsPending() bool { return c.Status == ClaimPending }

// Complete marks a pending claim as fulfilled.
func (c *RewardClaim) Complete() error {
	if !c.IsPending() {
		return ErrClaimNotPending
	}
	c.Status = ClaimCompleted
	return nil
}

// Cancel marks a pending claim as cancelled. The caller is responsible for
// triggering the coin refund (cross-aggregate effect).
func (c *RewardClaim) Cancel() error {
	if !c.IsPending() {
		return ErrClaimNotPending
	}
	c.Status = ClaimCancelled
	return nil
}

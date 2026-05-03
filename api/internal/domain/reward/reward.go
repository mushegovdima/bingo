package reward

import (
	"errors"
	"time"
)

// Domain invariant errors for Reward.
var (
	// ErrNotFound means a reward lookup found no record.
	ErrNotFound = errors.New("reward not found")
	// ErrRewardUnavailable means the reward cannot currently be purchased
	// (hidden or otherwise inactive).
	ErrRewardUnavailable = errors.New("reward is not available")
	// ErrInvalidCost means a non-positive cost was supplied.
	ErrInvalidCost = errors.New("reward cost must be positive")
	// ErrLimitExceeded means the reward's purchase cap has been reached.
	ErrLimitExceeded = errors.New("reward limit exceeded")
)

// RewardStatus enumerates visibility/availability states of a Reward.
type RewardStatus string

const (
	RewardAvailable RewardStatus = "available"
	RewardHidden    RewardStatus = "hidden"
)

// Reward is a purchasable item priced in season coins, scoped to a season.
// Limit may cap total purchases (nil = unlimited).
type Reward struct {
	ID          int64        `json:"id"`
	SeasonID    int64        `json:"season_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	CostCoins   int          `json:"cost_coins"`
	Limit       *int         `json:"limit"`
	Status      RewardStatus `json:"status"`
}

// IsPurchasable reports whether the reward is currently buyable.
func (r *Reward) IsPurchasable() bool { return r.Status == RewardAvailable }

// Hide marks the reward as not displayed in the catalogue.
func (r *Reward) Hide() { r.Status = RewardHidden }

// Show makes the reward visible and purchasable again.
func (r *Reward) Show() { r.Status = RewardAvailable }

// NewClaim creates a pending RewardClaim for the given user, snapshotting
// the current reward price into SpentCoins. Returns ErrRewardUnavailable
// when the reward is not purchasable. Stock/limit checks are the repository's
// responsibility (they require a count query).
func (r *Reward) NewClaim(userID int64, now time.Time) (*RewardClaim, error) {
	if !r.IsPurchasable() {
		return nil, ErrRewardUnavailable
	}
	if r.CostCoins <= 0 {
		return nil, ErrInvalidCost
	}
	return &RewardClaim{
		UserID:     userID,
		RewardID:   r.ID,
		Status:     ClaimPending,
		SpentCoins: r.CostCoins,
		CreatedAt:  now,
	}, nil
}

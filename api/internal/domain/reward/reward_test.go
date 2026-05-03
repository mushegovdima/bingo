package reward_test

import (
	"errors"
	"testing"
	"time"

	rewarddomain "go.mod/internal/domain/reward"
)

func TestReward_NewClaim(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	t.Run("snapshots price and creates pending claim", func(t *testing.T) {
		r := &rewarddomain.Reward{ID: 5, CostCoins: 200, Status: rewarddomain.RewardAvailable}
		c, err := r.NewClaim(42, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.UserID != 42 || c.RewardID != 5 || c.SpentCoins != 200 {
			t.Errorf("unexpected claim: %+v", c)
		}
		if c.Status != rewarddomain.ClaimPending {
			t.Errorf("Status: got %q, want pending", c.Status)
		}
		if !c.CreatedAt.Equal(now) {
			t.Errorf("CreatedAt: got %v, want %v", c.CreatedAt, now)
		}

		// price changes on the reward must not affect already-issued claims
		r.CostCoins = 999
		if c.SpentCoins != 200 {
			t.Errorf("claim price was not snapshotted: %d", c.SpentCoins)
		}
	})

	t.Run("rejects when hidden", func(t *testing.T) {
		r := &rewarddomain.Reward{CostCoins: 100, Status: rewarddomain.RewardHidden}
		if _, err := r.NewClaim(1, now); !errors.Is(err, rewarddomain.ErrRewardUnavailable) {
			t.Errorf("got err=%v, want ErrRewardUnavailable", err)
		}
	})

	t.Run("rejects non-positive cost", func(t *testing.T) {
		r := &rewarddomain.Reward{CostCoins: 0, Status: rewarddomain.RewardAvailable}
		if _, err := r.NewClaim(1, now); !errors.Is(err, rewarddomain.ErrInvalidCost) {
			t.Errorf("got err=%v, want ErrInvalidCost", err)
		}
	})
}

func TestReward_HideShow(t *testing.T) {
	r := &rewarddomain.Reward{Status: rewarddomain.RewardAvailable}
	r.Hide()
	if r.IsPurchasable() {
		t.Errorf("hidden reward must not be purchasable")
	}
	r.Show()
	if !r.IsPurchasable() {
		t.Errorf("shown reward must be purchasable")
	}
}

func TestRewardClaim_Complete(t *testing.T) {
	t.Run("transitions pending to completed", func(t *testing.T) {
		c := &rewarddomain.RewardClaim{Status: rewarddomain.ClaimPending}
		if err := c.Complete(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Status != rewarddomain.ClaimCompleted {
			t.Errorf("Status: got %q", c.Status)
		}
	})

	t.Run("rejects non-pending claims", func(t *testing.T) {
		for _, status := range []rewarddomain.RewardClaimStatus{
			rewarddomain.ClaimCompleted,
			rewarddomain.ClaimCancelled,
		} {
			c := &rewarddomain.RewardClaim{Status: status}
			if err := c.Complete(); !errors.Is(err, rewarddomain.ErrClaimNotPending) {
				t.Errorf("status=%q: got err=%v, want ErrClaimNotPending", status, err)
			}
		}
	})
}

func TestRewardClaim_Cancel(t *testing.T) {
	t.Run("transitions pending to cancelled", func(t *testing.T) {
		c := &rewarddomain.RewardClaim{Status: rewarddomain.ClaimPending}
		if err := c.Cancel(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Status != rewarddomain.ClaimCancelled {
			t.Errorf("Status: got %q", c.Status)
		}
	})

	t.Run("rejects non-pending claims", func(t *testing.T) {
		c := &rewarddomain.RewardClaim{Status: rewarddomain.ClaimCompleted}
		if err := c.Cancel(); !errors.Is(err, rewarddomain.ErrClaimNotPending) {
			t.Errorf("got err=%v, want ErrClaimNotPending", err)
		}
	})
}

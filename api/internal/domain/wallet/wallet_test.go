package wallet_test

import (
	"errors"
	"testing"
	"time"

	walletdomain "go.mod/internal/domain/wallet"
)

func TestSeasonMember_Credit(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	t.Run("increases balance and total earned", func(t *testing.T) {
		m := &walletdomain.SeasonMember{ID: 7, Balance: 100, TotalEarned: 200}
		tx, err := m.Credit(50, walletdomain.TransactionReasonTask, nil, "task #1", now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Balance != 150 {
			t.Errorf("Balance: got %d, want 150", m.Balance)
		}
		if m.TotalEarned != 250 {
			t.Errorf("TotalEarned: got %d, want 250", m.TotalEarned)
		}
		if !m.UpdatedAt.Equal(now) {
			t.Errorf("UpdatedAt: got %v, want %v", m.UpdatedAt, now)
		}
		if tx.Amount != 50 || tx.MemberID != 7 || tx.Reason != walletdomain.TransactionReasonTask {
			t.Errorf("unexpected tx: %+v", tx)
		}
	})

	t.Run("rejects non-positive amount", func(t *testing.T) {
		for _, amount := range []int{0, -1, -100} {
			m := &walletdomain.SeasonMember{Balance: 10}
			_, err := m.Credit(amount, walletdomain.TransactionReasonManual, nil, "", now)
			if !errors.Is(err, walletdomain.ErrNonPositiveAmount) {
				t.Errorf("amount=%d: got err=%v, want ErrNonPositiveAmount", amount, err)
			}
			if m.Balance != 10 {
				t.Errorf("amount=%d: balance mutated to %d", amount, m.Balance)
			}
		}
	})
}

func TestSeasonMember_Debit(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	t.Run("decreases balance, not total earned", func(t *testing.T) {
		m := &walletdomain.SeasonMember{Balance: 100, TotalEarned: 200}
		tx, err := m.Debit(30, walletdomain.TransactionReasonReward, nil, "prize", now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Balance != 70 {
			t.Errorf("Balance: got %d, want 70", m.Balance)
		}
		if m.TotalEarned != 200 {
			t.Errorf("TotalEarned must not change on debit: got %d", m.TotalEarned)
		}
		if tx.Amount != -30 {
			t.Errorf("tx.Amount: got %d, want -30", tx.Amount)
		}
	})

	t.Run("returns ErrInsufficientBalance when overdrawn", func(t *testing.T) {
		m := &walletdomain.SeasonMember{Balance: 10}
		_, err := m.Debit(11, walletdomain.TransactionReasonReward, nil, "", now)
		if !errors.Is(err, walletdomain.ErrInsufficientBalance) {
			t.Fatalf("got err=%v, want ErrInsufficientBalance", err)
		}
		if m.Balance != 10 {
			t.Errorf("balance mutated to %d on failure", m.Balance)
		}
	})

	t.Run("rejects non-positive amount", func(t *testing.T) {
		m := &walletdomain.SeasonMember{Balance: 10}
		_, err := m.Debit(0, walletdomain.TransactionReasonReward, nil, "", now)
		if !errors.Is(err, walletdomain.ErrNonPositiveAmount) {
			t.Errorf("got err=%v, want ErrNonPositiveAmount", err)
		}
	})

	t.Run("allows debit to exactly zero", func(t *testing.T) {
		m := &walletdomain.SeasonMember{Balance: 50}
		if _, err := m.Debit(50, walletdomain.TransactionReasonReward, nil, "", now); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Balance != 0 {
			t.Errorf("Balance: got %d, want 0", m.Balance)
		}
	})
}

func TestSeasonMember_Adjust(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	t.Run("positive delta credits", func(t *testing.T) {
		m := &walletdomain.SeasonMember{Balance: 10}
		tx, err := m.Adjust(5, walletdomain.TransactionReasonManual, nil, "bonus", now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Balance != 15 || tx.Amount != 5 {
			t.Errorf("balance=%d tx.Amount=%d", m.Balance, tx.Amount)
		}
	})

	t.Run("negative delta debits", func(t *testing.T) {
		m := &walletdomain.SeasonMember{Balance: 10}
		tx, err := m.Adjust(-4, walletdomain.TransactionReasonManual, nil, "penalty", now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Balance != 6 || tx.Amount != -4 {
			t.Errorf("balance=%d tx.Amount=%d", m.Balance, tx.Amount)
		}
	})

	t.Run("zero delta is rejected", func(t *testing.T) {
		m := &walletdomain.SeasonMember{Balance: 10}
		_, err := m.Adjust(0, walletdomain.TransactionReasonManual, nil, "", now)
		if !errors.Is(err, walletdomain.ErrNonPositiveAmount) {
			t.Errorf("got err=%v, want ErrNonPositiveAmount", err)
		}
	})

	t.Run("negative delta refuses overdraw", func(t *testing.T) {
		m := &walletdomain.SeasonMember{Balance: 5}
		_, err := m.Adjust(-10, walletdomain.TransactionReasonManual, nil, "", now)
		if !errors.Is(err, walletdomain.ErrInsufficientBalance) {
			t.Errorf("got err=%v, want ErrInsufficientBalance", err)
		}
		if m.Balance != 5 {
			t.Errorf("balance mutated to %d", m.Balance)
		}
	})
}

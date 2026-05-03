package balanceservice_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	wallet "go.mod/internal/contracts/wallet"
	seasondomain "go.mod/internal/domain/season"
	walletdomain "go.mod/internal/domain/wallet"
	balanceservice "go.mod/internal/services/balance"
)

// --- fake repo ---

type fakeBalanceRepo struct {
	getByUserAndSeasonFn func(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error)
	listTransactionsFn   func(ctx context.Context, balanceID int64) ([]walletdomain.Transaction, error)
	adjustAndRecordFn    func(ctx context.Context, userID, seasonID int64, amount int, reason walletdomain.TransactionReason, refID *int64, refTitle string) (*walletdomain.Transaction, error)
	spendAndRecordFn     func(ctx context.Context, userID, seasonID int64, amount int, refID *int64, refTitle string) (*walletdomain.Transaction, error)
	listByUserFn         func(ctx context.Context, userID int64) ([]walletdomain.SeasonMemberWithSeason, error)
	ensureBalanceFn      func(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error)
}

func (f *fakeBalanceRepo) GetByUserAndSeason(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error) {
	if f.getByUserAndSeasonFn != nil {
		return f.getByUserAndSeasonFn(ctx, userID, seasonID)
	}
	return nil, nil
}

func (f *fakeBalanceRepo) ListTransactions(ctx context.Context, balanceID int64) ([]walletdomain.Transaction, error) {
	if f.listTransactionsFn != nil {
		return f.listTransactionsFn(ctx, balanceID)
	}
	return nil, nil
}

func (f *fakeBalanceRepo) AdjustAndRecord(ctx context.Context, userID, seasonID int64, amount int, reason walletdomain.TransactionReason, refID *int64, refTitle string) (*walletdomain.Transaction, error) {
	if f.adjustAndRecordFn != nil {
		return f.adjustAndRecordFn(ctx, userID, seasonID, amount, reason, refID, refTitle)
	}
	return &walletdomain.Transaction{ID: 1, Amount: amount, Reason: reason}, nil
}

func (f *fakeBalanceRepo) SpendAndRecord(ctx context.Context, userID, seasonID int64, amount int, refID *int64, refTitle string) (*walletdomain.Transaction, error) {
	if f.spendAndRecordFn != nil {
		return f.spendAndRecordFn(ctx, userID, seasonID, amount, refID, refTitle)
	}
	return &walletdomain.Transaction{ID: 2, Amount: -amount}, nil
}

func (f *fakeBalanceRepo) ListByUser(ctx context.Context, userID int64) ([]walletdomain.SeasonMemberWithSeason, error) {
	if f.listByUserFn != nil {
		return f.listByUserFn(ctx, userID)
	}
	return nil, nil
}

func (f *fakeBalanceRepo) EnsureBalance(ctx context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error) {
	if f.ensureBalanceFn != nil {
		return f.ensureBalanceFn(ctx, userID, seasonID)
	}
	return nil, nil
}

func (f *fakeBalanceRepo) GetLeaderboardNeighbors(_ context.Context, _, _ int64) ([]walletdomain.LeaderboardEntry, error) {
	return nil, nil
}

func (f *fakeBalanceRepo) GetFullLeaderboard(_ context.Context, _, _ int64) ([]walletdomain.LeaderboardEntry, error) {
	return nil, nil
}

func newBalanceService(repo *fakeBalanceRepo) *balanceservice.BalanceService {
	return balanceservice.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// --- GetBalance ---

func TestBalanceService_GetBalance(t *testing.T) {
	t.Run("returns nil when balance does not exist", func(t *testing.T) {
		svc := newBalanceService(&fakeBalanceRepo{})

		b, err := svc.GetBalance(context.Background(), 1, 1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b != nil {
			t.Fatalf("expected nil, got %v", b)
		}
	})

	t.Run("returns domain balance when found", func(t *testing.T) {
		repo := &fakeBalanceRepo{
			getByUserAndSeasonFn: func(_ context.Context, _, _ int64) (*walletdomain.SeasonMember, error) {
				return &walletdomain.SeasonMember{
					ID:          10,
					UserID:      1,
					SeasonID:    2,
					Balance:     500,
					TotalEarned: 700,
				}, nil
			},
		}
		svc := newBalanceService(repo)

		b, err := svc.GetBalance(context.Background(), 1, 2)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b == nil {
			t.Fatal("expected non-nil balance")
		}
		if b.Balance != 500 {
			t.Errorf("expected balance 500, got %d", b.Balance)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("db error")
		svc := newBalanceService(&fakeBalanceRepo{
			getByUserAndSeasonFn: func(_ context.Context, _, _ int64) (*walletdomain.SeasonMember, error) {
				return nil, repoErr
			},
		})

		_, err := svc.GetBalance(context.Background(), 1, 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- GetTransactions ---

func TestBalanceService_GetTransactions(t *testing.T) {
	t.Run("returns ErrBalanceNotFound when balance does not exist", func(t *testing.T) {
		svc := newBalanceService(&fakeBalanceRepo{})

		_, err := svc.GetTransactions(context.Background(), 1, 1)

		if !errors.Is(err, balanceservice.ErrBalanceNotFound) {
			t.Fatalf("expected ErrBalanceNotFound, got: %v", err)
		}
	})

	t.Run("returns transactions for existing balance", func(t *testing.T) {
		repo := &fakeBalanceRepo{
			getByUserAndSeasonFn: func(_ context.Context, _, _ int64) (*walletdomain.SeasonMember, error) {
				return &walletdomain.SeasonMember{ID: 10}, nil
			},
			listTransactionsFn: func(_ context.Context, balanceID int64) ([]walletdomain.Transaction, error) {
				if balanceID != 10 {
					return nil, errors.New("wrong balanceID")
				}
				return []walletdomain.Transaction{
					{ID: 1, Amount: 100},
					{ID: 2, Amount: -50},
				}, nil
			},
		}
		svc := newBalanceService(repo)

		txs, err := svc.GetTransactions(context.Background(), 1, 1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(txs) != 2 {
			t.Fatalf("expected 2 transactions, got %d", len(txs))
		}
	})

	t.Run("propagates repo ListTransactions error", func(t *testing.T) {
		repoErr := errors.New("list error")
		repo := &fakeBalanceRepo{
			getByUserAndSeasonFn: func(_ context.Context, _, _ int64) (*walletdomain.SeasonMember, error) {
				return &walletdomain.SeasonMember{ID: 1}, nil
			},
			listTransactionsFn: func(_ context.Context, _ int64) ([]walletdomain.Transaction, error) {
				return nil, repoErr
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.GetTransactions(context.Background(), 1, 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- ChangeBalance ---

func TestBalanceService_ChangeBalance(t *testing.T) {
	t.Run("credits positive amount with manual reason", func(t *testing.T) {
		var calledReason walletdomain.TransactionReason
		var calledAmount int

		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, amount int, reason walletdomain.TransactionReason, _ *int64, _ string) (*walletdomain.Transaction, error) {
				calledAmount = amount
				calledReason = reason
				return &walletdomain.Transaction{Amount: amount, Reason: reason}, nil
			},
		}
		svc := newBalanceService(repo)

		tx, err := svc.ChangeBalance(context.Background(), wallet.ChangeRequest{
			UserID: 1, SeasonID: 1, Amount: 200, Note: "bonus",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calledAmount != 200 {
			t.Errorf("expected amount 200, got %d", calledAmount)
		}
		if calledReason != walletdomain.TransactionReasonManual {
			t.Errorf("expected reason 'manual', got %q", calledReason)
		}
		if tx.Amount != 200 {
			t.Errorf("expected tx amount 200, got %d", tx.Amount)
		}
	})

	t.Run("deducts with negative amount", func(t *testing.T) {
		var calledAmount int
		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, amount int, _ walletdomain.TransactionReason, _ *int64, _ string) (*walletdomain.Transaction, error) {
				calledAmount = amount
				return &walletdomain.Transaction{Amount: amount}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.ChangeBalance(context.Background(), wallet.ChangeRequest{
			Amount: -100,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calledAmount != -100 {
			t.Errorf("expected amount -100, got %d", calledAmount)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("adjust failed")
		svc := newBalanceService(&fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ walletdomain.TransactionReason, _ *int64, _ string) (*walletdomain.Transaction, error) {
				return nil, repoErr
			},
		})

		_, err := svc.ChangeBalance(context.Background(), wallet.ChangeRequest{Amount: 100})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- AddCoins ---

func TestBalanceService_AddCoins(t *testing.T) {
	t.Run("credits coins with correct reason and ref", func(t *testing.T) {
		refID := int64(42)
		var gotReason walletdomain.TransactionReason
		var gotRefID *int64
		var gotRefTitle string

		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, _ int, reason walletdomain.TransactionReason, rid *int64, refTitle string) (*walletdomain.Transaction, error) {
				gotReason = reason
				gotRefID = rid
				gotRefTitle = refTitle
				return &walletdomain.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.AddCoins(context.Background(), wallet.CreditRequest{
			UserID:   1,
			SeasonID: 1,
			Amount:   50,
			Reason:   walletdomain.TransactionReasonTask,
			RefID:    &refID,
			RefTitle: "Task: Write code",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotReason != walletdomain.TransactionReasonTask {
			t.Errorf("expected reason 'task', got %q", gotReason)
		}
		if gotRefID == nil || *gotRefID != 42 {
			t.Errorf("expected refID 42, got %v", gotRefID)
		}
		if gotRefTitle != "Task: Write code" {
			t.Errorf("unexpected refTitle: %q", gotRefTitle)
		}
	})
}

// --- SpendCoins ---

func TestBalanceService_SpendCoins(t *testing.T) {
	t.Run("deducts coins successfully", func(t *testing.T) {
		svc := newBalanceService(&fakeBalanceRepo{})

		tx, err := svc.SpendCoins(context.Background(), wallet.DebitRequest{
			UserID: 1, SeasonID: 1, Amount: 30,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx == nil {
			t.Fatal("expected non-nil transaction")
		}
	})

	t.Run("returns ErrInsufficientBalance when repo signals insufficient funds", func(t *testing.T) {
		svc := newBalanceService(&fakeBalanceRepo{
			spendAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ *int64, _ string) (*walletdomain.Transaction, error) {
				return nil, walletdomain.ErrInsufficientBalance
			},
		})

		_, err := svc.SpendCoins(context.Background(), wallet.DebitRequest{Amount: 9999})

		if !errors.Is(err, wallet.ErrInsufficientBalance) {
			t.Fatalf("expected ErrInsufficientBalance, got: %v", err)
		}
	})

	t.Run("propagates generic repo error", func(t *testing.T) {
		repoErr := errors.New("spend failed")
		svc := newBalanceService(&fakeBalanceRepo{
			spendAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ *int64, _ string) (*walletdomain.Transaction, error) {
				return nil, repoErr
			},
		})

		_, err := svc.SpendCoins(context.Background(), wallet.DebitRequest{Amount: 10})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- RefundCoins ---

func TestBalanceService_RefundCoins(t *testing.T) {
	t.Run("refunds with reward reason", func(t *testing.T) {
		var gotReason walletdomain.TransactionReason

		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, _ int, reason walletdomain.TransactionReason, _ *int64, _ string) (*walletdomain.Transaction, error) {
				gotReason = reason
				return &walletdomain.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.RefundCoins(context.Background(), wallet.DebitRequest{
			UserID: 1, SeasonID: 1, Amount: 30,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotReason != walletdomain.TransactionReasonReward {
			t.Errorf("expected reason 'reward', got %q", gotReason)
		}
	})

	t.Run("forwards amount as-is — service must not negate it", func(t *testing.T) {
		var gotAmount int
		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, amount int, _ walletdomain.TransactionReason, _ *int64, _ string) (*walletdomain.Transaction, error) {
				gotAmount = amount
				return &walletdomain.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.RefundCoins(context.Background(), wallet.DebitRequest{Amount: 75})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotAmount != 75 {
			t.Errorf("expected repo to receive +75, got %d (service must not negate refund amount)", gotAmount)
		}
	})

	t.Run("forwards userID, seasonID, refID and refTitle unchanged", func(t *testing.T) {
		refID := int64(99)
		var gotUserID, gotSeasonID int64
		var gotRefID *int64
		var gotRefTitle string

		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, userID, seasonID int64, _ int, _ walletdomain.TransactionReason, rid *int64, refTitle string) (*walletdomain.Transaction, error) {
				gotUserID = userID
				gotSeasonID = seasonID
				gotRefID = rid
				gotRefTitle = refTitle
				return &walletdomain.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.RefundCoins(context.Background(), wallet.DebitRequest{
			UserID:   7,
			SeasonID: 3,
			Amount:   40,
			RefID:    &refID,
			RefTitle: "Refund: Gold reward",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotUserID != 7 {
			t.Errorf("expected userID 7, got %d", gotUserID)
		}
		if gotSeasonID != 3 {
			t.Errorf("expected seasonID 3, got %d", gotSeasonID)
		}
		if gotRefID == nil || *gotRefID != 99 {
			t.Errorf("expected refID 99, got %v", gotRefID)
		}
		if gotRefTitle != "Refund: Gold reward" {
			t.Errorf("unexpected refTitle: %q", gotRefTitle)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("refund failed")
		svc := newBalanceService(&fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ walletdomain.TransactionReason, _ *int64, _ string) (*walletdomain.Transaction, error) {
				return nil, repoErr
			},
		})

		_, err := svc.RefundCoins(context.Background(), wallet.DebitRequest{Amount: 10})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- Transactional field-forwarding ---

func TestBalanceService_ChangeBalance_FieldForwarding(t *testing.T) {
	t.Run("forwards note as refTitle and keeps refID nil", func(t *testing.T) {
		var gotRefID *int64
		var gotRefTitle string

		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ walletdomain.TransactionReason, rid *int64, refTitle string) (*walletdomain.Transaction, error) {
				gotRefID = rid
				gotRefTitle = refTitle
				return &walletdomain.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.ChangeBalance(context.Background(), wallet.ChangeRequest{
			UserID: 1, SeasonID: 1, Amount: 100, Note: "manual top-up",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotRefID != nil {
			t.Errorf("expected nil refID for manual change, got %v", gotRefID)
		}
		if gotRefTitle != "manual top-up" {
			t.Errorf("expected refTitle 'manual top-up', got %q", gotRefTitle)
		}
	})

	t.Run("forwards userID and seasonID without mixing them up", func(t *testing.T) {
		var gotUserID, gotSeasonID int64

		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, userID, seasonID int64, _ int, _ walletdomain.TransactionReason, _ *int64, _ string) (*walletdomain.Transaction, error) {
				gotUserID = userID
				gotSeasonID = seasonID
				return &walletdomain.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.ChangeBalance(context.Background(), wallet.ChangeRequest{
			UserID: 42, SeasonID: 7, Amount: 50,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotUserID != 42 {
			t.Errorf("expected userID 42, got %d", gotUserID)
		}
		if gotSeasonID != 7 {
			t.Errorf("expected seasonID 7, got %d", gotSeasonID)
		}
	})
}

func TestBalanceService_AddCoins_FieldForwarding(t *testing.T) {
	t.Run("forwards userID and seasonID without mixing them up", func(t *testing.T) {
		var gotUserID, gotSeasonID int64

		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, userID, seasonID int64, _ int, _ walletdomain.TransactionReason, _ *int64, _ string) (*walletdomain.Transaction, error) {
				gotUserID = userID
				gotSeasonID = seasonID
				return &walletdomain.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.AddCoins(context.Background(), wallet.CreditRequest{
			UserID: 11, SeasonID: 5, Amount: 100, Reason: walletdomain.TransactionReasonEvent,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotUserID != 11 {
			t.Errorf("expected userID 11, got %d", gotUserID)
		}
		if gotSeasonID != 5 {
			t.Errorf("expected seasonID 5, got %d", gotSeasonID)
		}
	})

	t.Run("zero amount is forwarded — service does not short-circuit", func(t *testing.T) {
		called := false
		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, amount int, _ walletdomain.TransactionReason, _ *int64, _ string) (*walletdomain.Transaction, error) {
				called = true
				return &walletdomain.Transaction{Amount: amount}, nil
			},
		}
		svc := newBalanceService(repo)

		tx, err := svc.AddCoins(context.Background(), wallet.CreditRequest{
			UserID: 1, SeasonID: 1, Amount: 0, Reason: walletdomain.TransactionReasonTask,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Fatal("repo was not called for zero-amount AddCoins")
		}
		if tx.Amount != 0 {
			t.Errorf("expected amount 0, got %d", tx.Amount)
		}
	})
}

func TestBalanceService_SpendCoins_FieldForwarding(t *testing.T) {
	t.Run("forwards userID, seasonID, amount, refID and refTitle to repo", func(t *testing.T) {
		refID := int64(55)
		var gotUserID, gotSeasonID int64
		var gotAmount int
		var gotRefID *int64
		var gotRefTitle string

		repo := &fakeBalanceRepo{
			spendAndRecordFn: func(_ context.Context, userID, seasonID int64, amount int, rid *int64, refTitle string) (*walletdomain.Transaction, error) {
				gotUserID = userID
				gotSeasonID = seasonID
				gotAmount = amount
				gotRefID = rid
				gotRefTitle = refTitle
				return &walletdomain.Transaction{Amount: -amount}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.SpendCoins(context.Background(), wallet.DebitRequest{
			UserID:   9,
			SeasonID: 4,
			Amount:   120,
			RefID:    &refID,
			RefTitle: "Gold reward",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotUserID != 9 {
			t.Errorf("expected userID 9, got %d", gotUserID)
		}
		if gotSeasonID != 4 {
			t.Errorf("expected seasonID 4, got %d", gotSeasonID)
		}
		if gotAmount != 120 {
			t.Errorf("expected amount 120 (positive — repo negates internally), got %d", gotAmount)
		}
		if gotRefID == nil || *gotRefID != 55 {
			t.Errorf("expected refID 55, got %v", gotRefID)
		}
		if gotRefTitle != "Gold reward" {
			t.Errorf("unexpected refTitle: %q", gotRefTitle)
		}
	})

	t.Run("ErrInsufficientBalance is reachable via errors.Is even when repo wraps it", func(t *testing.T) {
		svc := newBalanceService(&fakeBalanceRepo{
			spendAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ *int64, _ string) (*walletdomain.Transaction, error) {
				return nil, fmt.Errorf("spend: %w", walletdomain.ErrInsufficientBalance)
			},
		})

		_, err := svc.SpendCoins(context.Background(), wallet.DebitRequest{Amount: 50})

		if !errors.Is(err, wallet.ErrInsufficientBalance) {
			t.Fatalf("expected ErrInsufficientBalance, got: %v", err)
		}
		if !errors.Is(err, walletdomain.ErrInsufficientBalance) {
			t.Fatalf("contracts and domain sentinels must be the same value")
		}
	})
}

// --- ListUserBalances ---

func TestBalanceService_ListUserBalances(t *testing.T) {
	t.Run("returns empty slice when user has no balances", func(t *testing.T) {
		svc := newBalanceService(&fakeBalanceRepo{})

		out, err := svc.ListUserBalances(context.Background(), 1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out) != 0 {
			t.Fatalf("expected empty slice, got %d", len(out))
		}
	})

	t.Run("forwards repo result unchanged", func(t *testing.T) {
		expected := []walletdomain.SeasonMemberWithSeason{
			{
				SeasonMember: walletdomain.SeasonMember{
					ID: 1, UserID: 7, SeasonID: 100, Balance: 500, TotalEarned: 800,
				},
				Season: seasondomain.Season{ID: 100, Title: "Spring"},
			},
			{
				SeasonMember: walletdomain.SeasonMember{ID: 2, UserID: 7, SeasonID: 200, Balance: 10},
			},
		}

		svc := newBalanceService(&fakeBalanceRepo{
			listByUserFn: func(_ context.Context, userID int64) ([]walletdomain.SeasonMemberWithSeason, error) {
				if userID != 7 {
					t.Errorf("expected userID 7, got %d", userID)
				}
				return expected, nil
			},
		})

		out, err := svc.ListUserBalances(context.Background(), 7)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out) != 2 {
			t.Fatalf("expected 2 items, got %d", len(out))
		}
		if out[0].Balance != 500 || out[0].TotalEarned != 800 {
			t.Errorf("unexpected first balance: %+v", out[0].SeasonMember)
		}
		if out[0].Season.ID != 100 || out[0].Season.Title != "Spring" {
			t.Errorf("expected season {ID:100 Title:Spring}, got %+v", out[0].Season)
		}
		if out[1].Season.ID != 0 || out[1].Season.Title != "" {
			t.Errorf("expected zero-value season for missing relation, got %+v", out[1].Season)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("list failed")
		svc := newBalanceService(&fakeBalanceRepo{
			listByUserFn: func(_ context.Context, _ int64) ([]walletdomain.SeasonMemberWithSeason, error) {
				return nil, repoErr
			},
		})

		_, err := svc.ListUserBalances(context.Background(), 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- JoinSeason ---

func TestBalanceService_JoinSeason(t *testing.T) {
	t.Run("forwards userID and seasonID and returns domain balance", func(t *testing.T) {
		var gotUserID, gotSeasonID int64
		svc := newBalanceService(&fakeBalanceRepo{
			ensureBalanceFn: func(_ context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error) {
				gotUserID, gotSeasonID = userID, seasonID
				return &walletdomain.SeasonMember{
					ID:       42,
					UserID:   userID,
					SeasonID: seasonID,
					Balance:  0,
				}, nil
			},
		})

		b, err := svc.JoinSeason(context.Background(), 11, 5)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotUserID != 11 || gotSeasonID != 5 {
			t.Errorf("forwarded args: got user=%d season=%d, want 11/5", gotUserID, gotSeasonID)
		}
		if b == nil {
			t.Fatal("expected non-nil balance")
		}
		if b.ID != 42 || b.UserID != 11 || b.SeasonID != 5 || b.Balance != 0 {
			t.Errorf("unexpected balance: %+v", b)
		}
	})

	t.Run("idempotent — returns existing wallet without error", func(t *testing.T) {
		calls := 0
		svc := newBalanceService(&fakeBalanceRepo{
			ensureBalanceFn: func(_ context.Context, userID, seasonID int64) (*walletdomain.SeasonMember, error) {
				calls++
				return &walletdomain.SeasonMember{
					ID:          1,
					UserID:      userID,
					SeasonID:    seasonID,
					Balance:     250,
					TotalEarned: 400,
				}, nil
			},
		})

		first, err := svc.JoinSeason(context.Background(), 1, 1)
		if err != nil {
			t.Fatalf("first call: %v", err)
		}
		second, err := svc.JoinSeason(context.Background(), 1, 1)
		if err != nil {
			t.Fatalf("second call: %v", err)
		}
		if calls != 2 {
			t.Errorf("expected repo called twice, got %d", calls)
		}
		if first.Balance != 250 || second.Balance != 250 {
			t.Errorf("expected pre-existing balance preserved, got %d / %d", first.Balance, second.Balance)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("ensure failed")
		svc := newBalanceService(&fakeBalanceRepo{
			ensureBalanceFn: func(_ context.Context, _, _ int64) (*walletdomain.SeasonMember, error) {
				return nil, repoErr
			},
		})

		_, err := svc.JoinSeason(context.Background(), 1, 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

package balanceservice_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	dbmodels "go.mod/internal/db"
	"go.mod/internal/db/repository"
	"go.mod/internal/domain"
	balanceservice "go.mod/internal/services/balance"
)

// --- fake repo ---

type fakeBalanceRepo struct {
	getByUserAndSeasonFn func(ctx context.Context, userID, seasonID int64) (*dbmodels.SeasonMember, error)
	listTransactionsFn   func(ctx context.Context, balanceID int64) ([]dbmodels.Transaction, error)
	adjustAndRecordFn    func(ctx context.Context, userID, seasonID int64, amount int, reason domain.TransactionReason, refID *int64, refTitle string) (*dbmodels.Transaction, error)
	spendAndRecordFn     func(ctx context.Context, userID, seasonID int64, amount int, refID *int64, refTitle string) (*dbmodels.Transaction, error)
}

func (f *fakeBalanceRepo) GetByUserAndSeason(ctx context.Context, userID, seasonID int64) (*dbmodels.SeasonMember, error) {
	if f.getByUserAndSeasonFn != nil {
		return f.getByUserAndSeasonFn(ctx, userID, seasonID)
	}
	return nil, nil
}

func (f *fakeBalanceRepo) ListTransactions(ctx context.Context, balanceID int64) ([]dbmodels.Transaction, error) {
	if f.listTransactionsFn != nil {
		return f.listTransactionsFn(ctx, balanceID)
	}
	return nil, nil
}

func (f *fakeBalanceRepo) AdjustAndRecord(ctx context.Context, userID, seasonID int64, amount int, reason domain.TransactionReason, refID *int64, refTitle string) (*dbmodels.Transaction, error) {
	if f.adjustAndRecordFn != nil {
		return f.adjustAndRecordFn(ctx, userID, seasonID, amount, reason, refID, refTitle)
	}
	return &dbmodels.Transaction{Entity: dbmodels.Entity{ID: 1}, Amount: amount, Reason: reason}, nil
}

func (f *fakeBalanceRepo) SpendAndRecord(ctx context.Context, userID, seasonID int64, amount int, refID *int64, refTitle string) (*dbmodels.Transaction, error) {
	if f.spendAndRecordFn != nil {
		return f.spendAndRecordFn(ctx, userID, seasonID, amount, refID, refTitle)
	}
	return &dbmodels.Transaction{Entity: dbmodels.Entity{ID: 2}, Amount: -amount}, nil
}

func (f *fakeBalanceRepo) ListByUser(ctx context.Context, userID int64) ([]*dbmodels.SeasonMember, error) {
	return nil, nil
}

func (f *fakeBalanceRepo) EnsureBalance(ctx context.Context, userID, seasonID int64) (*dbmodels.SeasonMember, error) {
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
			getByUserAndSeasonFn: func(_ context.Context, _, _ int64) (*dbmodels.SeasonMember, error) {
				return &dbmodels.SeasonMember{
					Entity:      dbmodels.Entity{ID: 10},
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
			getByUserAndSeasonFn: func(_ context.Context, _, _ int64) (*dbmodels.SeasonMember, error) {
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
			getByUserAndSeasonFn: func(_ context.Context, _, _ int64) (*dbmodels.SeasonMember, error) {
				return &dbmodels.SeasonMember{Entity: dbmodels.Entity{ID: 10}}, nil
			},
			listTransactionsFn: func(_ context.Context, balanceID int64) ([]dbmodels.Transaction, error) {
				if balanceID != 10 {
					return nil, errors.New("wrong balanceID")
				}
				return []dbmodels.Transaction{
					{Entity: dbmodels.Entity{ID: 1}, Amount: 100},
					{Entity: dbmodels.Entity{ID: 2}, Amount: -50},
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
			getByUserAndSeasonFn: func(_ context.Context, _, _ int64) (*dbmodels.SeasonMember, error) {
				return &dbmodels.SeasonMember{Entity: dbmodels.Entity{ID: 1}}, nil
			},
			listTransactionsFn: func(_ context.Context, _ int64) ([]dbmodels.Transaction, error) {
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
		var calledReason domain.TransactionReason
		var calledAmount int

		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, amount int, reason domain.TransactionReason, _ *int64, _ string) (*dbmodels.Transaction, error) {
				calledAmount = amount
				calledReason = reason
				return &dbmodels.Transaction{Amount: amount, Reason: reason}, nil
			},
		}
		svc := newBalanceService(repo)

		tx, err := svc.ChangeBalance(context.Background(), balanceservice.ChangeBalanceRequest{
			UserID: 1, SeasonID: 1, Amount: 200, Note: "bonus",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calledAmount != 200 {
			t.Errorf("expected amount 200, got %d", calledAmount)
		}
		if calledReason != domain.TransactionReasonManual {
			t.Errorf("expected reason 'manual', got %q", calledReason)
		}
		if tx.Amount != 200 {
			t.Errorf("expected tx amount 200, got %d", tx.Amount)
		}
	})

	t.Run("deducts with negative amount", func(t *testing.T) {
		var calledAmount int
		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, amount int, _ domain.TransactionReason, _ *int64, _ string) (*dbmodels.Transaction, error) {
				calledAmount = amount
				return &dbmodels.Transaction{Amount: amount}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.ChangeBalance(context.Background(), balanceservice.ChangeBalanceRequest{
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
			adjustAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ domain.TransactionReason, _ *int64, _ string) (*dbmodels.Transaction, error) {
				return nil, repoErr
			},
		})

		_, err := svc.ChangeBalance(context.Background(), balanceservice.ChangeBalanceRequest{Amount: 100})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- AddCoins ---

func TestBalanceService_AddCoins(t *testing.T) {
	t.Run("credits coins with correct reason and ref", func(t *testing.T) {
		refID := int64(42)
		var gotReason domain.TransactionReason
		var gotRefID *int64
		var gotRefTitle string

		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, _ int, reason domain.TransactionReason, rid *int64, refTitle string) (*dbmodels.Transaction, error) {
				gotReason = reason
				gotRefID = rid
				gotRefTitle = refTitle
				return &dbmodels.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.AddCoins(context.Background(), balanceservice.AddCoinsRequest{
			UserID:   1,
			SeasonID: 1,
			Amount:   50,
			Reason:   domain.TransactionReasonTask,
			RefID:    &refID,
			RefTitle: "Task: Write code",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotReason != domain.TransactionReasonTask {
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

		tx, err := svc.SpendCoins(context.Background(), balanceservice.SpendCoinsRequest{
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
			spendAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ *int64, _ string) (*dbmodels.Transaction, error) {
				return nil, repository.AdjustErrInsufficientBalance
			},
		})

		_, err := svc.SpendCoins(context.Background(), balanceservice.SpendCoinsRequest{Amount: 9999})

		if !errors.Is(err, balanceservice.ErrInsufficientBalance) {
			t.Fatalf("expected ErrInsufficientBalance, got: %v", err)
		}
	})

	t.Run("propagates generic repo error", func(t *testing.T) {
		repoErr := errors.New("spend failed")
		svc := newBalanceService(&fakeBalanceRepo{
			spendAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ *int64, _ string) (*dbmodels.Transaction, error) {
				return nil, repoErr
			},
		})

		_, err := svc.SpendCoins(context.Background(), balanceservice.SpendCoinsRequest{Amount: 10})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- RefundCoins ---

func TestBalanceService_RefundCoins(t *testing.T) {
	t.Run("refunds with reward reason", func(t *testing.T) {
		var gotReason domain.TransactionReason

		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, _ int, reason domain.TransactionReason, _ *int64, _ string) (*dbmodels.Transaction, error) {
				gotReason = reason
				return &dbmodels.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.RefundCoins(context.Background(), balanceservice.SpendCoinsRequest{
			UserID: 1, SeasonID: 1, Amount: 30,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotReason != domain.TransactionReasonReward {
			t.Errorf("expected reason 'reward', got %q", gotReason)
		}
	})

	t.Run("forwards amount as-is — service must not negate it", func(t *testing.T) {
		// The repo (AdjustAndRecord) handles sign semantics; service must pass the raw amount.
		var gotAmount int
		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, amount int, _ domain.TransactionReason, _ *int64, _ string) (*dbmodels.Transaction, error) {
				gotAmount = amount
				return &dbmodels.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.RefundCoins(context.Background(), balanceservice.SpendCoinsRequest{Amount: 75})

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
			adjustAndRecordFn: func(_ context.Context, userID, seasonID int64, _ int, _ domain.TransactionReason, rid *int64, refTitle string) (*dbmodels.Transaction, error) {
				gotUserID = userID
				gotSeasonID = seasonID
				gotRefID = rid
				gotRefTitle = refTitle
				return &dbmodels.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.RefundCoins(context.Background(), balanceservice.SpendCoinsRequest{
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
			adjustAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ domain.TransactionReason, _ *int64, _ string) (*dbmodels.Transaction, error) {
				return nil, repoErr
			},
		})

		_, err := svc.RefundCoins(context.Background(), balanceservice.SpendCoinsRequest{Amount: 10})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- Transactional field-forwarding ---

func TestBalanceService_ChangeBalance_FieldForwarding(t *testing.T) {
	t.Run("forwards note as refTitle and keeps refID nil", func(t *testing.T) {
		// ChangeBalance has no external reference — refID must always be nil.
		var gotRefID *int64
		var gotRefTitle string

		repo := &fakeBalanceRepo{
			adjustAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ domain.TransactionReason, rid *int64, refTitle string) (*dbmodels.Transaction, error) {
				gotRefID = rid
				gotRefTitle = refTitle
				return &dbmodels.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.ChangeBalance(context.Background(), balanceservice.ChangeBalanceRequest{
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
			adjustAndRecordFn: func(_ context.Context, userID, seasonID int64, _ int, _ domain.TransactionReason, _ *int64, _ string) (*dbmodels.Transaction, error) {
				gotUserID = userID
				gotSeasonID = seasonID
				return &dbmodels.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.ChangeBalance(context.Background(), balanceservice.ChangeBalanceRequest{
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
			adjustAndRecordFn: func(_ context.Context, userID, seasonID int64, _ int, _ domain.TransactionReason, _ *int64, _ string) (*dbmodels.Transaction, error) {
				gotUserID = userID
				gotSeasonID = seasonID
				return &dbmodels.Transaction{}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.AddCoins(context.Background(), balanceservice.AddCoinsRequest{
			UserID: 11, SeasonID: 5, Amount: 100, Reason: domain.TransactionReasonEvent,
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
			adjustAndRecordFn: func(_ context.Context, _, _ int64, amount int, _ domain.TransactionReason, _ *int64, _ string) (*dbmodels.Transaction, error) {
				called = true
				return &dbmodels.Transaction{Amount: amount}, nil
			},
		}
		svc := newBalanceService(repo)

		tx, err := svc.AddCoins(context.Background(), balanceservice.AddCoinsRequest{
			UserID: 1, SeasonID: 1, Amount: 0, Reason: domain.TransactionReasonTask,
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
			spendAndRecordFn: func(_ context.Context, userID, seasonID int64, amount int, rid *int64, refTitle string) (*dbmodels.Transaction, error) {
				gotUserID = userID
				gotSeasonID = seasonID
				gotAmount = amount
				gotRefID = rid
				gotRefTitle = refTitle
				return &dbmodels.Transaction{Amount: -amount}, nil
			},
		}
		svc := newBalanceService(repo)

		_, err := svc.SpendCoins(context.Background(), balanceservice.SpendCoinsRequest{
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

	t.Run("ErrInsufficientBalance is not double-wrapped — errors.Is resolves through chain", func(t *testing.T) {
		// Even if the repo wraps AdjustErrInsufficientBalance, service must expose ErrInsufficientBalance.
		svc := newBalanceService(&fakeBalanceRepo{
			spendAndRecordFn: func(_ context.Context, _, _ int64, _ int, _ *int64, _ string) (*dbmodels.Transaction, error) {
				return nil, fmt.Errorf("spend: %w", repository.AdjustErrInsufficientBalance)
			},
		})

		_, err := svc.SpendCoins(context.Background(), balanceservice.SpendCoinsRequest{Amount: 50})

		if !errors.Is(err, balanceservice.ErrInsufficientBalance) {
			t.Fatalf("expected ErrInsufficientBalance, got: %v", err)
		}
		// Must NOT leak the internal repo sentinel to callers.
		if errors.Is(err, repository.AdjustErrInsufficientBalance) {
			t.Errorf("repo-level sentinel must not leak beyond service boundary")
		}
	})
}

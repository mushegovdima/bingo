package rewardservice_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"unsafe"

	"github.com/uptrace/bun/driver/pgdriver"
	notificationcontract "go.mod/internal/contracts/notification"
	wallet "go.mod/internal/contracts/wallet"
	rewarddomain "go.mod/internal/domain/reward"
	walletdomain "go.mod/internal/domain/wallet"
	"go.mod/internal/notifications"
	rewardservice "go.mod/internal/services/reward"
)

// --- pgdriver FK error helper ---

type pgErrLayout struct{ m map[byte]string }

func pgFKErr(code string) error {
	src := pgErrLayout{m: map[byte]string{'C': code}}
	return *(*pgdriver.Error)(unsafe.Pointer(&src))
}

// --- fake repo ---

type fakeRewardRepo struct {
	insertFn            func(ctx context.Context, rw *rewarddomain.Reward) error
	updateFn            func(ctx context.Context, rw *rewarddomain.Reward, columns ...string) error
	deleteFn            func(ctx context.Context, id int64) error
	getByIDFn           func(ctx context.Context, id int64) (*rewarddomain.Reward, error)
	listBySeasonFn      func(ctx context.Context, seasonID int64) ([]rewarddomain.Reward, error)
	insertClaimFn       func(ctx context.Context, c *rewarddomain.RewardClaim) error
	updateClaimFn       func(ctx context.Context, c *rewarddomain.RewardClaim, columns ...string) error
	getClaimByIDFn      func(ctx context.Context, id int64) (*rewarddomain.RewardClaim, error)
	listClaimsByUserFn  func(ctx context.Context, userID int64) ([]rewarddomain.RewardClaim, error)
	listAllClaimsFn     func(ctx context.Context) ([]rewarddomain.RewardClaim, error)
	countActiveClaimsFn func(ctx context.Context, rewardID int64) (int, error)
}

func (f *fakeRewardRepo) Insert(ctx context.Context, rw *rewarddomain.Reward) error {
	if f.insertFn != nil {
		return f.insertFn(ctx, rw)
	}
	rw.ID = 1
	return nil
}

func (f *fakeRewardRepo) Update(ctx context.Context, rw *rewarddomain.Reward, columns ...string) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, rw, columns...)
	}
	return nil
}

func (f *fakeRewardRepo) Delete(ctx context.Context, id int64) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func (f *fakeRewardRepo) GetByID(ctx context.Context, id int64) (*rewarddomain.Reward, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeRewardRepo) ListBySeason(ctx context.Context, seasonID int64) ([]rewarddomain.Reward, error) {
	if f.listBySeasonFn != nil {
		return f.listBySeasonFn(ctx, seasonID)
	}
	return nil, nil
}

func (f *fakeRewardRepo) InsertClaim(ctx context.Context, c *rewarddomain.RewardClaim) error {
	if f.insertClaimFn != nil {
		return f.insertClaimFn(ctx, c)
	}
	c.ID = 1
	return nil
}

func (f *fakeRewardRepo) UpdateClaim(ctx context.Context, c *rewarddomain.RewardClaim, columns ...string) error {
	if f.updateClaimFn != nil {
		return f.updateClaimFn(ctx, c, columns...)
	}
	return nil
}

func (f *fakeRewardRepo) GetClaimByID(ctx context.Context, id int64) (*rewarddomain.RewardClaim, error) {
	if f.getClaimByIDFn != nil {
		return f.getClaimByIDFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeRewardRepo) ListClaimsByUser(ctx context.Context, userID int64) ([]rewarddomain.RewardClaim, error) {
	if f.listClaimsByUserFn != nil {
		return f.listClaimsByUserFn(ctx, userID)
	}
	return nil, nil
}

func (f *fakeRewardRepo) ListAllClaims(ctx context.Context) ([]rewarddomain.RewardClaim, error) {
	if f.listAllClaimsFn != nil {
		return f.listAllClaimsFn(ctx)
	}
	return nil, nil
}

func (f *fakeRewardRepo) CountActiveClaims(ctx context.Context, rewardID int64) (int, error) {
	if f.countActiveClaimsFn != nil {
		return f.countActiveClaimsFn(ctx, rewardID)
	}
	return 0, nil
}

// --- fake balance service ---

type fakeCoinsSpender struct {
	spendFn  func(ctx context.Context, req wallet.DebitRequest) (*walletdomain.Transaction, error)
	refundFn func(ctx context.Context, req wallet.DebitRequest) (*walletdomain.Transaction, error)
}

func (f *fakeCoinsSpender) SpendCoins(ctx context.Context, req wallet.DebitRequest) (*walletdomain.Transaction, error) {
	if f.spendFn != nil {
		return f.spendFn(ctx, req)
	}
	return &walletdomain.Transaction{}, nil
}

func (f *fakeCoinsSpender) RefundCoins(ctx context.Context, req wallet.DebitRequest) (*walletdomain.Transaction, error) {
	if f.refundFn != nil {
		return f.refundFn(ctx, req)
	}
	return &walletdomain.Transaction{}, nil
}

// --- helpers ---

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeRewardNotifier struct{}

func (fakeRewardNotifier) Notify(_ context.Context, _ notifications.Notification, _ notificationcontract.UserFilter) error {
	return nil
}

func newRewardService(repo *fakeRewardRepo, balance *fakeCoinsSpender) *rewardservice.RewardService {
	return rewardservice.NewService(repo, balance, fakeRewardNotifier{}, noopLogger())
}

func availableReward(id int64, cost int, limit *int) *rewarddomain.Reward {
	return &rewarddomain.Reward{
		ID:        id,
		Title:     "Test Reward",
		CostCoins: cost,
		Limit:     limit,
		Status:    rewarddomain.RewardAvailable,
	}
}

func intPtr(n int) *int { return &n }

// --- Create ---

func TestRewardService_Create(t *testing.T) {
	t.Run("creates reward with available status by default", func(t *testing.T) {
		svc := newRewardService(&fakeRewardRepo{}, &fakeCoinsSpender{})

		rw, err := svc.Create(context.Background(), &rewardservice.CreateRewardRequest{
			Title:     "Prize",
			CostCoins: 100,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rw == nil {
			t.Fatal("expected non-nil reward")
		}
		if rw.Status != rewarddomain.RewardAvailable {
			t.Errorf("expected status=available, got %q", rw.Status)
		}
	})

	t.Run("uses explicit status when provided", func(t *testing.T) {
		svc := newRewardService(&fakeRewardRepo{}, &fakeCoinsSpender{})

		rw, err := svc.Create(context.Background(), &rewardservice.CreateRewardRequest{
			Title:  "Hidden Prize",
			Status: rewarddomain.RewardHidden,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rw.Status != rewarddomain.RewardHidden {
			t.Errorf("expected status=hidden, got %q", rw.Status)
		}
	})

	t.Run("propagates repo insert error", func(t *testing.T) {
		repoErr := errors.New("insert error")
		svc := newRewardService(&fakeRewardRepo{
			insertFn: func(_ context.Context, _ *rewarddomain.Reward) error {
				return repoErr
			},
		}, &fakeCoinsSpender{})

		_, err := svc.Create(context.Background(), &rewardservice.CreateRewardRequest{Title: "x"})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- Update ---

func TestRewardService_Update(t *testing.T) {
	t.Run("updates provided fields only", func(t *testing.T) {
		newTitle := "Updated Title"
		newCost := 200
		var updatedColumns []string

		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, id int64) (*rewarddomain.Reward, error) {
				return availableReward(id, 100, nil), nil
			},
			updateFn: func(_ context.Context, _ *rewarddomain.Reward, columns ...string) error {
				updatedColumns = columns
				return nil
			},
		}

		rw, err := newRewardService(repo, &fakeCoinsSpender{}).Update(context.Background(), 1, &rewardservice.UpdateRewardRequest{
			Title:     &newTitle,
			CostCoins: &newCost,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rw.Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, rw.Title)
		}
		if len(updatedColumns) != 2 {
			t.Errorf("expected 2 columns updated, got %v", updatedColumns)
		}
	})

	t.Run("no-op when no fields provided — repo Update not called", func(t *testing.T) {
		updateCalled := false
		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, id int64) (*rewarddomain.Reward, error) {
				return availableReward(id, 100, nil), nil
			},
			updateFn: func(_ context.Context, _ *rewarddomain.Reward, _ ...string) error {
				updateCalled = true
				return nil
			},
		}

		_, err := newRewardService(repo, &fakeCoinsSpender{}).Update(context.Background(), 1, &rewardservice.UpdateRewardRequest{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updateCalled {
			t.Error("repo.Update must not be called when no fields change")
		}
	})

	t.Run("returns ErrNotFound when reward does not exist", func(t *testing.T) {
		_, err := newRewardService(&fakeRewardRepo{}, &fakeCoinsSpender{}).Update(
			context.Background(), 99, &rewardservice.UpdateRewardRequest{},
		)

		if !errors.Is(err, rewardservice.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("propagates repo GetByID error", func(t *testing.T) {
		repoErr := errors.New("db error")
		_, err := newRewardService(&fakeRewardRepo{
			getByIDFn: func(_ context.Context, _ int64) (*rewarddomain.Reward, error) {
				return nil, repoErr
			},
		}, &fakeCoinsSpender{}).Update(context.Background(), 1, &rewardservice.UpdateRewardRequest{})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})

	t.Run("propagates repo Update error", func(t *testing.T) {
		repoErr := errors.New("update failed")
		newTitle := "x"
		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, id int64) (*rewarddomain.Reward, error) {
				return availableReward(id, 100, nil), nil
			},
			updateFn: func(_ context.Context, _ *rewarddomain.Reward, _ ...string) error {
				return repoErr
			},
		}

		_, err := newRewardService(repo, &fakeCoinsSpender{}).Update(
			context.Background(), 1, &rewardservice.UpdateRewardRequest{Title: &newTitle},
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- Delete ---

func TestRewardService_Delete(t *testing.T) {
	t.Run("deletes successfully", func(t *testing.T) {
		if err := newRewardService(&fakeRewardRepo{}, &fakeCoinsSpender{}).Delete(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrHasRelations on FK violation", func(t *testing.T) {
		svc := newRewardService(&fakeRewardRepo{
			deleteFn: func(_ context.Context, _ int64) error {
				return pgFKErr("23503")
			},
		}, &fakeCoinsSpender{})

		if err := svc.Delete(context.Background(), 1); !errors.Is(err, rewardservice.ErrHasRelations) {
			t.Fatalf("expected ErrHasRelations, got: %v", err)
		}
	})

	t.Run("propagates generic repo error", func(t *testing.T) {
		repoErr := errors.New("delete failed")
		svc := newRewardService(&fakeRewardRepo{
			deleteFn: func(_ context.Context, _ int64) error { return repoErr },
		}, &fakeCoinsSpender{})

		if err := svc.Delete(context.Background(), 1); !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- SubmitClaim ---

func TestRewardService_SubmitClaim(t *testing.T) {
	t.Run("creates claim and spends coins", func(t *testing.T) {
		spendCalledWith := wallet.DebitRequest{}
		spendCalled := false

		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, id int64) (*rewarddomain.Reward, error) {
				return availableReward(id, 50, nil), nil
			},
		}
		balance := &fakeCoinsSpender{
			spendFn: func(_ context.Context, req wallet.DebitRequest) (*walletdomain.Transaction, error) {
				spendCalledWith = req
				spendCalled = true
				return &walletdomain.Transaction{}, nil
			},
		}

		claim, err := newRewardService(repo, balance).SubmitClaim(context.Background(), 1, 10, 1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claim == nil {
			t.Fatal("expected non-nil claim")
		}
		if claim.Status != rewarddomain.ClaimPending {
			t.Errorf("expected status=pending, got %q", claim.Status)
		}
		if !spendCalled {
			t.Error("expected SpendCoins to be called")
		}
		if spendCalledWith.Amount != 50 {
			t.Errorf("expected amount=50, got %d", spendCalledWith.Amount)
		}
	})

	t.Run("returns ErrRewardUnavailable when reward is nil", func(t *testing.T) {
		// repo returns nil reward
		_, err := newRewardService(&fakeRewardRepo{}, &fakeCoinsSpender{}).SubmitClaim(context.Background(), 1, 10, 99)

		if !errors.Is(err, rewardservice.ErrRewardUnavailable) {
			t.Fatalf("expected ErrRewardUnavailable, got: %v", err)
		}
	})

	t.Run("returns ErrRewardUnavailable when reward is hidden", func(t *testing.T) {
		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, id int64) (*rewarddomain.Reward, error) {
				rw := availableReward(id, 100, nil)
				rw.Status = rewarddomain.RewardHidden
				return rw, nil
			},
		}

		_, err := newRewardService(repo, &fakeCoinsSpender{}).SubmitClaim(context.Background(), 1, 10, 1)

		if !errors.Is(err, rewardservice.ErrRewardUnavailable) {
			t.Fatalf("expected ErrRewardUnavailable (hidden), got: %v", err)
		}
	})

	t.Run("returns ErrLimitExceeded when all slots are taken", func(t *testing.T) {
		limit := 3
		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, id int64) (*rewarddomain.Reward, error) {
				return availableReward(id, 50, &limit), nil
			},
			countActiveClaimsFn: func(_ context.Context, _ int64) (int, error) {
				return 3, nil // all 3 slots occupied
			},
		}

		_, err := newRewardService(repo, &fakeCoinsSpender{}).SubmitClaim(context.Background(), 1, 10, 1)

		if !errors.Is(err, rewardservice.ErrLimitExceeded) {
			t.Fatalf("expected ErrLimitExceeded, got: %v", err)
		}
	})

	t.Run("allows submission when limit is nil (unlimited)", func(t *testing.T) {
		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, id int64) (*rewarddomain.Reward, error) {
				return availableReward(id, 50, nil), nil // no limit
			},
		}

		claim, err := newRewardService(repo, &fakeCoinsSpender{}).SubmitClaim(context.Background(), 1, 10, 1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claim == nil {
			t.Fatal("expected claim, got nil")
		}
	})

	t.Run("returns ErrInsufficientBalance when spend fails with balance error", func(t *testing.T) {
		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, id int64) (*rewarddomain.Reward, error) {
				return availableReward(id, 1000, nil), nil
			},
		}
		balance := &fakeCoinsSpender{
			spendFn: func(_ context.Context, _ wallet.DebitRequest) (*walletdomain.Transaction, error) {
				return nil, wallet.ErrInsufficientBalance
			},
		}

		_, err := newRewardService(repo, balance).SubmitClaim(context.Background(), 1, 10, 1)

		if !errors.Is(err, rewardservice.ErrInsufficientBalance) {
			t.Fatalf("expected ErrInsufficientBalance, got: %v", err)
		}
	})

	t.Run("propagates unexpected spend error", func(t *testing.T) {
		spendErr := errors.New("payment gateway down")
		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, id int64) (*rewarddomain.Reward, error) {
				return availableReward(id, 50, nil), nil
			},
		}
		balance := &fakeCoinsSpender{
			spendFn: func(_ context.Context, _ wallet.DebitRequest) (*walletdomain.Transaction, error) {
				return nil, spendErr
			},
		}

		_, err := newRewardService(repo, balance).SubmitClaim(context.Background(), 1, 10, 1)

		if !errors.Is(err, spendErr) {
			t.Fatalf("expected spend error, got: %v", err)
		}
	})

	t.Run("propagates repo GetByID error", func(t *testing.T) {
		repoErr := errors.New("db error")
		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, _ int64) (*rewarddomain.Reward, error) {
				return nil, repoErr
			},
		}

		_, err := newRewardService(repo, &fakeCoinsSpender{}).SubmitClaim(context.Background(), 1, 10, 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- UpdateClaimStatus ---

func TestRewardService_UpdateClaimStatus(t *testing.T) {
	t.Run("updates claim status to completed without refund", func(t *testing.T) {
		refundCalled := false
		repo := &fakeRewardRepo{
			getClaimByIDFn: func(_ context.Context, id int64) (*rewarddomain.RewardClaim, error) {
				return &rewarddomain.RewardClaim{
					ID:         id,
					UserID:     1,
					RewardID:   10,
					Status:     rewarddomain.ClaimPending,
					SpentCoins: 100,
				}, nil
			},
		}
		balance := &fakeCoinsSpender{
			refundFn: func(_ context.Context, _ wallet.DebitRequest) (*walletdomain.Transaction, error) {
				refundCalled = true
				return nil, nil
			},
		}

		claim, err := newRewardService(repo, balance).UpdateClaimStatus(
			context.Background(), 1,
			&rewardservice.UpdateClaimRequest{Status: rewarddomain.ClaimCompleted},
			5,
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claim.Status != rewarddomain.ClaimCompleted {
			t.Errorf("expected status=completed, got %q", claim.Status)
		}
		if refundCalled {
			t.Error("refund must not happen for completion transition")
		}
	})

	t.Run("refunds coins when transitioning to cancelled", func(t *testing.T) {
		refundCalledWith := wallet.DebitRequest{}
		rewardTitle := "Prize"

		repo := &fakeRewardRepo{
			getClaimByIDFn: func(_ context.Context, id int64) (*rewarddomain.RewardClaim, error) {
				return &rewarddomain.RewardClaim{
					ID:         id,
					UserID:     1,
					RewardID:   10,
					Status:     rewarddomain.ClaimPending,
					SpentCoins: 150,
				}, nil
			},
			getByIDFn: func(_ context.Context, id int64) (*rewarddomain.Reward, error) {
				rw := availableReward(id, 150, nil)
				rw.Title = rewardTitle
				return rw, nil
			},
		}
		balance := &fakeCoinsSpender{
			refundFn: func(_ context.Context, req wallet.DebitRequest) (*walletdomain.Transaction, error) {
				refundCalledWith = req
				return &walletdomain.Transaction{}, nil
			},
		}

		_, err := newRewardService(repo, balance).UpdateClaimStatus(
			context.Background(), 1,
			&rewardservice.UpdateClaimRequest{Status: rewarddomain.ClaimCancelled},
			5,
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if refundCalledWith.Amount != 150 {
			t.Errorf("expected refund amount=150, got %d", refundCalledWith.Amount)
		}
		if refundCalledWith.UserID != 1 {
			t.Errorf("expected refund userID=1, got %d", refundCalledWith.UserID)
		}
		if refundCalledWith.SeasonID != 5 {
			t.Errorf("expected seasonID=5, got %d", refundCalledWith.SeasonID)
		}
	})

	t.Run("does not refund when claim is already cancelled", func(t *testing.T) {
		refundCalled := false
		repo := &fakeRewardRepo{
			getClaimByIDFn: func(_ context.Context, id int64) (*rewarddomain.RewardClaim, error) {
				// already cancelled
				return &rewarddomain.RewardClaim{
					ID:       id,
					Status:   rewarddomain.ClaimCancelled,
					RewardID: 10,
				}, nil
			},
		}
		balance := &fakeCoinsSpender{
			refundFn: func(_ context.Context, _ wallet.DebitRequest) (*walletdomain.Transaction, error) {
				refundCalled = true
				return nil, nil
			},
		}

		_, err := newRewardService(repo, balance).UpdateClaimStatus(
			context.Background(), 1,
			&rewardservice.UpdateClaimRequest{Status: rewarddomain.ClaimCancelled},
			5,
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if refundCalled {
			t.Error("refund must not be called when already cancelled")
		}
	})

	t.Run("returns ErrClaimNotFound when claim does not exist", func(t *testing.T) {
		_, err := newRewardService(&fakeRewardRepo{}, &fakeCoinsSpender{}).UpdateClaimStatus(
			context.Background(), 99,
			&rewardservice.UpdateClaimRequest{Status: rewarddomain.ClaimCompleted},
			1,
		)

		if !errors.Is(err, rewardservice.ErrClaimNotFound) {
			t.Fatalf("expected ErrClaimNotFound, got: %v", err)
		}
	})

	t.Run("propagates repo GetClaimByID error", func(t *testing.T) {
		repoErr := errors.New("db error")
		repo := &fakeRewardRepo{
			getClaimByIDFn: func(_ context.Context, _ int64) (*rewarddomain.RewardClaim, error) {
				return nil, repoErr
			},
		}

		_, err := newRewardService(repo, &fakeCoinsSpender{}).UpdateClaimStatus(
			context.Background(), 1,
			&rewardservice.UpdateClaimRequest{Status: rewarddomain.ClaimCompleted},
			1,
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})

	t.Run("propagates repo UpdateClaim error", func(t *testing.T) {
		repoErr := errors.New("update failed")
		repo := &fakeRewardRepo{
			getClaimByIDFn: func(_ context.Context, id int64) (*rewarddomain.RewardClaim, error) {
				return &rewarddomain.RewardClaim{ID: id, Status: rewarddomain.ClaimPending, RewardID: 1}, nil
			},
			updateClaimFn: func(_ context.Context, _ *rewarddomain.RewardClaim, _ ...string) error {
				return repoErr
			},
		}

		_, err := newRewardService(repo, &fakeCoinsSpender{}).UpdateClaimStatus(
			context.Background(), 1,
			&rewardservice.UpdateClaimRequest{Status: rewarddomain.ClaimCompleted},
			1,
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- GetByID ---

func TestRewardService_GetByID(t *testing.T) {
	t.Run("returns nil when not found", func(t *testing.T) {
		rw, err := newRewardService(&fakeRewardRepo{}, &fakeCoinsSpender{}).GetByID(context.Background(), 1)

		if err != nil || rw != nil {
			t.Fatalf("expected (nil, nil), got (%v, %v)", rw, err)
		}
	})

	t.Run("returns domain reward when found", func(t *testing.T) {
		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, id int64) (*rewarddomain.Reward, error) {
				return availableReward(id, 100, nil), nil
			},
		}

		rw, err := newRewardService(repo, &fakeCoinsSpender{}).GetByID(context.Background(), 5)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rw == nil || rw.ID != 5 {
			t.Fatalf("expected reward id=5, got %v", rw)
		}
	})
}

// --- ListBySeason ---

func TestRewardService_ListBySeason(t *testing.T) {
	t.Run("returns domain rewards for season", func(t *testing.T) {
		repo := &fakeRewardRepo{
			listBySeasonFn: func(_ context.Context, seasonID int64) ([]rewarddomain.Reward, error) {
				if seasonID != 3 {
					return nil, errors.New("wrong seasonID")
				}
				return []rewarddomain.Reward{
					{ID: 1},
					{ID: 2},
				}, nil
			},
		}

		rewards, err := newRewardService(repo, &fakeCoinsSpender{}).ListBySeason(context.Background(), 3)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rewards) != 2 {
			t.Fatalf("expected 2 rewards, got %d", len(rewards))
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("list error")
		repo := &fakeRewardRepo{
			listBySeasonFn: func(_ context.Context, _ int64) ([]rewarddomain.Reward, error) {
				return nil, repoErr
			},
		}

		_, err := newRewardService(repo, &fakeCoinsSpender{}).ListBySeason(context.Background(), 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

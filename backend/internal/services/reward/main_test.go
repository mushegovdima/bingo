package rewardservice_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"unsafe"

	"github.com/uptrace/bun/driver/pgdriver"
	dbmodels "go.mod/internal/db"
	"go.mod/internal/domain"
	balanceservice "go.mod/internal/services/balance"
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
	insertFn           func(ctx context.Context, rw *dbmodels.Reward) error
	updateFn           func(ctx context.Context, rw *dbmodels.Reward, columns ...string) error
	deleteFn           func(ctx context.Context, id int64) error
	getByIDFn          func(ctx context.Context, id int64) (*dbmodels.Reward, error)
	listBySeasonFn   func(ctx context.Context, seasonID int64) ([]dbmodels.Reward, error)
	insertClaimFn      func(ctx context.Context, c *dbmodels.RewardClaim) error
	updateClaimFn      func(ctx context.Context, c *dbmodels.RewardClaim, columns ...string) error
	getClaimByIDFn     func(ctx context.Context, id int64) (*dbmodels.RewardClaim, error)
	listClaimsByUserFn func(ctx context.Context, userID int64) ([]dbmodels.RewardClaim, error)
	listAllClaimsFn    func(ctx context.Context) ([]dbmodels.RewardClaim, error)
	countActiveClaimsFn func(ctx context.Context, rewardID int64) (int, error)
}

func (f *fakeRewardRepo) Insert(ctx context.Context, rw *dbmodels.Reward) error {
	if f.insertFn != nil {
		return f.insertFn(ctx, rw)
	}
	rw.ID = 1
	return nil
}

func (f *fakeRewardRepo) Update(ctx context.Context, rw *dbmodels.Reward, columns ...string) error {
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

func (f *fakeRewardRepo) GetByID(ctx context.Context, id int64) (*dbmodels.Reward, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeRewardRepo) ListBySeason(ctx context.Context, seasonID int64) ([]dbmodels.Reward, error) {
	if f.listBySeasonFn != nil {
		return f.listBySeasonFn(ctx, seasonID)
	}
	return nil, nil
}

func (f *fakeRewardRepo) InsertClaim(ctx context.Context, c *dbmodels.RewardClaim) error {
	if f.insertClaimFn != nil {
		return f.insertClaimFn(ctx, c)
	}
	c.ID = 1
	return nil
}

func (f *fakeRewardRepo) UpdateClaim(ctx context.Context, c *dbmodels.RewardClaim, columns ...string) error {
	if f.updateClaimFn != nil {
		return f.updateClaimFn(ctx, c, columns...)
	}
	return nil
}

func (f *fakeRewardRepo) GetClaimByID(ctx context.Context, id int64) (*dbmodels.RewardClaim, error) {
	if f.getClaimByIDFn != nil {
		return f.getClaimByIDFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeRewardRepo) ListClaimsByUser(ctx context.Context, userID int64) ([]dbmodels.RewardClaim, error) {
	if f.listClaimsByUserFn != nil {
		return f.listClaimsByUserFn(ctx, userID)
	}
	return nil, nil
}

func (f *fakeRewardRepo) ListAllClaims(ctx context.Context) ([]dbmodels.RewardClaim, error) {
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
	spendFn  func(ctx context.Context, req balanceservice.SpendCoinsRequest) (*domain.Transaction, error)
	refundFn func(ctx context.Context, req balanceservice.SpendCoinsRequest) (*domain.Transaction, error)
}

func (f *fakeCoinsSpender) SpendCoins(ctx context.Context, req balanceservice.SpendCoinsRequest) (*domain.Transaction, error) {
	if f.spendFn != nil {
		return f.spendFn(ctx, req)
	}
	return &domain.Transaction{}, nil
}

func (f *fakeCoinsSpender) RefundCoins(ctx context.Context, req balanceservice.SpendCoinsRequest) (*domain.Transaction, error) {
	if f.refundFn != nil {
		return f.refundFn(ctx, req)
	}
	return &domain.Transaction{}, nil
}

// --- helpers ---

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newRewardService(repo *fakeRewardRepo, balance *fakeCoinsSpender) *rewardservice.RewardService {
	return rewardservice.NewService(repo, balance, noopLogger())
}

func availableReward(id int64, cost int, limit *int) *dbmodels.Reward {
	return &dbmodels.Reward{
		Entity:    dbmodels.Entity{ID: id},
		Title:     "Test Reward",
		CostCoins: cost,
		Limit:     limit,
		Status:    domain.RewardAvailable,
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
		if rw.Status != domain.RewardAvailable {
			t.Errorf("expected status=available, got %q", rw.Status)
		}
	})

	t.Run("uses explicit status when provided", func(t *testing.T) {
		svc := newRewardService(&fakeRewardRepo{}, &fakeCoinsSpender{})

		rw, err := svc.Create(context.Background(), &rewardservice.CreateRewardRequest{
			Title:  "Hidden Prize",
			Status: domain.RewardHidden,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rw.Status != domain.RewardHidden {
			t.Errorf("expected status=hidden, got %q", rw.Status)
		}
	})

	t.Run("propagates repo insert error", func(t *testing.T) {
		repoErr := errors.New("insert error")
		svc := newRewardService(&fakeRewardRepo{
			insertFn: func(_ context.Context, _ *dbmodels.Reward) error {
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
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Reward, error) {
				return availableReward(id, 100, nil), nil
			},
			updateFn: func(_ context.Context, _ *dbmodels.Reward, columns ...string) error {
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
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Reward, error) {
				return availableReward(id, 100, nil), nil
			},
			updateFn: func(_ context.Context, _ *dbmodels.Reward, _ ...string) error {
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
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Reward, error) {
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
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Reward, error) {
				return availableReward(id, 100, nil), nil
			},
			updateFn: func(_ context.Context, _ *dbmodels.Reward, _ ...string) error {
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
		spendCalledWith := balanceservice.SpendCoinsRequest{}
		spendCalled := false

		repo := &fakeRewardRepo{
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Reward, error) {
				return availableReward(id, 50, nil), nil
			},
		}
		balance := &fakeCoinsSpender{
			spendFn: func(_ context.Context, req balanceservice.SpendCoinsRequest) (*domain.Transaction, error) {
				spendCalledWith = req
				spendCalled = true
				return &domain.Transaction{}, nil
			},
		}

		claim, err := newRewardService(repo, balance).SubmitClaim(context.Background(), 1, 10, 1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claim == nil {
			t.Fatal("expected non-nil claim")
		}
		if claim.Status != domain.ClaimPending {
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
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Reward, error) {
				rw := availableReward(id, 100, nil)
				rw.Status = domain.RewardHidden
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
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Reward, error) {
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
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Reward, error) {
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
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Reward, error) {
				return availableReward(id, 1000, nil), nil
			},
		}
		balance := &fakeCoinsSpender{
			spendFn: func(_ context.Context, _ balanceservice.SpendCoinsRequest) (*domain.Transaction, error) {
				return nil, balanceservice.ErrInsufficientBalance
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
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Reward, error) {
				return availableReward(id, 50, nil), nil
			},
		}
		balance := &fakeCoinsSpender{
			spendFn: func(_ context.Context, _ balanceservice.SpendCoinsRequest) (*domain.Transaction, error) {
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
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Reward, error) {
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
			getClaimByIDFn: func(_ context.Context, id int64) (*dbmodels.RewardClaim, error) {
				return &dbmodels.RewardClaim{
					Entity:     dbmodels.Entity{ID: id},
					UserID:     1,
					RewardID:   10,
					Status:     domain.ClaimPending,
					SpentCoins: 100,
				}, nil
			},
		}
		balance := &fakeCoinsSpender{
			refundFn: func(_ context.Context, _ balanceservice.SpendCoinsRequest) (*domain.Transaction, error) {
				refundCalled = true
				return nil, nil
			},
		}

		claim, err := newRewardService(repo, balance).UpdateClaimStatus(
			context.Background(), 1,
			&rewardservice.UpdateClaimRequest{Status: domain.ClaimCompleted},
			5,
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claim.Status != domain.ClaimCompleted {
			t.Errorf("expected status=completed, got %q", claim.Status)
		}
		if refundCalled {
			t.Error("refund must not happen for completion transition")
		}
	})

	t.Run("refunds coins when transitioning to cancelled", func(t *testing.T) {
		refundCalledWith := balanceservice.SpendCoinsRequest{}
		rewardTitle := "Prize"

		repo := &fakeRewardRepo{
			getClaimByIDFn: func(_ context.Context, id int64) (*dbmodels.RewardClaim, error) {
				return &dbmodels.RewardClaim{
					Entity:     dbmodels.Entity{ID: id},
					UserID:     1,
					RewardID:   10,
					Status:     domain.ClaimPending,
					SpentCoins: 150,
				}, nil
			},
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Reward, error) {
				rw := availableReward(id, 150, nil)
				rw.Title = rewardTitle
				return rw, nil
			},
		}
		balance := &fakeCoinsSpender{
			refundFn: func(_ context.Context, req balanceservice.SpendCoinsRequest) (*domain.Transaction, error) {
				refundCalledWith = req
				return &domain.Transaction{}, nil
			},
		}

		_, err := newRewardService(repo, balance).UpdateClaimStatus(
			context.Background(), 1,
			&rewardservice.UpdateClaimRequest{Status: domain.ClaimCancelled},
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
			getClaimByIDFn: func(_ context.Context, id int64) (*dbmodels.RewardClaim, error) {
				// already cancelled
				return &dbmodels.RewardClaim{
					Entity:   dbmodels.Entity{ID: id},
					Status:   domain.ClaimCancelled,
					RewardID: 10,
				}, nil
			},
		}
		balance := &fakeCoinsSpender{
			refundFn: func(_ context.Context, _ balanceservice.SpendCoinsRequest) (*domain.Transaction, error) {
				refundCalled = true
				return nil, nil
			},
		}

		_, err := newRewardService(repo, balance).UpdateClaimStatus(
			context.Background(), 1,
			&rewardservice.UpdateClaimRequest{Status: domain.ClaimCancelled},
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
			&rewardservice.UpdateClaimRequest{Status: domain.ClaimCompleted},
			1,
		)

		if !errors.Is(err, rewardservice.ErrClaimNotFound) {
			t.Fatalf("expected ErrClaimNotFound, got: %v", err)
		}
	})

	t.Run("propagates repo GetClaimByID error", func(t *testing.T) {
		repoErr := errors.New("db error")
		repo := &fakeRewardRepo{
			getClaimByIDFn: func(_ context.Context, _ int64) (*dbmodels.RewardClaim, error) {
				return nil, repoErr
			},
		}

		_, err := newRewardService(repo, &fakeCoinsSpender{}).UpdateClaimStatus(
			context.Background(), 1,
			&rewardservice.UpdateClaimRequest{Status: domain.ClaimCompleted},
			1,
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})

	t.Run("propagates repo UpdateClaim error", func(t *testing.T) {
		repoErr := errors.New("update failed")
		repo := &fakeRewardRepo{
			getClaimByIDFn: func(_ context.Context, id int64) (*dbmodels.RewardClaim, error) {
				return &dbmodels.RewardClaim{Entity: dbmodels.Entity{ID: id}, Status: domain.ClaimPending, RewardID: 1}, nil
			},
			updateClaimFn: func(_ context.Context, _ *dbmodels.RewardClaim, _ ...string) error {
				return repoErr
			},
		}

		_, err := newRewardService(repo, &fakeCoinsSpender{}).UpdateClaimStatus(
			context.Background(), 1,
			&rewardservice.UpdateClaimRequest{Status: domain.ClaimCompleted},
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
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Reward, error) {
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
			listBySeasonFn: func(_ context.Context, seasonID int64) ([]dbmodels.Reward, error) {
				if seasonID != 3 {
					return nil, errors.New("wrong seasonID")
				}
				return []dbmodels.Reward{
					{Entity: dbmodels.Entity{ID: 1}},
					{Entity: dbmodels.Entity{ID: 2}},
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
			listBySeasonFn: func(_ context.Context, _ int64) ([]dbmodels.Reward, error) {
				return nil, repoErr
			},
		}

		_, err := newRewardService(repo, &fakeCoinsSpender{}).ListBySeason(context.Background(), 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

package submissionservice_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	dbmodels "go.mod/internal/db"
	"go.mod/internal/domain"
	balanceservice "go.mod/internal/services/balance"
	submissionservice "go.mod/internal/services/submission"
)

// --- fake repo ---

type fakeSubmissionRepo struct {
	insertFn    func(ctx context.Context, s *dbmodels.TaskSubmission) error
	updateFn    func(ctx context.Context, s *dbmodels.TaskSubmission, columns ...string) error
	deleteFn    func(ctx context.Context, id int64) error
	getByIDFn   func(ctx context.Context, id int64) (*dbmodels.TaskSubmission, error)
	listByUserFn func(ctx context.Context, userID int64) ([]dbmodels.TaskSubmission, error)
	listAllFn   func(ctx context.Context) ([]dbmodels.TaskSubmission, error)
}

func (f *fakeSubmissionRepo) Insert(ctx context.Context, s *dbmodels.TaskSubmission) error {
	if f.insertFn != nil {
		return f.insertFn(ctx, s)
	}
	s.ID = 1
	return nil
}

func (f *fakeSubmissionRepo) Update(ctx context.Context, s *dbmodels.TaskSubmission, columns ...string) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, s, columns...)
	}
	return nil
}

func (f *fakeSubmissionRepo) Delete(ctx context.Context, id int64) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func (f *fakeSubmissionRepo) GetByID(ctx context.Context, id int64) (*dbmodels.TaskSubmission, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeSubmissionRepo) ListByUser(ctx context.Context, userID int64) ([]dbmodels.TaskSubmission, error) {
	if f.listByUserFn != nil {
		return f.listByUserFn(ctx, userID)
	}
	return nil, nil
}

func (f *fakeSubmissionRepo) ListAll(ctx context.Context) ([]dbmodels.TaskSubmission, error) {
	if f.listAllFn != nil {
		return f.listAllFn(ctx)
	}
	return nil, nil
}

// --- fake task finder ---

type fakeTaskFinder struct {
	getByIDFn func(ctx context.Context, id int64) (*domain.Task, error)
}

func (f *fakeTaskFinder) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return nil, nil
}

// --- fake balance service ---

type fakeCoinsAccruer struct {
	addCoinsFn func(ctx context.Context, req balanceservice.AddCoinsRequest) (*domain.Transaction, error)
}

func (f *fakeCoinsAccruer) AddCoins(ctx context.Context, req balanceservice.AddCoinsRequest) (*domain.Transaction, error) {
	if f.addCoinsFn != nil {
		return f.addCoinsFn(ctx, req)
	}
	return &domain.Transaction{}, nil
}

// --- helpers ---

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newSubmissionService(repo *fakeSubmissionRepo, tasks *fakeTaskFinder, balance *fakeCoinsAccruer) *submissionservice.SubmissionService {
	return submissionservice.NewService(repo, tasks, balance, noopLogger())
}

func taskWithCoins(id int64, coins int) *domain.Task {
	return &domain.Task{ID: id, Title: "Task", RewardCoins: coins, SeasonID: 1}
}

// --- Create ---

func TestSubmissionService_Create(t *testing.T) {
	t.Run("creates approved submission and credits coins", func(t *testing.T) {
		var addCoinsCalledWith balanceservice.AddCoinsRequest
		coinsCalled := false

		repo := &fakeSubmissionRepo{}
		tasks := &fakeTaskFinder{
			getByIDFn: func(_ context.Context, id int64) (*domain.Task, error) {
				return taskWithCoins(id, 100), nil
			},
		}
		balance := &fakeCoinsAccruer{
			addCoinsFn: func(_ context.Context, req balanceservice.AddCoinsRequest) (*domain.Transaction, error) {
				addCoinsCalledWith = req
				coinsCalled = true
				return &domain.Transaction{}, nil
			},
		}

		sub, err := newSubmissionService(repo, tasks, balance).Create(
			context.Background(), 99,
			&submissionservice.CreateRequest{UserID: 1, TaskID: 5, SeasonID: 10},
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sub == nil {
			t.Fatal("expected non-nil submission")
		}
		if sub.Status != domain.SubmissionApproved {
			t.Errorf("expected status=approved, got %q", sub.Status)
		}
		if !coinsCalled {
			t.Error("expected AddCoins to be called")
		}
		if addCoinsCalledWith.Amount != 100 {
			t.Errorf("expected coins=100, got %d", addCoinsCalledWith.Amount)
		}
		if addCoinsCalledWith.SeasonID != 10 {
			t.Errorf("expected seasonID=10, got %d", addCoinsCalledWith.SeasonID)
		}
	})

	t.Run("skips coin crediting when task has zero reward", func(t *testing.T) {
		coinsCalled := false
		balance := &fakeCoinsAccruer{
			addCoinsFn: func(_ context.Context, _ balanceservice.AddCoinsRequest) (*domain.Transaction, error) {
				coinsCalled = true
				return nil, nil
			},
		}
		tasks := &fakeTaskFinder{
			getByIDFn: func(_ context.Context, id int64) (*domain.Task, error) {
				return taskWithCoins(id, 0), nil
			},
		}

		sub, err := newSubmissionService(&fakeSubmissionRepo{}, tasks, balance).Create(
			context.Background(), 99,
			&submissionservice.CreateRequest{UserID: 1, TaskID: 5, SeasonID: 10},
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if coinsCalled {
			t.Error("AddCoins must not be called for zero-reward tasks")
		}
		if sub.Status != domain.SubmissionApproved {
			t.Errorf("expected approved, got %q", sub.Status)
		}
	})

	t.Run("returns error when task not found", func(t *testing.T) {
		tasks := &fakeTaskFinder{} // returns nil, nil

		_, err := newSubmissionService(&fakeSubmissionRepo{}, tasks, &fakeCoinsAccruer{}).Create(
			context.Background(), 99,
			&submissionservice.CreateRequest{UserID: 1, TaskID: 5, SeasonID: 10},
		)

		if err == nil {
			t.Fatal("expected error when task not found")
		}
	})

	t.Run("propagates task finder error", func(t *testing.T) {
		repoErr := errors.New("task db error")
		tasks := &fakeTaskFinder{
			getByIDFn: func(_ context.Context, _ int64) (*domain.Task, error) {
				return nil, repoErr
			},
		}

		_, err := newSubmissionService(&fakeSubmissionRepo{}, tasks, &fakeCoinsAccruer{}).Create(
			context.Background(), 99,
			&submissionservice.CreateRequest{UserID: 1, TaskID: 5, SeasonID: 10},
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected task finder error, got: %v", err)
		}
	})

	t.Run("propagates repo insert error", func(t *testing.T) {
		repoErr := errors.New("insert failed")
		repo := &fakeSubmissionRepo{
			insertFn: func(_ context.Context, _ *dbmodels.TaskSubmission) error {
				return repoErr
			},
		}
		tasks := &fakeTaskFinder{
			getByIDFn: func(_ context.Context, id int64) (*domain.Task, error) {
				return taskWithCoins(id, 50), nil
			},
		}

		_, err := newSubmissionService(repo, tasks, &fakeCoinsAccruer{}).Create(
			context.Background(), 99,
			&submissionservice.CreateRequest{UserID: 1, TaskID: 5, SeasonID: 10},
		)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})

	t.Run("coin crediting error is non-fatal — submission still returned", func(t *testing.T) {
		balance := &fakeCoinsAccruer{
			addCoinsFn: func(_ context.Context, _ balanceservice.AddCoinsRequest) (*domain.Transaction, error) {
				return nil, errors.New("balance service unavailable")
			},
		}
		tasks := &fakeTaskFinder{
			getByIDFn: func(_ context.Context, id int64) (*domain.Task, error) {
				return taskWithCoins(id, 100), nil
			},
		}

		sub, err := newSubmissionService(&fakeSubmissionRepo{}, tasks, balance).Create(
			context.Background(), 99,
			&submissionservice.CreateRequest{UserID: 1, TaskID: 5, SeasonID: 10},
		)

		// coin failure is logged but the submission itself should succeed
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if sub == nil {
			t.Fatal("expected submission to be returned despite balance error")
		}
	})
}

// --- Delete ---

func TestSubmissionService_Delete(t *testing.T) {
	t.Run("deletes existing submission", func(t *testing.T) {
		repo := &fakeSubmissionRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.TaskSubmission, error) {
				return &dbmodels.TaskSubmission{Entity: dbmodels.Entity{ID: 1}}, nil
			},
		}

		if err := newSubmissionService(repo, &fakeTaskFinder{}, &fakeCoinsAccruer{}).Delete(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrNotFound when submission does not exist", func(t *testing.T) {
		err := newSubmissionService(&fakeSubmissionRepo{}, &fakeTaskFinder{}, &fakeCoinsAccruer{}).Delete(context.Background(), 99)

		if !errors.Is(err, submissionservice.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("propagates repo GetByID error", func(t *testing.T) {
		repoErr := errors.New("db read error")
		repo := &fakeSubmissionRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.TaskSubmission, error) {
				return nil, repoErr
			},
		}

		err := newSubmissionService(repo, &fakeTaskFinder{}, &fakeCoinsAccruer{}).Delete(context.Background(), 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})

	t.Run("propagates repo Delete error", func(t *testing.T) {
		repoErr := errors.New("delete failed")
		repo := &fakeSubmissionRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.TaskSubmission, error) {
				return &dbmodels.TaskSubmission{Entity: dbmodels.Entity{ID: 1}}, nil
			},
			deleteFn: func(_ context.Context, _ int64) error {
				return repoErr
			},
		}

		err := newSubmissionService(repo, &fakeTaskFinder{}, &fakeCoinsAccruer{}).Delete(context.Background(), 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- GetByID ---

func TestSubmissionService_GetByID(t *testing.T) {
	t.Run("returns nil when not found", func(t *testing.T) {
		sub, err := newSubmissionService(&fakeSubmissionRepo{}, &fakeTaskFinder{}, &fakeCoinsAccruer{}).GetByID(context.Background(), 1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sub != nil {
			t.Fatalf("expected nil, got %v", sub)
		}
	})

	t.Run("returns domain submission when found", func(t *testing.T) {
		repo := &fakeSubmissionRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.TaskSubmission, error) {
				return &dbmodels.TaskSubmission{Entity: dbmodels.Entity{ID: 7}, UserID: 2, TaskID: 3, Status: domain.SubmissionApproved}, nil
			},
		}

		sub, err := newSubmissionService(repo, &fakeTaskFinder{}, &fakeCoinsAccruer{}).GetByID(context.Background(), 7)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sub.ID != 7 || sub.UserID != 2 {
			t.Fatalf("unexpected domain mapping: %+v", sub)
		}
	})
}

// --- ListByUser ---

func TestSubmissionService_ListByUser(t *testing.T) {
	t.Run("returns submissions for a user", func(t *testing.T) {
		repo := &fakeSubmissionRepo{
			listByUserFn: func(_ context.Context, userID int64) ([]dbmodels.TaskSubmission, error) {
				if userID != 5 {
					t.Errorf("expected userID=5, got %d", userID)
				}
				return []dbmodels.TaskSubmission{
					{Entity: dbmodels.Entity{ID: 1}},
					{Entity: dbmodels.Entity{ID: 2}},
				}, nil
			},
		}

		subs, err := newSubmissionService(repo, &fakeTaskFinder{}, &fakeCoinsAccruer{}).ListByUser(context.Background(), 5)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(subs) != 2 {
			t.Fatalf("expected 2 submissions, got %d", len(subs))
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("list error")
		repo := &fakeSubmissionRepo{
			listByUserFn: func(_ context.Context, _ int64) ([]dbmodels.TaskSubmission, error) {
				return nil, repoErr
			},
		}

		_, err := newSubmissionService(repo, &fakeTaskFinder{}, &fakeCoinsAccruer{}).ListByUser(context.Background(), 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- ListAll ---

func TestSubmissionService_ListAll(t *testing.T) {
	t.Run("returns all submissions", func(t *testing.T) {
		repo := &fakeSubmissionRepo{
			listAllFn: func(_ context.Context) ([]dbmodels.TaskSubmission, error) {
				return []dbmodels.TaskSubmission{
					{Entity: dbmodels.Entity{ID: 1}},
					{Entity: dbmodels.Entity{ID: 2}},
					{Entity: dbmodels.Entity{ID: 3}},
				}, nil
			},
		}

		subs, err := newSubmissionService(repo, &fakeTaskFinder{}, &fakeCoinsAccruer{}).ListAll(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(subs) != 3 {
			t.Fatalf("expected 3, got %d", len(subs))
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("list all error")
		repo := &fakeSubmissionRepo{
			listAllFn: func(_ context.Context) ([]dbmodels.TaskSubmission, error) {
				return nil, repoErr
			},
		}

		_, err := newSubmissionService(repo, &fakeTaskFinder{}, &fakeCoinsAccruer{}).ListAll(context.Background())

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

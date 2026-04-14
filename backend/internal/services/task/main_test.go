package taskservice_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"unsafe"

	"github.com/uptrace/bun/driver/pgdriver"
	dbmodels "go.mod/internal/db"
	taskservice "go.mod/internal/services/task"
)

// --- pgdriver FK error helper ---

type pgErrLayout struct{ m map[byte]string }

func pgFKErr(code string) error {
	src := pgErrLayout{m: map[byte]string{'C': code}}
	return *(*pgdriver.Error)(unsafe.Pointer(&src))
}

// --- fake repo ---

type fakeTaskRepo struct {
	insertFn         func(ctx context.Context, t *dbmodels.Task) error
	updateFn         func(ctx context.Context, t *dbmodels.Task, columns ...string) error
	deleteFn         func(ctx context.Context, id int64) error
	getByIDFn        func(ctx context.Context, id int64) (*dbmodels.Task, error)
	listBySeasonFn func(ctx context.Context, seasonID int64) ([]dbmodels.Task, error)
}

func (f *fakeTaskRepo) Insert(ctx context.Context, t *dbmodels.Task) error {
	if f.insertFn != nil {
		return f.insertFn(ctx, t)
	}
	t.ID = 1
	return nil
}

func (f *fakeTaskRepo) Update(ctx context.Context, t *dbmodels.Task, columns ...string) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, t, columns...)
	}
	return nil
}

func (f *fakeTaskRepo) Delete(ctx context.Context, id int64) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func (f *fakeTaskRepo) GetByID(ctx context.Context, id int64) (*dbmodels.Task, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeTaskRepo) ListBySeason(ctx context.Context, seasonID int64) ([]dbmodels.Task, error) {
	if f.listBySeasonFn != nil {
		return f.listBySeasonFn(ctx, seasonID)
	}
	return nil, nil
}

func newTaskService(repo *fakeTaskRepo) *taskservice.TaskService {
	return taskservice.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func existingTask() *dbmodels.Task {
	return &dbmodels.Task{
		Entity:      dbmodels.Entity{ID: 1},
		SeasonID:  10,
		Category:    "quiz",
		Title:       "Old Title",
		Description: "Old desc",
		RewardCoins: 100,
		SortOrder:   0,
		Metadata:    json.RawMessage(`{}`),
		IsActive:    true,
	}
}

// --- Create ---

func TestTaskService_Create(t *testing.T) {
	t.Run("creates task with provided metadata", func(t *testing.T) {
		svc := newTaskService(&fakeTaskRepo{})

		task, err := svc.Create(context.Background(), &taskservice.CreateRequest{
			SeasonID:  10,
			Category:    "quiz",
			Title:       "Do a quiz",
			Description: "Answer all questions",
			RewardCoins: 50,
			Metadata:    json.RawMessage(`{"key":"value"}`),
			IsActive:    true,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task == nil {
			t.Fatal("expected non-nil task")
		}
		if task.Title != "Do a quiz" {
			t.Errorf("expected title %q, got %q", "Do a quiz", task.Title)
		}
	})

	t.Run("defaults metadata to {} when empty", func(t *testing.T) {
		var gotMetadata json.RawMessage
		repo := &fakeTaskRepo{
			insertFn: func(_ context.Context, t *dbmodels.Task) error {
				gotMetadata = t.Metadata
				t.ID = 1
				return nil
			},
		}
		svc := newTaskService(repo)

		_, err := svc.Create(context.Background(), &taskservice.CreateRequest{
			Title: "Task without metadata",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(gotMetadata) != "{}" {
			t.Errorf("expected metadata {}, got %s", gotMetadata)
		}
	})

	t.Run("propagates repo insert error", func(t *testing.T) {
		repoErr := errors.New("insert failed")
		svc := newTaskService(&fakeTaskRepo{
			insertFn: func(_ context.Context, _ *dbmodels.Task) error {
				return repoErr
			},
		})

		_, err := svc.Create(context.Background(), &taskservice.CreateRequest{Title: "x"})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- Update ---

func TestTaskService_Update(t *testing.T) {
	t.Run("updates only provided fields", func(t *testing.T) {
		newTitle := "New Title"
		newReward := 200
		var updatedColumns []string

		repo := &fakeTaskRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Task, error) {
				return existingTask(), nil
			},
			updateFn: func(_ context.Context, _ *dbmodels.Task, columns ...string) error {
				updatedColumns = columns
				return nil
			},
		}
		svc := newTaskService(repo)

		task, err := svc.Update(context.Background(), 1, &taskservice.UpdateRequest{
			Title:       &newTitle,
			RewardCoins: &newReward,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, task.Title)
		}
		if task.RewardCoins != 200 {
			t.Errorf("expected reward 200, got %d", task.RewardCoins)
		}
		if len(updatedColumns) != 2 {
			t.Errorf("expected 2 updated columns, got %v", updatedColumns)
		}
	})

	t.Run("no-op when no fields provided — repo Update not called", func(t *testing.T) {
		updateCalled := false
		repo := &fakeTaskRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Task, error) {
				return existingTask(), nil
			},
			updateFn: func(_ context.Context, _ *dbmodels.Task, _ ...string) error {
				updateCalled = true
				return nil
			},
		}
		svc := newTaskService(repo)

		_, err := svc.Update(context.Background(), 1, &taskservice.UpdateRequest{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updateCalled {
			t.Error("repo.Update must not be called when no fields change")
		}
	})

	t.Run("returns ErrNotFound when task does not exist", func(t *testing.T) {
		svc := newTaskService(&fakeTaskRepo{})

		_, err := svc.Update(context.Background(), 99, &taskservice.UpdateRequest{})

		if !errors.Is(err, taskservice.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("propagates repo GetByID error", func(t *testing.T) {
		repoErr := errors.New("db read error")
		svc := newTaskService(&fakeTaskRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Task, error) {
				return nil, repoErr
			},
		})

		_, err := svc.Update(context.Background(), 1, &taskservice.UpdateRequest{})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})

	t.Run("propagates repo Update error", func(t *testing.T) {
		repoErr := errors.New("update failed")
		newTitle := "x"
		svc := newTaskService(&fakeTaskRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Task, error) {
				return existingTask(), nil
			},
			updateFn: func(_ context.Context, _ *dbmodels.Task, _ ...string) error {
				return repoErr
			},
		})

		_, err := svc.Update(context.Background(), 1, &taskservice.UpdateRequest{Title: &newTitle})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- Delete ---

func TestTaskService_Delete(t *testing.T) {
	t.Run("deletes successfully", func(t *testing.T) {
		if err := newTaskService(&fakeTaskRepo{}).Delete(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrHasRelations on FK violation", func(t *testing.T) {
		svc := newTaskService(&fakeTaskRepo{
			deleteFn: func(_ context.Context, _ int64) error {
				return pgFKErr("23503")
			},
		})

		err := svc.Delete(context.Background(), 1)

		if !errors.Is(err, taskservice.ErrHasRelations) {
			t.Fatalf("expected ErrHasRelations, got: %v", err)
		}
	})

	t.Run("propagates generic repo error", func(t *testing.T) {
		repoErr := errors.New("delete failed")
		svc := newTaskService(&fakeTaskRepo{
			deleteFn: func(_ context.Context, _ int64) error { return repoErr },
		})

		if err := svc.Delete(context.Background(), 1); !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- GetByID ---

func TestTaskService_GetByID(t *testing.T) {
	t.Run("returns domain task when found", func(t *testing.T) {
		repo := &fakeTaskRepo{
			getByIDFn: func(_ context.Context, id int64) (*dbmodels.Task, error) {
				if id != 42 {
					t.Errorf("expected id=42, got %d", id)
				}
				return &dbmodels.Task{Entity: dbmodels.Entity{ID: 42}, Title: "Found Task"}, nil
			},
		}

		task, err := newTaskService(repo).GetByID(context.Background(), 42)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task == nil || task.ID != 42 {
			t.Fatalf("expected task id=42, got %v", task)
		}
	})

	t.Run("returns nil when task not found", func(t *testing.T) {
		task, err := newTaskService(&fakeTaskRepo{}).GetByID(context.Background(), 99)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task != nil {
			t.Fatalf("expected nil, got %v", task)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("db error")
		svc := newTaskService(&fakeTaskRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Task, error) {
				return nil, repoErr
			},
		})

		_, err := svc.GetByID(context.Background(), 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- ListBySeason ---

func TestTaskService_ListBySeason(t *testing.T) {
	t.Run("returns mapped domain tasks", func(t *testing.T) {
		repo := &fakeTaskRepo{
			listBySeasonFn: func(_ context.Context, seasonID int64) ([]dbmodels.Task, error) {
				if seasonID != 10 {
					t.Errorf("expected seasonID=10, got %d", seasonID)
				}
				return []dbmodels.Task{
					{Entity: dbmodels.Entity{ID: 1}, Title: "T1"},
					{Entity: dbmodels.Entity{ID: 2}, Title: "T2"},
				}, nil
			},
		}

		tasks, err := newTaskService(repo).ListBySeason(context.Background(), 10)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
	})

	t.Run("returns empty slice when no tasks", func(t *testing.T) {
		tasks, err := newTaskService(&fakeTaskRepo{}).ListBySeason(context.Background(), 1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tasks) != 0 {
			t.Fatalf("expected empty, got %d", len(tasks))
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("db error")
		svc := newTaskService(&fakeTaskRepo{
			listBySeasonFn: func(_ context.Context, _ int64) ([]dbmodels.Task, error) {
				return nil, repoErr
			},
		})

		_, err := svc.ListBySeason(context.Background(), 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- Domain mapping ---

func TestTaskService_DomainMapping(t *testing.T) {
	t.Run("maps task fields correctly", func(t *testing.T) {
		expected := &dbmodels.Task{
			Entity:      dbmodels.Entity{ID: 99},
			SeasonID:  5,
			Category:    "sport",
			Title:       "Run 5km",
			Description: "Complete a 5km run",
			RewardCoins: 150,
			SortOrder:   3,
			IsActive:    true,
			Metadata:    json.RawMessage(`{"distance":5}`),
		}

		repo := &fakeTaskRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Task, error) {
				return expected, nil
			},
		}

		task, _ := newTaskService(repo).GetByID(context.Background(), 99)

		if task.ID != 99 || task.SeasonID != 5 || task.Title != "Run 5km" {
			t.Errorf("unexpected domain mapping: %+v", task)
		}
		if task.RewardCoins != 150 {
			t.Errorf("expected RewardCoins=150, got %d", task.RewardCoins)
		}
	})
}

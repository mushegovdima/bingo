package seasonservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	dbmodels "go.mod/internal/db"
	"go.mod/internal/domain"
	seasonservice "go.mod/internal/services/season"
)

// --- fake repo ---

type fakeSeasonRepo struct {
	insertFn    func(ctx context.Context, c *dbmodels.Season) error
	updateFn    func(ctx context.Context, c *dbmodels.Season, columns ...string) error
	deleteFn    func(ctx context.Context, id int64) error
	getByIDFn   func(ctx context.Context, id int64) (*dbmodels.Season, error)
	getActiveFn func(ctx context.Context) (*dbmodels.Season, error)
}

func (f *fakeSeasonRepo) Insert(ctx context.Context, c *dbmodels.Season) error {
	if f.insertFn != nil {
		return f.insertFn(ctx, c)
	}
	c.ID = 1
	return nil
}

func (f *fakeSeasonRepo) Update(ctx context.Context, c *dbmodels.Season, columns ...string) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, c, columns...)
	}
	return nil
}

func (f *fakeSeasonRepo) Delete(ctx context.Context, id int64) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func (f *fakeSeasonRepo) GetByID(ctx context.Context, id int64) (*dbmodels.Season, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeSeasonRepo) GetActive(ctx context.Context) (*dbmodels.Season, error) {
	if f.getActiveFn != nil {
		return f.getActiveFn(ctx)
	}
	return nil, nil
}

func (f *fakeSeasonRepo) List(ctx context.Context) ([]*dbmodels.Season, error) {
	return nil, nil
}

func (f *fakeSeasonRepo) ListActive(ctx context.Context) ([]*dbmodels.Season, error) {
	return nil, nil
}

// --- helpers ---

func newSeasonService(repo *fakeSeasonRepo) *seasonservice.SeasonService {
	return seasonservice.NewService(repo, noopLogger())
}

var now = time.Now()
var later = now.Add(24 * time.Hour)

func newCreateReq() *seasonservice.CreateRequest {
	return &seasonservice.CreateRequest{
		Title:     "Test Season",
		StartDate: now,
		EndDate:   later,
		IsActive:  false,
	}
}

// --- Create ---

func TestSeasonService_Create(t *testing.T) {
	t.Run("creates season and returns domain model", func(t *testing.T) {
		svc := newSeasonService(&fakeSeasonRepo{})

		c, err := svc.Create(context.Background(), newCreateReq())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Fatal("expected non-nil season")
		}
		if c.Title != "Test Season" {
			t.Errorf("expected title %q, got %q", "Test Season", c.Title)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("db down")
		repo := &fakeSeasonRepo{
			insertFn: func(_ context.Context, _ *dbmodels.Season) error {
				return repoErr
			},
		}
		svc := newSeasonService(repo)

		_, err := svc.Create(context.Background(), newCreateReq())

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- Update ---

func TestSeasonService_Update(t *testing.T) {
	existingSeason := func() *dbmodels.Season {
		return &dbmodels.Season{
			Entity:    dbmodels.Entity{ID: 1},
			Title:     "Old Title",
			StartDate: now,
			EndDate:   later,
			IsActive:  false,
		}
	}

	t.Run("updates only provided fields", func(t *testing.T) {
		newTitle := "New Title"
		var updatedColumns []string

		repo := &fakeSeasonRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Season, error) {
				return existingSeason(), nil
			},
			updateFn: func(_ context.Context, _ *dbmodels.Season, columns ...string) error {
				updatedColumns = columns
				return nil
			},
		}
		svc := newSeasonService(repo)

		c, err := svc.Update(context.Background(), 1, &seasonservice.UpdateRequest{Title: &newTitle})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, c.Title)
		}
		if len(updatedColumns) != 1 || updatedColumns[0] != "title" {
			t.Errorf("expected columns [title], got %v", updatedColumns)
		}
	})

	t.Run("no-op when no fields provided — repo Update not called", func(t *testing.T) {
		updateCalled := false
		repo := &fakeSeasonRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Season, error) {
				return existingSeason(), nil
			},
			updateFn: func(_ context.Context, _ *dbmodels.Season, _ ...string) error {
				updateCalled = true
				return nil
			},
		}
		svc := newSeasonService(repo)

		_, err := svc.Update(context.Background(), 1, &seasonservice.UpdateRequest{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updateCalled {
			t.Error("repo.Update must not be called when no fields change")
		}
	})

	t.Run("returns ErrNotFound when season does not exist", func(t *testing.T) {
		repo := &fakeSeasonRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Season, error) {
				return nil, nil
			},
		}
		svc := newSeasonService(repo)

		_, err := svc.Update(context.Background(), 99, &seasonservice.UpdateRequest{})

		if !errors.Is(err, seasonservice.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("propagates repo GetByID error", func(t *testing.T) {
		repoErr := errors.New("db error")
		repo := &fakeSeasonRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Season, error) {
				return nil, repoErr
			},
		}
		svc := newSeasonService(repo)

		_, err := svc.Update(context.Background(), 1, &seasonservice.UpdateRequest{})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})

	t.Run("propagates repo Update error", func(t *testing.T) {
		repoErr := errors.New("update failed")
		newTitle := "x"
		repo := &fakeSeasonRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Season, error) {
				return existingSeason(), nil
			},
			updateFn: func(_ context.Context, _ *dbmodels.Season, _ ...string) error {
				return repoErr
			},
		}
		svc := newSeasonService(repo)

		_, err := svc.Update(context.Background(), 1, &seasonservice.UpdateRequest{Title: &newTitle})

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})

	t.Run("updates all fields at once", func(t *testing.T) {
		newTitle := "Full"
		isActive := true
		newEnd := later.Add(time.Hour)
		var updatedColumns []string

		repo := &fakeSeasonRepo{
			getByIDFn: func(_ context.Context, _ int64) (*dbmodels.Season, error) {
				return existingSeason(), nil
			},
			updateFn: func(_ context.Context, _ *dbmodels.Season, columns ...string) error {
				updatedColumns = columns
				return nil
			},
		}
		svc := newSeasonService(repo)

		_, err := svc.Update(context.Background(), 1, &seasonservice.UpdateRequest{
			Title:    &newTitle,
			IsActive: &isActive,
			EndDate:  &newEnd,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(updatedColumns) != 3 {
			t.Errorf("expected 3 columns, got %v", updatedColumns)
		}
	})
}

// --- Delete ---

func TestSeasonService_Delete(t *testing.T) {
	t.Run("deletes successfully", func(t *testing.T) {
		svc := newSeasonService(&fakeSeasonRepo{})

		if err := svc.Delete(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrHasRelations on FK violation", func(t *testing.T) {
		repo := &fakeSeasonRepo{
			deleteFn: func(_ context.Context, _ int64) error {
				return pgFKErr("23503")
			},
		}
		svc := newSeasonService(repo)

		err := svc.Delete(context.Background(), 1)

		if !errors.Is(err, seasonservice.ErrHasRelations) {
			t.Fatalf("expected ErrHasRelations, got: %v", err)
		}
	})

	t.Run("propagates generic repo error", func(t *testing.T) {
		repoErr := errors.New("unexpected db error")
		repo := &fakeSeasonRepo{
			deleteFn: func(_ context.Context, _ int64) error { return repoErr },
		}
		svc := newSeasonService(repo)

		err := svc.Delete(context.Background(), 1)

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- GetActive ---

func TestSeasonService_GetActive(t *testing.T) {
	t.Run("returns active season", func(t *testing.T) {
		repo := &fakeSeasonRepo{
			getActiveFn: func(_ context.Context) (*dbmodels.Season, error) {
				return &dbmodels.Season{Entity: dbmodels.Entity{ID: 5}, Title: "Active"}, nil
			},
		}
		svc := newSeasonService(repo)

		c, err := svc.GetActive(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil || c.ID != 5 {
			t.Fatalf("expected season id=5, got: %v", c)
		}
	})

	t.Run("returns nil when no active season", func(t *testing.T) {
		svc := newSeasonService(&fakeSeasonRepo{})

		c, err := svc.GetActive(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c != nil {
			t.Fatalf("expected nil, got: %v", c)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("db error")
		repo := &fakeSeasonRepo{
			getActiveFn: func(_ context.Context) (*dbmodels.Season, error) {
				return nil, repoErr
			},
		}
		svc := newSeasonService(repo)

		_, err := svc.GetActive(context.Background())

		if !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got: %v", err)
		}
	})
}

// --- toDomain mapping ---

func TestSeasonService_DomainMapping(t *testing.T) {
	t.Run("domain fields are correctly mapped from db model", func(t *testing.T) {
		var got *domain.Season
		repo := &fakeSeasonRepo{
			getActiveFn: func(_ context.Context) (*dbmodels.Season, error) {
				return &dbmodels.Season{
					Entity:    dbmodels.Entity{ID: 42},
					Title:     "Mapped",
					StartDate: now,
					EndDate:   later,
					IsActive:  true,
				}, nil
			},
		}
		svc := newSeasonService(repo)

		got, _ = svc.GetActive(context.Background())

		if got.ID != 42 {
			t.Errorf("ID mismatch: got %d", got.ID)
		}
		if !got.IsActive {
			t.Error("IsActive should be true")
		}
	})
}

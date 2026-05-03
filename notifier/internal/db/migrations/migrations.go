// Package migrations bundles the SQL files for the notifier database schema
// and exposes a thin wrapper around bun's migrator so cmd/migrator can drive it.
package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

//go:embed *.sql
var EmbeddedFiles embed.FS

type Migrator struct {
	m *migrate.Migrator
}

func NewMigrator(db *bun.DB, fsys fs.FS) (*Migrator, error) {
	migs := migrate.NewMigrations()
	if err := migs.Discover(fsys); err != nil {
		return nil, fmt.Errorf("discover migrations: %w", err)
	}
	m := migrate.NewMigrator(db, migs)
	if err := m.Init(context.Background()); err != nil {
		return nil, fmt.Errorf("init migrator: %w", err)
	}
	return &Migrator{m: m}, nil
}

func (p *Migrator) Migrate(ctx context.Context) (int, error) {
	group, err := p.m.Migrate(ctx)
	if err != nil {
		return 0, err
	}
	return len(group.Migrations.Applied()), nil
}

func (p *Migrator) Rollback(ctx context.Context) error {
	_, err := p.m.Rollback(ctx)
	return err
}

func (p *Migrator) Status(ctx context.Context) error {
	ms, err := p.m.MigrationsWithStatus(ctx)
	if err != nil {
		return err
	}
	for _, mig := range ms {
		state := "pending"
		if !mig.MigratedAt.IsZero() {
			state = "applied " + mig.MigratedAt.Format(time.RFC3339)
		}
		fmt.Printf("%-40s %s\n", mig.Name, state)
	}
	return nil
}

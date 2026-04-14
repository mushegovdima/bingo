package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"time"

	"github.com/uptrace/bun/migrate"
	"go.mod/internal/db"
)

type migrator struct {
	db       *db.DB
	migrator *migrate.Migrator
}

//go:embed *.sql
var EmbeddedFiles embed.FS

func NewMigrator(p *db.DB, fsys fs.FS) (*migrator, error) {
	migs := migrate.NewMigrations()
	if err := migs.Discover(fsys); err != nil {
		return nil, err
	}
	m := migrate.NewMigrator(p.DB(), migs)
	return &migrator{
		db:       p,
		migrator: m,
	}, m.Init(context.Background())
}

func (p *migrator) Migrate(ctx context.Context) (int, error) {
	group, err := p.migrator.Migrate(ctx)
	return len(group.Migrations.Applied()), err
}

func (p *migrator) Rollback(ctx context.Context) error {
	_, err := p.migrator.Rollback(ctx)
	return err
}

func (p *migrator) MigrateStatus(ctx context.Context) error {
	ms, err := p.migrator.MigrationsWithStatus(ctx)
	if err != nil {
		return err
	}
	for _, m := range ms {
		state := "pending"
		if !m.MigratedAt.IsZero() {
			state = "applied " + m.MigratedAt.Format(time.RFC3339)
		}
		fmt.Printf("%-40s %s\n", m.Name, state)
	}
	return nil
}

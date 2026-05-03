// Command migrator applies notifier database migrations. Mirrors api/cmd/migrator
// so docker-compose can run a one-shot job before starting the notifier service.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"notifier/internal/config"
	"notifier/internal/db/migrations"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	env := flag.String("env", "prod", "environment (dev|prod)")
	configPath := flag.String("config", "", "path to config file (default: {env}.env)")
	flag.Parse()

	cfg, err := config.LoadConfig(*env, *configPath)
	if err != nil {
		logger.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DBConnectionString)))
	defer sqlDB.Close()
	bunDB := bun.NewDB(sqlDB, pgdialect.New())

	if err := bunDB.PingContext(context.Background()); err != nil {
		logger.Error("failed to connect to db", slog.Any("error", err))
		os.Exit(1)
	}

	m, err := migrations.NewMigrator(bunDB, migrations.EmbeddedFiles)
	if err != nil {
		logger.Error("failed to create migrator", slog.Any("error", err))
		os.Exit(1)
	}

	cmd := "up"
	if args := flag.Args(); len(args) > 0 {
		cmd = args[0]
	}

	ctx := context.Background()
	switch cmd {
	case "up":
		cnt, err := m.Migrate(ctx)
		if err != nil {
			logger.Error("migrate up", slog.Any("error", err))
			os.Exit(1)
		}
		fmt.Printf("migrations applied. count: %d\n", cnt)
	case "rollback":
		if err := m.Rollback(ctx); err != nil {
			logger.Error("migrate rollback", slog.Any("error", err))
			os.Exit(1)
		}
		fmt.Println("last migration rolled back")
	case "status":
		if err := m.Status(ctx); err != nil {
			logger.Error("migrate status", slog.Any("error", err))
			os.Exit(1)
		}
	default:
		logger.Error("unknown command", slog.String("cmd", cmd))
		os.Exit(2)
	}
}

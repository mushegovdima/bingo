package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"go.mod/internal/config"
	"go.mod/internal/db"
	"go.mod/internal/db/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	configPath := flag.String("config", "", "path to config file (default: {env}.env)")
	env := flag.String("env", "prod", "environment (dev|prod)")
	migrationsPath := flag.String("migrations-path", "", "path to migrations directory (default: embedded)")
	flag.Parse()

	cfg, err := config.LoadConfig(*env, *configPath)
	if err != nil {
		logger.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	pg, err := db.NewDB(cfg, logger)
	if err != nil {
		logger.Error("failed to connect to postgres", slog.Any("error", err))
		os.Exit(1)
	}
	defer pg.Close()

	ctx := context.Background()

	var fsys fs.FS
	if *migrationsPath != "" {
		fsys = os.DirFS(*migrationsPath)
	} else {
		fsys = migrations.EmbeddedFiles
	}

	m, err := migrations.NewMigrator(pg, fsys)
	if err != nil {
		logger.Error("failed to create migrator", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("migrator started", slog.Any("command", os.Args))

	cmd := "up"
	if args := flag.Args(); len(args) > 0 {
		cmd = args[0]
	}

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
		if err := m.MigrateStatus(ctx); err != nil {
			logger.Error("migrate status", slog.Any("error", err))
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\nusage: migrator [up|rollback|status]\n", cmd)
		os.Exit(1)
	}
}

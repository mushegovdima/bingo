package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"go.mod/internal/config"
	"go.mod/internal/db"
	dbmodels "go.mod/internal/db"
	"go.mod/internal/domain"
)

func main() {
	env := flag.String("env", "dev", "environment (dev|prod)")
	configPath := flag.String("config", "", "path to config file (default: {env}.env)")
	username := flag.String("username", "", "telegram username (without @), required")
	rolesRaw := flag.String("roles", "", "comma-separated roles to set, e.g. manager,resident")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if *username == "" || *rolesRaw == "" {
		fmt.Fprintln(os.Stderr, "usage: setroles -username <username> -roles <role1,role2>")
		fmt.Fprintln(os.Stderr, "       valid roles: manager, resident")
		os.Exit(1)
	}

	var roles []domain.UserRole
	for _, raw := range strings.Split(*rolesRaw, ",") {
		r := domain.UserRole(strings.TrimSpace(raw))
		switch r {
		case domain.Manager, domain.Resident:
			roles = append(roles, r)
		default:
			fmt.Fprintf(os.Stderr, "unknown role %q (valid: manager, resident)\n", raw)
			os.Exit(1)
		}
	}

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
	bunDB := pg.DB()

	user := &dbmodels.User{}
	err = bunDB.NewSelect().Model(user).Where("username = ?", *username).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		fmt.Fprintf(os.Stderr, "user %q not found\n", *username)
		os.Exit(1)
	}
	if err != nil {
		logger.Error("failed to query user", slog.Any("error", err))
		os.Exit(1)
	}

	user.Roles = roles
	_, err = bunDB.NewUpdate().Model(user).
		Column("roles").
		Where("id = ?", user.ID).
		Exec(ctx)
	if err != nil {
		logger.Error("failed to update roles", slog.Any("error", err))
		os.Exit(1)
	}

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = string(r)
	}
	fmt.Printf("OK: user %q (id=%d) roles set to [%s]\n", user.Username, user.ID, strings.Join(roleNames, ", "))
}

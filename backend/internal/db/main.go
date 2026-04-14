package db

import (
	"log/slog"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.mod/internal/config"
	"go.mod/internal/domain"

	"database/sql"
)

// DB wrapper for database connection and operations
type DB struct {
	domain.Database
	logger *slog.Logger
	db     *bun.DB
	sql    *sql.DB
}

func NewDB(config *config.Config, logger *slog.Logger) (*DB, error) {
	db, sqlDB, err := connectDB(config)
	if err != nil {
		return nil, err
	}

	return &DB{
		logger: logger,
		db:     db,
		sql:    sqlDB,
	}, nil
}

func (db *DB) Close() error {
	return db.sql.Close()
}

func (db *DB) DB() *bun.DB {
	return db.db
}

func connectDB(config *config.Config) (*bun.DB, *sql.DB, error) {
	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(config.DBConnectionString)))

	db := bun.NewDB(sqlDB, pgdialect.New())
	if err := db.Ping(); err != nil {
		return nil, nil, err
	}

	return db, sqlDB, nil
}

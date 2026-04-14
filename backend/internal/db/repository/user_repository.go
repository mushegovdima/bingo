package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/uptrace/bun"
	dbmodels "go.mod/internal/db"
	"go.mod/internal/services/cache"
)

type UserRepository struct {
	db     *bun.DB
	logger *slog.Logger
	cache  *cache.LocalCache[int64, *dbmodels.User]
}

func NewUserRepository(ctx context.Context, db *bun.DB, logger *slog.Logger) *UserRepository {
	// todo: change to distributed cache if needed, e.g. RedisCache[string, *dbmodels.User]
	c := cache.NewLocalCache[int64, *dbmodels.User](ctx, 3*time.Minute)
	return &UserRepository{db: db, logger: logger, cache: c}
}

func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*dbmodels.User, error) {
	op := "db.userrepository.GetByTelegramID"
	log := r.logger.With(slog.String("op", op), slog.Int64("telegram_id", telegramID))

	user := &dbmodels.User{}
	err := r.db.NewSelect().Model(user).Where("telegram_id = ?", telegramID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		log.DebugContext(ctx, "user not found")
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	log.DebugContext(ctx, "user found", slog.Int64("user_id", user.ID))
	return user, nil
}

func (r *UserRepository) Insert(ctx context.Context, user *dbmodels.User) error {
	op := "db.userrepository.Insert"
	log := r.logger.With(slog.String("op", op), slog.Int64("telegram_id", user.TelegramID))

	_, err := r.db.NewInsert().Model(user).Returning("id").Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	log.InfoContext(ctx, "user inserted", slog.Int64("user_id", user.ID))
	return nil
}

func (r *UserRepository) Update(ctx context.Context, user *dbmodels.User, columns ...string) error {
	op := "db.userrepository.Update"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", user.ID))

	_, err := r.db.NewUpdate().Model(user).Column(columns...).Where("id = ?", user.ID).Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("columns", columns), slog.Any("error", err))
		return err
	}
	log.InfoContext(ctx, "user updated", slog.Any("columns", columns))
	r.cache.Delete(user.ID)
	return nil
}

func (r *UserRepository) GetById(ctx context.Context, id int64) (*dbmodels.User, error) {
	op := "db.userrepository.GetById"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", id))

	if u, ok := r.cache.Get(id); ok {
		return u, nil
	}

	user := &dbmodels.User{}
	err := r.db.NewSelect().Model(user).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		log.DebugContext(ctx, "user not found")
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}

	r.cache.Set(id, user)
	log.DebugContext(ctx, "user found", slog.Int64("telegram_id", user.TelegramID))
	return user, nil
}

func (r *UserRepository) List(ctx context.Context) ([]*dbmodels.User, error) {
	op := "db.userrepository.List"
	log := r.logger.With(slog.String("op", op))

	var users []*dbmodels.User
	if err := r.db.NewSelect().Model(&users).OrderExpr("id ASC").Scan(ctx); err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	log.DebugContext(ctx, "users listed", slog.Int("count", len(users)))
	return users, nil
}

func (r *UserRepository) SetIsBlocked(ctx context.Context, id int64, isBlocked bool) error {
	op := "db.userrepository.SetIsBlocked"
	log := r.logger.With(slog.String("op", op), slog.Int64("user_id", id), slog.Bool("is_blocked", isBlocked))

	_, err := r.db.NewUpdate().TableExpr("users").
		Set("is_blocked = ?", isBlocked).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return err
	}
	log.InfoContext(ctx, "user is_blocked updated")
	r.cache.Delete(id)
	return nil
}

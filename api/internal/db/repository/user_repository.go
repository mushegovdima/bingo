package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.mod/internal/cache"
	dbmodels "go.mod/internal/db"
	"log/slog"
	"time"
)

// UserFilter is a flat AND-conjunction of predicates evaluated against the users table.
// It lives in the repository because it describes a query shape, not a business concept;
// callers (e.g. the notification worker) translate their domain filter into this type.
type UserFilter struct {
	// IDs restricts the result to a fixed set of user ids.
	IDs []int64
	// Roles restricts the result to users having any of the listed roles.
	Roles []string
	// ExcludeBlocked drops users with is_blocked = true.
	ExcludeBlocked bool
	// OnlyWithTelegram drops users with telegram_id = 0.
	OnlyWithTelegram bool
}

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

// ListByFilter returns up to limit users matching the filter, with id > afterID.
// The result is ordered by id ASC so callers can paginate by feeding the last id back as afterID.
// All non-zero filter fields combine with logical AND.
func (r *UserRepository) ListByFilter(ctx context.Context, filter UserFilter, afterID int64, limit int) ([]*dbmodels.User, error) {
	op := "db.userrepository.ListByFilter"
	log := r.logger.With(slog.String("op", op))

	if limit <= 0 {
		limit = 500
	}

	var users []*dbmodels.User
	q := r.db.NewSelect().Model(&users).Where("id > ?", afterID)

	if len(filter.IDs) > 0 {
		q = q.Where("id IN (?)", bun.In(filter.IDs))
	}
	if len(filter.Roles) > 0 {
		q = q.Where("roles && ?", pgdialect.Array(filter.Roles))
	}
	if filter.ExcludeBlocked {
		q = q.Where("is_blocked = FALSE")
	}
	if filter.OnlyWithTelegram {
		q = q.Where("telegram_id <> 0")
	}

	if err := q.OrderExpr("id ASC").Limit(limit).Scan(ctx); err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, err
	}
	log.DebugContext(ctx, "users listed by filter", slog.Int("count", len(users)), slog.Int64("after_id", afterID))
	return users, nil
}

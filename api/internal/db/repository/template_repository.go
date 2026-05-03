package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/uptrace/bun"
	dbmodels "go.mod/internal/db"
	templatedomain "go.mod/internal/domain/template"
)

// TemplateRepository is the persistence layer for message_templates and
// message_template_history. Cache lives in the service layer.
type TemplateRepository struct {
	db     *bun.DB
	logger *slog.Logger
}

func NewTemplateRepository(db *bun.DB, logger *slog.Logger) *TemplateRepository {
	return &TemplateRepository{db: db, logger: logger}
}

// GetByCodename returns the template with the given codename, or nil if not found.
func (r *TemplateRepository) GetByCodename(ctx context.Context, codename string) (*templatedomain.Template, error) {
	op := "db.templaterepository.GetByCodename"
	log := r.logger.With(slog.String("op", op), slog.String("codename", codename))

	row := &dbmodels.Template{}
	err := r.db.NewSelect().Model(row).Where("codename = ?", codename).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return toDomain(row), nil
}

// List returns all templates ordered by codename.
func (r *TemplateRepository) List(ctx context.Context) ([]*templatedomain.Template, error) {
	op := "db.templaterepository.List"
	log := r.logger.With(slog.String("op", op))

	var rows []*dbmodels.Template
	if err := r.db.NewSelect().Model(&rows).OrderExpr("codename ASC").Scan(ctx); err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	out := make([]*templatedomain.Template, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomain(row))
	}
	return out, nil
}

// UpdateBody applies a new body to the template and appends a history row.
// Both writes happen in a single transaction.
func (r *TemplateRepository) UpdateBody(
	ctx context.Context,
	codename string,
	body string,
	changedBy int64,
) (*templatedomain.Template, error) {
	op := "db.templaterepository.UpdateBody"
	log := r.logger.With(slog.String("op", op), slog.String("codename", codename))

	now := time.Now()
	var result *templatedomain.Template

	if err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		row := &dbmodels.Template{}
		if err := tx.NewSelect().Model(row).Where("codename = ?", codename).For("UPDATE").Scan(ctx); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return templatedomain.ErrNotFound
			}
			return err
		}

		row.Body = body
		row.UpdatedAt = now

		if _, err := tx.NewUpdate().Model(row).
			Column("body", "updated_at").
			Where("id = ?", row.ID).
			Exec(ctx); err != nil {
			return err
		}

		hist := &dbmodels.TemplateHistory{
			TemplateID: row.ID,
			Body:       body,
			ChangedBy:  changedBy,
			ChangedAt:  now,
		}
		if _, err := tx.NewInsert().Model(hist).Exec(ctx); err != nil {
			return err
		}

		result = toDomain(row)
		return nil
	}); err != nil {
		log.ErrorContext(ctx, "update failed", slog.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.InfoContext(ctx, "template body updated", slog.String("codename", codename))
	return result, nil
}

// ListHistory returns all body-change history entries for a template, newest first.
func (r *TemplateRepository) ListHistory(ctx context.Context, codename string) ([]*templatedomain.TemplateHistory, error) {
	op := "db.templaterepository.ListHistory"
	log := r.logger.With(slog.String("op", op), slog.String("codename", codename))

	row := &dbmodels.Template{}
	err := r.db.NewSelect().Model(row).Column("id").Where("codename = ?", codename).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, templatedomain.ErrNotFound
	}
	if err != nil {
		log.ErrorContext(ctx, "resolve id failed", slog.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var rows []*dbmodels.TemplateHistory
	if err := r.db.NewSelect().Model(&rows).
		Where("template_id = ?", row.ID).
		OrderExpr("changed_at DESC").
		Scan(ctx); err != nil {
		log.ErrorContext(ctx, "query failed", slog.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	out := make([]*templatedomain.TemplateHistory, 0, len(rows))
	for _, h := range rows {
		out = append(out, toHistoryDomain(h))
	}
	return out, nil
}

func toDomain(row *dbmodels.Template) *templatedomain.Template {
	return &templatedomain.Template{
		ID:        row.ID,
		Codename:  row.Codename,
		Body:      row.Body,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func toHistoryDomain(row *dbmodels.TemplateHistory) *templatedomain.TemplateHistory {
	return &templatedomain.TemplateHistory{
		ID:         row.ID,
		TemplateID: row.TemplateID,
		Body:       row.Body,
		ChangedBy:  row.ChangedBy,
		ChangedAt:  row.ChangedAt,
	}
}

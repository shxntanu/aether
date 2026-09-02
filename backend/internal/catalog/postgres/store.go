package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shxntanu/aether/backend/internal/domain"
	"github.com/shxntanu/aether/backend/migrations"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct {
	db  *sql.DB
	orm *gorm.DB
}

const schemaMigrationsTable = "schema_migrations_tbl"

var _ domain.Repository = (*Store)(nil)

func Open(ctx context.Context, dataSourceName string) (*Store, error) {
	orm, err := gorm.Open(gormpostgres.Open(dataSourceName), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL catalog ORM: %w", err)
	}
	db, err := orm.DB()
	if err != nil {
		return nil, fmt.Errorf("get PostgreSQL catalog connection: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping PostgreSQL catalog: %w", err)
	}

	store := &Store{db: db, orm: orm}
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) WithinTransaction(
	ctx context.Context,
	operation func(domain.Repository) error,
) error {
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		transactionalStore := &Store{db: s.db, orm: tx}
		return operation(transactionalStore)
	})
	if err != nil {
		return fmt.Errorf("catalog transaction: %w", err)
	}
	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
		DO $$
		BEGIN
			IF to_regclass('public.schema_migrations') IS NOT NULL
				AND to_regclass('public.schema_migrations_tbl') IS NULL THEN
				ALTER TABLE schema_migrations RENAME TO schema_migrations_tbl;
			END IF;
			IF to_regclass('public.members') IS NOT NULL
				AND to_regclass('public.members_tbl') IS NULL THEN
				ALTER TABLE members RENAME TO members_tbl;
			END IF;
			IF to_regclass('public.documents') IS NOT NULL
				AND to_regclass('public.documents_tbl') IS NULL THEN
				ALTER TABLE documents RENAME TO documents_tbl;
			END IF;
			IF to_regclass('public.tags') IS NOT NULL
				AND to_regclass('public.tags_tbl') IS NULL THEN
				ALTER TABLE tags RENAME TO tags_tbl;
			END IF;
			IF to_regclass('public.document_tags') IS NOT NULL
				AND to_regclass('public.document_tags_tbl') IS NULL THEN
				ALTER TABLE document_tags RENAME TO document_tags_tbl;
			END IF;
			IF to_regclass('public.audit_events') IS NOT NULL
				AND to_regclass('public.audit_events_tbl') IS NULL THEN
				ALTER TABLE audit_events RENAME TO audit_events_tbl;
			END IF;
			IF to_regclass('public.sessions') IS NOT NULL
				AND to_regclass('public.sessions_tbl') IS NULL THEN
				ALTER TABLE sessions RENAME TO sessions_tbl;
			END IF;
			IF to_regclass('public.auth_flows') IS NOT NULL
				AND to_regclass('public.auth_flows_tbl') IS NULL THEN
				ALTER TABLE auth_flows RENAME TO auth_flows_tbl;
			END IF;
			IF to_regclass('public.upload_requests') IS NOT NULL
				AND to_regclass('public.upload_requests_tbl') IS NULL THEN
				ALTER TABLE upload_requests RENAME TO upload_requests_tbl;
			END IF;
		END $$;

		CREATE TABLE IF NOT EXISTS `+schemaMigrationsTable+` (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL
		)`); err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}

	available, err := migrations.PostgreSQL()
	if err != nil {
		return err
	}
	for _, migration := range available {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", migration.Version, err)
		}

		var applied bool
		checkQuery := "SELECT EXISTS (SELECT 1 FROM " + schemaMigrationsTable + " " +
			"WHERE version = $1)"
		if err := tx.QueryRowContext(ctx, checkQuery, migration.Version).Scan(&applied); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("check migration %s: %w", migration.Version, err)
		}
		if applied {
			if err := tx.Rollback(); err != nil {
				return fmt.Errorf("close migration %s check: %w", migration.Version, err)
			}
			continue
		}

		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", migration.Version, err)
		}
		recordQuery := "INSERT INTO " + schemaMigrationsTable + " " +
			"(version, applied_at) VALUES ($1, CURRENT_TIMESTAMP)"
		if _, err := tx.ExecContext(ctx, recordQuery, migration.Version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", migration.Version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", migration.Version, err)
		}
	}
	return nil
}

func translateError(operation string, err error) error {
	if err == nil {
		return nil
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return fmt.Errorf("%s: %w", operation, domain.ErrAlreadyExists)
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return fmt.Errorf("%s: %w", operation, domain.ErrAlreadyExists)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

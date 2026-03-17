//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

func mustDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	cfg.MaxConns = 4
	cfg.MinConns = 0
	cfg.MaxConnIdleTime = 30 * time.Second
	cfg.MaxConnLifetime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}

	t.Cleanup(func() { pool.Close() })
	return pool
}

func ensureSchema(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	applyMigrations(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Clean between tests (idempotent).
	_, _ = db.Exec(ctx, `TRUNCATE TABLE attachments;`)
	_, _ = db.Exec(ctx, `TRUNCATE TABLE messages;`)
	_, _ = db.Exec(ctx, `TRUNCATE TABLE bug_reports;`)
	_, _ = db.Exec(ctx, `TRUNCATE TABLE refresh_tokens;`)
	_, _ = db.Exec(ctx, `TRUNCATE TABLE users;`)
}

func applyMigrations(t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// tests/integration -> repo root
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	srcURL := "file://" + filepath.ToSlash(filepath.Join(root, "migrations"))

	sqlDB := stdlib.OpenDBFromPool(db)
	defer func() { _ = sqlDB.Close() }()

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		t.Fatalf("migrate driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(srcURL, "postgres", driver)
	if err != nil {
		t.Fatalf("migrate new: %v", err)
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up: %v", err)
	}
	_, _ = m.Close()
}

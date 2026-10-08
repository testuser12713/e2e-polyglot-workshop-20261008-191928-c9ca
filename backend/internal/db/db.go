// Package db opens the PostgreSQL pool and applies the SQL migrations.
package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationsLockKey is the advisory-lock key that serializes migration runs.
// go test runs packages in parallel and several processes may share one
// database, and PostgreSQL does not tolerate two concurrent
// CREATE ... IF NOT EXISTS on the same objects (it can raise a duplicate
// catalog-key error). A session-level advisory lock makes the whole pass safe.
const migrationsLockKey int64 = 0x776f726b73686f70 // "workshop"

// Open parses DATABASE_URL, builds a pool and verifies it can reach the
// database. It never falls back to anything else: PostgreSQL is required.
func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return pool, nil
}

// migrationLockKey is the advisory-lock key that serializes concurrent
// migration runs. It spells "workshop" in ASCII.
const migrationLockKey int64 = 0x776f726b73686f70

// RunMigrations applies every migrations/*.sql file that has not been applied
// yet, in file-name order. When dir is empty the directory is located by
// walking up from the working directory, so both the server and the tests find
// backend/migrations.
//
// The whole run happens in one transaction that first takes the advisory lock
// migrationLockKey. Several processes apply migrations to the same database at
// the same time — parallel `go test` packages, or more than one API instance —
// and PostgreSQL's CREATE TABLE IF NOT EXISTS is not safe under that race: two
// runs both pass the existence check and the loser dies with a duplicate
// pg_type/pg_class error. The advisory transaction lock serializes the runs and
// is released automatically when the transaction ends.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	// Hold a session-level advisory lock on a dedicated connection for the
	// whole pass, so concurrent test processes serialize instead of racing on
	// the schema_migrations DDL below.
	lockConn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration lock connection: %w", err)
	}
	defer lockConn.Release()
	if _, err := lockConn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationsLockKey); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		if _, err := lockConn.Exec(context.Background(),
			`SELECT pg_advisory_unlock($1)`, migrationsLockKey); err != nil {
			log.Printf("release migration lock: %v", err)
		}
	}()

	if dir == "" {
		dir = findMigrationsDir()
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return fmt.Errorf("list migrations in %s: %w", dir, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no migrations found in %s", dir)
	}
	sort.Strings(files)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migrations: %w", err)
	}
	// A rollback after a successful commit is a no-op, so this only cleans up
	// the error paths.
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLockKey); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}

	if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".sql")
		var applied bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied {
			continue
		}
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		// With no arguments pgx uses the simple protocol, so a file may contain
		// several statements.
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}

// findMigrationsDir walks up from the working directory looking for a
// "migrations" directory, so the server (cwd=backend) and the tests
// (cwd=backend/internal/<pkg>) both locate backend/migrations.
func findMigrationsDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "migrations"
	}
	for {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "migrations"
		}
		dir = parent
	}
}

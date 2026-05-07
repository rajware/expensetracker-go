// Package postgres provides a PostgreSQL-backed implementation of the domain
// repository interfaces.
//
// Usage:
//
//	store, err := postgres.Open(ctx, "postgres://user:pass@host/dbname")
//	if err != nil { ... }
//	defer store.Close()
//
//	userRepo := store.UserRepository()
//	expenseRepo := store.ExpenseRepository()
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib" // register the pgx driver under "pgx"
)

// advisoryLockKey is an arbitrary application-wide integer used as the
// PostgreSQL advisory lock key. It must be unique across all applications
// sharing the same database server.
const advisoryLockKey = 8472983645

// currentSchemaVersion is the schema version this binary requires.
// Ready returns an error if the database schema is older than this.
const currentSchemaVersion = 1

// Store holds the database connection pool and vends repository implementations.
type Store struct {
	db *sql.DB
}

// Open connects to the PostgreSQL database at the given URL, runs migrations
// under an advisory lock so that concurrent instances do not conflict, and
// returns a ready-to-use Store.
//
// The advisory lock is acquired on a dedicated connection and released
// when that connection closes — whether by normal completion,
// context cancellation, or process crash.
func Open(ctx context.Context, url string) (*Store, error) {
	log.Println("Connecting to PostgreSQL database...")
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("postgres.Open: %w", err)
	}

	// Use the caller's context only for the connectivity check.
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres.Open ping: %w", err)
	}
	log.Println("PostgreSQL connection established.")

	// Migration must not be bounded by the caller's startup deadline.
	// Lock waiters need time to acquire the advisory lock and run steps
	// after the first replica completes migration.
	if err := migrate(context.Background(), db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close releases the database connection pool.
func (s *Store) Close() error {
	return s.db.Close()
}

// Ready implements healthroutes.Checker. It pings the database and verifies
// the schema is at least the version this binary requires.
func (s *Store) Ready(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return err
	}
	var version int
	err := s.db.QueryRowContext(ctx, `SELECT version FROM schema_version`).Scan(&version)
	if err != nil {
		return fmt.Errorf("postgres.Ready read schema version: %w", err)
	}
	if version < currentSchemaVersion {
		return fmt.Errorf("postgres.Ready: schema version %d is below required %d", version, currentSchemaVersion)
	}
	return nil
}

// UserRepository returns a UserRepository backed by this store.
func (s *Store) UserRepository() *UserRepository {
	return &UserRepository{db: s.db}
}

// ExpenseRepository returns an ExpenseRepository backed by this store.
func (s *Store) ExpenseRepository() *ExpenseRepository {
	return &ExpenseRepository{db: s.db}
}

// migrate acquires an advisory lock on a dedicated connection, applies all
// schema migrations, then removes the lock and closes the connection.
func migrate(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("postgres.migrate acquire connection: %w", err)
	}
	defer conn.Close()

	log.Println("Acquiring database migration lock...")
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockKey); err != nil {
		return fmt.Errorf("postgres.migrate advisory lock: %w", err)
	}
	log.Println("Acquired database migration lock.")

	defer func() {
		log.Println("Releasing database migration lock.")
		conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, advisoryLockKey)
	}()

	log.Println("Starting schema migration...")
	if err := migrateSchema(ctx, conn); err != nil {
		return err
	}
	log.Println("Schema migration completed successfully.")

	return nil
}

// migrateSchema applies schema migrations sequentially up to currentSchemaVersion.
// Each step is applied only if the schema version is below that step's number.
//
// Step 1 is special: it must handle the case where schema_version does not yet
// exist (a v1.0.0 database), so it checks for the table before reading it.
func migrateSchema(ctx context.Context, conn *sql.Conn) error {
	version, err := readSchemaVersion(ctx, conn)
	if err != nil {
		return err
	}

	if version < 1 {
		if err := migrateStep1(ctx, conn); err != nil {
			return err
		}
	}

	return nil
}

// readSchemaVersion returns the current schema version, or 0 if the
// schema_version table does not yet exist (a v1.0.0 database).
func readSchemaVersion(ctx context.Context, conn *sql.Conn) (int, error) {
	var count int
	err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'schema_version'`,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("postgres.migrate check schema_version table: %w", err)
	}
	if count == 0 {
		return 0, nil
	}
	var version int
	if err := conn.QueryRowContext(ctx, `SELECT version FROM schema_version`).Scan(&version); err != nil {
		return 0, fmt.Errorf("postgres.migrate read schema version: %w", err)
	}
	return version, nil
}

// migrateStep1 creates the schema_version table, the users table, and the
// expenses table (all idempotent), then records schema version 1.
// ON DELETE CASCADE handles the user→expense cascade contract defined in
// domain.UserRepository.Delete.
func migrateStep1(ctx context.Context, conn *sql.Conn) error {
	_, err := conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS users (
			id            TEXT PRIMARY KEY,
			username      TEXT NOT NULL UNIQUE,
			display_name  TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS expenses (
			id          TEXT PRIMARY KEY,
			owner_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			occurred_at TIMESTAMPTZ NOT NULL,
			description TEXT NOT NULL,
			amount      DOUBLE PRECISION NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL
		);

		DELETE FROM schema_version;
		INSERT INTO schema_version (version) VALUES (1);
	`)
	if err != nil {
		return fmt.Errorf("postgres.migrate step 1: %w", err)
	}
	return nil
}

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

// Ready implements healthroutes.Checker. It pings the database to confirm
// connectivity. Open already applies all migrations before returning, so a
// successful ping means the store is fully ready.
func (s *Store) Ready(ctx context.Context) error {
	return s.db.PingContext(ctx)
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
	if err := applySchema(ctx, conn); err != nil {
		return err
	}
	log.Println("Schema migration completed successfully.")

	return nil
}

// applySchema creates the schema if it does not already exist.
// ON DELETE CASCADE handles the user→expense cascade contract defined in
// domain.UserRepository.Delete.
func applySchema(ctx context.Context, conn *sql.Conn) error {
	_, err := conn.ExecContext(ctx, `
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
	`)
	if err != nil {
		return fmt.Errorf("postgres.migrate apply schema: %w", err)
	}
	return nil
}

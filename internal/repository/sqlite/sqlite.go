// Package sqlite provides a SQLite-backed implementation of the domain
// repository interfaces.
//
// Usage:
//
//	store, err := sqlite.Open("./data/expense_tracker.db")
//	if err != nil { ... }
//	defer store.Close()
//
//	userRepo := store.UserRepository()
//	expenseRepo := store.ExpenseRepository()
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // register the sqlite driver
)

const defaultPath = "./data/expense_tracker.db"

// Store holds the database connection and vends repository implementations.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at the given path, creates the
// schema if needed, and returns a ready-to-use Store.
// Pass an empty string to use the default path.
func Open(path string) (*Store, error) {
	if path == "" {
		path = defaultPath
	}

	// Ensure parent paths
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	log.Printf("Opening SQLite database at %s...\n", path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite.Open: %w", err)
	}

	// Verify the connection is live.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite.Open ping: %w", err)
	}
	log.Println("SQLite database opened.")

	store := &Store{db: db}
	if err := store.init(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// Close releases the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// UserRepository returns a UserRepository backed by this store.
func (s *Store) UserRepository() *UserRepository {
	return &UserRepository{db: s.db}
}

// ExpenseRepository returns an ExpenseRepository backed by this store.
func (s *Store) ExpenseRepository() *ExpenseRepository {
	return &ExpenseRepository{db: s.db}
}

// Ready implements healthroutes.Checker. It pings the database to confirm
// connectivity. For SQLite, Open already applies all migrations synchronously,
// so a successful ping means the store is fully ready.
func (s *Store) Ready(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// init creates the schema if it does not already exist.
// Foreign key enforcement is enabled here; ON DELETE CASCADE handles the
// user→expense cascade contract defined in domain.UserRepository.Delete.
func (s *Store) init() error {
	log.Println("Starting SQLite schema migration...")
	_, err := s.db.Exec(`PRAGMA foreign_keys = ON`)
	if err != nil {
		return fmt.Errorf("sqlite.init enable foreign keys: %w", err)
	}

	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id           TEXT PRIMARY KEY,
			username     TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS expenses (
			id          TEXT PRIMARY KEY,
			owner_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			occurred_at TEXT NOT NULL,
			description TEXT NOT NULL,
			amount      REAL NOT NULL,
			created_at  TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("sqlite.init create schema: %w", err)
	}
	log.Println("SQLite schema migration completed successfully.")
	return nil
}

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
//	categoryRepo := store.CategoryRepository()
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

// currentSchemaVersion is the schema version this binary requires.
// Ready returns an error if the database schema is older than this.
const currentSchemaVersion = 2

// Store holds the database connection and vends repository implementations.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at the given path, runs all
// pending schema migrations, and returns a Store whose schema is guaranteed
// to be at currentSchemaVersion.
// Pass an empty string to use the default path.
func Open(path string) (*Store, error) {
	if path == "" {
		path = defaultPath
	}

	// Ensure parent paths exist.
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
	if err := store.migrate(); err != nil {
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

// CategoryRepository returns a CategoryRepository backed by this store.
func (s *Store) CategoryRepository() *CategoryRepository {
	return &CategoryRepository{db: s.db}
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
		return fmt.Errorf("sqlite.Ready read schema version: %w", err)
	}
	if version < currentSchemaVersion {
		return fmt.Errorf("sqlite.Ready: schema version %d is below required %d", version, currentSchemaVersion)
	}
	return nil
}

// migrate applies schema migrations sequentially up to currentSchemaVersion.
// Each step is applied only if the schema version is below that step's number.
//
// Step 1 is special: it must handle the case where schema_version does not yet
// exist (a v1.0.0 database), so it checks for the table before reading it.
//
// Foreign key enforcement is enabled here; ON DELETE CASCADE handles the
// user→expense cascade contract defined in domain.UserRepository.Delete.
func (s *Store) migrate() error {
	log.Println("Starting SQLite schema migration...")
	_, err := s.db.Exec(`PRAGMA foreign_keys = ON`)
	if err != nil {
		return fmt.Errorf("sqlite.init enable foreign keys: %w", err)
	}

	version, err := s.readSchemaVersion()
	if err != nil {
		return err
	}

	if version < 1 {
		if err := s.migrateStep1(); err != nil {
			return err
		}
	}

	if version < 2 {
		if err := s.migrateStep2(); err != nil {
			return err
		}
	}

	return nil
}

// readSchemaVersion returns the current schema version, or 0 if the
// schema_version table does not yet exist (a v1.0.0 database).
func (s *Store) readSchemaVersion() (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_version'`,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("sqlite.migrate check schema_version table: %w", err)
	}
	if count == 0 {
		return 0, nil
	}
	var version int
	if err := s.db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); err != nil {
		return 0, fmt.Errorf("sqlite.migrate read schema version: %w", err)
	}
	return version, nil
}

// migrateStep1 creates the schema_version table, the users table, and the
// expenses table (all idempotent), then records schema version 1.
func (s *Store) migrateStep1() error {
	_, err := s.db.Exec(`
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
			occurred_at TEXT NOT NULL,
			description TEXT NOT NULL,
			amount      REAL NOT NULL,
			created_at  TEXT NOT NULL
		);

		DELETE FROM schema_version;
		INSERT INTO schema_version (version) VALUES (1);
	`)
	if err != nil {
		return fmt.Errorf("sqlite.migrate step 1: %w", err)
	}

	log.Println("SQLite schema migration completed successfully.")
	return nil
}

// migrateStep2 adds the categories table, adds category_id to expenses,
// seeds the system user and Uncategorised category, then records schema
// version 2.
//
// category_id on expenses is nullable and defaults to UncategorisedCategoryID,
// so existing rows are automatically assigned to Uncategorised.
//
// On category delete, a trigger reclassifies affected expenses to Uncategorised,
// fulfilling the CategoryRepository.Delete contract without requiring
// application-level transactions.
func (s *Store) migrateStep2() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id       TEXT PRIMARY KEY,
			name     TEXT NOT NULL UNIQUE COLLATE NOCASE,
			owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE
		);

		ALTER TABLE expenses
			ADD COLUMN category_id TEXT
			DEFAULT '00000000-0000-0000-0000-000000000002';

		INSERT OR IGNORE INTO users (id, username, display_name, password_hash)
			VALUES ('00000000-0000-0000-0000-000000000001', 'system', 'System', '!');

		INSERT OR IGNORE INTO categories (id, name, owner_id)
			VALUES (
				'00000000-0000-0000-0000-000000000002',
				'Uncategorised',
				'00000000-0000-0000-0000-000000000001'
			);

		CREATE TRIGGER IF NOT EXISTS reclassify_expenses_on_category_delete
			BEFORE DELETE ON categories
			BEGIN
				UPDATE expenses
				SET category_id = '00000000-0000-0000-0000-000000000002'
				WHERE category_id = OLD.id
				  AND OLD.id != '00000000-0000-0000-0000-000000000002';
			END;

		DELETE FROM schema_version;
		INSERT INTO schema_version (version) VALUES (2);
	`)
	if err != nil {
		return fmt.Errorf("sqlite.migrate step 2: %w", err)
	}
	return nil
}

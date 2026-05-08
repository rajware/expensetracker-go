package sqlite_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rajware/expensetracker-go/internal/domain"
	"github.com/rajware/expensetracker-go/internal/domain/domaintest"
	"github.com/rajware/expensetracker-go/internal/repository/sqlite"
)

// TestSQLiteStore runs the full domain contract test suite against the SQLite
// repository implementations. Each sub-test gets a fresh in-memory database,
// so tests are fully isolated.
func TestSQLiteStore(t *testing.T) {
	domaintest.RunSuite(t, func() domaintest.TestApp {
		// ":memory:" gives a fresh, empty SQLite database for each call.
		store, err := sqlite.Open(":memory:")
		if err != nil {
			t.Fatalf("sqlite.Open: %v", err)
		}
		t.Cleanup(func() { store.Close() })

		return domaintest.TestApp{
			UserService:     domain.NewUserService(store.UserRepository()),
			ExpenseService:  domain.NewExpenseService(store.ExpenseRepository(), store.CategoryRepository()),
			CategoryService: domain.NewCategoryService(store.CategoryRepository()),
		}
	})
}

// TestOpenCreatesFile verifies that Open creates a database file at the given
// path, and that data written to it persists after closing and reopening.
func TestOpenCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	// Open and write a user.
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}

	us := domain.NewUserService(store.UserRepository())
	user, err := us.SignUp(t.Context(), "alice", "", "password123")
	if err != nil {
		t.Fatalf("SignUp: %v", err)
	}
	aliceID := user.ID
	store.Close()

	// Verify the file exists on disk.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("expected database file to exist on disk")
	}

	// Reopen and verify the data is still there.
	store2, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("sqlite.Open (reopen): %v", err)
	}
	defer store2.Close()

	us2 := domain.NewUserService(store2.UserRepository())
	view, err := us2.QueryByID(t.Context(), aliceID)
	if err != nil {
		t.Fatalf("QueryByID after reopen: %v", err)
	}
	if view.Username != "alice" {
		t.Errorf("expected username %q, got %q", "alice", view.Username)
	}
}

// TestOpenInvalidPath verifies that Open returns an error when given a path
// that cannot be created (e.g. a directory that does not exist and cannot be
// made, or a path with no write permission).
func TestOpenInvalidPath(t *testing.T) {
	// Use a path whose parent directory does not exist.
	path := "/nonexistent-directory/expense_tracker.db"
	_, err := sqlite.Open(path)
	if err == nil {
		t.Fatal("expected error when opening invalid path, got nil")
	}
}

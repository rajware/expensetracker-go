package postgres_test

import (
	"database/sql"
	"testing"

	"github.com/rajware/expensetracker-go/internal/domain"
	"github.com/rajware/expensetracker-go/internal/domain/domaintest"
	"github.com/rajware/expensetracker-go/internal/repository/postgres"
)

// TestPostgresStore runs the full domain contract test suite against the Postgres
// repository implementations.
func TestPostgresStore(t *testing.T) {
	domaintest.RunSuite(t, func() domaintest.TestApp {
		url := "postgres://testuser:testpassword@localhost:15432"
		// // ":memory:" gives a fresh, empty SQLite database for each call.
		// store, err := sqlite.Open(":memory:")
		db, err := sql.Open("pgx", url)
		if err != nil {
			t.Fatalf("postgres.pre-Open: %v", err)
		}
		defer db.Close()

		_, err = db.ExecContext(t.Context(), "DROP DATABASE IF EXISTS test_expensedb WITH (FORCE);")
		if err != nil {
			t.Fatalf("postgres.DropDB: %v", err)
		}

		_, err = db.ExecContext(t.Context(), "CREATE DATABASE test_expensedb;")
		if err != nil {
			t.Fatalf("postgres.RecreateDB: %v", err)
		}

		store, err := postgres.Open(t.Context(), url+"/test_expensedb")
		if err != nil {
			t.Fatalf("postgres.Open: %v", err)
		}

		t.Cleanup(func() { store.Close() })

		return domaintest.TestApp{
			UserService:    domain.NewUserService(store.UserRepository()),
			ExpenseService: domain.NewExpenseService(store.ExpenseRepository()),
		}
	})
}

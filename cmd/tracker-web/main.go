package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rajware/expensetracker-go/internal/api/rest"
	"github.com/rajware/expensetracker-go/internal/auth/cookie"
	"github.com/rajware/expensetracker-go/internal/domain"
	"github.com/rajware/expensetracker-go/internal/healthroutes"
	"github.com/rajware/expensetracker-go/internal/repository/postgres"
	"github.com/rajware/expensetracker-go/internal/repository/sqlite"
	"github.com/rajware/expensetracker-go/internal/ui/spa"
	"github.com/rajware/expensetracker-go/internal/webserver"
)

// version will be set by the build process.
// "latest" indicates non-build-process compile.
var version = "latest"

var (
	addrFlag     = flag.String("addr", "", "address the server listens on (default: :8080)")
	hmacKeyFlag  = flag.String("hmac-key", "", "HMAC signing key for auth tokens")
	dbDriverFlag = flag.String("db-driver", "", "database driver to use: sqlite or postgres (default: sqlite)")
	dbPathFlag   = flag.String("db-path", "", "path to the SQLite database file (default: data/expense_tracker.db)")
	dbURLFlag    = flag.String("db-url", "", "PostgreSQL connection URL (required when driver is postgres)")
)

// getOption returns the first non-empty value among: the flag, the named
// environment variable, and the default.
func getOption(flagValue, envVar, defaultValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return defaultValue
}

func main() {
	title := fmt.Sprintf("Expense Tracker v%v", version)
	log.Println(title)

	flag.Parse()

	addr := getOption(*addrFlag, "ET_ADDR", ":8080")
	hmacKey := getOption(*hmacKeyFlag, "ET_HMAC_KEY", "")
	dbDriver := getOption(*dbDriverFlag, "ET_DB_DRIVER", "sqlite")
	dbPath := getOption(*dbPathFlag, "ET_DB_PATH", "data/expense_tracker.db")
	dbURL := getOption(*dbURLFlag, "ET_DB_URL", "")

	if _, _, err := net.SplitHostPort(addr); err != nil {
		log.Fatalf("invalid addr %q: must be :PORT or HOST:PORT\n", addr)
	}

	if hmacKey == "" {
		log.Fatalln("HMAC key must be set via -hmac-key or ET_HMAC_KEY")
	}

	var (
		userService     domain.UserService
		expenseService  domain.ExpenseService
		categoryService domain.CategoryService
		checker         healthroutes.Checker
	)

	switch dbDriver {
	case "sqlite":
		if filepath.IsAbs(dbPath) || strings.HasPrefix(filepath.Clean(dbPath), "..") {
			log.Fatalln("db-path must be a relative path within the working directory")
		}

		store, err := sqlite.Open(dbPath)
		if err != nil {
			log.Fatalln(err)
		}
		userService = domain.NewUserService(store.UserRepository())
		categoryService = domain.NewCategoryService(store.CategoryRepository())
		expenseService = domain.NewExpenseService(store.ExpenseRepository(), store.CategoryRepository())
		checker = store

	case "postgres":
		if dbURL == "" {
			log.Fatalln("db-url must be set via -db-url or ET_DB_URL when using the postgres driver")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		store, err := postgres.Open(ctx, dbURL)
		if err != nil {
			log.Fatalln(err)
		}
		userService = domain.NewUserService(store.UserRepository())
		categoryService = domain.NewCategoryService(store.CategoryRepository())
		expenseService = domain.NewExpenseService(store.ExpenseRepository(), store.CategoryRepository())
		checker = store

	default:
		log.Fatalf("unsupported db driver: %q\n", dbDriver)
	}

	cookieAuth := cookie.New([]byte(hmacKey), 2*time.Minute, false)
	restHandler := rest.NewHandler(userService, expenseService, categoryService, cookieAuth, cookieAuth)
	healthHandler := healthroutes.NewHandler(checker)
	spaHandler := spa.NewHandler()

	ws := webserver.New(title, &webserver.Options{ListenAddress: addr})
	ws.HandlerMux().Handle("/api/", http.StripPrefix("/api", restHandler))
	ws.HandlerMux().Handle("/healthz/", http.StripPrefix("/healthz", healthHandler))
	ws.HandlerMux().Handle("/", spaHandler)

	ws.ListenAndServe()
}

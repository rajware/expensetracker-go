// Package rest provides an HTTP handler for the expense tracker API.
// Mount the handler returned by NewHandler on any path using a ServeMux.
package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rajware/expensetracker-go/internal/auth"
	"github.com/rajware/expensetracker-go/internal/domain"
)

type handler struct {
	users     domain.UserService
	expenses  domain.ExpenseService
	auth      auth.Authenticator
	transport TokenTransport
}

// NewHandler constructs and returns the REST API handler.
// Mount it on a path prefix using http.ServeMux:
//
//	mux.Handle("/api/", http.StripPrefix("/api", NewHandler(...)))
func NewHandler(
	users domain.UserService,
	expenses domain.ExpenseService,
	a auth.Authenticator,
	t TokenTransport,
) http.Handler {
	h := &handler{users: users, expenses: expenses, auth: a, transport: t}

	mux := http.NewServeMux()

	// Unauthenticated routes.
	mux.HandleFunc("POST /users/signup", h.handleSignUp)
	mux.HandleFunc("POST /users/signin", h.handleSignIn)

	// Authenticated routes.
	protected := func(next http.HandlerFunc) http.Handler {
		return authMiddleware(a, t, next)
	}
	mux.Handle("GET /users/me", protected(h.handleGetMe))
	mux.Handle("PATCH /users/me", protected(h.handleUpdateMe))
	mux.Handle("DELETE /users/me", protected(h.handleDeleteMe))
	mux.Handle("POST /users/me/keepalive", protected(h.handleKeepalive))
	mux.Handle("POST /users/me/signout", protected(h.handleSignOut))
	mux.Handle("POST /expenses", protected(h.handleAddExpense))
	mux.Handle("GET /expenses", protected(h.handleQueryExpenses))
	mux.Handle("GET /expenses/{id}", protected(h.handleGetExpense))
	mux.Handle("PATCH /expenses/{id}", protected(h.handleUpdateExpense))
	mux.Handle("DELETE /expenses/{id}", protected(h.handleDeleteExpense))

	return mux
}

// --- helpers ---

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeJSONWithStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrExpenseNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, domain.ErrExpenseNotOwned):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, domain.ErrInvalidCredentials):
		http.Error(w, err.Error(), http.StatusUnauthorized)
	case errors.Is(err, domain.ErrUsernameTaken),
		errors.Is(err, domain.ErrUsernameEmpty),
		errors.Is(err, domain.ErrPasswordTooShort),
		errors.Is(err, domain.ErrDescriptionEmpty),
		errors.Is(err, domain.ErrAmountNotPositive):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

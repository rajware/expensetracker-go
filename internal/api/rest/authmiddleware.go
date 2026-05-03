package rest

import (
	"context"
	"net/http"

	"github.com/rajware/expensetracker-go/internal/auth"
)

// contextKey is an unexported type for context keys in this package,
// preventing collisions with keys from other packages.
type contextKey int

const userIDKey contextKey = 0

// authMiddleware wraps a handler, requiring a valid auth token.
// On success, it injects the user ID into the request context.
// On failure, it responds with 401 and does not call next.
func authMiddleware(a auth.Authenticator, t TokenTransport, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := t.ExtractToken(r)
		userID, err := a.VerifyToken(token)
		if err != nil {
			http.Error(w, "unauthorised", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// userIDFromContext retrieves the user ID injected by authMiddleware.
func userIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

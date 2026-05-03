// Package auth defines shared authentication abstractions for use across
// API layers (REST, gRPC, etc.).
package auth

import "errors"

// ErrInvalidToken is returned when a token is missing, malformed, or expired.
var ErrInvalidToken = errors.New("invalid or expired token")

// Authenticator issues and verifies opaque auth tokens.
// Implementations are responsible for signing and expiry.
type Authenticator interface {
	// IssueToken creates a signed token encoding the given user ID.
	IssueToken(userID string) (string, error)

	// VerifyToken validates a token and returns the user ID it encodes.
	// Returns ErrInvalidToken if the token is missing, malformed, or expired.
	VerifyToken(token string) (string, error)
}

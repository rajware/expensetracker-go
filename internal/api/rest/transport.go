package rest

import "net/http"

// TokenTransport handles reading and writing auth tokens over HTTP.
// Implementations decide the delivery mechanism (cookie, header, etc.).
type TokenTransport interface {
	// SetToken writes the token to the response.
	SetToken(w http.ResponseWriter, token string)

	// ClearToken removes the token from the response.
	ClearToken(w http.ResponseWriter)

	// ExtractToken retrieves the raw token from the request.
	// Returns an empty string if no token is present.
	ExtractToken(r *http.Request) string
}

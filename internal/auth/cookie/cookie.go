// Package cookie provides cookie-based implementations of auth.Authenticator
// and rest.TokenTransport. Tokens are stateless HMAC-SHA256 signed strings
// stored in HTTP cookies.
package cookie

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rajware/expensetracker-go/internal/auth"
)

const cookieName = "auth_token"
const tokenSeparator = ":"

// Authenticator issues and verifies HMAC-SHA256 signed auth tokens.
// Token format: <b64(userID)>:<expiry_unix>:<signature>
//
// - userID is expected to be a UUID (safe to expose).
// - expiry is a Unix timestamp.
// - signature is HMAC-SHA256 over "<b64(userID)>:<expiry>".
type Authenticator struct {
	key    []byte
	ttl    time.Duration
	secure bool // whether to set Secure flag on cookies
}

// New constructs a cookie Authenticator with the given signing key,
// token TTL, and Secure flag (true for production, false for local dev).
func New(key []byte, ttl time.Duration, secure bool) *Authenticator {
	return &Authenticator{key: key, ttl: ttl, secure: secure}
}

// IssueToken creates a signed token encoding the given user ID.
// Tokens are short-lived; revocation is handled by TTL expiry + refresh.
func (a *Authenticator) IssueToken(userID string) (string, error) {
	expiry := time.Now().Add(a.ttl).Unix()
	payload := buildPayload(userID, expiry)
	sig := sign(a.key, payload)
	return payload + tokenSeparator + sig, nil
}

// VerifyToken checks format, signature, and expiry.
// Returns the user ID if valid, otherwise ErrInvalidToken.
func (a *Authenticator) VerifyToken(token string) (string, error) {
	userID, expiry, sig, ok := parseToken(token)
	if !ok {
		return "", auth.ErrInvalidToken
	}
	expected := sign(a.key, buildPayload(userID, expiry))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", auth.ErrInvalidToken
	}
	if time.Now().Unix() > expiry {
		return "", auth.ErrInvalidToken
	}
	return userID, nil
}

// SetToken writes the auth token as an HTTP cookie.
// Secure flag is configurable to support local dev vs. production.
func (a *Authenticator) SetToken(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(a.ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   a.secure,
	})
}

// ClearToken removes the auth cookie from the client.
func (a *Authenticator) ClearToken(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   a.secure,
	})
}

// ExtractToken reads the auth token from the request cookie.
// Returns an empty string if the cookie is absent.
func (a *Authenticator) ExtractToken(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// --- internal helpers ---

func buildPayload(userID string, expiry int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(userID)) +
		tokenSeparator + itoa(expiry)
}

func sign(key []byte, payload string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func parseToken(token string) (userID string, expiry int64, sig string, ok bool) {
	parts := strings.SplitN(token, tokenSeparator, 3)
	if len(parts) != 3 {
		return "", 0, "", false
	}
	userIDBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", 0, "", false
	}
	expiry, err = atoi(parts[1])
	if err != nil {
		return "", 0, "", false
	}
	return string(userIDBytes), expiry, parts[2], true
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

func atoi(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

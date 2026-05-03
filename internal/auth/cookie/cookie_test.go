package cookie_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rajware/expensetracker-go/internal/auth"
	"github.com/rajware/expensetracker-go/internal/auth/cookie"
)

func TestExpiredTokenRejected(t *testing.T) {
	a := cookie.New([]byte("test-key"), -1*time.Second, false)
	token, err := a.IssueToken("user-123")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	_, err = a.VerifyToken(token)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if err != auth.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestSetToken_WritesCookie(t *testing.T) {
	a := cookie.New([]byte("test-key"), time.Minute, false)
	token, err := a.IssueToken("user-123")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	w := httptest.NewRecorder()
	a.SetToken(w, token)

	resp := w.Result()
	c := findCookie(resp, "auth_token")
	if c == nil {
		t.Fatal("expected auth_token cookie to be set")
	}
	if c.Value != token {
		t.Errorf("expected cookie value %q, got %q", token, c.Value)
	}
	if !c.HttpOnly {
		t.Error("expected HttpOnly to be true")
	}
	if c.MaxAge <= 0 {
		t.Errorf("expected positive MaxAge, got %d", c.MaxAge)
	}
}

func TestClearToken_ClearsCookie(t *testing.T) {
	a := cookie.New([]byte("test-key"), time.Minute, false)

	w := httptest.NewRecorder()
	a.ClearToken(w)

	resp := w.Result()
	c := findCookie(resp, "auth_token")
	if c == nil {
		t.Fatal("expected auth_token cookie in response")
	}
	if c.MaxAge != -1 {
		t.Errorf("expected MaxAge -1, got %d", c.MaxAge)
	}
}

func findCookie(resp *http.Response, name string) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

package domaintest

import (
	"errors"
	"testing"

	"github.com/rajware/expensetracker-go/internal/domain"
)

// TestApp wraps all services.
type TestApp struct {
	UserService domain.UserService
}

// NewTestApp creates a TestApp with initialized
// services.
func NewTestApp(us domain.UserRepository) TestApp {
	return TestApp{
		UserService: domain.NewUserService(us),
	}
}

// RunSuite runs the full domain contract test suite against all
// services. The factory function is called once per sub-test to
// provide a clean, empty state. This prevents tests from
// interfering with each other.
//
// Usage in a storage plugin's test file:
//
//	func TestSQLite(t *testing.T) {
//	   domaintest.RunSuite(t, func() domaintest.TestApp {
//	       // open a fresh in-memory SQLite database and create repos
//	    // and Services
//	   return NewTestApp(
//	            // repos
//		      )
//	}
func RunSuite(t *testing.T, factory func() TestApp) {
	t.Helper()

	// UserService tests
	t.Run("SignUp_Success", func(t *testing.T) { testSignUpSuccess(t, factory) })
	t.Run("SignUp_EmptyUsername", func(t *testing.T) { testSignUpEmptyUsername(t, factory) })
	t.Run("SignUp_PasswordTooShort", func(t *testing.T) { testSignUpPasswordTooShort(t, factory) })
	t.Run("SignUp_DuplicateUsername", func(t *testing.T) { testSignUpDuplicateUsername(t, factory) })
	t.Run("SignIn_Success", func(t *testing.T) { testSignInSuccess(t, factory) })
	t.Run("SignIn_WrongPassword", func(t *testing.T) { testSignInWrongPassword(t, factory) })
	t.Run("SignIn_UnknownUsername", func(t *testing.T) { testSignInUnknownUsername(t, factory) })
	t.Run("GetByID_Success", func(t *testing.T) { testGetByIDSuccess(t, factory) })
	t.Run("GetByID_NotFound", func(t *testing.T) { testGetByIDNotFound(t, factory) })
	t.Run("Close_DeletesUser", func(t *testing.T) { testCloseDeletesUser(t, factory) })
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustSignUp(t *testing.T, us domain.UserService, username, password string) *domain.User {
	t.Helper()
	u, err := us.SignUp(t.Context(), username, "", password)
	if err != nil {
		t.Fatalf("mustSignUp: %v", err)
	}
	return u
}

// ---------------------------------------------------------------------------
// UserService tests
// ---------------------------------------------------------------------------

func testSignUpSuccess(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	u, err := us.SignUp(t.Context(), "alice", "Alice A", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", u.Username)
	}
	if u.ID == "" {
		t.Error("expected non-empty ID")
	}
	if u.PasswordHash == "" {
		t.Error("expected non-empty PasswordHash")
	}
	if u.PasswordHash == "password123" {
		t.Error("password must be hashed, not stored in plaintext")
	}
}

func testSignUpEmptyUsername(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	_, err := us.SignUp(t.Context(), "", "", "password123")
	if !errors.Is(err, domain.ErrUsernameEmpty) {
		t.Errorf("expected ErrUsernameEmpty, got %v", err)
	}
}

func testSignUpPasswordTooShort(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	_, err := us.SignUp(t.Context(), "alice", "", "short")
	if !errors.Is(err, domain.ErrPasswordTooShort) {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func testSignUpDuplicateUsername(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	mustSignUp(t, us, "alice", "password123")
	_, err := us.SignUp(t.Context(), "alice", "", "password123")
	if !errors.Is(err, domain.ErrUsernameTaken) {
		t.Errorf("expected ErrUsernameTaken, got %v", err)
	}
}

func testSignInSuccess(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	mustSignUp(t, us, "alice", "password123")
	u, err := us.SignIn(t.Context(), "alice", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", u.Username)
	}
}

func testSignInWrongPassword(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	mustSignUp(t, us, "alice", "password123")
	_, err := us.SignIn(t.Context(), "alice", "wrongpassword")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func testSignInUnknownUsername(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	_, err := us.SignIn(t.Context(), "nobody", "password123")
	// Must return ErrInvalidCredentials, not ErrUserNotFound —
	// to avoid revealing whether the username exists.
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func testGetByIDSuccess(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	created := mustSignUp(t, us, "alice", "password123")
	found, err := us.GetByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, found.ID)
	}
}

func testGetByIDNotFound(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	_, err := us.GetByID(t.Context(), "nonexistent-id")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func testCloseDeletesUser(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	alice := mustSignUp(t, us, "alice", "password123")

	if err := us.CloseAccountByID(t.Context(), alice.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := us.GetByID(t.Context(), alice.ID)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound after Close, got %v", err)
	}
}

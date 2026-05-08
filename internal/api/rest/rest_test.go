package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rajware/expensetracker-go/internal/api/rest"
	"github.com/rajware/expensetracker-go/internal/auth/cookie"
	"github.com/rajware/expensetracker-go/internal/domain"
	"github.com/rajware/expensetracker-go/internal/domain/domaintest"
)

// ---------------------------------------------------------------------------
// Test infrastructure
// ---------------------------------------------------------------------------

// testEnv holds a fully wired handler and the services it uses,
// so individual tests can interact with the domain directly to set up state.
type testEnv struct {
	handler    http.Handler
	users      domain.UserService
	expenses   domain.ExpenseService
	categories domain.CategoryService
	auth       *cookie.Authenticator
}

func newTestEnv() testEnv {
	app := domaintest.NewMockApp()
	a := cookie.New([]byte("test-key"), time.Minute, false)
	h := rest.NewHandler(app.UserService, app.ExpenseService, app.CategoryService, a, a)
	return testEnv{
		handler:    h,
		users:      app.UserService,
		expenses:   app.ExpenseService,
		categories: app.CategoryService,
		auth:       a,
	}
}

// do sends a request to the handler and returns the response.
func (e *testEnv) do(r *http.Request) *http.Response {
	w := httptest.NewRecorder()
	e.handler.ServeHTTP(w, r)
	return w.Result()
}

// jsonBody encodes v as JSON and returns it as a *bytes.Reader.
func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("jsonBody: %v", err)
	}
	return bytes.NewReader(b)
}

// mustSignUp creates a user and returns their auth cookie, ready to attach to requests.
func mustSignUp(t *testing.T, e *testEnv, username, password string) *http.Cookie {
	t.Helper()
	// Sign up via the domain directly to avoid coupling tests to the HTTP sign-up path.
	user, err := e.users.SignUp(t.Context(), username, "", password)
	if err != nil {
		t.Fatalf("mustSignUp: %v", err)
	}
	token, err := e.auth.IssueToken(user.ID)
	if err != nil {
		t.Fatalf("mustSignUp IssueToken: %v", err)
	}
	return &http.Cookie{Name: "auth_token", Value: token}
}

// withCookie attaches a cookie to a request.
func withCookie(r *http.Request, c *http.Cookie) *http.Request {
	r.AddCookie(c)
	return r
}

// findCookie returns the named cookie from the response, or nil if absent.
func findCookie(resp *http.Response, name string) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Unauthenticated endpoints
// ---------------------------------------------------------------------------

func TestSignUp_Success(t *testing.T) {
	e := newTestEnv()
	body := jsonBody(t, map[string]string{
		"username": "alice",
		"password": "password123",
	})
	r := httptest.NewRequest(http.MethodPost, "/users/signup", body)
	resp := e.do(r)
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestSignUp_BadRequest(t *testing.T) {
	e := newTestEnv()
	r := httptest.NewRequest(http.MethodPost, "/users/signup", bytes.NewReader([]byte("not json")))
	resp := e.do(r)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSignUp_UnprocessableEntity(t *testing.T) {
	e := newTestEnv()
	body := jsonBody(t, map[string]string{
		"username": "",
		"password": "password123",
	})
	r := httptest.NewRequest(http.MethodPost, "/users/signup", body)
	resp := e.do(r)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestSignIn_Success(t *testing.T) {
	e := newTestEnv()
	mustSignUp(t, &e, "alice", "password123")

	body := jsonBody(t, map[string]string{
		"username": "alice",
		"password": "password123",
	})
	r := httptest.NewRequest(http.MethodPost, "/users/signin", body)
	resp := e.do(r)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
	if findCookie(resp, "auth_token") == nil {
		t.Error("expected auth_token cookie to be set")
	}
}

func TestSignIn_InvalidCredentials(t *testing.T) {
	e := newTestEnv()
	mustSignUp(t, &e, "alice", "password123")

	body := jsonBody(t, map[string]string{
		"username": "alice",
		"password": "wrongpassword",
	})
	r := httptest.NewRequest(http.MethodPost, "/users/signin", body)
	resp := e.do(r)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Auth middleware
// ---------------------------------------------------------------------------

func TestProtectedRoute_NoCookie(t *testing.T) {
	e := newTestEnv()
	r := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	resp := e.do(r)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestProtectedRoute_InvalidCookie(t *testing.T) {
	e := newTestEnv()
	r := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	r.AddCookie(&http.Cookie{Name: "auth_token", Value: "invalid-token"})
	resp := e.do(r)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// User endpoints
// ---------------------------------------------------------------------------

func TestGetMe(t *testing.T) {
	e := newTestEnv()
	cookie := mustSignUp(t, &e, "alice", "password123")

	r := withCookie(httptest.NewRequest(http.MethodGet, "/users/me", nil), cookie)
	resp := e.do(r)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var view domain.UserView
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if view.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", view.Username)
	}
}

func TestUpdateMe(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	body := jsonBody(t, map[string]string{"display_name": "Alice A"})
	r := withCookie(httptest.NewRequest(http.MethodPatch, "/users/me", body), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var view domain.UserView
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if view.DisplayName != "Alice A" {
		t.Errorf("expected display name 'Alice A', got %q", view.DisplayName)
	}
}

func TestDeleteMe(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	r := withCookie(httptest.NewRequest(http.MethodDelete, "/users/me", nil), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}

	authCookie := findCookie(resp, "auth_token")
	if authCookie == nil {
		t.Fatal("expected auth_token cookie in response")
	}
	if authCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1 (cleared), got %d", authCookie.MaxAge)
	}
}

func TestKeepalive(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	r := withCookie(httptest.NewRequest(http.MethodPost, "/users/me/keepalive", nil), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
	if findCookie(resp, "auth_token") == nil {
		t.Error("expected auth_token cookie to be reissued")
	}
}

func TestSignOut(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	r := withCookie(httptest.NewRequest(http.MethodPost, "/users/me/signout", nil), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}

	authCookie := findCookie(resp, "auth_token")
	if authCookie == nil {
		t.Fatal("expected auth_token cookie in response")
	}
	if authCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1 (cleared), got %d", authCookie.MaxAge)
	}
}

// ---------------------------------------------------------------------------
// Expense endpoints
// ---------------------------------------------------------------------------

func TestAddExpense_Success(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	body := jsonBody(t, map[string]any{
		"occurred_at": time.Date(2024, time.March, 5, 0, 0, 0, 0, time.UTC),
		"description": "Coffee",
		"amount":      4.50,
	})
	r := withCookie(httptest.NewRequest(http.MethodPost, "/expenses", body), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestAddExpense_UnprocessableEntity(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	body := jsonBody(t, map[string]any{
		"occurred_at": time.Date(2024, time.March, 5, 0, 0, 0, 0, time.UTC),
		"description": "",
		"amount":      4.50,
	})
	r := withCookie(httptest.NewRequest(http.MethodPost, "/expenses", body), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestGetExpense_Success(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	// Add via HTTP to get a real expense ID.
	body := jsonBody(t, map[string]any{
		"occurred_at": time.Date(2024, time.March, 5, 0, 0, 0, 0, time.UTC),
		"description": "Coffee",
		"amount":      4.50,
	})
	addReq := withCookie(httptest.NewRequest(http.MethodPost, "/expenses", body), c)
	addResp := e.do(addReq)
	if addResp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d", addResp.StatusCode)
	}

	var expense domain.ExpenseView
	if err := json.NewDecoder(addResp.Body).Decode(&expense); err != nil {
		t.Fatalf("decode add response: %v", err)
	}

	r := withCookie(httptest.NewRequest(http.MethodGet, "/expenses/"+expense.ID, nil), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetExpense_NotFound(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	r := withCookie(httptest.NewRequest(http.MethodGet, "/expenses/nonexistent-id", nil), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetExpense_WrongOwner(t *testing.T) {
	e := newTestEnv()
	aliceCookie := mustSignUp(t, &e, "alice", "password123")
	bobCookie := mustSignUp(t, &e, "bob", "password123")

	// Alice adds an expense.
	body := jsonBody(t, map[string]any{
		"occurred_at": time.Date(2024, time.March, 5, 0, 0, 0, 0, time.UTC),
		"description": "Coffee",
		"amount":      4.50,
	})
	addReq := withCookie(httptest.NewRequest(http.MethodPost, "/expenses", body), aliceCookie)
	addResp := e.do(addReq)
	if addResp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d", addResp.StatusCode)
	}
	var expense domain.ExpenseView
	if err := json.NewDecoder(addResp.Body).Decode(&expense); err != nil {
		t.Fatalf("decode add response: %v", err)
	}

	// Bob tries to fetch it.
	r := withCookie(httptest.NewRequest(http.MethodGet, "/expenses/"+expense.ID, nil), bobCookie)
	resp := e.do(r)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestUpdateExpense_Success(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	// Add an expense.
	body := jsonBody(t, map[string]any{
		"occurred_at": time.Date(2024, time.March, 5, 0, 0, 0, 0, time.UTC),
		"description": "Coffee",
		"amount":      4.50,
	})
	addReq := withCookie(httptest.NewRequest(http.MethodPost, "/expenses", body), c)
	addResp := e.do(addReq)
	if addResp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d", addResp.StatusCode)
	}
	var expense domain.ExpenseView
	if err := json.NewDecoder(addResp.Body).Decode(&expense); err != nil {
		t.Fatalf("decode add response: %v", err)
	}

	// Update it.
	updateBody := jsonBody(t, map[string]any{
		"occurred_at": time.Date(2024, time.March, 6, 0, 0, 0, 0, time.UTC),
		"description": "Fancy Coffee",
		"amount":      6.00,
	})
	r := withCookie(httptest.NewRequest(http.MethodPatch, "/expenses/"+expense.ID, updateBody), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDeleteExpense_Success(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	// Add an expense.
	body := jsonBody(t, map[string]any{
		"occurred_at": time.Date(2024, time.March, 5, 0, 0, 0, 0, time.UTC),
		"description": "Coffee",
		"amount":      4.50,
	})
	addReq := withCookie(httptest.NewRequest(http.MethodPost, "/expenses", body), c)
	addResp := e.do(addReq)
	if addResp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d", addResp.StatusCode)
	}
	var expense domain.ExpenseView
	if err := json.NewDecoder(addResp.Body).Decode(&expense); err != nil {
		t.Fatalf("decode add response: %v", err)
	}

	// Delete it.
	r := withCookie(httptest.NewRequest(http.MethodDelete, "/expenses/"+expense.ID, nil), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestQueryExpenses_NoFilter(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	// Add two expenses.
	for _, desc := range []string{"Coffee", "Lunch"} {
		body := jsonBody(t, map[string]any{
			"occurred_at": time.Date(2024, time.March, 5, 0, 0, 0, 0, time.UTC),
			"description": desc,
			"amount":      10.0,
		})
		req := withCookie(httptest.NewRequest(http.MethodPost, "/expenses", body), c)
		if resp := e.do(req); resp.StatusCode != http.StatusCreated {
			t.Fatalf("setup: expected 201, got %d", resp.StatusCode)
		}
	}

	r := withCookie(httptest.NewRequest(http.MethodGet, "/expenses", nil), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result domain.ExpenseResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.TotalCount != 2 {
		t.Errorf("expected TotalCount 2, got %d", result.TotalCount)
	}
}

func TestQueryExpenses_BadQueryParam(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	r := withCookie(httptest.NewRequest(http.MethodGet, "/expenses?from=not-a-date", nil), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Category endpoints
// ---------------------------------------------------------------------------

func TestAddCategory_Success(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	body := jsonBody(t, map[string]string{"name": "Food"})
	r := withCookie(httptest.NewRequest(http.MethodPost, "/categories", body), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var view domain.CategoryView
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if view.Name != "Food" {
		t.Errorf("expected name 'Food', got %q", view.Name)
	}
}

func TestAddCategory_DuplicateName(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	// First one succeeds.
	e.do(withCookie(httptest.NewRequest(http.MethodPost, "/categories", jsonBody(t, map[string]string{"name": "Food"})), c))

	// Second one fails.
	body := jsonBody(t, map[string]string{"name": "food"}) // case-insensitive
	r := withCookie(httptest.NewRequest(http.MethodPost, "/categories", body), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestQueryCategories_All(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	e.do(withCookie(httptest.NewRequest(http.MethodPost, "/categories", jsonBody(t, map[string]string{"name": "Food"})), c))
	e.do(withCookie(httptest.NewRequest(http.MethodPost, "/categories", jsonBody(t, map[string]string{"name": "Transport"})), c))

	r := withCookie(httptest.NewRequest(http.MethodGet, "/categories", nil), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var views []domain.CategoryView
	if err := json.NewDecoder(resp.Body).Decode(&views); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// 2 we added + 1 system "Uncategorised"
	if len(views) != 3 {
		t.Errorf("expected 3 categories, got %d", len(views))
	}
}

func TestUpdateCategory_Success(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	// Add.
	resp1 := e.do(withCookie(httptest.NewRequest(http.MethodPost, "/categories", jsonBody(t, map[string]string{"name": "Food"})), c))
	var cat domain.CategoryView
	json.NewDecoder(resp1.Body).Decode(&cat)

	// Update.
	body := jsonBody(t, map[string]string{"name": "Groceries"})
	r := withCookie(httptest.NewRequest(http.MethodPatch, "/categories/"+cat.ID, body), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var updated domain.CategoryView
	json.NewDecoder(resp.Body).Decode(&updated)
	if updated.Name != "Groceries" {
		t.Errorf("expected name 'Groceries', got %q", updated.Name)
	}
}

func TestDeleteCategory_Success(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	// Add.
	resp1 := e.do(withCookie(httptest.NewRequest(http.MethodPost, "/categories", jsonBody(t, map[string]string{"name": "Food"})), c))
	var cat domain.CategoryView
	json.NewDecoder(resp1.Body).Decode(&cat)

	// Delete.
	r := withCookie(httptest.NewRequest(http.MethodDelete, "/categories/"+cat.ID, nil), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestDeleteCategory_Uncategorised(t *testing.T) {
	e := newTestEnv()
	c := mustSignUp(t, &e, "alice", "password123")

	r := withCookie(httptest.NewRequest(http.MethodDelete, "/categories/"+domain.UncategorisedCategoryID, nil), c)
	resp := e.do(r)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

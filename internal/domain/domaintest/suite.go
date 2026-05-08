package domaintest

import (
	"errors"
	"testing"
	"time"

	"github.com/rajware/expensetracker-go/internal/domain"
)

// TestApp wraps all services.
type TestApp struct {
	UserService     domain.UserService
	ExpenseService  domain.ExpenseService
	CategoryService domain.CategoryService
}

// RunSuite runs the full domain contract test suite against all
// services. The factory function is called once per sub-test to
// provide a clean, empty state. This prevents tests from
// interfering with each other.
//
// Usage in a storage plugin's test file:
//
//	func TestSQLite(t *testing.T) {
//	    domaintest.RunSuite(
//	     t,
//	        func() domaintest.TestApp {
//	            // open a fresh in-memory SQLite database and create repos
//	            // and Services
//	            return NewTestApp(
//	                 // repos
//	 	       )
//	        }
//	    )
//	}
func RunSuite(t *testing.T, factory func() TestApp) {
	t.Helper()

	// UserService tests
	t.Run("SignUp_Success", func(t *testing.T) { testSignUpSuccess(t, factory) })
	t.Run("SignUp_EmptyUsername", func(t *testing.T) { testSignUpEmptyUsername(t, factory) })
	t.Run("SignUp_PasswordTooShort", func(t *testing.T) { testSignUpPasswordTooShort(t, factory) })
	t.Run("SignUp_DuplicateUsername", func(t *testing.T) { testSignUpDuplicateUsername(t, factory) })
	t.Run("SignUp_ReservedUsername", func(t *testing.T) { testSignUpReservedUsername(t, factory) })
	t.Run("SignIn_Success", func(t *testing.T) { testSignInSuccess(t, factory) })
	t.Run("SignIn_WrongPassword", func(t *testing.T) { testSignInWrongPassword(t, factory) })
	t.Run("SignIn_UnknownUsername", func(t *testing.T) { testSignInUnknownUsername(t, factory) })
	t.Run("QueryByID_Success", func(t *testing.T) { testQueryByIDSuccess(t, factory) })
	t.Run("QueryByID_NotFound", func(t *testing.T) { testQueryByIDNotFound(t, factory) })
	t.Run("Close_DeletesUser", func(t *testing.T) { testCloseDeletesUser(t, factory) })

	// ExpenseService tests
	t.Run("Add_Success", func(t *testing.T) { testAddSuccess(t, factory) })
	t.Run("Add_EmptyDescription", func(t *testing.T) { testAddEmptyDescription(t, factory) })
	t.Run("Add_NonPositiveAmount", func(t *testing.T) { testAddNonPositiveAmount(t, factory) })
	t.Run("Update_Success", func(t *testing.T) { testUpdateSuccess(t, factory) })
	t.Run("Update_WrongOwner", func(t *testing.T) { testUpdateWrongOwner(t, factory) })
	t.Run("Delete_Success", func(t *testing.T) { testDeleteSuccess(t, factory) })
	t.Run("Delete_WrongOwner", func(t *testing.T) { testDeleteWrongOwner(t, factory) })

	// User/Expense cascade tests
	t.Run("Close_DeletesUserAndExpenses", func(t *testing.T) { testCloseDeletesUserAndExpenses(t, factory) })

	// Query tests
	t.Run("Query_NoFilter", func(t *testing.T) { testQueryNoFilter(t, factory) })
	t.Run("Query_IsolatesByUser", func(t *testing.T) { testQueryIsolatesByUser(t, factory) })
	t.Run("Query_DateFilter", func(t *testing.T) { testQueryDateFilter(t, factory) })
	t.Run("Query_SortByAmount", func(t *testing.T) { testQuerySortByAmount(t, factory) })
	t.Run("Query_Pagination", func(t *testing.T) { testQueryPagination(t, factory) })
	t.Run("QueryByID_Success", func(t *testing.T) { testQueryByIDExpenseSuccess(t, factory) })
	t.Run("QueryByID_NotFound", func(t *testing.T) { testQueryByIDExpenseNotFound(t, factory) })
	t.Run("QueryByID_WrongOwner", func(t *testing.T) { testQueryByIDExpenseWrongOwner(t, factory) })

	RunCategorySuite(t, factory)
}

// RunCategorySuite runs the CategoryService contract tests.
// It is called by RunSuite but can also be called independently.
func RunCategorySuite(t *testing.T, factory func() TestApp) {
	t.Helper()

	t.Run("Category_Add_Success", func(t *testing.T) { testCategoryAddSuccess(t, factory) })
	t.Run("Category_Add_EmptyName", func(t *testing.T) { testCategoryAddEmptyName(t, factory) })
	t.Run("Category_Add_DuplicateName", func(t *testing.T) { testCategoryAddDuplicateName(t, factory) })
	t.Run("Category_Add_CaseInsensitiveDuplicate", func(t *testing.T) { testCategoryAddCaseInsensitiveDuplicate(t, factory) })
	t.Run("Category_Update_Success", func(t *testing.T) { testCategoryUpdateSuccess(t, factory) })
	t.Run("Category_Update_WrongOwner", func(t *testing.T) { testCategoryUpdateWrongOwner(t, factory) })
	t.Run("Category_Update_NotFound", func(t *testing.T) { testCategoryUpdateNotFound(t, factory) })
	t.Run("Category_Delete_Success", func(t *testing.T) { testCategoryDeleteSuccess(t, factory) })
	t.Run("Category_Delete_WrongOwner", func(t *testing.T) { testCategoryDeleteWrongOwner(t, factory) })
	t.Run("Category_Delete_Uncategorised", func(t *testing.T) { testCategoryDeleteUncategorised(t, factory) })
	t.Run("Category_Query_All", func(t *testing.T) { testCategoryQueryAll(t, factory) })
	t.Run("Category_Query_Prefix", func(t *testing.T) { testCategoryQueryPrefix(t, factory) })
	t.Run("Category_Delete_ReclassifiesExpenses", func(t *testing.T) { testCategoryDeleteReclassifiesExpenses(t, factory) })
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

func mustAdd(t *testing.T, es domain.ExpenseService, ownerID string, occurredAt time.Time, description string, amount float64) *domain.ExpenseView {
	t.Helper()
	e, err := es.Add(t.Context(), ownerID, occurredAt, description, amount, "")
	if err != nil {
		t.Fatalf("mustAdd: %v", err)
	}
	return e
}

func mustAddCategory(t *testing.T, cs domain.CategoryService, ownerID, name string) domain.CategoryView {
	t.Helper()
	v, err := cs.Add(t.Context(), ownerID, name)
	if err != nil {
		t.Fatalf("mustAddCategory: %v", err)
	}
	return v
}

// makeDate constructs a UTC timestamp for a given date, for use in tests.
func makeDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
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

func testSignUpReservedUsername(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	_, err := us.SignUp(t.Context(), "system", "", "password123")
	if !errors.Is(err, domain.ErrUsernameReserved) {
		t.Errorf("expected ErrUsernameReserved, got %v", err)
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

func testQueryByIDSuccess(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	created := mustSignUp(t, us, "alice", "password123")
	found, err := us.QueryByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, found.ID)
	}
}

func testQueryByIDNotFound(t *testing.T, factory func() TestApp) {
	us := factory().UserService
	_, err := us.QueryByID(t.Context(), "nonexistent-id")
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

	_, err := us.QueryByID(t.Context(), alice.ID)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound after Close, got %v", err)
	}
}

func testCloseDeletesUserAndExpenses(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	mustAdd(t, es, alice.ID, makeDate(2024, time.January, 10), "Lunch", 12.50)
	mustAdd(t, es, alice.ID, makeDate(2024, time.January, 11), "Dinner", 30.00)

	if err := us.CloseAccountByID(t.Context(), alice.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := us.QueryByID(t.Context(), alice.ID)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound after Close, got %v", err)
	}

	result, err := es.Query(t.Context(), alice.ID, domain.ExpenseQuery{})
	if err != nil {
		t.Fatalf("unexpected error querying after Close: %v", err)
	}
	if result.TotalCount != 0 {
		t.Errorf("expected 0 expenses after Close, got %d", result.TotalCount)
	}
}

// ---------------------------------------------------------------------------
// ExpenseService tests
// ---------------------------------------------------------------------------

func testAddSuccess(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	e, err := es.Add(t.Context(), alice.ID, makeDate(2024, time.March, 5), "Coffee", 4.50, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.ID == "" {
		t.Error("expected non-empty expense ID")
	}
	if e.OwnerID != alice.ID {
		t.Errorf("expected OwnerID %q, got %q", alice.ID, e.OwnerID)
	}
	if e.CategoryID != domain.UncategorisedCategoryID {
		t.Errorf("expected default CategoryID %q, got %q", domain.UncategorisedCategoryID, e.CategoryID)
	}
}

func testAddEmptyDescription(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	_, err := es.Add(t.Context(), alice.ID, makeDate(2024, time.March, 5), "", 4.50, "")
	if !errors.Is(err, domain.ErrDescriptionEmpty) {
		t.Errorf("expected ErrDescriptionEmpty, got %v", err)
	}
}

func testAddNonPositiveAmount(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")

	_, err := es.Add(t.Context(), alice.ID, makeDate(2024, time.March, 5), "Coffee", 0, "")
	if !errors.Is(err, domain.ErrAmountNotPositive) {
		t.Errorf("expected ErrAmountNotPositive for zero amount, got %v", err)
	}

	_, err = es.Add(t.Context(), alice.ID, makeDate(2024, time.March, 5), "Coffee", -5, "")
	if !errors.Is(err, domain.ErrAmountNotPositive) {
		t.Errorf("expected ErrAmountNotPositive for negative amount, got %v", err)
	}
}

func testUpdateSuccess(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	e := mustAdd(t, es, alice.ID, makeDate(2024, time.March, 5), "Coffee", 4.50)

	updated, err := es.Update(t.Context(), alice.ID, e.ID, "Fancy Coffee", makeDate(2024, time.March, 6), 6.00, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Description != "Fancy Coffee" {
		t.Errorf("expected description 'Fancy Coffee', got %q", updated.Description)
	}
	if updated.Amount != 6.00 {
		t.Errorf("expected amount 6.00, got %f", updated.Amount)
	}
}

func testUpdateWrongOwner(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	bob := mustSignUp(t, us, "bob", "password123")
	e := mustAdd(t, es, alice.ID, makeDate(2024, time.March, 5), "Coffee", 4.50)

	_, err := es.Update(t.Context(), bob.ID, e.ID, "Stolen Coffee", makeDate(2024, time.March, 5), 4.50, "")
	if !errors.Is(err, domain.ErrExpenseNotOwned) {
		t.Errorf("expected ErrExpenseNotOwned, got %v", err)
	}
}

func testDeleteSuccess(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	e := mustAdd(t, es, alice.ID, makeDate(2024, time.March, 5), "Coffee", 4.50)

	if err := es.Delete(t.Context(), alice.ID, e.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := es.Query(t.Context(), alice.ID, domain.ExpenseQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalCount != 0 {
		t.Errorf("expected 0 expenses after delete, got %d", result.TotalCount)
	}
}

func testDeleteWrongOwner(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	bob := mustSignUp(t, us, "bob", "password123")
	e := mustAdd(t, es, alice.ID, makeDate(2024, time.March, 5), "Coffee", 4.50)

	err := es.Delete(t.Context(), bob.ID, e.ID)
	if !errors.Is(err, domain.ErrExpenseNotOwned) {
		t.Errorf("expected ErrExpenseNotOwned, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Query tests
// ---------------------------------------------------------------------------

func testQueryNoFilter(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	mustAdd(t, es, alice.ID, makeDate(2024, time.January, 1), "A", 10)
	mustAdd(t, es, alice.ID, makeDate(2024, time.January, 2), "B", 20)
	mustAdd(t, es, alice.ID, makeDate(2024, time.January, 3), "C", 30)

	result, err := es.Query(t.Context(), alice.ID, domain.ExpenseQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalCount != 3 {
		t.Errorf("expected TotalCount 3, got %d", result.TotalCount)
	}
}

func testQueryIsolatesByUser(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	bob := mustSignUp(t, us, "bob", "password123")
	mustAdd(t, es, alice.ID, makeDate(2024, time.January, 1), "Alice's lunch", 10)
	mustAdd(t, es, bob.ID, makeDate(2024, time.January, 1), "Bob's lunch", 10)

	result, _ := es.Query(t.Context(), alice.ID, domain.ExpenseQuery{})
	if result.TotalCount != 1 {
		t.Errorf("expected 1 expense for alice, got %d", result.TotalCount)
	}
}

func testQueryDateFilter(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	mustAdd(t, es, alice.ID, makeDate(2024, time.January, 1), "Jan", 10)
	mustAdd(t, es, alice.ID, makeDate(2024, time.February, 1), "Feb", 20)
	mustAdd(t, es, alice.ID, makeDate(2024, time.March, 1), "Mar", 30)

	from := makeDate(2024, time.February, 1)
	to := makeDate(2024, time.February, 28)
	result, err := es.Query(t.Context(), alice.ID, domain.ExpenseQuery{From: &from, To: &to})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalCount != 1 {
		t.Errorf("expected 1 expense in February, got %d", result.TotalCount)
	}
}

func testQuerySortByAmount(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	mustAdd(t, es, alice.ID, makeDate(2024, time.January, 1), "Mid", 50)
	mustAdd(t, es, alice.ID, makeDate(2024, time.January, 2), "Low", 10)
	mustAdd(t, es, alice.ID, makeDate(2024, time.January, 3), "High", 100)

	result, _ := es.Query(t.Context(), alice.ID, domain.ExpenseQuery{SortBy: domain.SortByAmount})
	for i := 1; i < len(result.Expenses); i++ {
		if result.Expenses[i].Amount < result.Expenses[i-1].Amount {
			t.Errorf("expected ascending sort by amount, got out of order at index %d", i)
			break
		}
	}
}

func testQueryPagination(t *testing.T, factory func() TestApp) {
	app := factory()
	us := app.UserService
	es := app.ExpenseService
	alice := mustSignUp(t, us, "alice", "password123")
	for i := 1; i <= 5; i++ {
		mustAdd(t, es, alice.ID, makeDate(2024, time.January, i), "Expense", float64(i))
	}

	result, err := es.Query(t.Context(), alice.ID, domain.ExpenseQuery{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalCount != 5 {
		t.Errorf("expected TotalCount 5, got %d", result.TotalCount)
	}
	if len(result.Expenses) != 2 {
		t.Errorf("expected 2 expenses on page 1, got %d", len(result.Expenses))
	}

	result, _ = es.Query(t.Context(), alice.ID, domain.ExpenseQuery{Page: 3, PageSize: 2})
	if len(result.Expenses) != 1 {
		t.Errorf("expected 1 expense on last page, got %d", len(result.Expenses))
	}
}

func testQueryByIDExpenseSuccess(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	e := mustAdd(t, app.ExpenseService, alice.ID, makeDate(2024, time.March, 5), "Coffee", 4.50)

	view, err := app.ExpenseService.QueryByID(t.Context(), alice.ID, e.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.ID != e.ID {
		t.Errorf("expected ID %q, got %q", e.ID, view.ID)
	}
}

func testQueryByIDExpenseNotFound(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")

	_, err := app.ExpenseService.QueryByID(t.Context(), alice.ID, "nonexistent-id")
	if !errors.Is(err, domain.ErrExpenseNotFound) {
		t.Errorf("expected ErrExpenseNotFound, got %v", err)
	}
}

func testQueryByIDExpenseWrongOwner(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	bob := mustSignUp(t, app.UserService, "bob", "password123")
	e := mustAdd(t, app.ExpenseService, alice.ID, makeDate(2024, time.March, 5), "Coffee", 4.50)

	_, err := app.ExpenseService.QueryByID(t.Context(), bob.ID, e.ID)
	if !errors.Is(err, domain.ErrExpenseNotOwned) {
		t.Errorf("expected ErrExpenseNotOwned, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// CategoryService tests
// ---------------------------------------------------------------------------

func testCategoryAddSuccess(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	v := mustAddCategory(t, app.CategoryService, alice.ID, "Food")
	if v.ID == "" {
		t.Error("expected non-empty category ID")
	}
	if v.Name != "Food" {
		t.Errorf("expected name 'Food', got %q", v.Name)
	}
	if v.OwnerID != alice.ID {
		t.Errorf("expected OwnerID %q, got %q", alice.ID, v.OwnerID)
	}
}

func testCategoryAddEmptyName(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	_, err := app.CategoryService.Add(t.Context(), alice.ID, "")
	if !errors.Is(err, domain.ErrCategoryNameEmpty) {
		t.Errorf("expected ErrCategoryNameEmpty, got %v", err)
	}
}

func testCategoryAddDuplicateName(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	mustAddCategory(t, app.CategoryService, alice.ID, "Food")
	_, err := app.CategoryService.Add(t.Context(), alice.ID, "Food")
	if !errors.Is(err, domain.ErrCategoryNameTaken) {
		t.Errorf("expected ErrCategoryNameTaken, got %v", err)
	}
}

func testCategoryAddCaseInsensitiveDuplicate(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	mustAddCategory(t, app.CategoryService, alice.ID, "Food")
	_, err := app.CategoryService.Add(t.Context(), alice.ID, "food")
	if !errors.Is(err, domain.ErrCategoryNameTaken) {
		t.Errorf("expected ErrCategoryNameTaken for case-insensitive duplicate, got %v", err)
	}
}

func testCategoryUpdateSuccess(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	v := mustAddCategory(t, app.CategoryService, alice.ID, "Food")

	updated, err := app.CategoryService.Update(t.Context(), alice.ID, v.ID, "Groceries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Groceries" {
		t.Errorf("expected name 'Groceries', got %q", updated.Name)
	}
}

func testCategoryUpdateWrongOwner(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	bob := mustSignUp(t, app.UserService, "bob", "password123")
	v := mustAddCategory(t, app.CategoryService, alice.ID, "Food")

	_, err := app.CategoryService.Update(t.Context(), bob.ID, v.ID, "Stolen")
	if !errors.Is(err, domain.ErrCategoryNotOwned) {
		t.Errorf("expected ErrCategoryNotOwned, got %v", err)
	}
}

func testCategoryUpdateNotFound(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")

	_, err := app.CategoryService.Update(t.Context(), alice.ID, "nonexistent-id", "Whatever")
	if !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Errorf("expected ErrCategoryNotFound, got %v", err)
	}
}

func testCategoryDeleteSuccess(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	v := mustAddCategory(t, app.CategoryService, alice.ID, "Food")

	if err := app.CategoryService.Delete(t.Context(), alice.ID, v.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := app.CategoryService.Update(t.Context(), alice.ID, v.ID, "Whatever")
	if !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Errorf("expected ErrCategoryNotFound after delete, got %v", err)
	}
}

func testCategoryDeleteWrongOwner(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	bob := mustSignUp(t, app.UserService, "bob", "password123")
	v := mustAddCategory(t, app.CategoryService, alice.ID, "Food")

	err := app.CategoryService.Delete(t.Context(), bob.ID, v.ID)
	if !errors.Is(err, domain.ErrCategoryNotOwned) {
		t.Errorf("expected ErrCategoryNotOwned, got %v", err)
	}
}

func testCategoryDeleteUncategorised(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")

	err := app.CategoryService.Delete(t.Context(), alice.ID, domain.UncategorisedCategoryID)
	if !errors.Is(err, domain.ErrCategoryNotDeletable) {
		t.Errorf("expected ErrCategoryNotDeletable, got %v", err)
	}
}

func testCategoryQueryAll(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	mustAddCategory(t, app.CategoryService, alice.ID, "Food")
	mustAddCategory(t, app.CategoryService, alice.ID, "Transport")

	// Blank prefix returns all categories (including the seeded Uncategorised).
	views, err := app.CategoryService.Query(t.Context(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(views) < 2 {
		t.Errorf("expected at least 2 categories, got %d", len(views))
	}
}

func testCategoryQueryPrefix(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	mustAddCategory(t, app.CategoryService, alice.ID, "Food")
	mustAddCategory(t, app.CategoryService, alice.ID, "Fuel")
	mustAddCategory(t, app.CategoryService, alice.ID, "Transport")

	views, err := app.CategoryService.Query(t.Context(), "F")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(views) != 2 {
		t.Errorf("expected 2 categories with prefix 'F', got %d", len(views))
	}
}

func testCategoryQueryPrefixCaseInsensitive(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	mustAddCategory(t, app.CategoryService, alice.ID, "Food")

	views, err := app.CategoryService.Query(t.Context(), "fo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(views) != 1 {
		t.Errorf("expected 1 category with prefix 'fo', got %d", len(views))
	}
}

func testCategoryDeleteReclassifiesExpenses(t *testing.T, factory func() TestApp) {
	app := factory()
	alice := mustSignUp(t, app.UserService, "alice", "password123")
	food := mustAddCategory(t, app.CategoryService, alice.ID, "Food")

	// Add an expense in the Food category.
	e, err := app.ExpenseService.Add(t.Context(), alice.ID, makeDate(2024, time.January, 1), "Lunch", 12.50, food.ID)
	if err != nil {
		t.Fatalf("unexpected error adding expense: %v", err)
	}

	// Delete the Food category.
	if err := app.CategoryService.Delete(t.Context(), alice.ID, food.ID); err != nil {
		t.Fatalf("unexpected error deleting category: %v", err)
	}

	// The expense should now be in Uncategorised.
	view, err := app.ExpenseService.QueryByID(t.Context(), alice.ID, e.ID)
	if err != nil {
		t.Fatalf("unexpected error querying expense: %v", err)
	}
	if view.CategoryID != domain.UncategorisedCategoryID {
		t.Errorf("expected CategoryID %q after reclassification, got %q", domain.UncategorisedCategoryID, view.CategoryID)
	}
}

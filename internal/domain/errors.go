package domain

import "errors"

// Sentinel errors let callers use errors.Is() for precise error handling
// without importing infrastructure packages.
var (
	// User errors
	ErrUserNotFound       = errors.New("user not found")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUsernameEmpty      = errors.New("username must not be empty")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrUsernameReserved   = errors.New("username is reserved")

	// Expense errors
	ErrExpenseNotFound   = errors.New("expense not found")
	ErrExpenseNotOwned   = errors.New("expense does not belong to this user")
	ErrDescriptionEmpty  = errors.New("description must not be empty")
	ErrAmountNotPositive = errors.New("amount must be greater than zero")

	// Category errors
	ErrCategoryNotFound     = errors.New("category not found")
	ErrCategoryNameTaken    = errors.New("category name already taken")
	ErrCategoryNameEmpty    = errors.New("category name must not be empty")
	ErrCategoryNotOwned     = errors.New("category does not belong to this user")
	ErrCategoryNotDeletable = errors.New("category cannot be deleted")
)

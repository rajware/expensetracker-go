package domain

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User is the person who owns expenses.
// PasswordHash stores a bcrypt hash — never the plaintext password.
type User struct {
	ID           string
	Username     string
	DisplayName  string
	PasswordHash string
}

// UserView is the "display" representation of an User.
// It does not include PasswordHash for security.
type UserView struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

func NewUserView(user *User) UserView {
	return UserView{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
	}
}

// UserRepository is implemented by storage plugins
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	UpdatePassword(ctx context.Context, id, hash string) error
	// Delete needs to ensure that all data owned by this user is also
	// deleted.
	Delete(ctx context.Context, id string) error
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
}

// UserService contains business logic for User operations.
// It depends only on the UserRepository interface, not on any concrete storage.
type UserService struct {
	users UserRepository
}

// NewUserService constructs a UserService with the given repository.
func NewUserService(users UserRepository) UserService {
	return UserService{users: users}
}

// SignUp creates a new user account. Returns the created User on success.
func (s UserService) SignUp(ctx context.Context, username, displayName, password string) (*User, error) {
	if username == "" {
		return nil, ErrUsernameEmpty
	}
	if username == "system" {
		return nil, ErrUsernameReserved
	}
	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           uuid.NewString(),
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: string(hash),
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err // includes ErrUsernameTaken from the repository
	}

	return user, nil
}

// SignIn verifies credentials and returns the matching User.
// Returns ErrInvalidCredentials if the username or password is wrong.
func (s UserService) SignIn(ctx context.Context, username, password string) (*User, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		// Don't reveal whether the username exists.
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// QueryByID returns a view of the User with the given ID.
func (s UserService) QueryByID(ctx context.Context, id string) (*UserView, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	result := NewUserView(user)
	return &result, nil
}

// UpdateDisplayName updates the display name of the User with the
// given ID. It can be used to remove a display name by passing an
// empty string.
func (s UserService) UpdateDisplayName(ctx context.Context, id string, newDisplayName string) (*UserView, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.DisplayName = newDisplayName
	err = s.users.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	result := NewUserView(user)
	return &result, nil
}

// CloseAccountByID permanently deletes the user's account.
//
// It delegates to UserRepository.Delete, which is contractually required to
// also remove all expenses owned by this user. See the UserRepository.Delete
// doc comment for how storage backends are expected to fulfil that contract.
func (s UserService) CloseAccountByID(ctx context.Context, id string) error {
	return s.users.Delete(ctx, id)
}

// ChangePassword updates the password for the user with the given ID.
// It verifies the current password before applying the change.
func (s UserService) ChangePassword(ctx context.Context, id, oldPassword, newPassword string) error {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.users.UpdatePassword(ctx, id, string(hash))
}

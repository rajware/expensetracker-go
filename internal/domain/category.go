package domain

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// Category is a label that groups expenses. Categories are owned by a user,
// except for system categories (owned by SystemUserID) which are shared
// across all users.
type Category struct {
	ID      string
	Name    string
	OwnerID string // User.ID of the owner
}

// CategoryView is the "display" representation of a Category.
// Defined separately from Category to allow the two to diverge in the future.
type CategoryView struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnerID string `json:"owner_id"`
}

func NewCategoryView(c Category) CategoryView {
	return CategoryView{
		ID:      c.ID,
		Name:    c.Name,
		OwnerID: c.OwnerID,
	}
}

// CategoryRepository is the storage contract for Category entities.
type CategoryRepository interface {
	// Create persists a new Category.
	Create(ctx context.Context, category *Category) error

	// GetByID returns the Category with the given ID, or ErrCategoryNotFound.
	GetByID(ctx context.Context, id string) (*Category, error)

	// GetByName returns the Category with the given name (case-insensitive),
	// or ErrCategoryNotFound.
	GetByName(ctx context.Context, name string) (*Category, error)

	// Update persists changes to an existing Category.
	Update(ctx context.Context, category *Category) error

	// Delete removes the Category with the given ID.
	// Contract: all expenses in this category must be reclassified to
	// UncategorisedCategoryID before or during deletion.
	Delete(ctx context.Context, id string) error

	// Query returns all categories whose names start with prefix
	// (case-insensitive). An empty prefix returns all categories.
	Query(ctx context.Context, prefix string) ([]*Category, error)
}

// CategoryService contains business logic for Category operations.
type CategoryService struct {
	categories CategoryRepository
}

// NewCategoryService constructs a CategoryService with the given repository.
func NewCategoryService(categories CategoryRepository) CategoryService {
	return CategoryService{categories: categories}
}

// Add creates a new category owned by ownerID.
func (s CategoryService) Add(ctx context.Context, ownerID, name string) (CategoryView, error) {
	if strings.TrimSpace(name) == "" {
		return CategoryView{}, ErrCategoryNameEmpty
	}

	category := &Category{
		ID:      uuid.NewString(),
		Name:    name,
		OwnerID: ownerID,
	}

	if err := s.categories.Create(ctx, category); err != nil {
		return CategoryView{}, err
	}

	return NewCategoryView(*category), nil
}

// Query returns all categories whose names start with prefix.
// An empty prefix returns all categories.
func (s CategoryService) Query(ctx context.Context, prefix string) ([]CategoryView, error) {
	cats, err := s.categories.Query(ctx, prefix)
	if err != nil {
		return nil, err
	}

	views := make([]CategoryView, len(cats))
	for i, c := range cats {
		views[i] = NewCategoryView(*c)
	}
	return views, nil
}

// Update changes the name of a category. Only the owner may update it.
func (s CategoryService) Update(ctx context.Context, ownerID, id, name string) (CategoryView, error) {
	if strings.TrimSpace(name) == "" {
		return CategoryView{}, ErrCategoryNameEmpty
	}

	cat, err := s.categories.GetByID(ctx, id)
	if err != nil {
		return CategoryView{}, err
	}
	if cat.OwnerID != ownerID {
		return CategoryView{}, ErrCategoryNotOwned
	}

	cat.Name = name
	if err := s.categories.Update(ctx, cat); err != nil {
		return CategoryView{}, err
	}

	return NewCategoryView(*cat), nil
}

// Delete removes a category. Only the owner may delete it.
// The Uncategorised category cannot be deleted.
func (s CategoryService) Delete(ctx context.Context, ownerID, id string) error {
	if id == UncategorisedCategoryID {
		return ErrCategoryNotDeletable
	}

	cat, err := s.categories.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if cat.OwnerID != ownerID {
		return ErrCategoryNotOwned
	}

	return s.categories.Delete(ctx, id)
}

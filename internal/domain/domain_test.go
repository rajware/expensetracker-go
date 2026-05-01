package domain_test

import (
	"testing"

	"github.com/rajware/expensetracker-go/internal/domain/domaintest"
)

func TestUserService(t *testing.T) {
	domaintest.RunSuite(t, func() domaintest.TestApp {
		return domaintest.NewTestApp(
			domaintest.NewMockUserRepository(),
		)
	})
}

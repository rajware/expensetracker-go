package domain_test

import (
	"testing"

	"github.com/rajware/expensetracker-go/internal/domain/domaintest"
)

func TestAllServices(t *testing.T) {
	domaintest.RunSuite(t, func() domaintest.TestApp {
		return domaintest.NewMockApp()
	})
}

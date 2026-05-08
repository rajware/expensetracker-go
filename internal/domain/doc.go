// Package domain contains the core business entities and logic for the expense tracker.
package domain

const SchemaVersion = 2

// SystemUserID is the fixed ID of the built-in system user.
// The system user owns system-managed categories such as Uncategorised.
const SystemUserID = "00000000-0000-0000-0000-000000000001"

// UncategorisedCategoryID is the fixed ID of the built-in "Uncategorised" category.
// It is owned by the system user and cannot be deleted.
const UncategorisedCategoryID = "00000000-0000-0000-0000-000000000002"

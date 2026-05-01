package sqlite

import (
	"errors"

	"modernc.org/sqlite"
	lib "modernc.org/sqlite/lib"
)

// isUniqueConstraintError reports whether err is a SQLite UNIQUE constraint
// violation, identified by error code SQLITE_CONSTRAINT_UNIQUE (2067).
func isUniqueConstraintError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	return sqliteErr.Code() == lib.SQLITE_CONSTRAINT_UNIQUE
}

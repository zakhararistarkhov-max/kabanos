package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// Sentinel errors returned by repositories so services can react without
// importing pgx-specific types.
var (
	ErrNotFound = errors.New("resource not found")
	ErrConflict = errors.New("resource already exists")
)

// IsUniqueViolation reports whether err is a Postgres unique-constraint error
// (SQLSTATE 23505). Repositories use it to translate DB conflicts into
// ErrConflict.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

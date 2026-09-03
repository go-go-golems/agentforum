package store

import (
	"context"
	"database/sql"
)

// dbtx is the minimal interface satisfied by both *sql.DB and *sql.Tx, so the
// atomic create methods can run their many statements inside one transaction
// while reusing the same helpers as single-statement methods.
type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

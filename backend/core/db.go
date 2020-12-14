package core

import (
	"context"
	"database/sql"
)

type (
	DBExecutor interface {
		Exec(query string, args ...interface{}) (sql.Result, error)
		ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
		Query(query string, args ...interface{}) (*sql.Rows, error)
		QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
		QueryRow(query string, args ...interface{}) *sql.Row
		QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	}

	DB interface {
		DBExecutor

		Begin() (*sql.Tx, error)
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	}

	DBTransactor interface {
		DBExecutor

		Commit() error
		Rollback() error
	}
)

type DBOrdering struct {
	Field     string
	Ascending bool
}

func (ord DBOrdering) String() string {
	direction := "DESC"
	if ord.Ascending {
		direction = "ASC"
	}
	return ord.Field + " " + direction
}

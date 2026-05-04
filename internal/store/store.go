package store

import (
	"database/sql"
	"errors"
)

type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

var ErrNotFound = errors.New("not found")

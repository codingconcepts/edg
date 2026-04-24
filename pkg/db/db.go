package db

import (
	"context"
	"io"
)

type RowIterator interface {
	Columns() ([]string, error)
	ColumnTypes() ([]ColumnType, error)
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

type ColumnType interface {
	DatabaseTypeName() string
}

type Executor interface {
	QueryContext(ctx context.Context, query string, args ...any) (RowIterator, error)
	ExecContext(ctx context.Context, query string, args ...any) error
}

type Transactor interface {
	BeginTx(ctx context.Context) (Transaction, error)
}

type Transaction interface {
	Executor
	Commit() error
	Rollback() error
}

type Preparer interface {
	PrepareContext(ctx context.Context, query string) (PreparedStatement, error)
}

type PreparedStatement interface {
	QueryContext(ctx context.Context, args ...any) (RowIterator, error)
	ExecContext(ctx context.Context, args ...any) error
	Close() error
}

type DB interface {
	Executor
	io.Closer
}

type GenericColumnType struct {
	Name string
}

func (g *GenericColumnType) DatabaseTypeName() string { return g.Name }

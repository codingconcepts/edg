package db

import (
	"context"
	"database/sql"
)

type SQDB struct {
	DB *sql.DB
}

func NewSQDB(db *sql.DB) *SQDB { return &SQDB{DB: db} }

func (s *SQDB) QueryContext(ctx context.Context, query string, args ...any) (RowIterator, error) {
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRowIterator{rows: rows}, nil
}

func (s *SQDB) ExecContext(ctx context.Context, query string, args ...any) error {
	_, err := s.DB.ExecContext(ctx, query, args...)
	return err
}

func (s *SQDB) BeginTx(ctx context.Context) (Transaction, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &sqlTransaction{tx: tx}, nil
}

func (s *SQDB) PrepareContext(ctx context.Context, query string) (PreparedStatement, error) {
	stmt, err := s.DB.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &sqlPreparedStmt{stmt: stmt}, nil
}

func (s *SQDB) Close() error { return s.DB.Close() }

type sqlRowIterator struct {
	rows *sql.Rows
}

func NewSQLRowIterator(rows *sql.Rows) RowIterator {
	return &sqlRowIterator{rows: rows}
}

func (r *sqlRowIterator) Columns() ([]string, error) { return r.rows.Columns() }

func (r *sqlRowIterator) ColumnTypes() ([]ColumnType, error) {
	cts, err := r.rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	result := make([]ColumnType, len(cts))
	for i, ct := range cts {
		result[i] = ct
	}
	return result, nil
}

func (r *sqlRowIterator) Next() bool             { return r.rows.Next() }
func (r *sqlRowIterator) Scan(dest ...any) error  { return r.rows.Scan(dest...) }
func (r *sqlRowIterator) Close() error            { return r.rows.Close() }
func (r *sqlRowIterator) Err() error              { return r.rows.Err() }

type sqlTransaction struct {
	tx *sql.Tx
}

func (t *sqlTransaction) QueryContext(ctx context.Context, query string, args ...any) (RowIterator, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRowIterator{rows: rows}, nil
}

func (t *sqlTransaction) ExecContext(ctx context.Context, query string, args ...any) error {
	_, err := t.tx.ExecContext(ctx, query, args...)
	return err
}

func (t *sqlTransaction) Commit() error   { return t.tx.Commit() }
func (t *sqlTransaction) Rollback() error { return t.tx.Rollback() }

type sqlPreparedStmt struct {
	stmt *sql.Stmt
}

func (p *sqlPreparedStmt) QueryContext(ctx context.Context, args ...any) (RowIterator, error) {
	rows, err := p.stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRowIterator{rows: rows}, nil
}

func (p *sqlPreparedStmt) ExecContext(ctx context.Context, args ...any) error {
	_, err := p.stmt.ExecContext(ctx, args...)
	return err
}

func (p *sqlPreparedStmt) Close() error { return p.stmt.Close() }

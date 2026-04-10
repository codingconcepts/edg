package env

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/codingconcepts/edg/pkg/config"
)

func (e *Env) Query(ctx context.Context, db *sql.DB, q *config.Query, args ...any) error {
	rows, err := db.QueryContext(ctx, q.Query, args...)
	if err != nil {
		return fmt.Errorf("running statement: %w", err)
	}

	data, err := ReadRows(rows)
	if err != nil {
		return fmt.Errorf("reading rows: %w", err)
	}

	e.SetEnv(q.Name, data)

	return nil
}

func (e *Env) Exec(ctx context.Context, db *sql.DB, q *config.Query, args ...any) error {
	_, err := db.ExecContext(ctx, q.Query, args...)
	if err != nil {
		return fmt.Errorf("running statement: %w", err)
	}

	return nil
}

func ReadRows(rows *sql.Rows) ([]map[string]any, error) {
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("getting columns: %w", err)
	}

	for i, c := range columns {
		columns[i] = strings.ToLower(c)
	}

	var results []map[string]any

	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		result := make(map[string]any, len(columns))
		for i, col := range columns {
			result[col] = values[i]
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return results, nil
}

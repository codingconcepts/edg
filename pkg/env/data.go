package env

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/db"
)

func (e *Env) Query(ctx context.Context, ex db.Executor, q *config.Query, args ...any) error {
	rows, err := ex.QueryContext(ctx, q.Query, args...)
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

func (e *Env) Exec(ctx context.Context, ex db.Executor, q *config.Query, args ...any) error {
	err := ex.ExecContext(ctx, q.Query, args...)
	if err != nil {
		return fmt.Errorf("running statement: %w", err)
	}

	return nil
}

func (e *Env) QueryPrepared(ctx context.Context, stmt db.PreparedStatement, q *config.Query, args ...any) error {
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("running prepared statement: %w", err)
	}

	data, err := ReadRows(rows)
	if err != nil {
		return fmt.Errorf("reading rows: %w", err)
	}

	e.SetEnv(q.Name, data)

	return nil
}

func (e *Env) ExecPrepared(ctx context.Context, stmt db.PreparedStatement, q *config.Query, args ...any) error {
	err := stmt.ExecContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("running prepared statement: %w", err)
	}

	return nil
}

func ReadRows(rows db.RowIterator) ([]map[string]any, error) {
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("getting columns: %w", err)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("getting column types: %w", err)
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
			switch v := values[i].(type) {
			case []byte:
				result[col] = normalizeBytes(v, columnTypes[i].DatabaseTypeName())
			default:
				result[col] = values[i]
			}
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return results, nil
}

// normalizeBytes converts raw []byte values from database drivers into
// portable Go types. UNIQUEIDENTIFIER columns get the wire-format bytes
// (first 3 groups little-endian) decoded into a canonical UUID string.
// Binary/blob columns are preserved as []byte. All other []byte values
// are converted to string.
func normalizeBytes(b []byte, dbTypeName string) any {
	switch dbTypeName {
	case "UNIQUEIDENTIFIER":
		if len(b) == 16 {
			return fmt.Sprintf("%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X",
				binary.LittleEndian.Uint32(b[0:4]),
				binary.LittleEndian.Uint16(b[4:6]),
				binary.LittleEndian.Uint16(b[6:8]),
				b[8], b[9],
				b[10], b[11], b[12], b[13], b[14], b[15])
		}
	case "BLOB", "BYTEA", "BYTES", "BINARY", "VARBINARY", "RAW",
		"TINYBLOB", "MEDIUMBLOB", "LONGBLOB", "IMAGE", "LONG RAW":
		dst := make([]byte, len(b))
		copy(dst, b)
		return dst
	}
	return string(b)
}

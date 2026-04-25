package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Inspect reads the schema metadata for the given driver and database/schema
// name. It returns a slice of Tables with column types, defaults, primary keys,
// and foreign key references populated.
func Inspect(ctx context.Context, db *sql.DB, driver, database string) ([]Table, error) {
	switch driver {
	case "pgx", "dsql":
		return inspectPostgres(ctx, db, database)
	case "mysql":
		return inspectMySQL(ctx, db, database)
	case "mssql":
		return inspectMSSQL(ctx, db, database)
	case "oracle":
		return inspectOracle(ctx, db, database)
	case "spanner":
		return inspectSpanner(ctx, db, database)
	default:
		return nil, fmt.Errorf("unsupported driver for init: %s", driver)
	}
}

func buildResult(tableMap map[string]*Table, order []string) []Table {
	tables := make([]Table, 0, len(order))
	for _, name := range order {
		tables = append(tables, *tableMap[name])
	}
	return tables
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

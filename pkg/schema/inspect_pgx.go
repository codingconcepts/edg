package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func inspectPostgres(ctx context.Context, db *sql.DB, schema string) ([]Table, error) {
	tableMap, tableOrder, err := pgxTables(ctx, db, schema)
	if err != nil {
		return nil, err
	}
	if err := pgxColumns(ctx, db, schema, tableMap); err != nil {
		return nil, err
	}
	if err := pgxPrimaryKeys(ctx, db, schema, tableMap); err != nil {
		return nil, err
	}
	if err := pgxForeignKeys(ctx, db, schema, tableMap); err != nil {
		return nil, err
	}

	// Try SHOW CREATE TABLE (CockroachDB). Falls back to pg_catalog
	// on PostgreSQL where that syntax is unsupported.
	if !pgxFetchDDL(ctx, db, tableMap, tableOrder) {
		if err := pgxFetchDDLFromCatalog(ctx, db, schema, tableMap, tableOrder); err != nil {
			return nil, err
		}
	}

	return buildResult(tableMap, tableOrder), nil
}

func pgxTables(ctx context.Context, db *sql.DB, schema string) (map[string]*Table, []string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_type = 'BASE TABLE'
		ORDER BY table_name`, schema)
	if err != nil {
		return nil, nil, fmt.Errorf("querying tables: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*Table)
	var order []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, nil, fmt.Errorf("scanning table: %w", err)
		}
		tableMap[name] = &Table{Name: name}
		order = append(order, name)
	}
	return tableMap, order, rows.Err()
}

func pgxColumns(ctx context.Context, db *sql.DB, schema string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name, column_name, data_type,
		       character_maximum_length, numeric_precision, numeric_scale,
		       is_nullable, COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = $1
		ORDER BY table_name, ordinal_position`, schema)
	if err != nil {
		return fmt.Errorf("querying columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName, dataType, isNullable, colDefault string
		var charMaxLen, numPrec, numScale sql.NullInt64
		if err := rows.Scan(&tableName, &colName, &dataType, &charMaxLen, &numPrec, &numScale, &isNullable, &colDefault); err != nil {
			return fmt.Errorf("scanning column: %w", err)
		}
		t, ok := tableMap[tableName]
		if !ok {
			continue
		}
		colDefault = strings.TrimSpace(colDefault)
		t.Columns = append(t.Columns, Column{
			Name:       colName,
			DataType:   pgxBuildType(dataType, charMaxLen, numPrec, numScale),
			IsNullable: isNullable == "YES",
			Default:    pgxCleanDefault(colDefault),
			IsGenerated: strings.Contains(strings.ToLower(colDefault), "nextval(") ||
				strings.Contains(strings.ToLower(colDefault), "unique_rowid("),
		})
	}
	return rows.Err()
}

func pgxPrimaryKeys(ctx context.Context, db *sql.DB, schema string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT tc.table_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		  AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = $1`, schema)
	if err != nil {
		return fmt.Errorf("querying primary keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName string
		if err := rows.Scan(&tableName, &colName); err != nil {
			return fmt.Errorf("scanning pk: %w", err)
		}
		t, ok := tableMap[tableName]
		if !ok {
			continue
		}
		for i := range t.Columns {
			if t.Columns[i].Name == colName {
				t.Columns[i].IsPK = true
				break
			}
		}
	}
	return rows.Err()
}

func pgxForeignKeys(ctx context.Context, db *sql.DB, schema string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT
		    kcu.table_name,
		    kcu.column_name,
		    ccu.table_name  AS ref_table,
		    ccu.column_name AS ref_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		  AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
		  ON tc.constraint_name = ccu.constraint_name
		  AND tc.table_schema = ccu.constraint_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = $1`, schema)
	if err != nil {
		return fmt.Errorf("querying foreign keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName, refTable, refCol string
		if err := rows.Scan(&tableName, &colName, &refTable, &refCol); err != nil {
			return fmt.Errorf("scanning fk: %w", err)
		}
		t, ok := tableMap[tableName]
		if !ok {
			continue
		}
		for i := range t.Columns {
			if t.Columns[i].Name == colName {
				t.Columns[i].Ref = refTable + "." + refCol
				break
			}
		}
	}
	return rows.Err()
}

func pgxBuildType(dataType string, charMaxLen, numPrec, numScale sql.NullInt64) string {
	switch dataType {
	case "integer":
		return "INT"
	case "bigint":
		return "BIGINT"
	case "smallint":
		return "SMALLINT"
	case "character varying":
		if charMaxLen.Valid {
			return fmt.Sprintf("VARCHAR(%d)", charMaxLen.Int64)
		}
		return "STRING"
	case "character":
		if charMaxLen.Valid {
			return fmt.Sprintf("CHAR(%d)", charMaxLen.Int64)
		}
		return "CHAR"
	case "text":
		return "STRING"
	case "uuid":
		return "UUID"
	case "boolean":
		return "BOOL"
	case "numeric":
		if numPrec.Valid && numScale.Valid {
			return fmt.Sprintf("DECIMAL(%d,%d)", numPrec.Int64, numScale.Int64)
		}
		return "DECIMAL"
	case "double precision":
		return "FLOAT8"
	case "real":
		return "FLOAT4"
	case "timestamp without time zone":
		return "TIMESTAMP"
	case "timestamp with time zone":
		return "TIMESTAMPTZ"
	case "date":
		return "DATE"
	case "bytea":
		return "BYTES"
	case "jsonb":
		return "JSONB"
	case "json":
		return "JSON"
	default:
		return strings.ToUpper(dataType)
	}
}

func pgxCleanDefault(def string) string {
	// Strip PostgreSQL type casts like ::character varying for readability.
	if before, _, ok := strings.Cut(def, "::"); ok {
		return before
	}
	return def
}

// pgxFetchDDL tries SHOW CREATE TABLE (CockroachDB). Returns false on
// PostgreSQL where that syntax is unsupported.
func pgxFetchDDL(ctx context.Context, db *sql.DB, tableMap map[string]*Table, order []string) bool {
	for _, name := range order {
		var tblName, stmt string
		err := db.QueryRowContext(ctx, "SHOW CREATE TABLE "+name).Scan(&tblName, &stmt)
		if err != nil {
			return false
		}
		tableMap[name].CreateStmt = strings.TrimRight(stmt, ";\n\r\t ")
	}
	return true
}

// pgxFetchDDLFromCatalog builds CREATE TABLE DDL from pg_catalog views.
// Used on PostgreSQL where SHOW CREATE TABLE is unavailable.
func pgxFetchDDLFromCatalog(ctx context.Context, db *sql.DB, schema string, tableMap map[string]*Table, order []string) error {
	for _, name := range order {
		qualified := schema + "." + name

		// Column definitions.
		colRows, err := db.QueryContext(ctx, `
			SELECT
				a.attname,
				pg_catalog.format_type(a.atttypid, a.atttypmod),
				a.attnotnull,
				COALESCE(pg_catalog.pg_get_expr(d.adbin, d.adrelid), '')
			FROM pg_catalog.pg_attribute a
			LEFT JOIN pg_catalog.pg_attrdef d
			  ON a.attrelid = d.adrelid AND a.attnum = d.adnum
			WHERE a.attrelid = $1::regclass
			  AND a.attnum > 0
			  AND NOT a.attisdropped
			ORDER BY a.attnum`, qualified)
		if err != nil {
			return fmt.Errorf("querying pg_catalog columns for %s: %w", name, err)
		}

		var lines []string
		for colRows.Next() {
			var colName, dataType, defaultVal string
			var notNull bool
			if err := colRows.Scan(&colName, &dataType, &notNull, &defaultVal); err != nil {
				colRows.Close()
				return fmt.Errorf("scanning pg_catalog column for %s: %w", name, err)
			}
			line := fmt.Sprintf("  %s %s", colName, dataType)
			if notNull {
				line += " NOT NULL"
			}
			if defaultVal != "" {
				line += " DEFAULT " + defaultVal
			}
			lines = append(lines, line)
		}
		colRows.Close()
		if err := colRows.Err(); err != nil {
			return err
		}

		// Constraints (PK, FK, UNIQUE, CHECK).
		conRows, err := db.QueryContext(ctx, `
			SELECT pg_catalog.pg_get_constraintdef(c.oid)
			FROM pg_catalog.pg_constraint c
			WHERE c.conrelid = $1::regclass
			ORDER BY c.contype, c.conname`, qualified)
		if err != nil {
			return fmt.Errorf("querying pg_catalog constraints for %s: %w", name, err)
		}

		for conRows.Next() {
			var conDef string
			if err := conRows.Scan(&conDef); err != nil {
				conRows.Close()
				return fmt.Errorf("scanning pg_catalog constraint for %s: %w", name, err)
			}
			lines = append(lines, "  "+conDef)
		}
		conRows.Close()
		if err := conRows.Err(); err != nil {
			return err
		}

		tableMap[name].CreateStmt = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n)", name, strings.Join(lines, ",\n"))
	}
	return nil
}

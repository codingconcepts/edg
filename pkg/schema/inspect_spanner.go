package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func inspectSpanner(ctx context.Context, db *sql.DB, schema string) ([]Table, error) {
	tableMap, tableOrder, err := spannerTables(ctx, db, schema)
	if err != nil {
		return nil, err
	}
	if err := spannerColumns(ctx, db, schema, tableMap); err != nil {
		return nil, err
	}
	if err := spannerPrimaryKeys(ctx, db, schema, tableMap); err != nil {
		return nil, err
	}
	if err := spannerForeignKeys(ctx, db, schema, tableMap); err != nil {
		return nil, err
	}
	if err := spannerBuildDDL(tableMap, tableOrder); err != nil {
		return nil, err
	}
	return buildResult(tableMap, tableOrder), nil
}

func spannerTables(ctx context.Context, db *sql.DB, schema string) (map[string]*Table, []string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = @p1 AND table_type = 'BASE TABLE'
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

func spannerColumns(ctx context.Context, db *sql.DB, schema string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name, column_name, spanner_type,
		       is_nullable, COALESCE(column_default, ''),
		       COALESCE(is_generated, 'NEVER')
		FROM information_schema.columns
		WHERE table_schema = @p1
		ORDER BY table_name, ordinal_position`, schema)
	if err != nil {
		return fmt.Errorf("querying columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName, spannerType, isNullable, colDefault, isGenerated string
		if err := rows.Scan(&tableName, &colName, &spannerType, &isNullable, &colDefault, &isGenerated); err != nil {
			return fmt.Errorf("scanning column: %w", err)
		}
		t, ok := tableMap[tableName]
		if !ok {
			continue
		}
		colDefault = strings.TrimSpace(colDefault)
		t.Columns = append(t.Columns, Column{
			Name:        colName,
			DataType:    spannerType,
			IsNullable:  isNullable == "YES",
			Default:     colDefault,
			IsGenerated: isGenerated != "NEVER",
		})
	}
	return rows.Err()
}

func spannerPrimaryKeys(ctx context.Context, db *sql.DB, schema string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT kcu.table_name, kcu.column_name
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.table_constraints tc
		  ON kcu.constraint_name = tc.constraint_name
		  AND kcu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = @p1`, schema)
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

func spannerForeignKeys(ctx context.Context, db *sql.DB, schema string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT
		    kcu.table_name,
		    kcu.column_name,
		    ctu.table_name AS ref_table,
		    ctu.column_name AS ref_column
		FROM information_schema.referential_constraints rc
		JOIN information_schema.key_column_usage kcu
		  ON rc.constraint_name = kcu.constraint_name
		  AND rc.constraint_schema = kcu.constraint_schema
		JOIN (
		    SELECT constraint_name, constraint_schema, table_name, column_name
		    FROM information_schema.key_column_usage
		) ctu
		  ON rc.unique_constraint_name = ctu.constraint_name
		  AND rc.unique_constraint_schema = ctu.constraint_schema
		WHERE rc.constraint_schema = @p1`, schema)
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

func spannerBuildDDL(tableMap map[string]*Table, order []string) error {
	for _, name := range order {
		t := tableMap[name]
		var lines []string
		for _, col := range t.Columns {
			line := fmt.Sprintf("  %s %s", col.Name, col.DataType)
			if !col.IsNullable && !col.IsPK {
				line += " NOT NULL"
			}
			if col.Default != "" {
				line += " DEFAULT (" + col.Default + ")"
			}
			lines = append(lines, line)
		}

		var pkCols []string
		for _, col := range t.Columns {
			if col.IsPK {
				pkCols = append(pkCols, col.Name)
			}
		}

		stmt := fmt.Sprintf("CREATE TABLE %s (\n%s\n) PRIMARY KEY (%s)",
			name, strings.Join(lines, ",\n"), strings.Join(pkCols, ", "))
		t.CreateStmt = stmt
	}
	return nil
}

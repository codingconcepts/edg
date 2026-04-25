package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func inspectMySQL(ctx context.Context, db *sql.DB, database string) ([]Table, error) {
	tableMap, tableOrder, err := mysqlTables(ctx, db, database)
	if err != nil {
		return nil, err
	}
	if err := mysqlColumns(ctx, db, database, tableMap); err != nil {
		return nil, err
	}
	if err := mysqlForeignKeys(ctx, db, database, tableMap); err != nil {
		return nil, err
	}
	if err := mysqlFetchDDL(ctx, db, tableMap, tableOrder); err != nil {
		return nil, err
	}

	tables := make([]Table, 0, len(tableOrder))
	for _, name := range tableOrder {
		tables = append(tables, *tableMap[name])
	}
	return tables, nil
}

func mysqlTables(ctx context.Context, db *sql.DB, database string) (map[string]*Table, []string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ? AND table_type = 'BASE TABLE'
		ORDER BY table_name`, database)
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

func mysqlColumns(ctx context.Context, db *sql.DB, database string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name, column_name, column_type,
		       is_nullable, COALESCE(column_default, ''), column_key,
		       COALESCE(extra, '')
		FROM information_schema.columns
		WHERE table_schema = ?
		ORDER BY table_name, ordinal_position`, database)
	if err != nil {
		return fmt.Errorf("querying columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName, colType, isNullable, colDefault, colKey, extra string
		if err := rows.Scan(&tableName, &colName, &colType, &isNullable, &colDefault, &colKey, &extra); err != nil {
			return fmt.Errorf("scanning column: %w", err)
		}
		t, ok := tableMap[tableName]
		if !ok {
			continue
		}
		t.Columns = append(t.Columns, Column{
			Name:        colName,
			DataType:    strings.ToUpper(colType),
			IsNullable:  isNullable == "YES",
			Default:     mysqlFormatDefault(strings.TrimSpace(colDefault), colType, extra),
			IsPK:        colKey == "PRI",
			IsGenerated: strings.Contains(strings.ToLower(extra), "auto_increment"),
		})
	}
	return rows.Err()
}

func mysqlForeignKeys(ctx context.Context, db *sql.DB, database string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name, column_name,
		       referenced_table_name, referenced_column_name
		FROM information_schema.key_column_usage
		WHERE table_schema = ?
		  AND referenced_table_name IS NOT NULL`, database)
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

func mysqlFormatDefault(def, colType, extra string) string {
	if def == "" || strings.EqualFold(def, "NULL") {
		return ""
	}
	// Expression defaults (MySQL 8.0+).
	if strings.Contains(strings.ToLower(extra), "default_generated") {
		return "(" + def + ")"
	}
	dl := strings.ToLower(def)
	if dl == "current_timestamp" || strings.Contains(dl, "(") {
		return def
	}
	ct := strings.ToLower(colType)
	if containsAny(ct, "char", "text", "varchar", "enum", "set") {
		return "'" + def + "'"
	}
	return def
}

func mysqlFetchDDL(ctx context.Context, db *sql.DB, tableMap map[string]*Table, order []string) error {
	for _, name := range order {
		var tblName, stmt string
		if err := db.QueryRowContext(ctx, "SHOW CREATE TABLE `"+name+"`").Scan(&tblName, &stmt); err != nil {
			return fmt.Errorf("fetching DDL for %s: %w", name, err)
		}
		tableMap[name].CreateStmt = strings.TrimRight(stmt, ";\n\r\t ")
	}
	return nil
}

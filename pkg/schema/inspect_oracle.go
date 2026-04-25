package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func inspectOracle(ctx context.Context, db *sql.DB, owner string) ([]Table, error) {
	owner = strings.ToUpper(owner)
	tableMap, tableOrder, err := oracleTables(ctx, db, owner)
	if err != nil {
		return nil, err
	}
	if err := oracleColumns(ctx, db, owner, tableMap); err != nil {
		return nil, err
	}
	if err := oraclePrimaryKeys(ctx, db, owner, tableMap); err != nil {
		return nil, err
	}
	if err := oracleForeignKeys(ctx, db, owner, tableMap); err != nil {
		return nil, err
	}
	if err := oracleFetchDDL(ctx, db, owner, tableMap, tableOrder); err != nil {
		return nil, err
	}
	return buildResult(tableMap, tableOrder), nil
}

func oracleTables(ctx context.Context, db *sql.DB, owner string) (map[string]*Table, []string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM all_tables
		WHERE owner = :1
		ORDER BY table_name`, owner)
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
		name = strings.ToLower(name)
		tableMap[name] = &Table{Name: name}
		order = append(order, name)
	}
	return tableMap, order, rows.Err()
}

func oracleColumns(ctx context.Context, db *sql.DB, owner string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name, column_name, data_type,
		       char_length, data_precision, data_scale,
		       nullable, data_default
		FROM all_tab_columns
		WHERE owner = :1
		ORDER BY table_name, column_id`, owner)
	if err != nil {
		return fmt.Errorf("querying columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName, dataType, nullable string
		var colDefault sql.NullString
		var charLen, dataPrecision, dataScale sql.NullInt64
		if err := rows.Scan(&tableName, &colName, &dataType, &charLen, &dataPrecision, &dataScale, &nullable, &colDefault); err != nil {
			return fmt.Errorf("scanning column: %w", err)
		}
		name := strings.ToLower(tableName)
		t, ok := tableMap[name]
		if !ok {
			continue
		}
		def := strings.TrimSpace(colDefault.String)
		dl := strings.ToUpper(def)
		t.Columns = append(t.Columns, Column{
			Name:        strings.ToLower(colName),
			DataType:    oracleBuildType(dataType, charLen, dataPrecision, dataScale),
			IsNullable:  nullable == "Y",
			Default:     oracleCleanDefault(def),
			IsGenerated: strings.Contains(dl, "ISEQ") || strings.Contains(dl, "NEXTVAL"),
		})
	}
	return rows.Err()
}

func oraclePrimaryKeys(ctx context.Context, db *sql.DB, owner string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT cc.table_name, cc.column_name
		FROM all_constraints c
		JOIN all_cons_columns cc ON c.constraint_name = cc.constraint_name AND c.owner = cc.owner
		WHERE c.constraint_type = 'P'
		  AND c.owner = :1`, owner)
	if err != nil {
		return fmt.Errorf("querying primary keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName string
		if err := rows.Scan(&tableName, &colName); err != nil {
			return fmt.Errorf("scanning pk: %w", err)
		}
		name := strings.ToLower(tableName)
		t, ok := tableMap[name]
		if !ok {
			continue
		}
		col := strings.ToLower(colName)
		for i := range t.Columns {
			if t.Columns[i].Name == col {
				t.Columns[i].IsPK = true
				break
			}
		}
	}
	return rows.Err()
}

func oracleForeignKeys(ctx context.Context, db *sql.DB, owner string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT
		    cc.table_name,
		    cc.column_name,
		    rcc.table_name  AS ref_table,
		    rcc.column_name AS ref_column
		FROM all_constraints c
		JOIN all_cons_columns cc ON c.constraint_name = cc.constraint_name AND c.owner = cc.owner
		JOIN all_constraints rc ON c.r_constraint_name = rc.constraint_name AND c.r_owner = rc.owner
		JOIN all_cons_columns rcc ON rc.constraint_name = rcc.constraint_name AND rc.owner = rcc.owner
		  AND cc.position = rcc.position
		WHERE c.constraint_type = 'R'
		  AND c.owner = :1`, owner)
	if err != nil {
		return fmt.Errorf("querying foreign keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName, refTable, refCol string
		if err := rows.Scan(&tableName, &colName, &refTable, &refCol); err != nil {
			return fmt.Errorf("scanning fk: %w", err)
		}
		name := strings.ToLower(tableName)
		t, ok := tableMap[name]
		if !ok {
			continue
		}
		col := strings.ToLower(colName)
		for i := range t.Columns {
			if t.Columns[i].Name == col {
				t.Columns[i].Ref = strings.ToLower(refTable) + "." + strings.ToLower(refCol)
				break
			}
		}
	}
	return rows.Err()
}

func oracleBuildType(dataType string, charLen, dataPrecision, dataScale sql.NullInt64) string {
	switch strings.ToUpper(dataType) {
	case "NUMBER":
		if dataPrecision.Valid {
			if dataScale.Valid && dataScale.Int64 > 0 {
				return fmt.Sprintf("NUMBER(%d,%d)", dataPrecision.Int64, dataScale.Int64)
			}
			return fmt.Sprintf("NUMBER(%d)", dataPrecision.Int64)
		}
		return "NUMBER"
	case "VARCHAR2":
		if charLen.Valid && charLen.Int64 > 0 {
			return fmt.Sprintf("VARCHAR2(%d)", charLen.Int64)
		}
		return "VARCHAR2(255)"
	case "CHAR":
		if charLen.Valid && charLen.Int64 > 0 {
			return fmt.Sprintf("CHAR(%d)", charLen.Int64)
		}
		return "CHAR(1)"
	case "NVARCHAR2":
		if charLen.Valid && charLen.Int64 > 0 {
			return fmt.Sprintf("NVARCHAR2(%d)", charLen.Int64)
		}
		return "NVARCHAR2(255)"
	case "CLOB":
		return "CLOB"
	case "BLOB":
		return "BLOB"
	case "DATE":
		return "DATE"
	case "RAW":
		if charLen.Valid && charLen.Int64 > 0 {
			return fmt.Sprintf("RAW(%d)", charLen.Int64)
		}
		return "RAW(16)"
	default:
		dt := strings.ToUpper(dataType)
		if strings.HasPrefix(dt, "TIMESTAMP") {
			return "TIMESTAMP"
		}
		return dt
	}
}

func oracleCleanDefault(def string) string {
	if def == "" || strings.EqualFold(def, "NULL") {
		return ""
	}
	return def
}

func oracleFetchDDL(ctx context.Context, db *sql.DB, owner string, tableMap map[string]*Table, order []string) error {
	// Suppress storage/segment/tablespace clauses for cleaner output.
	_, _ = db.ExecContext(ctx, `BEGIN
		DBMS_METADATA.SET_TRANSFORM_PARAM(DBMS_METADATA.SESSION_TRANSFORM, 'SEGMENT_ATTRIBUTES', FALSE);
		DBMS_METADATA.SET_TRANSFORM_PARAM(DBMS_METADATA.SESSION_TRANSFORM, 'SQLTERMINATOR', FALSE);
		DBMS_METADATA.SET_TRANSFORM_PARAM(DBMS_METADATA.SESSION_TRANSFORM, 'STORAGE', FALSE);
		DBMS_METADATA.SET_TRANSFORM_PARAM(DBMS_METADATA.SESSION_TRANSFORM, 'TABLESPACE', FALSE);
	END;`)

	for _, name := range order {
		tblName := strings.ToUpper(name)
		var stmt string
		err := db.QueryRowContext(ctx,
			"SELECT DBMS_METADATA.GET_DDL('TABLE', :1, :2) FROM DUAL",
			tblName, owner).Scan(&stmt)
		if err != nil {
			return fmt.Errorf("fetching DDL for %s: %w", name, err)
		}
		tableMap[name].CreateStmt = strings.TrimRight(strings.TrimSpace(stmt), ";\n\r\t ")
	}
	return nil
}

package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func inspectMSSQL(ctx context.Context, db *sql.DB, dbSchema string) ([]Table, error) {
	tableMap, tableOrder, err := mssqlTables(ctx, db, dbSchema)
	if err != nil {
		return nil, err
	}
	if err := mssqlColumns(ctx, db, dbSchema, tableMap); err != nil {
		return nil, err
	}
	if err := mssqlPrimaryKeys(ctx, db, dbSchema, tableMap); err != nil {
		return nil, err
	}
	if err := mssqlForeignKeys(ctx, db, dbSchema, tableMap); err != nil {
		return nil, err
	}
	if err := mssqlIdentityColumns(ctx, db, dbSchema, tableMap); err != nil {
		return nil, err
	}
	if err := mssqlFetchDDL(ctx, db, dbSchema, tableMap, tableOrder); err != nil {
		return nil, err
	}
	return buildResult(tableMap, tableOrder), nil
}

func mssqlTables(ctx context.Context, db *sql.DB, dbSchema string) (map[string]*Table, []string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ? AND table_type = 'BASE TABLE'
		ORDER BY table_name`, dbSchema)
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

func mssqlColumns(ctx context.Context, db *sql.DB, dbSchema string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name, column_name, data_type,
		       character_maximum_length, numeric_precision, numeric_scale,
		       is_nullable, COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = ?
		ORDER BY table_name, ordinal_position`, dbSchema)
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
		colDefault = mssqlCleanDefault(strings.TrimSpace(colDefault))
		t.Columns = append(t.Columns, Column{
			Name:       colName,
			DataType:   mssqlBuildType(dataType, charMaxLen, numPrec, numScale),
			IsNullable: isNullable == "YES",
			Default:    colDefault,
		})
	}
	return rows.Err()
}

func mssqlPrimaryKeys(ctx context.Context, db *sql.DB, dbSchema string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT tc.table_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		  AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = ?`, dbSchema)
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

func mssqlForeignKeys(ctx context.Context, db *sql.DB, dbSchema string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT
		    kcu1.table_name,
		    kcu1.column_name,
		    kcu2.table_name  AS ref_table,
		    kcu2.column_name AS ref_column
		FROM information_schema.referential_constraints rc
		JOIN information_schema.key_column_usage kcu1
		  ON rc.constraint_name = kcu1.constraint_name
		  AND rc.constraint_schema = kcu1.constraint_schema
		JOIN information_schema.key_column_usage kcu2
		  ON rc.unique_constraint_name = kcu2.constraint_name
		  AND rc.unique_constraint_schema = kcu2.constraint_schema
		  AND kcu1.ordinal_position = kcu2.ordinal_position
		WHERE rc.constraint_schema = ?`, dbSchema)
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

func mssqlIdentityColumns(ctx context.Context, db *sql.DB, dbSchema string, tableMap map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
		SELECT t.name AS table_name, c.name AS column_name
		FROM sys.columns c
		JOIN sys.tables t ON c.object_id = t.object_id
		JOIN sys.schemas s ON t.schema_id = s.schema_id
		WHERE s.name = ? AND c.is_identity = 1`, dbSchema)
	if err != nil {
		return fmt.Errorf("querying identity columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName string
		if err := rows.Scan(&tableName, &colName); err != nil {
			return fmt.Errorf("scanning identity: %w", err)
		}
		t, ok := tableMap[tableName]
		if !ok {
			continue
		}
		for i := range t.Columns {
			if t.Columns[i].Name == colName {
				t.Columns[i].IsGenerated = true
				break
			}
		}
	}
	return rows.Err()
}

func mssqlBuildType(dataType string, charMaxLen, numPrec, numScale sql.NullInt64) string {
	switch strings.ToLower(dataType) {
	case "int":
		return "INT"
	case "bigint":
		return "BIGINT"
	case "smallint":
		return "SMALLINT"
	case "tinyint":
		return "TINYINT"
	case "bit":
		return "BIT"
	case "nvarchar":
		if charMaxLen.Valid {
			if charMaxLen.Int64 == -1 {
				return "NVARCHAR(MAX)"
			}
			return fmt.Sprintf("NVARCHAR(%d)", charMaxLen.Int64)
		}
		return "NVARCHAR(255)"
	case "varchar":
		if charMaxLen.Valid {
			if charMaxLen.Int64 == -1 {
				return "VARCHAR(MAX)"
			}
			return fmt.Sprintf("VARCHAR(%d)", charMaxLen.Int64)
		}
		return "VARCHAR(255)"
	case "nchar":
		if charMaxLen.Valid {
			return fmt.Sprintf("NCHAR(%d)", charMaxLen.Int64)
		}
		return "NCHAR(1)"
	case "char":
		if charMaxLen.Valid {
			return fmt.Sprintf("CHAR(%d)", charMaxLen.Int64)
		}
		return "CHAR(1)"
	case "uniqueidentifier":
		return "UNIQUEIDENTIFIER"
	case "datetime2":
		return "DATETIME2"
	case "datetime":
		return "DATETIME"
	case "date":
		return "DATE"
	case "decimal", "numeric":
		if numPrec.Valid && numScale.Valid {
			return fmt.Sprintf("DECIMAL(%d,%d)", numPrec.Int64, numScale.Int64)
		}
		return "DECIMAL"
	case "float":
		return "FLOAT"
	case "real":
		return "REAL"
	case "varbinary":
		if charMaxLen.Valid {
			if charMaxLen.Int64 == -1 {
				return "VARBINARY(MAX)"
			}
			return fmt.Sprintf("VARBINARY(%d)", charMaxLen.Int64)
		}
		return "VARBINARY"
	case "ntext":
		return "NVARCHAR(MAX)"
	default:
		return strings.ToUpper(dataType)
	}
}

func mssqlCleanDefault(def string) string {
	if def == "" {
		return ""
	}
	// MSSQL wraps defaults in parens: ((0)), (getdate()), (newid()).
	for {
		if len(def) < 2 || def[0] != '(' || def[len(def)-1] != ')' {
			break
		}
		depth := 0
		match := true
		for i, c := range def {
			switch c {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 && i < len(def)-1 {
					match = false
				}
			}
			if !match {
				break
			}
		}
		if !match {
			break
		}
		def = def[1 : len(def)-1]
	}
	if strings.EqualFold(def, "NULL") {
		return ""
	}
	return def
}

// mssqlFetchDDL builds CREATE TABLE DDL from sys catalog views.
func mssqlFetchDDL(ctx context.Context, db *sql.DB, dbSchema string, tableMap map[string]*Table, order []string) error {
	for _, name := range order {
		// Column definitions.
		colRows, err := db.QueryContext(ctx, `
			SELECT
				c.name,
				TYPE_NAME(c.user_type_id),
				c.max_length,
				c.precision,
				c.scale,
				c.is_nullable,
				c.is_identity,
				ISNULL(dc.definition, '')
			FROM sys.columns c
			JOIN sys.tables t ON c.object_id = t.object_id
			JOIN sys.schemas s ON t.schema_id = s.schema_id
			LEFT JOIN sys.default_constraints dc ON c.default_object_id = dc.object_id
			WHERE s.name = ? AND t.name = ?
			ORDER BY c.column_id`, dbSchema, name)
		if err != nil {
			return fmt.Errorf("querying sys.columns for %s: %w", name, err)
		}

		var lines []string
		for colRows.Next() {
			var colName, dataType, defaultDef string
			var maxLen, prec, scale int
			var isNullable, isIdentity bool
			if err := colRows.Scan(&colName, &dataType, &maxLen, &prec, &scale, &isNullable, &isIdentity, &defaultDef); err != nil {
				colRows.Close()
				return fmt.Errorf("scanning sys.columns for %s: %w", name, err)
			}
			line := fmt.Sprintf("  %s %s", colName, mssqlDDLType(dataType, maxLen, prec, scale))
			if isIdentity {
				line += " IDENTITY(1,1)"
			}
			if !isNullable {
				line += " NOT NULL"
			}
			defaultDef = mssqlCleanDefault(defaultDef)
			if defaultDef != "" {
				line += " DEFAULT " + defaultDef
			}
			lines = append(lines, line)
		}
		colRows.Close()
		if err := colRows.Err(); err != nil {
			return err
		}

		// Primary key.
		pkRows, err := db.QueryContext(ctx, `
			SELECT COL_NAME(ic.object_id, ic.column_id)
			FROM sys.key_constraints kc
			JOIN sys.index_columns ic
			  ON kc.parent_object_id = ic.object_id
			  AND kc.unique_index_id = ic.index_id
			JOIN sys.tables t ON kc.parent_object_id = t.object_id
			JOIN sys.schemas s ON t.schema_id = s.schema_id
			WHERE s.name = ? AND t.name = ? AND kc.type = 'PK'
			ORDER BY ic.key_ordinal`, dbSchema, name)
		if err != nil {
			return fmt.Errorf("querying sys.key_constraints for %s: %w", name, err)
		}

		var pkCols []string
		for pkRows.Next() {
			var col string
			if err := pkRows.Scan(&col); err != nil {
				pkRows.Close()
				return fmt.Errorf("scanning pk for %s: %w", name, err)
			}
			pkCols = append(pkCols, col)
		}
		pkRows.Close()
		if err := pkRows.Err(); err != nil {
			return err
		}
		if len(pkCols) > 0 {
			lines = append(lines, fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
		}

		// Foreign keys.
		fkRows, err := db.QueryContext(ctx, `
			SELECT
				COL_NAME(fkc.parent_object_id, fkc.parent_column_id),
				OBJECT_NAME(fkc.referenced_object_id),
				COL_NAME(fkc.referenced_object_id, fkc.referenced_column_id)
			FROM sys.foreign_key_columns fkc
			JOIN sys.tables t ON fkc.parent_object_id = t.object_id
			JOIN sys.schemas s ON t.schema_id = s.schema_id
			WHERE s.name = ? AND t.name = ?
			ORDER BY fkc.constraint_column_id`, dbSchema, name)
		if err != nil {
			return fmt.Errorf("querying sys.foreign_key_columns for %s: %w", name, err)
		}

		for fkRows.Next() {
			var col, refTable, refCol string
			if err := fkRows.Scan(&col, &refTable, &refCol); err != nil {
				fkRows.Close()
				return fmt.Errorf("scanning fk for %s: %w", name, err)
			}
			lines = append(lines, fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s(%s)", col, refTable, refCol))
		}
		fkRows.Close()
		if err := fkRows.Err(); err != nil {
			return err
		}

		tableMap[name].CreateStmt = fmt.Sprintf("CREATE TABLE %s (\n%s\n)", name, strings.Join(lines, ",\n"))
	}
	return nil
}

func mssqlDDLType(dataType string, maxLen, prec, scale int) string {
	switch strings.ToLower(dataType) {
	case "nvarchar", "nchar":
		if maxLen == -1 {
			return strings.ToUpper(dataType) + "(MAX)"
		}
		return fmt.Sprintf("%s(%d)", strings.ToUpper(dataType), maxLen/2)
	case "varchar", "char", "varbinary", "binary":
		if maxLen == -1 {
			return strings.ToUpper(dataType) + "(MAX)"
		}
		return fmt.Sprintf("%s(%d)", strings.ToUpper(dataType), maxLen)
	case "decimal", "numeric":
		return fmt.Sprintf("%s(%d,%d)", strings.ToUpper(dataType), prec, scale)
	default:
		return strings.ToUpper(dataType)
	}
}

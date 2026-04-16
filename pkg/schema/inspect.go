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
	default:
		return nil, fmt.Errorf("unsupported driver for init: %s", driver)
	}
}

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

package schema

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type initConfig struct {
	Globals initGlobals `yaml:"globals"`
	Up      []initQuery `yaml:"up"`
	Seed    []initQuery `yaml:"seed,omitempty"`
	Deseed  []initQuery `yaml:"deseed"`
	Down    []initQuery `yaml:"down"`
}

type initGlobals struct {
	Rows      int `yaml:"rows"`
	BatchSize int `yaml:"batch_size"`
}

type initQuery struct {
	Name  string   `yaml:"name"`
	Type  string   `yaml:"type,omitempty"`
	Count string   `yaml:"count,omitempty"`
	Args  []string `yaml:"args,omitempty"`
	Query string   `yaml:"query"`
}

// Generate produces a YAML config string with up, seed, deseed, and down
// sections for the given tables and driver.
func Generate(tables []Table, driver string) string {
	applyCheckBounds(tables)

	cfg := initConfig{
		Globals: initGlobals{Rows: 100, BatchSize: 100},
		Up:      buildUp(tables, driver),
		Seed:    buildSeed(tables, driver),
		Deseed:  buildDeseed(tables, driver),
		Down:    buildDown(tables, driver),
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Sprintf("# error: %v\n", err)
	}
	enc.Close()
	return buf.String()
}

func buildUp(tables []Table, driver string) []initQuery {
	queries := make([]initQuery, 0, len(tables))
	for _, t := range tables {
		q := initQuery{Name: "create_" + t.Name}
		switch driver {
		case "oracle":
			q.Type = "exec"
			q.Query = wrapOracleCreate(t.CreateStmt)
		default:
			q.Query = t.CreateStmt
		}
		queries = append(queries, q)
	}
	return queries
}

func buildSeed(tables []Table, driver string) []initQuery {
	var queries []initQuery
	for _, t := range tables {
		cols := seedableColumns(t)
		if len(cols) == 0 {
			continue
		}

		args := make([]string, len(cols))
		colNames := make([]string, len(cols))
		placeholders := make([]string, len(cols))
		for i, col := range cols {
			args[i] = exprForColumn(col)
			colNames[i] = col.Name
			placeholders[i] = placeholder(driver, i+1)
		}

		queries = append(queries, initQuery{
			Name:  "populate_" + t.Name,
			Type:  "exec",
			Count: "rows",
			Args:  args,
			Query: fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
				t.Name, strings.Join(colNames, ", "), strings.Join(placeholders, ", ")),
		})
	}
	return queries
}

func buildDeseed(tables []Table, driver string) []initQuery {
	queries := make([]initQuery, 0, len(tables))
	for i := len(tables) - 1; i >= 0; i-- {
		t := tables[i]
		q := initQuery{
			Name: deseedVerb(driver) + "_" + t.Name,
			Type: "exec",
		}
		switch driver {
		case "pgx", "dsql":
			q.Query = "TRUNCATE TABLE " + t.Name
			if i == 0 {
				q.Query += " CASCADE"
			}
		case "oracle":
			q.Query = "TRUNCATE TABLE " + t.Name
		case "spanner":
			q.Query = "DELETE FROM " + t.Name + " WHERE TRUE"
		default:
			q.Query = "DELETE FROM " + t.Name
		}
		queries = append(queries, q)
	}
	return queries
}

func deseedVerb(driver string) string {
	switch driver {
	case "pgx", "dsql", "oracle":
		return "truncate"
	case "spanner":
		return "delete"
	default:
		return "delete"
	}
}

func buildDown(tables []Table, driver string) []initQuery {
	queries := make([]initQuery, 0, len(tables))
	for i := len(tables) - 1; i >= 0; i-- {
		t := tables[i]
		q := initQuery{
			Name: "drop_" + t.Name,
			Type: "exec",
		}
		switch driver {
		case "oracle":
			q.Query = fmt.Sprintf(
				"BEGIN\n  EXECUTE IMMEDIATE 'DROP TABLE %s CASCADE CONSTRAINTS PURGE';\nEXCEPTION WHEN OTHERS THEN\n  IF SQLCODE != -942 THEN RAISE; END IF;\nEND;",
				t.Name)
		default:
			q.Query = "DROP TABLE IF EXISTS " + t.Name
		}
		queries = append(queries, q)
	}
	return queries
}

func wrapOracleCreate(stmt string) string {
	escaped := strings.ReplaceAll(stmt, "'", "''")
	return fmt.Sprintf(
		"BEGIN\n  EXECUTE IMMEDIATE '\n%s\n  ';\nEXCEPTION WHEN OTHERS THEN\n  IF SQLCODE != -955 THEN RAISE; END IF;\nEND;",
		indentLines(escaped, "    "))
}

func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// seedableColumns returns columns that need explicit values during seed
// (skipping auto-generated columns).
func seedableColumns(t Table) []Column {
	var cols []Column
	for _, col := range t.Columns {
		if col.IsGenerated || isAutoDefault(col.Default) {
			continue
		}
		cols = append(cols, col)
	}
	return cols
}

func isAutoDefault(def string) bool {
	if def == "" {
		return false
	}
	d := strings.ToLower(def)
	autoPatterns := []string{
		"gen_random_uuid()", "uuid()", "newid()", "unique_rowid()",
		"now()", "current_timestamp", "getdate()", "systimestamp",
		"nextval(",
	}
	for _, p := range autoPatterns {
		if strings.Contains(d, p) {
			return true
		}
	}
	return false
}

func exprForColumn(col Column) string {
	dt := strings.ToLower(col.DataType)

	// UUID types.
	if containsAny(dt, "uuid", "uniqueidentifier") {
		return "uuid_v4()"
	}

	// Boolean.
	if containsAny(dt, "bool", "bit") {
		return "gen('bool')"
	}

	// Decimal / float (check before int since DECIMAL doesn't contain "int").
	if containsAny(dt, "decimal", "numeric", "float", "double", "real") ||
		(strings.Contains(dt, "number") && strings.Contains(dt, ",")) {
		return "uniform(1.0, 100.0)"
	}

	// Integer.
	if containsAny(dt, "int", "serial") || dt == "number" ||
		(strings.HasPrefix(dt, "number(") && !strings.Contains(dt, ",")) {
		if col.CheckMin != nil && col.CheckMax != nil {
			return fmt.Sprintf("uniform(%d, %d)", *col.CheckMin, *col.CheckMax)
		}
		return "uniform(1, 1000)"
	}

	// String.
	if containsAny(dt, "char", "text", "string", "clob", "nvarchar", "varchar") {
		return "gen('sentence:3')"
	}

	// Date / time.
	if containsAny(dt, "timestamp", "datetime", "date", "time") {
		return "gen('date')"
	}

	// Binary.
	if containsAny(dt, "blob", "bytea", "binary", "varbinary", "raw", "bytes") {
		return "gen('sentence:3')"
	}

	// JSON.
	if containsAny(dt, "json") {
		return "gen('sentence:3')"
	}
	return "gen('sentence:3')"
}

var checkBetweenRe = regexp.MustCompile(`(?i)CHECK\s*\(\s*"?(\w+)"?\s+BETWEEN\s+(-?\d+)(?::+[\w]+)?\s+AND\s+(-?\d+)(?::+[\w]+)?\s*\)`)

// applyCheckBounds parses CHECK BETWEEN constraints from each table's DDL
// and sets CheckMin/CheckMax on the matching columns.
func applyCheckBounds(tables []Table) {
	for i := range tables {
		matches := checkBetweenRe.FindAllStringSubmatch(tables[i].CreateStmt, -1)
		for _, m := range matches {
			colName := strings.ToLower(m[1])
			min, err1 := strconv.ParseInt(m[2], 10, 64)
			max, err2 := strconv.ParseInt(m[3], 10, 64)
			if err1 != nil || err2 != nil {
				continue
			}
			for j := range tables[i].Columns {
				if strings.ToLower(tables[i].Columns[j].Name) == colName {
					tables[i].Columns[j].CheckMin = &min
					tables[i].Columns[j].CheckMax = &max
					break
				}
			}
		}
	}
}

func placeholder(driver string, n int) string {
	switch driver {
	case "pgx", "dsql":
		return fmt.Sprintf("$%d", n)
	case "mysql":
		return "?"
	case "mssql", "spanner":
		return fmt.Sprintf("@p%d", n)
	case "oracle":
		return fmt.Sprintf(":%d", n)
	default:
		return fmt.Sprintf("$%d", n)
	}
}

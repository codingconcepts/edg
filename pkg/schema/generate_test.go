package schema

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerate_BasicTable(t *testing.T) {
	tables := []Table{
		{
			Name:       "users",
			CreateStmt: "CREATE TABLE users (id UUID PRIMARY KEY, name VARCHAR(255) NOT NULL)",
			Columns: []Column{
				{Name: "id", DataType: "UUID", IsPK: true},
				{Name: "name", DataType: "VARCHAR(255)"},
			},
		},
	}

	out := Generate(tables, "pgx")

	// Must be valid YAML.
	var cfg initConfig
	require.NoError(t, yaml.Unmarshal([]byte(out), &cfg))

	assert.Equal(t, 100, cfg.Globals.Rows)
	assert.Equal(t, 100, cfg.Globals.BatchSize)

	// Up.
	require.Len(t, cfg.Up, 1)
	assert.Equal(t, "create_users", cfg.Up[0].Name)
	assert.Contains(t, cfg.Up[0].Query, "CREATE TABLE users")

	// Seed.
	require.Len(t, cfg.Seed, 1)
	assert.Equal(t, "populate_users", cfg.Seed[0].Name)
	assert.Equal(t, "exec", cfg.Seed[0].Type)
	assert.Equal(t, "rows", cfg.Seed[0].Count)
	assert.Contains(t, cfg.Seed[0].Query, "INSERT INTO users")
	assert.Len(t, cfg.Seed[0].Args, 2)

	// Deseed.
	require.Len(t, cfg.Deseed, 1)
	assert.Equal(t, "truncate_users", cfg.Deseed[0].Name)

	// Down.
	require.Len(t, cfg.Down, 1)
	assert.Equal(t, "drop_users", cfg.Down[0].Name)
	assert.Contains(t, cfg.Down[0].Query, "DROP TABLE IF EXISTS users")
}

func TestGenerate_Oracle(t *testing.T) {
	tables := []Table{
		{
			Name:       "items",
			CreateStmt: "CREATE TABLE items (id NUMBER(10) PRIMARY KEY)",
			Columns: []Column{
				{Name: "id", DataType: "NUMBER(10)", IsPK: true},
			},
		},
	}

	out := Generate(tables, "oracle")

	var cfg initConfig
	require.NoError(t, yaml.Unmarshal([]byte(out), &cfg))

	// Up wraps in BEGIN/EXCEPTION block.
	require.Len(t, cfg.Up, 1)
	assert.Equal(t, "exec", cfg.Up[0].Type)
	assert.Contains(t, cfg.Up[0].Query, "BEGIN")
	assert.Contains(t, cfg.Up[0].Query, "EXECUTE IMMEDIATE")
	assert.Contains(t, cfg.Up[0].Query, "SQLCODE != -955")

	// Down also wraps.
	require.Len(t, cfg.Down, 1)
	assert.Contains(t, cfg.Down[0].Query, "CASCADE CONSTRAINTS PURGE")

	// Seed uses Oracle placeholders.
	require.Len(t, cfg.Seed, 1)
	assert.Contains(t, cfg.Seed[0].Query, ":1")
}

func TestGenerate_EmptySeed(t *testing.T) {
	tables := []Table{
		{
			Name:       "audit_log",
			CreateStmt: "CREATE TABLE audit_log (id INT)",
			Columns: []Column{
				{Name: "id", DataType: "INT", IsGenerated: true},
			},
		},
	}

	out := Generate(tables, "pgx")

	var cfg initConfig
	require.NoError(t, yaml.Unmarshal([]byte(out), &cfg))

	// All columns are generated → seed section omitted.
	assert.Empty(t, cfg.Seed)
}

func TestBuildUp(t *testing.T) {
	tables := []Table{
		{Name: "a", CreateStmt: "CREATE TABLE a (id INT)"},
		{Name: "b", CreateStmt: "CREATE TABLE b (id INT)"},
	}

	queries := buildUp(tables, "pgx")
	require.Len(t, queries, 2)
	assert.Equal(t, "create_a", queries[0].Name)
	assert.Equal(t, "CREATE TABLE a (id INT)", queries[0].Query)
	assert.Empty(t, queries[0].Type)
}

func TestBuildUp_Oracle(t *testing.T) {
	tables := []Table{
		{Name: "t", CreateStmt: "CREATE TABLE t (id NUMBER)"},
	}

	queries := buildUp(tables, "oracle")
	require.Len(t, queries, 1)
	assert.Equal(t, "exec", queries[0].Type)
	assert.Contains(t, queries[0].Query, "EXECUTE IMMEDIATE")
}

func TestBuildSeed_Placeholders(t *testing.T) {
	tables := []Table{
		{Name: "t", Columns: []Column{
			{Name: "a", DataType: "INT"},
			{Name: "b", DataType: "INT"},
		}},
	}

	tests := []struct {
		driver       string
		placeholders []string
	}{
		{"pgx", []string{"$1", "$2"}},
		{"dsql", []string{"$1", "$2"}},
		{"mysql", []string{"?", "?"}},
		{"mssql", []string{"@p1", "@p2"}},
		{"oracle", []string{":1", ":2"}},
	}

	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			queries := buildSeed(tables, tt.driver)
			require.Len(t, queries, 1)
			for _, ph := range tt.placeholders {
				assert.Contains(t, queries[0].Query, ph)
			}
		})
	}
}

func TestBuildSeed_SkipsAutoColumns(t *testing.T) {
	tables := []Table{
		{Name: "t", Columns: []Column{
			{Name: "id", DataType: "INT", IsGenerated: true},
			{Name: "uuid_col", DataType: "UUID", Default: "gen_random_uuid()"},
			{Name: "created", DataType: "TIMESTAMP", Default: "now()"},
			{Name: "name", DataType: "VARCHAR(255)"},
		}},
	}

	queries := buildSeed(tables, "pgx")
	require.Len(t, queries, 1)
	assert.Len(t, queries[0].Args, 1)
	assert.Contains(t, queries[0].Query, "name")
	assert.NotContains(t, queries[0].Query, "id,")
	assert.NotContains(t, queries[0].Query, "uuid_col")
	assert.NotContains(t, queries[0].Query, "created")
}

func TestBuildDeseed_ReverseOrder(t *testing.T) {
	tables := []Table{
		{Name: "parent"},
		{Name: "child"},
	}

	queries := buildDeseed(tables, "pgx")
	require.Len(t, queries, 2)
	assert.Equal(t, "truncate_child", queries[0].Name)
	assert.Equal(t, "truncate_parent", queries[1].Name)
}

func TestBuildDeseed_Drivers(t *testing.T) {
	tables := []Table{{Name: "t"}}

	tests := []struct {
		driver string
		verb   string
		query  string
	}{
		{"pgx", "truncate", "TRUNCATE TABLE t CASCADE"},
		{"dsql", "truncate", "TRUNCATE TABLE t CASCADE"},
		{"mysql", "delete", "DELETE FROM t"},
		{"mssql", "delete", "DELETE FROM t"},
		{"oracle", "truncate", "TRUNCATE TABLE t"},
	}

	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			queries := buildDeseed(tables, tt.driver)
			require.Len(t, queries, 1)
			assert.Equal(t, tt.verb+"_t", queries[0].Name)
			assert.Equal(t, tt.query, queries[0].Query)
		})
	}
}

func TestBuildDeseed_CascadeOnlyOnRoot(t *testing.T) {
	tables := []Table{
		{Name: "parent"},
		{Name: "child"},
		{Name: "grandchild"},
	}

	queries := buildDeseed(tables, "pgx")
	require.Len(t, queries, 3)

	// Only the root table (last in reverse = index 0 in original = parent) gets CASCADE.
	for _, q := range queries {
		if q.Name == "truncate_parent" {
			assert.Contains(t, q.Query, "CASCADE")
		} else {
			assert.NotContains(t, q.Query, "CASCADE")
		}
	}
}

func TestBuildDown_ReverseOrder(t *testing.T) {
	tables := []Table{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
	}

	queries := buildDown(tables, "pgx")
	require.Len(t, queries, 3)
	assert.Equal(t, "drop_c", queries[0].Name)
	assert.Equal(t, "drop_b", queries[1].Name)
	assert.Equal(t, "drop_a", queries[2].Name)
}

func TestBuildDown_Oracle(t *testing.T) {
	tables := []Table{{Name: "t"}}

	queries := buildDown(tables, "oracle")
	require.Len(t, queries, 1)
	assert.Contains(t, queries[0].Query, "CASCADE CONSTRAINTS PURGE")
	assert.Contains(t, queries[0].Query, "SQLCODE != -942")
}

func TestExprForColumn(t *testing.T) {
	tests := []struct {
		dataType string
		want     string
	}{
		{"UUID", "uuid_v4()"},
		{"uniqueidentifier", "uuid_v4()"},
		{"BOOL", "gen('bool')"},
		{"BIT", "gen('bool')"},
		{"DECIMAL(10,2)", "uniform(1.0, 100.0)"},
		{"FLOAT", "uniform(1.0, 100.0)"},
		{"DOUBLE PRECISION", "uniform(1.0, 100.0)"},
		{"REAL", "uniform(1.0, 100.0)"},
		{"NUMBER(10,2)", "uniform(1.0, 100.0)"},
		{"INT", "uniform(1, 1000)"},
		{"BIGINT", "uniform(1, 1000)"},
		{"SERIAL", "uniform(1, 1000)"},
		{"NUMBER", "uniform(1, 1000)"},
		{"NUMBER(10)", "uniform(1, 1000)"},
		{"VARCHAR(255)", "gen('sentence:3')"},
		{"TEXT", "gen('sentence:3')"},
		{"NVARCHAR(100)", "gen('sentence:3')"},
		{"CLOB", "gen('sentence:3')"},
		{"TIMESTAMP", "gen('date')"},
		{"DATETIME", "gen('date')"},
		{"DATE", "gen('date')"},
		{"TIMESTAMPTZ", "gen('date')"},
		{"BYTEA", "gen('sentence:3')"},
		{"BLOB", "gen('sentence:3')"},
		{"VARBINARY(256)", "gen('sentence:3')"},
		{"RAW(16)", "gen('sentence:3')"},
		{"JSONB", "gen('sentence:3')"},
		{"JSON", "gen('sentence:3')"},
		{"SOMETHING_ELSE", "gen('sentence:3')"},
	}

	for _, tt := range tests {
		t.Run(tt.dataType, func(t *testing.T) {
			col := Column{Name: "x", DataType: tt.dataType}
			assert.Equal(t, tt.want, exprForColumn(col))
		})
	}
}

func TestApplyCheckBounds(t *testing.T) {
	tests := []struct {
		name      string
		ddl       string
		col       string
		wantMin   int64
		wantMax   int64
		wantFound bool
	}{
		{
			name:      "standard BETWEEN",
			ddl:       "CREATE TABLE t (rating INT NOT NULL CHECK (rating BETWEEN 1 AND 5))",
			col:       "rating",
			wantMin:   1,
			wantMax:   5,
			wantFound: true,
		},
		{
			name:      "CockroachDB type cast",
			ddl:       "CREATE TABLE t (rating INT8 NOT NULL, CONSTRAINT check_rating CHECK (rating BETWEEN 1:::INT8 AND 5:::INT8))",
			col:       "rating",
			wantMin:   1,
			wantMax:   5,
			wantFound: true,
		},
		{
			name:      "quoted column",
			ddl:       `CREATE TABLE t ("score" INT CHECK ("score" BETWEEN 0 AND 100))`,
			col:       "score",
			wantMin:   0,
			wantMax:   100,
			wantFound: true,
		},
		{
			name:      "no CHECK constraint",
			ddl:       "CREATE TABLE t (rating INT NOT NULL)",
			col:       "rating",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tables := []Table{{
				Name:       "t",
				CreateStmt: tt.ddl,
				Columns:    []Column{{Name: tt.col, DataType: "INT"}},
			}}

			applyCheckBounds(tables)

			col := tables[0].Columns[0]
			if !tt.wantFound {
				assert.Nil(t, col.CheckMin)
				assert.Nil(t, col.CheckMax)
				return
			}
			require.NotNil(t, col.CheckMin)
			require.NotNil(t, col.CheckMax)
			assert.Equal(t, tt.wantMin, *col.CheckMin)
			assert.Equal(t, tt.wantMax, *col.CheckMax)
		})
	}
}

func TestExprForColumn_WithCheckBounds(t *testing.T) {
	min, max := int64(1), int64(5)
	col := Column{Name: "rating", DataType: "INT", CheckMin: &min, CheckMax: &max}
	assert.Equal(t, "uniform(1, 5)", exprForColumn(col))
}

func TestSeedableColumns(t *testing.T) {
	tbl := Table{
		Columns: []Column{
			{Name: "id", IsGenerated: true},
			{Name: "uuid", Default: "gen_random_uuid()"},
			{Name: "created", Default: "now()"},
			{Name: "seq", Default: "nextval('my_seq')"},
			{Name: "name", DataType: "VARCHAR(255)"},
			{Name: "age", DataType: "INT", Default: "0"},
		},
	}

	cols := seedableColumns(tbl)
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.Name
	}
	assert.Equal(t, []string{"name", "age"}, names)
}

func TestIsAutoDefault(t *testing.T) {
	auto := []string{
		"gen_random_uuid()",
		"uuid()",
		"newid()",
		"unique_rowid()",
		"now()",
		"current_timestamp",
		"CURRENT_TIMESTAMP",
		"getdate()",
		"systimestamp",
		"nextval('my_seq'::regclass)",
	}
	for _, d := range auto {
		assert.True(t, isAutoDefault(d), "expected auto: %s", d)
	}

	notAuto := []string{
		"",
		"0",
		"42",
		"'hello'",
		"true",
	}
	for _, d := range notAuto {
		assert.False(t, isAutoDefault(d), "expected not auto: %s", d)
	}
}

func TestPlaceholder(t *testing.T) {
	assert.Equal(t, "$1", placeholder("pgx", 1))
	assert.Equal(t, "$3", placeholder("dsql", 3))
	assert.Equal(t, "?", placeholder("mysql", 1))
	assert.Equal(t, "?", placeholder("mysql", 5))
	assert.Equal(t, "@p2", placeholder("mssql", 2))
	assert.Equal(t, ":1", placeholder("oracle", 1))
	assert.Equal(t, "$1", placeholder("unknown", 1))
}

func TestWrapOracleCreate(t *testing.T) {
	stmt := "CREATE TABLE t (id NUMBER PRIMARY KEY, name VARCHAR2(100) DEFAULT 'foo')"
	wrapped := wrapOracleCreate(stmt)

	assert.True(t, strings.HasPrefix(wrapped, "BEGIN"))
	assert.Contains(t, wrapped, "EXECUTE IMMEDIATE")
	assert.Contains(t, wrapped, "SQLCODE != -955")
	// Single quotes in the DDL should be escaped.
	assert.Contains(t, wrapped, "''foo''")
}

func TestIndentLines(t *testing.T) {
	input := "line1\nline2\n\nline4"
	got := indentLines(input, "  ")
	assert.Equal(t, "  line1\n  line2\n\n  line4", got)
}

func TestDeseedVerb(t *testing.T) {
	assert.Equal(t, "truncate", deseedVerb("pgx"))
	assert.Equal(t, "truncate", deseedVerb("dsql"))
	assert.Equal(t, "truncate", deseedVerb("oracle"))
	assert.Equal(t, "delete", deseedVerb("mysql"))
	assert.Equal(t, "delete", deseedVerb("mssql"))
}

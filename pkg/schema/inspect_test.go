package schema

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPgxBuildType(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		charMax  sql.NullInt64
		numPrec  sql.NullInt64
		numScale sql.NullInt64
		want     string
	}{
		{"integer", "integer", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "INT"},
		{"bigint", "bigint", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "BIGINT"},
		{"smallint", "smallint", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "SMALLINT"},
		{"varchar with len", "character varying", sql.NullInt64{Int64: 255, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "VARCHAR(255)"},
		{"varchar no len", "character varying", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "STRING"},
		{"char with len", "character", sql.NullInt64{Int64: 1, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "CHAR(1)"},
		{"char no len", "character", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "CHAR"},
		{"text", "text", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "STRING"},
		{"uuid", "uuid", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "UUID"},
		{"boolean", "boolean", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "BOOL"},
		{"numeric with prec/scale", "numeric", sql.NullInt64{}, sql.NullInt64{Int64: 10, Valid: true}, sql.NullInt64{Int64: 2, Valid: true}, "DECIMAL(10,2)"},
		{"numeric no prec", "numeric", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "DECIMAL"},
		{"double precision", "double precision", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "FLOAT8"},
		{"real", "real", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "FLOAT4"},
		{"timestamp", "timestamp without time zone", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "TIMESTAMP"},
		{"timestamptz", "timestamp with time zone", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "TIMESTAMPTZ"},
		{"date", "date", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "DATE"},
		{"bytea", "bytea", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "BYTES"},
		{"jsonb", "jsonb", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "JSONB"},
		{"json", "json", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "JSON"},
		{"unknown uppercased", "citext", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "CITEXT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pgxBuildType(tt.dataType, tt.charMax, tt.numPrec, tt.numScale)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPgxCleanDefault(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"42", "42"},
		{"'hello'::character varying", "'hello'"},
		{"nextval('seq'::regclass)", "nextval('seq'"},
		{"gen_random_uuid()", "gen_random_uuid()"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, pgxCleanDefault(tt.input))
		})
	}
}

func TestMssqlBuildType(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		charMax  sql.NullInt64
		numPrec  sql.NullInt64
		numScale sql.NullInt64
		want     string
	}{
		{"int", "int", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "INT"},
		{"bigint", "bigint", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "BIGINT"},
		{"smallint", "smallint", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "SMALLINT"},
		{"tinyint", "tinyint", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "TINYINT"},
		{"bit", "bit", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "BIT"},
		{"nvarchar with len", "nvarchar", sql.NullInt64{Int64: 100, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "NVARCHAR(100)"},
		{"nvarchar max", "nvarchar", sql.NullInt64{Int64: -1, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "NVARCHAR(MAX)"},
		{"nvarchar no len", "nvarchar", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "NVARCHAR(255)"},
		{"varchar with len", "varchar", sql.NullInt64{Int64: 50, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "VARCHAR(50)"},
		{"varchar max", "varchar", sql.NullInt64{Int64: -1, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "VARCHAR(MAX)"},
		{"varchar no len", "varchar", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "VARCHAR(255)"},
		{"nchar with len", "nchar", sql.NullInt64{Int64: 10, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "NCHAR(10)"},
		{"nchar no len", "nchar", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "NCHAR(1)"},
		{"char with len", "char", sql.NullInt64{Int64: 5, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "CHAR(5)"},
		{"char no len", "char", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "CHAR(1)"},
		{"uniqueidentifier", "uniqueidentifier", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "UNIQUEIDENTIFIER"},
		{"datetime2", "datetime2", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "DATETIME2"},
		{"datetime", "datetime", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "DATETIME"},
		{"date", "date", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "DATE"},
		{"decimal", "decimal", sql.NullInt64{}, sql.NullInt64{Int64: 18, Valid: true}, sql.NullInt64{Int64: 4, Valid: true}, "DECIMAL(18,4)"},
		{"decimal no prec", "decimal", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "DECIMAL"},
		{"float", "float", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "FLOAT"},
		{"real", "real", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "REAL"},
		{"varbinary with len", "varbinary", sql.NullInt64{Int64: 256, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "VARBINARY(256)"},
		{"varbinary max", "varbinary", sql.NullInt64{Int64: -1, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "VARBINARY(MAX)"},
		{"varbinary no len", "varbinary", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "VARBINARY"},
		{"ntext", "ntext", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "NVARCHAR(MAX)"},
		{"unknown uppercased", "xml", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "XML"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mssqlBuildType(tt.dataType, tt.charMax, tt.numPrec, tt.numScale)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMssqlCleanDefault(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"null", "NULL", ""},
		{"null mixed case", "Null", ""},
		{"single wrapped int", "(0)", "0"},
		{"double wrapped int", "((0))", "0"},
		{"wrapped getdate", "(getdate())", "getdate()"},
		{"wrapped newid", "(newid())", "newid()"},
		{"no parens", "42", "42"},
		{"unbalanced outer", "(a)(b)", "(a)(b)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mssqlCleanDefault(tt.input))
		})
	}
}

func TestMssqlDDLType(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		maxLen   int
		prec     int
		scale    int
		want     string
	}{
		{"nvarchar max", "nvarchar", -1, 0, 0, "NVARCHAR(MAX)"},
		{"nvarchar 200 bytes", "nvarchar", 200, 0, 0, "NVARCHAR(100)"},
		{"nchar 40 bytes", "nchar", 40, 0, 0, "NCHAR(20)"},
		{"varchar max", "varchar", -1, 0, 0, "VARCHAR(MAX)"},
		{"varchar 100", "varchar", 100, 0, 0, "VARCHAR(100)"},
		{"char 10", "char", 10, 0, 0, "CHAR(10)"},
		{"varbinary max", "varbinary", -1, 0, 0, "VARBINARY(MAX)"},
		{"varbinary 512", "varbinary", 512, 0, 0, "VARBINARY(512)"},
		{"binary 16", "binary", 16, 0, 0, "BINARY(16)"},
		{"decimal", "decimal", 0, 18, 4, "DECIMAL(18,4)"},
		{"numeric", "numeric", 0, 10, 2, "NUMERIC(10,2)"},
		{"int passthrough", "int", 4, 10, 0, "INT"},
		{"bigint passthrough", "bigint", 8, 19, 0, "BIGINT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mssqlDDLType(tt.dataType, tt.maxLen, tt.prec, tt.scale)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOracleBuildType(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		charLen  sql.NullInt64
		prec     sql.NullInt64
		scale    sql.NullInt64
		want     string
	}{
		{"number no prec", "NUMBER", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "NUMBER"},
		{"number with prec", "NUMBER", sql.NullInt64{}, sql.NullInt64{Int64: 10, Valid: true}, sql.NullInt64{}, "NUMBER(10)"},
		{"number with prec and scale", "NUMBER", sql.NullInt64{}, sql.NullInt64{Int64: 10, Valid: true}, sql.NullInt64{Int64: 2, Valid: true}, "NUMBER(10,2)"},
		{"number scale zero", "NUMBER", sql.NullInt64{}, sql.NullInt64{Int64: 10, Valid: true}, sql.NullInt64{Int64: 0, Valid: true}, "NUMBER(10)"},
		{"varchar2", "VARCHAR2", sql.NullInt64{Int64: 100, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "VARCHAR2(100)"},
		{"varchar2 no len", "VARCHAR2", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "VARCHAR2(255)"},
		{"char", "CHAR", sql.NullInt64{Int64: 5, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "CHAR(5)"},
		{"char no len", "CHAR", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "CHAR(1)"},
		{"nvarchar2", "NVARCHAR2", sql.NullInt64{Int64: 50, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "NVARCHAR2(50)"},
		{"nvarchar2 no len", "NVARCHAR2", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "NVARCHAR2(255)"},
		{"clob", "CLOB", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "CLOB"},
		{"blob", "BLOB", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "BLOB"},
		{"date", "DATE", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "DATE"},
		{"raw", "RAW", sql.NullInt64{Int64: 32, Valid: true}, sql.NullInt64{}, sql.NullInt64{}, "RAW(32)"},
		{"raw no len", "RAW", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "RAW(16)"},
		{"timestamp prefix", "TIMESTAMP(6)", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "TIMESTAMP"},
		{"unknown uppercased", "xmltype", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "XMLTYPE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := oracleBuildType(tt.dataType, tt.charLen, tt.prec, tt.scale)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOracleCleanDefault(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"NULL", ""},
		{"null", ""},
		{"0", "0"},
		{"SYSTIMESTAMP", "SYSTIMESTAMP"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, oracleCleanDefault(tt.input))
		})
	}
}

func TestMysqlFormatDefault(t *testing.T) {
	tests := []struct {
		name    string
		def     string
		colType string
		extra   string
		want    string
	}{
		{"empty", "", "int", "", ""},
		{"null", "NULL", "int", "", ""},
		{"null mixed case", "Null", "varchar(255)", "", ""},
		{"int default", "42", "int", "", "42"},
		{"string default", "hello", "varchar(255)", "", "'hello'"},
		{"text default", "world", "text", "", "'world'"},
		{"enum default", "active", "enum('active','inactive')", "", "'active'"},
		{"current_timestamp", "CURRENT_TIMESTAMP", "datetime", "", "CURRENT_TIMESTAMP"},
		{"function call", "uuid()", "char(36)", "", "uuid()"},
		{"expression default", "uuid()", "char(36)", "DEFAULT_GENERATED", "(uuid())"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mysqlFormatDefault(tt.def, tt.colType, tt.extra)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildResult(t *testing.T) {
	tableMap := map[string]*Table{
		"a": {Name: "a"},
		"b": {Name: "b"},
		"c": {Name: "c"},
	}
	order := []string{"c", "a", "b"}

	result := buildResult(tableMap, order)
	assert.Equal(t, []string{"c", "a", "b"}, []string{result[0].Name, result[1].Name, result[2].Name})
}

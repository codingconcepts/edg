package schema

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

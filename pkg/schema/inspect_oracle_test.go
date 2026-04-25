package schema

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

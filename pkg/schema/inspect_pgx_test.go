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

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

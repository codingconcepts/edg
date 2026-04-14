package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstant(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"string", "hello"},
		{"int", 42},
		{"nil", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Constant(tt.input)
			assert.Equal(t, tt.input, got)
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"number:1,10", "{number:1,10}"},
		{"{number:1,10}", "{number:1,10}"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Wrap(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		want   float64
		expErr string
	}{
		{"float64", 3.14, 3.14, ""},
		{"int", 42, 42.0, ""},
		{"int64", int64(99), 99.0, ""},
		{"unsupported", "hello", 0, `cannot convert "hello" to float: strconv.ParseFloat: parsing "hello": invalid syntax`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToFloat(tt.input)
			if tt.expErr != "" {
				require.EqualError(t, err, tt.expErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBatch(t *testing.T) {
	result, err := Batch(3)
	require.NoError(t, err)
	require.Len(t, result, 3)
	for i, row := range result {
		require.Len(t, row, 1)
		assert.Equal(t, i, row[0])
	}
}

func TestBatch_Zero(t *testing.T) {
	result, err := Batch(0)
	require.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestCond(t *testing.T) {
	assert.Equal(t, "yes", Cond(true, "yes", "no"))
	assert.Equal(t, "no", Cond(false, "yes", "no"))
}

func TestCond_NonBool(t *testing.T) {
	// Non-bool predicate should return falseVal.
	assert.Equal(t, "no", Cond("truthy", "yes", "no"))
}

func TestNullable_AlwaysNull(t *testing.T) {
	// probability=1.0 should always return nil.
	for range 100 {
		got, err := Nullable("value", 1.0)
		require.NoError(t, err)
		require.Nil(t, got)
	}
}

func TestNullable_NeverNull(t *testing.T) {
	// probability=0.0 should always return the value.
	for range 100 {
		got, err := Nullable("value", 0.0)
		require.NoError(t, err)
		require.Equal(t, "value", got)
	}
}

func TestNullable_InvalidProbability(t *testing.T) {
	_, err := Nullable("value", "not_a_number")
	require.Error(t, err)
}

func TestCoalesce(t *testing.T) {
	assert.Equal(t, "first", Coalesce(nil, nil, "first", "second"))
}

func TestCoalesce_AllNil(t *testing.T) {
	assert.Nil(t, Coalesce(nil, nil))
}

func TestTemplate(t *testing.T) {
	got := Tmpl("ORD-%05d-%s", 42, "abc")
	assert.Equal(t, "ORD-00042-abc", got)
}

func TestSQLFormatValue(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		driver string
		want   string
	}{
		{"nil", nil, "", "NULL"},
		{"int", 42, "", "42"},
		{"int64", int64(9999999999), "", "9999999999"},
		{"float64", 3.14, "", "3.14"},
		{"string", "hello", "", "'hello'"},
		{"string with quote", "it's", "", "'it''s'"},
		{"bool", true, "", "'true'"},
		{"uuid string", "550e8400-e29b-41d4-a716-446655440000", "", "'550e8400-e29b-41d4-a716-446655440000'"},
		{"raw sql", RawSQL("NOW()"), "", "NOW()"},
		{"bytes", []byte{0xDE, 0xAD, 0xBE, 0xEF}, "", "X'deadbeef'"},
		{"bytes mssql", []byte{0xDE, 0xAD, 0xBE, 0xEF}, "mssql", "0xdeadbeef"},
		{"empty bytes", []byte{}, "", "X''"},
		{"empty bytes mssql", []byte{}, "mssql", "0x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SQLFormatValue(tt.input, tt.driver)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBatchFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"nil", nil, "NULL"},
		{"string", "hello", "hello"},
		{"string with quote", "it's", "it''s"},
		{"int", 42, "42"},
		{"int64", int64(9999999999), "9999999999"},
		{"float64", 3.14, "3.14"},
		{"bool", true, "true"},
		{"uuid string", "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000"},
		{"bytes", []byte{0xCA, 0xFE}, "cafe"},
		{"empty bytes", []byte{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BatchFormatValue(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBatchJoinJSON(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{"empty", []string{}, "[]"},
		{"single value", []string{"hello"}, `["hello"]`},
		{"multiple values", []string{"a", "b", "c"}, `["a","b","c"]`},
		{"with NULL", []string{"a", "NULL", "c"}, `["a",null,"c"]`},
		{"all NULL", []string{"NULL", "NULL"}, `[null,null]`},
		{"with quotes", []string{`he said "hi"`, "ok"}, `["he said \"hi\"","ok"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BatchJoinJSON(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func BenchmarkToInt(b *testing.B) {
	cases := []struct {
		name  string
		input any
	}{
		{"int", 42},
		{"float64", 42.0},
		{"int64", int64(42)},
		{"unsupported", "42"},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				_, _ = ToInt(tc.input)
			}
		})
	}
}

func BenchmarkWrap(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{"needs_wrap", "number:1,100"},
		{"already_wrapped", "{number:1,100}"},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				Wrap(tc.input)
			}
		})
	}
}

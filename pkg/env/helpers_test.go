package env

import (
	"os"
	"testing"

	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSep(t *testing.T) {
	cases := []struct {
		driver string
		exp    convert.RawSQL
	}{
		{"pgx", "chr(31)"},
		{"dsql", "chr(31)"},
		{"mysql", "CHAR(31)"},
		{"mssql", "CHAR(31)"},
		{"spanner", "CODE_POINTS_TO_STRING([31])"},
		{"oracle", "codepoints-to-string(31)"},
	}

	for _, c := range cases {
		t.Run(c.driver, func(t *testing.T) {
			e := &Env{driver: c.driver}
			assert.Equal(t, c.exp, e.sep())
		})
	}
}

func TestEnviron(t *testing.T) {
	cases := []struct {
		name      string
		envKeySet string
		envKeyGet string
		envVal    string
		expErr    string
	}{
		{
			name:      "missing env var",
			envKeySet: "ABC",
			envKeyGet: "DEF",
			envVal:    "123",
			expErr:    `missing environment variable: "DEF"`,
		},
		{
			name:      "valid env var",
			envKeySet: "ABC",
			envKeyGet: "ABC",
			envVal:    "123",
			expErr:    "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test.CleanupEnv(t, c.envKeySet)
			os.Setenv(c.envKeySet, c.envVal)

			act, err := environ(c.envKeyGet)

			if c.expErr != "" {
				require.EqualError(t, err, c.expErr)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, c.envVal, act)
		})
	}
}

func TestEnvironNil(t *testing.T) {
	cases := []struct {
		name      string
		envKeySet string
		envKeyGet string
		envVal    string
		exp       any
	}{
		{
			name:      "missing env var returns nil",
			envKeySet: "ABC",
			envKeyGet: "DEF",
			envVal:    "123",
			exp:       nil,
		},
		{
			name:      "valid env var returns string",
			envKeySet: "ABC",
			envKeyGet: "ABC",
			envVal:    "123",
			exp:       "123",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test.CleanupEnv(t, c.envKeySet)
			os.Setenv(c.envKeySet, c.envVal)

			act := environNil(c.envKeyGet)
			assert.Equal(t, c.exp, act)
		})
	}
}

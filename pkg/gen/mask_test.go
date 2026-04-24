package gen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMask_Deterministic(t *testing.T) {
	tests := []struct {
		name  string
		input any
		args  []any
	}{
		{"same string", "hello@example.com", nil},
		{"integer input", 42, nil},
		{"base64 mode", "hello", []any{"base64"}},
		{"email mode", "alice@corp.io", []any{"email", 6}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v1, err := Mask(tc.input, tc.args...)
			require.NoError(t, err)

			v2, err := Mask(tc.input, tc.args...)
			require.NoError(t, err)

			assert.Equal(t, v1, v2)
		})
	}
}

func TestMask_DifferentInputs(t *testing.T) {
	v1, err := Mask("alice@example.com")
	require.NoError(t, err)

	v2, err := Mask("bob@example.com")
	require.NoError(t, err)

	assert.NotEqual(t, v1, v2)
}

func TestMask_Length(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		args    []any
		wantLen int
	}{
		{"default", "test", nil, 16},
		{"custom 8", "test", []any{8}, 8},
		{"custom 32", "test", []any{32}, 32},
		{"float length", "test", []any{8.0}, 8},
		{"clamp zero to 1", "test", []any{0}, 1},
		{"clamp oversized to 64", "test", []any{999}, 64},
		{"non-string input", 42, nil, 16},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := Mask(tc.input, tc.args...)
			require.NoError(t, err)
			assert.Len(t, v, tc.wantLen)
		})
	}
}

func TestMask_Modes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		args    []any
		wantLen int
		wantVal string
	}{
		{"hex default", "test", []any{"hex"}, 16, ""},
		{"hex custom length", "test", []any{"hex", 8}, 8, ""},
		{"base64 default", "test", []any{"base64"}, 16, ""},
		{"base64 custom length", "test", []any{"base64", 8}, 8, ""},
		{"base32 default", "test", []any{"base32"}, 16, ""},
		{"base32 custom length", "test", []any{"base32", 8}, 8, ""},
		{"asterisk default", "test", []any{"asterisk"}, 16, "****************"},
		{"asterisk custom length", "test", []any{"asterisk", 5}, 5, "*****"},
		{"redact", "sensitive data", []any{"redact"}, 10, "[REDACTED]"},
		{"redact ignores length", "other data", []any{"redact", 99}, 10, "[REDACTED]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := Mask(tc.input, tc.args...)
			require.NoError(t, err)
			assert.Len(t, v, tc.wantLen)
			if tc.wantVal != "" {
				assert.Equal(t, tc.wantVal, v)
			}
		})
	}
}

func TestMask_EmailMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		args    []any
		wantVal string
	}{
		{"default length", "john@example.com", []any{"email"}, "****************@example.com"},
		{"custom length", "john@example.com", []any{"email", 4}, "****@example.com"},
		{"no @ sign", "not-an-email", []any{"email"}, "****************"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := Mask(tc.input, tc.args...)
			require.NoError(t, err)
			assert.Equal(t, tc.wantVal, v)
			if strings.Contains(tc.input, "@") {
				assert.True(t, strings.HasSuffix(v, "@example.com"))
			}
		})
	}
}

func TestMask_Errors(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		args      []any
		wantInErr string
	}{
		{"invalid single arg type", "test", []any{true}, "expected mode"},
		{"invalid mode arg type", "test", []any{8, "hex"}, "expected mode"},
		{"invalid length arg type", "test", []any{"hex", true}, "expected length"},
		{"unknown mode", "test", []any{"rot13"}, "unknown mode"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Mask(tc.input, tc.args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantInErr)
		})
	}
}

func TestMask_BackwardsCompat(t *testing.T) {
	v1, err := Mask("compat-test")
	require.NoError(t, err)

	v2, err := Mask("compat-test", 8)
	require.NoError(t, err)

	assert.Equal(t, v1[:8], v2, "shorter length should be prefix of default")
}

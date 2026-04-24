package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMask_Deterministic(t *testing.T) {
	v1, err := Mask("hello@example.com")
	require.NoError(t, err)

	v2, err := Mask("hello@example.com")
	require.NoError(t, err)

	assert.Equal(t, v1, v2, "same input should produce same output")
}

func TestMask_DifferentInputs(t *testing.T) {
	v1, err := Mask("alice@example.com")
	require.NoError(t, err)

	v2, err := Mask("bob@example.com")
	require.NoError(t, err)

	assert.NotEqual(t, v1, v2, "different inputs should produce different outputs")
}

func TestMask_DefaultLength(t *testing.T) {
	v, err := Mask("test")
	require.NoError(t, err)
	assert.Len(t, v, 16, "default mask length should be 16")
}

func TestMask_CustomLength(t *testing.T) {
	v, err := Mask("test", 8)
	require.NoError(t, err)
	assert.Len(t, v, 8, "mask with length 8")

	v2, err := Mask("test", 32)
	require.NoError(t, err)
	assert.Len(t, v2, 32, "mask with length 32")
}

func TestMask_CustomLengthFloat(t *testing.T) {
	v, err := Mask("test", 8.0)
	require.NoError(t, err)
	assert.Len(t, v, 8)
}

func TestMask_InvalidLengthType(t *testing.T) {
	_, err := Mask("test", "bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected int")
}

func TestMask_NonStringInput(t *testing.T) {
	v1, err := Mask(42)
	require.NoError(t, err)
	assert.Len(t, v1, 16)

	v2, err := Mask(42)
	require.NoError(t, err)
	assert.Equal(t, v1, v2, "integer input should be deterministic")
}

func TestMask_ClampLength(t *testing.T) {
	v, err := Mask("test", 0)
	require.NoError(t, err)
	assert.Len(t, v, 1, "length 0 should be clamped to 1")

	v2, err := Mask("test", 999)
	require.NoError(t, err)
	assert.Len(t, v2, 64, "length > hash should be clamped to hash length")
}

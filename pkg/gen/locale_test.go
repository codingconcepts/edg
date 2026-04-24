package gen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenLocale_AllFields(t *testing.T) {
	fields := []string{"first_name", "last_name", "name", "city", "street", "phone", "zip", "address"}
	for _, locale := range SupportedLocales() {
		for _, field := range fields {
			v, err := GenLocale(field, locale)
			require.NoError(t, err, "field=%s locale=%s", field, locale)
			assert.NotEmpty(t, v, "field=%s locale=%s", field, locale)
		}
	}
}

func TestGenLocale_FieldAliases(t *testing.T) {
	aliases := map[string]string{
		"firstname":    "first_name",
		"lastname":     "last_name",
		"full_name":    "name",
		"fullname":     "name",
		"street_name":  "street",
		"streetname":   "street",
		"phone_number": "phone",
		"phonenumber":  "phone",
		"zip_code":     "zip",
		"zipcode":      "zip",
		"postal_code":  "zip",
		"postalcode":   "zip",
	}

	for alias, canonical := range aliases {
		v1, err := GenLocale(alias, "en_US")
		require.NoError(t, err, "alias=%s", alias)
		assert.NotEmpty(t, v1, "alias=%s", alias)

		v2, err := GenLocale(canonical, "en_US")
		require.NoError(t, err, "canonical=%s", canonical)
		assert.NotEmpty(t, v2, "canonical=%s", canonical)
	}
}

func TestGenLocale_LocaleAliases(t *testing.T) {
	aliases := map[string]string{
		"ja": "ja_JP", "jp": "ja_JP",
		"de": "de_DE",
		"en": "en_US", "us": "en_US",
		"fr": "fr_FR",
	}
	for alias, canonical := range aliases {
		v, err := GenLocale("name", alias)
		require.NoError(t, err, "alias=%s", alias)
		assert.NotEmpty(t, v, "alias=%s", alias)

		v2, err := GenLocale("name", canonical)
		require.NoError(t, err, "canonical=%s", canonical)
		assert.NotEmpty(t, v2, "canonical=%s", canonical)
	}
}

func TestGenLocale_HyphenatedLocale(t *testing.T) {
	v, err := GenLocale("name", "ja-JP")
	require.NoError(t, err)
	assert.NotEmpty(t, v)
}

func TestGenLocale_EasternNameOrder(t *testing.T) {
	for range 100 {
		name, err := GenLocale("name", "ja_JP")
		require.NoError(t, err)
		assert.NotContains(t, name, " ", "Japanese names should not contain spaces")
	}
}

func TestGenLocale_WesternNameOrder(t *testing.T) {
	for range 100 {
		name, err := GenLocale("name", "en_US")
		require.NoError(t, err)
		parts := strings.SplitN(name, " ", 2)
		assert.Len(t, parts, 2, "Western names should have first and last: %q", name)
	}
}

func TestGenLocale_UnknownLocale(t *testing.T) {
	_, err := GenLocale("name", "xx_XX")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown locale")
}

func TestGenLocale_UnknownField(t *testing.T) {
	_, err := GenLocale("ssn", "en_US")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

func TestGenLocale_Variety(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		v, _ := GenLocale("first_name", "de_DE")
		seen[v] = true
	}
	assert.Greater(t, len(seen), 5, "expected variety in generated names, got %d unique", len(seen))
}

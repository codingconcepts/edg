package gen

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGen(t *testing.T) {
	result, err := Gen("number:1,100")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = Gen("{number:1,100}")
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGenBatch(t *testing.T) {
	result, err := GenBatch(25, 10, "email")
	require.NoError(t, err)
	require.Len(t, result, 3)

	// First two batches should have 10 emails each.
	for _, i := range []int{0, 1} {
		csv := string(result[i][0].(convert.RawSQL))
		parts := strings.Split(csv, convert.Sep)
		assert.Len(t, parts, 10, "batch %d has %d values, want 10", i, len(parts))
	}

	// Last batch should have 5 emails (remainder).
	csv := string(result[2][0].(convert.RawSQL))
	parts := strings.Split(csv, convert.Sep)
	assert.Len(t, parts, 5, "last batch has %d values, want 5", len(parts))

	// All emails across all batches should be unique.
	seen := map[string]bool{}
	for _, row := range result {
		for _, v := range strings.Split(string(row[0].(convert.RawSQL)), convert.Sep) {
			assert.False(t, seen[v], "GenBatch produced duplicate: %s", v)
			seen[v] = true
		}
	}
	assert.Equal(t, 25, len(seen), "GenBatch produced %d unique values, want 25", len(seen))
}

func TestGenBatch_ExactMultiple(t *testing.T) {
	result, err := GenBatch(20, 10, "email")
	require.NoError(t, err)
	require.Len(t, result, 2)
	for i, row := range result {
		parts := strings.Split(string(row[0].(convert.RawSQL)), convert.Sep)
		assert.Len(t, parts, 10, "batch %d has %d values, want 10", i, len(parts))
	}
}

func TestFloatRand(t *testing.T) {
	for range 1000 {
		v, err := FloatRand(1.0, 10.0, 2)
		require.NoError(t, err)
		require.True(t, v >= 1.0 && v <= 10.0, "FloatRand(1.0, 10.0, 2) = %v, out of range", v)
		scaled := v * 100
		require.True(t, math.Abs(scaled-math.Round(scaled)) <= 0.0001, "FloatRand precision 2: %v not rounded correctly", v)
	}
}

func TestUniformRand(t *testing.T) {
	for range 1000 {
		v, err := UniformRand(5.0, 15.0)
		require.NoError(t, err)
		require.True(t, v >= 5.0 && v < 15.0, "UniformRand(5.0, 15.0) = %v, out of range", v)
	}
}

func TestZipfRand(t *testing.T) {
	bins := make([]int, 100)
	for range 10000 {
		v, err := ZipfRand(2.0, 1.0, 99)
		require.NoError(t, err)
		require.True(t, v >= 0 && v <= 99, "ZipfRand out of range: %d", v)
		bins[v]++
	}
	assert.Greater(t, bins[0], bins[99], "ZipfRand not skewed: bin[0]=%d, bin[99]=%d", bins[0], bins[99])
}

func TestZipfRand_InvalidParams(t *testing.T) {
	v, err := ZipfRand(0.5, 1.0, 100)
	if err != nil {
		// Invalid params returning an error is acceptable.
		return
	}
	assert.Equal(t, uint64(0), v, "ZipfRand with s <= 1 = %d, want 0", v)
}

func TestGenRegex(t *testing.T) {
	result := GenRegex("[A-Z]{3}-[0-9]{4}")
	matched, _ := regexp.MatchString(`^[A-Z]{3}-[0-9]{4}$`, result)
	assert.True(t, matched, "GenRegex = %q, does not match pattern", result)
}

func TestJsonObj(t *testing.T) {
	result, err := JsonObj("name", "alice", "age", 30)
	require.NoError(t, err)

	var m map[string]any
	err = json.Unmarshal([]byte(result), &m)
	require.NoError(t, err)
	assert.Equal(t, "alice", m["name"])
	assert.Equal(t, float64(30), m["age"])
}

func TestJsonObj_OddArgs(t *testing.T) {
	_, err := JsonObj("key1", "val1", "key2")
	require.Error(t, err)
}

func TestJsonArr(t *testing.T) {
	result, err := JsonArr(3, 3, "email")
	require.NoError(t, err)

	var arr []any
	err = json.Unmarshal([]byte(result), &arr)
	require.NoError(t, err)
	assert.Len(t, arr, 3)
}

func TestGenPoint(t *testing.T) {
	centerLat := 51.5074
	centerLon := -0.1278
	radiusKM := 10.0

	for range 100 {
		p, err := GenPoint(centerLat, centerLon, radiusKM)
		require.NoError(t, err)
		lat := p["lat"].(float64)
		lon := p["lon"].(float64)

		dLat := (lat - centerLat) * math.Pi / 180
		dLon := (lon - centerLon) * math.Pi / 180
		a := math.Sin(dLat/2)*math.Sin(dLat/2) +
			math.Cos(centerLat*math.Pi/180)*math.Cos(lat*math.Pi/180)*
				math.Sin(dLon/2)*math.Sin(dLon/2)
		dist := 6371.0 * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
		require.True(t, dist <= radiusKM+0.01, "GenPoint distance %.4f km exceeds radius", dist)
	}
}

func TestGenPointWKT(t *testing.T) {
	centerLat := 51.5074
	centerLon := -0.1278
	radiusKM := 10.0

	for range 100 {
		wkt, err := GenPointWKT(centerLat, centerLon, radiusKM)
		require.NoError(t, err)

		var lon, lat float64
		_, err = fmt.Sscanf(wkt, "POINT(%f %f)", &lon, &lat)
		require.NoError(t, err, "GenPointWKT returned invalid WKT %q", wkt)

		dLat := (lat - centerLat) * math.Pi / 180
		dLon := (lon - centerLon) * math.Pi / 180
		a := math.Sin(dLat/2)*math.Sin(dLat/2) +
			math.Cos(centerLat*math.Pi/180)*math.Cos(lat*math.Pi/180)*
				math.Sin(dLon/2)*math.Sin(dLon/2)
		dist := 6371.0 * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
		require.True(t, dist <= radiusKM+0.01, "GenPointWKT distance %.4f km exceeds radius", dist)
	}
}

func TestRandTimestamp(t *testing.T) {
	min := "2020-01-01T00:00:00Z"
	max := "2025-01-01T00:00:00Z"
	result, err := RandTimestamp(min, max)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	ts, err := time.Parse(time.RFC3339, result)
	require.NoError(t, err)
	minT, _ := time.Parse(time.RFC3339, min)
	maxT, _ := time.Parse(time.RFC3339, max)
	assert.True(t, !ts.Before(minT) && !ts.After(maxT), "RandTimestamp %v not in range [%v, %v]", ts, minT, maxT)
}

func TestRandTimestamp_InvalidInput(t *testing.T) {
	_, err := RandTimestamp("bad", "2025-01-01T00:00:00Z")
	require.Error(t, err)
}

func TestRandDuration(t *testing.T) {
	result, err := RandDuration("1h", "24h")
	require.NoError(t, err)
	require.NotEmpty(t, result)

	d, err := time.ParseDuration(result)
	require.NoError(t, err)
	assert.True(t, d >= time.Hour && d <= 24*time.Hour, "RandDuration %v not in range [1h, 24h]", d)
}

func TestRandDuration_InvalidInput(t *testing.T) {
	_, err := RandDuration("bad", "24h")
	require.Error(t, err)
}

func TestDateRand(t *testing.T) {
	result, err := DateRand("2006-01-02", "2020-01-01T00:00:00Z", "2025-01-01T00:00:00Z")
	require.NoError(t, err)
	require.NotEmpty(t, result)

	ts, err := time.Parse("2006-01-02", result)
	require.NoError(t, err)
	assert.True(t, ts.Year() >= 2020 && ts.Year() <= 2025, "DateRand year %d not in range [2020, 2025]", ts.Year())
}

func TestDateOffset(t *testing.T) {
	result, err := DateOffset("1h")
	require.NoError(t, err)
	require.NotEmpty(t, result)

	ts, err := time.Parse(time.RFC3339, result)
	require.NoError(t, err)

	expected := time.Now().Add(time.Hour).UTC()
	diff := ts.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	assert.True(t, diff <= 2*time.Second, "DateOffset('1h') = %v, expected ~%v", ts, expected)
}

func TestDateOffset_InvalidInput(t *testing.T) {
	_, err := DateOffset("bad")
	require.Error(t, err)
}

func TestExpRand(t *testing.T) {
	const (
		rate = 1.0
		min  = 0.0
		max  = 100.0
		n    = 10000
	)

	sum := 0.0
	for range n {
		v, err := ExpRand(rate, min, max)
		require.NoError(t, err)
		require.True(t, v >= min && v <= max, "ExpRand value %v outside [%.0f, %.0f]", v, min, max)
		sum += v
	}

	// Exponential with rate=1 has mean=1.
	observedMean := sum / n
	assert.True(t, observedMean >= 0.5 && observedMean <= 1.5, "observed mean = %.2f, want ~1.0", observedMean)
}

func TestExpRandF(t *testing.T) {
	for range 100 {
		v, err := ExpRandF(0.5, 0.0, 100.0, 2)
		require.NoError(t, err)
		require.True(t, v >= 0 && v <= 100, "ExpRandF value %v outside [0, 100]", v)
		scaled := v * 100
		require.True(t, math.Abs(scaled-math.Round(scaled)) <= 0.0001, "ExpRandF precision 2: %v not rounded correctly", v)
	}
}

func TestLognormRand(t *testing.T) {
	const (
		mu    = 3.0
		sigma = 0.5
		min   = 1.0
		max   = 500.0
		n     = 10000
	)

	for range n {
		v, err := LognormRand(mu, sigma, min, max)
		require.NoError(t, err)
		require.True(t, v >= min && v <= max, "LognormRand value %v outside [%.0f, %.0f]", v, min, max)
	}
}

func TestLognormRandF(t *testing.T) {
	for range 100 {
		v, err := LognormRandF(2.0, 0.5, 1.0, 100.0, 2)
		require.NoError(t, err)
		require.True(t, v >= 1 && v <= 100, "LognormRandF value %v outside [1, 100]", v)
		scaled := v * 100
		require.True(t, math.Abs(scaled-math.Round(scaled)) <= 0.0001, "LognormRandF precision 2: %v not rounded correctly", v)
	}
}

func TestGenBytes(t *testing.T) {
	result, err := GenBytes(8)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(result, `\x`), "GenBytes(8) = %q, want \\x prefix", result)
	require.Equal(t, 18, len(result), "GenBytes(8) length = %d, want 18", len(result)) // \x + 16 hex chars
}

func TestGenBlob(t *testing.T) {
	result, err := GenBlob(16)
	require.NoError(t, err)
	require.Len(t, result, 16)
}

func TestGenBit(t *testing.T) {
	result, err := GenBit(8)
	require.NoError(t, err)
	require.Equal(t, 8, len(result), "GenBit(8) length = %d, want 8", len(result))
	matched, _ := regexp.MatchString(`^[01]+$`, result)
	require.True(t, matched, "GenBit(8) = %q, contains non-bit characters", result)
}

func TestGenBool(t *testing.T) {
	trueCount := 0
	for range 1000 {
		if GenBool() {
			trueCount++
		}
	}
	require.True(t, trueCount > 0 && trueCount < 1000, "GenBool returned the same value 1000 times (true=%d)", trueCount)
}

func TestGenVarBit(t *testing.T) {
	for range 100 {
		result, err := GenVarBit(16)
		require.NoError(t, err)
		require.True(t, len(result) >= 1 && len(result) <= 16, "GenVarBit(16) length = %d, want 1-16", len(result))
	}
}

func TestGenInet(t *testing.T) {
	result, err := GenInet("10.0.0.0/8")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(result, "10."), "GenInet('10.0.0.0/8') = %q, want 10.x.x.x", result)
}

func TestGenInet_InvalidCIDR(t *testing.T) {
	_, err := GenInet("bad")
	require.Error(t, err)
}

func TestGenArray(t *testing.T) {
	result, err := GenArray(3, 3, "number:1,100")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(result, "{") && strings.HasSuffix(result, "}"), "GenArray = %q, want {}-wrapped", result)
	inner := result[1 : len(result)-1]
	parts := strings.Split(inner, ",")
	assert.Len(t, parts, 3, "GenArray(3,3) produced %d elements, want 3", len(parts))
}

func TestGenArray_Range(t *testing.T) {
	for range 100 {
		result, err := GenArray(2, 5, "letter")
		require.NoError(t, err)
		inner := result[1 : len(result)-1]
		parts := strings.Split(inner, ",")
		require.True(t, len(parts) >= 2 && len(parts) <= 5, "GenArray(2,5) produced %d elements, want 2-5", len(parts))
	}
}

func TestGenTime(t *testing.T) {
	result, err := GenTime("08:00:00", "17:00:00")
	require.NoError(t, err)
	require.True(t, result >= "08:00:00" && result <= "17:00:00", "GenTime = %q, out of range", result)
}

func TestGenTime_InvalidInput(t *testing.T) {
	_, err := GenTime("bad", "17:00:00")
	require.Error(t, err)
}

func TestGenTimez(t *testing.T) {
	result, err := GenTimez("08:00:00", "17:00:00")
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(result, "+00:00"), "GenTimez = %q, want +00:00 suffix", result)
	timePart := strings.TrimSuffix(result, "+00:00")
	require.True(t, timePart >= "08:00:00" && timePart <= "17:00:00", "GenTimez time part = %q, out of range", timePart)
}

func TestGenTimez_InvalidInput(t *testing.T) {
	_, err := GenTimez("bad", "17:00:00")
	require.Error(t, err)
}

func TestValidatePattern(t *testing.T) {
	valid := []string{"email", "number:1,100", "firstname", "sentence:5", "lastname"}
	for _, p := range valid {
		assert.NoError(t, ValidatePattern(p), "ValidatePattern(%q) unexpected error", p)
	}

	invalid := []string{"notafunction", "emaill", "nope:1,2"}
	for _, p := range invalid {
		assert.Error(t, ValidatePattern(p), "ValidatePattern(%q) expected error, got nil", p)
	}
}

func TestGenLtree(t *testing.T) {
	result, err := GenLtree("Top", "Science", "Astronomy")
	require.NoError(t, err)
	assert.Equal(t, "Top.Science.Astronomy", result)
}

func TestGenLtree_SinglePart(t *testing.T) {
	result, err := GenLtree("Root")
	require.NoError(t, err)
	assert.Equal(t, "Root", result)
}

func TestGenLtree_SkipsNilAndEmpty(t *testing.T) {
	result, err := GenLtree(nil, "Top", "", "Child")
	require.NoError(t, err)
	assert.Equal(t, "Top.Child", result)
}

func TestGenLtree_SanitizesLabels(t *testing.T) {
	result, err := GenLtree("Top", "next-gen", "my team")
	require.NoError(t, err)
	assert.Equal(t, "Top.next_gen.my_team", result)
}

func TestGenLtree_AllEmpty(t *testing.T) {
	_, err := GenLtree(nil, "")
	require.Error(t, err)
}

func BenchmarkGenBatch(b *testing.B) {
	cases := []struct {
		name  string
		total int
		batch int
	}{
		{"total_10/batch_10", 10, 10},
		{"total_100/batch_10", 100, 10},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				_, _ = GenBatch(tc.total, tc.batch, "number:1,1000")
			}
		})
	}
}

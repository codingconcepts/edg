package random

import (
	"math"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFloat(t *testing.T) {
	for range 1000 {
		v := Float(1.0, 10.0, 2)
		require.GreaterOrEqual(t, v, 1.0)
		require.LessOrEqual(t, v, 10.0)

		// Verify precision: multiply by 100, should be an integer.
		scaled := v * 100
		require.LessOrEqual(t, math.Abs(scaled-math.Round(scaled)), 0.0001, "not rounded to 2 decimal places: %v", v)
	}
}

func TestFloat_ZeroPrecision(t *testing.T) {
	for range 100 {
		v := Float(1.0, 10.0, 0)
		require.Equal(t, math.Round(v), v, "expected integer")
	}
}

func TestUniform(t *testing.T) {
	for range 1000 {
		v := Uniform(5.0, 15.0)
		require.GreaterOrEqual(t, v, 5.0)
		require.Less(t, v, 15.0)
	}
}

func TestZipf(t *testing.T) {
	bins := make([]int, 100)
	for range 10000 {
		v, err := Zipf(2.0, 1.0, 99)
		require.NoError(t, err)
		require.GreaterOrEqual(t, v, 0)
		require.LessOrEqual(t, v, 99)
		bins[v]++
	}

	// Zipfian: lower values should be much more frequent.
	assert.Greater(t, bins[0], bins[99], "Zipf not skewed")
}

func TestZipf_InvalidParams(t *testing.T) {
	// s <= 1 should return an error (NewZipf returns nil).
	_, err := Zipf(0.5, 1.0, 100)
	require.Error(t, err)
}

func TestNorm_DefaultPrecision(t *testing.T) {
	const (
		mean   = 50
		stddev = 10
		min    = 1.0
		max    = 100.0
		n      = 10000
	)

	sum := 0.0
	for range n {
		v, err := Norm(mean, stddev, min, max)
		require.NoError(t, err)
		require.GreaterOrEqual(t, v, min)
		require.LessOrEqual(t, v, max)
		// Default precision 0: should be a whole number.
		require.Equal(t, math.Round(v), v, "expected whole number")
		sum += v
	}

	observedMean := sum / n
	assert.GreaterOrEqual(t, observedMean, float64(mean-2))
	assert.LessOrEqual(t, observedMean, float64(mean+2))
}

func TestNorm_WithPrecision(t *testing.T) {
	for range 1000 {
		v, err := Norm(50, 10, 1, 100, 2)
		require.NoError(t, err)
		require.GreaterOrEqual(t, v, 1.0)
		require.LessOrEqual(t, v, 100.0)

		// Verify 2 decimal places.
		scaled := v * 100
		require.LessOrEqual(t, math.Abs(scaled-math.Round(scaled)), 0.0001, "not rounded to 2 decimal places: %v", v)
	}
}

func TestNorm_Distribution(t *testing.T) {
	const (
		mean   = 500.0
		stddev = 50.0
		min    = 1.0
		max    = 1000.0
		n      = 50000
	)

	within1 := 0
	within2 := 0

	for range n {
		v, err := Norm(mean, stddev, min, max)
		require.NoError(t, err)
		dist := math.Abs(v - mean)
		switch {
		case dist <= stddev:
			within1++
			within2++
		case dist <= 2*stddev:
			within2++
		}
	}

	pct1 := float64(within1) / n
	pct2 := float64(within2) / n

	assert.GreaterOrEqual(t, pct1, 0.65)
	assert.LessOrEqual(t, pct1, 0.71)
	assert.GreaterOrEqual(t, pct2, 0.93)
	assert.LessOrEqual(t, pct2, 0.97)
}

func TestExp_DefaultPrecision(t *testing.T) {
	const (
		rate = 1.0
		min  = 0.0
		max  = 100.0
		n    = 10000
	)

	sum := 0.0
	for range n {
		v, err := Exp(rate, min, max)
		require.NoError(t, err)
		require.GreaterOrEqual(t, v, min)
		require.LessOrEqual(t, v, max)
		// Default precision 0: should be a whole number.
		require.Equal(t, math.Round(v), v, "expected whole number")
		sum += v
	}

	// Exponential with rate=1 has mean=1, but clamped to [0,100] shouldn't
	// shift the mean much. Observed mean should be close to 1.
	observedMean := sum / n
	assert.GreaterOrEqual(t, observedMean, 0.5)
	assert.LessOrEqual(t, observedMean, 1.5)
}

func TestExp_WithPrecision(t *testing.T) {
	for range 1000 {
		v, err := Exp(0.5, 0, 100, 2)
		require.NoError(t, err)
		require.GreaterOrEqual(t, v, 0.0)
		require.LessOrEqual(t, v, 100.0)

		// Verify 2 decimal places.
		scaled := v * 100
		require.LessOrEqual(t, math.Abs(scaled-math.Round(scaled)), 0.0001, "not rounded to 2 decimal places: %v", v)
	}
}

func TestExp_Distribution(t *testing.T) {
	const (
		rate = 0.5
		min  = 0.0
		max  = 1000.0
		n    = 50000
	)

	// Exponential distribution: most values should be small.
	// With rate=0.5 and precision=0, values round to whole numbers.
	// "Rounded <= 2" captures continuous values in [0, 2.5),
	// so P ~ 1 - e^(-0.5*2.5) ~ 71.3%.
	belowMedian := 0
	for range n {
		v, err := Exp(rate, min, max)
		require.NoError(t, err)
		if v <= 2 {
			belowMedian++
		}
	}

	pct := float64(belowMedian) / n
	assert.GreaterOrEqual(t, pct, 0.67)
	assert.LessOrEqual(t, pct, 0.76)
}

func TestLogNorm_DefaultPrecision(t *testing.T) {
	const (
		mu    = 3.0
		sigma = 0.5
		min   = 1.0
		max   = 500.0
		n     = 10000
	)

	for range n {
		v, err := LogNorm(mu, sigma, min, max)
		require.NoError(t, err)
		require.GreaterOrEqual(t, v, min)
		require.LessOrEqual(t, v, max)
		// Default precision 0: should be a whole number.
		require.Equal(t, math.Round(v), v, "expected whole number")
	}
}

func TestLogNorm_WithPrecision(t *testing.T) {
	for range 1000 {
		v, err := LogNorm(2.0, 0.5, 1, 100, 2)
		require.NoError(t, err)
		require.GreaterOrEqual(t, v, 1.0)
		require.LessOrEqual(t, v, 100.0)

		scaled := v * 100
		require.LessOrEqual(t, math.Abs(scaled-math.Round(scaled)), 0.0001, "not rounded to 2 decimal places: %v", v)
	}
}

func TestLogNorm_Distribution(t *testing.T) {
	const (
		mu    = 3.0  // underlying normal mean
		sigma = 0.5  // underlying normal stddev
		min   = 1.0
		max   = 500.0
		n     = 50000
	)

	// For log-normal, the median is exp(mu) = exp(3) ~ 20.09.
	// Roughly 50% of values should be below the median.
	expectedMedian := math.Exp(mu)
	belowMedian := 0
	sum := 0.0

	for range n {
		v, err := LogNorm(mu, sigma, min, max)
		require.NoError(t, err)
		sum += v
		if v <= expectedMedian {
			belowMedian++
		}
	}

	pct := float64(belowMedian) / n
	assert.GreaterOrEqual(t, pct, 0.45)
	assert.LessOrEqual(t, pct, 0.55)

	// Mean of log-normal is exp(mu + sigma^2/2) ~ exp(3.125) ~ 22.76
	expectedMean := math.Exp(mu + sigma*sigma/2)
	observedMean := sum / n
	tolerance := expectedMean * 0.1
	assert.LessOrEqual(t, math.Abs(observedMean-expectedMean), tolerance)
}

func TestRegex(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
	}{
		{"product_code", "[A-Z]{3}-[0-9]{4}"},
		{"hex_color", "#[0-9a-f]{6}"},
		{"us_zip", "[0-9]{5}"},
		{"email_like", "[a-z]{5,10}@[a-z]{3,8}\\.(com|net|org)"},
		{"ip_octet", "(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)"},
		{"uuid_like", "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"},
		{"phone", "\\+1-[0-9]{3}-[0-9]{3}-[0-9]{4}"},
		{"alphanum", "[A-Za-z0-9]{8,16}"},
		{"single_char", "[aeiou]"},
		{"fixed_literal", "ABC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Regex(tt.pattern)
			matched, err := regexp.MatchString("^"+tt.pattern+"$", result)
			require.NoError(t, err, "invalid regex")
			assert.True(t, matched, "Regex(%q) = %q, does not match pattern", tt.pattern, result)
		})
	}
}

func TestPoint(t *testing.T) {
	centerLat := 51.5074
	centerLon := -0.1278
	radiusKM := 10.0

	for range 100 {
		lat, lon := Point(centerLat, centerLon, radiusKM)
		dist := haversineKM(centerLat, centerLon, lat, lon)
		require.LessOrEqual(t, dist, radiusKM+0.01, "point (%.6f, %.6f) distance %.4f km exceeds radius", lat, lon, dist)
	}
}

func TestPoint_ZeroRadius(t *testing.T) {
	lat, lon := Point(40.0, -74.0, 0)
	assert.InDelta(t, 40.0, lat, 0.0001)
	assert.InDelta(t, -74.0, lon, 0.0001)
}

func TestTimestamp(t *testing.T) {
	min := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	max := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for range 100 {
		ts := Timestamp(min, max)
		require.False(t, ts.Before(min) || ts.After(max), "timestamp not in range: %v", ts)
	}
}

func TestTimestamp_Equal(t *testing.T) {
	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	got := Timestamp(ts, ts)
	assert.True(t, got.Equal(ts), "want %v, got %v", ts, got)
}

func TestTimestamp_SwapsMinMax(t *testing.T) {
	min := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	max := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	ts := Timestamp(max, min)
	require.False(t, ts.Before(min) || ts.After(max), "timestamp with swapped args not in range: %v", ts)
}

func TestDuration(t *testing.T) {
	min := 1 * time.Hour
	max := 24 * time.Hour

	for range 100 {
		d := Duration(min, max)
		require.GreaterOrEqual(t, d, min)
		require.LessOrEqual(t, d, max)
	}
}

func TestDuration_Equal(t *testing.T) {
	d := 5 * time.Minute
	got := Duration(d, d)
	assert.Equal(t, d, got)
}

func TestDuration_SwapsMinMax(t *testing.T) {
	min := 1 * time.Hour
	max := 24 * time.Hour

	d := Duration(max, min)
	require.GreaterOrEqual(t, d, min)
	require.LessOrEqual(t, d, max)
}

func TestBytes(t *testing.T) {
	for range 100 {
		result := Bytes(16)
		require.True(t, strings.HasPrefix(result, `\x`), "want \\x prefix, got %q", result)
		// \x prefix + 32 hex chars = 34 total
		require.Equal(t, 34, len(result))
	}
}

func TestBytes_Zero(t *testing.T) {
	result := Bytes(0)
	assert.Equal(t, `\x`, result)
}

func TestBit(t *testing.T) {
	for range 100 {
		result := Bit(8)
		require.Equal(t, 8, len(result))
		for _, c := range result {
			require.True(t, c == '0' || c == '1', "contains non-bit character: %q", result)
		}
	}
}

func TestVarBit(t *testing.T) {
	for range 100 {
		result := VarBit(16)
		require.GreaterOrEqual(t, len(result), 1)
		require.LessOrEqual(t, len(result), 16)
		for _, c := range result {
			require.True(t, c == '0' || c == '1', "contains non-bit character: %q", result)
		}
	}
}

func TestVarBit_Zero(t *testing.T) {
	result := VarBit(0)
	assert.Equal(t, "", result)
}

func TestInet_IPv4(t *testing.T) {
	for range 100 {
		result, err := Inet("192.168.1.0/24")
		require.NoError(t, err)
		ip := net.ParseIP(result)
		require.NotNil(t, ip, "invalid IP: %q", result)
		_, network, _ := net.ParseCIDR("192.168.1.0/24")
		require.True(t, network.Contains(ip), "IP %v not in 192.168.1.0/24", result)
	}
}

func TestInet_IPv6(t *testing.T) {
	result, err := Inet("fd00::/64")
	require.NoError(t, err)
	ip := net.ParseIP(result)
	require.NotNil(t, ip, "invalid IP: %q", result)
	_, network, _ := net.ParseCIDR("fd00::/64")
	require.True(t, network.Contains(ip), "IP %v not in fd00::/64", result)
}

func TestInet_InvalidCIDR(t *testing.T) {
	_, err := Inet("not-a-cidr")
	require.Error(t, err)
}

func TestTimeOfDay(t *testing.T) {
	for range 100 {
		result, err := TimeOfDay("08:00:00", "17:30:00")
		require.NoError(t, err)
		require.Equal(t, 8, len(result), "want HH:MM:SS format, got %q", result)
		// Verify it parses back.
		_, err = time.Parse("15:04:05", result)
		require.NoError(t, err, "invalid time %q", result)
		require.GreaterOrEqual(t, result, "08:00:00")
		require.LessOrEqual(t, result, "17:30:00")
	}
}

func TestTimeOfDay_Equal(t *testing.T) {
	result, err := TimeOfDay("12:00:00", "12:00:00")
	require.NoError(t, err)
	assert.Equal(t, "12:00:00", result)
}

func TestTimeOfDay_SwapsMinMax(t *testing.T) {
	result, err := TimeOfDay("17:00:00", "08:00:00")
	require.NoError(t, err)
	require.GreaterOrEqual(t, result, "08:00:00")
	require.LessOrEqual(t, result, "17:00:00")
}

func TestTimeOfDay_InvalidInput(t *testing.T) {
	_, err := TimeOfDay("bad", "17:00:00")
	require.Error(t, err)
}

func TestSeed_Deterministic(t *testing.T) {
	// Same seed must produce the same sequence of values.
	Seed(12345)
	first := make([]float64, 10)
	for i := range first {
		first[i] = Uniform(0, 1000)
	}

	Seed(12345)
	for i := range first {
		v := Uniform(0, 1000)
		require.Equal(t, first[i], v, "index %d", i)
	}
}

func TestSeed_DifferentSeeds(t *testing.T) {
	Seed(1)
	a := Uniform(0, 1000)

	Seed(2)
	b := Uniform(0, 1000)

	assert.NotEqual(t, a, b, "different seeds produced the same value")
}

func TestSeed_Float(t *testing.T) {
	Seed(100)
	a := Float(1.0, 10.0, 2)

	Seed(100)
	b := Float(1.0, 10.0, 2)

	assert.Equal(t, a, b, "Float not deterministic")
}

func TestSeed_Norm(t *testing.T) {
	Seed(200)
	a, err := Norm(50, 10, 1, 100)
	require.NoError(t, err)

	Seed(200)
	b, err := Norm(50, 10, 1, 100)
	require.NoError(t, err)

	assert.Equal(t, a, b, "Norm not deterministic")
}

func TestSeed_Zipf(t *testing.T) {
	Seed(300)
	a, err := Zipf(2.0, 1.0, 99)
	require.NoError(t, err)

	Seed(300)
	b, err := Zipf(2.0, 1.0, 99)
	require.NoError(t, err)

	assert.Equal(t, a, b, "Zipf not deterministic")
}

func TestSeed_Timestamp(t *testing.T) {
	min := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	max := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	Seed(400)
	a := Timestamp(min, max)

	Seed(400)
	b := Timestamp(min, max)

	assert.True(t, a.Equal(b), "Timestamp not deterministic: %v != %v", a, b)
}

func TestSeed_Regex(t *testing.T) {
	Seed(500)
	a := Regex("[A-Z]{5}")

	Seed(500)
	b := Regex("[A-Z]{5}")

	assert.Equal(t, a, b, "Regex not deterministic")
}

func TestSeed_UUIDv4(t *testing.T) {
	Seed(600)
	a := UUIDv4()

	Seed(600)
	b := UUIDv4()

	assert.Equal(t, a, b, "UUIDv4 not deterministic")
}

func TestSeed_Bytes(t *testing.T) {
	Seed(700)
	a := Bytes(16)

	Seed(700)
	b := Bytes(16)

	assert.Equal(t, a, b, "Bytes not deterministic")
}

func TestSeed_Inet(t *testing.T) {
	Seed(800)
	a, err := Inet("10.0.0.0/8")
	require.NoError(t, err)

	Seed(800)
	b, err := Inet("10.0.0.0/8")
	require.NoError(t, err)

	assert.Equal(t, a, b, "Inet not deterministic")
}

func TestRngReader(t *testing.T) {
	Seed(42)
	rr := &rngReader{Rng}
	buf := make([]byte, 32)
	n, err := rr.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 32, n)

	// Re-seed and verify deterministic reads.
	Seed(42)
	rr2 := &rngReader{Rng}
	buf2 := make([]byte, 32)
	rr2.Read(buf2)

	require.Equal(t, buf, buf2, "rngReader not deterministic")
}

// haversineKM computes the great-circle distance between two points in km.
func haversineKM(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := degreesToRadians(lat2 - lat1)
	dLon := degreesToRadians(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKM * c
}

package random

import (
	"math"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestFloat(t *testing.T) {
	for range 1000 {
		v := Float(1.0, 10.0, 2)
		if v < 1.0 || v > 10.0 {
			t.Fatalf("Float(1.0, 10.0, 2) = %v, out of range", v)
		}

		// Verify precision: multiply by 100, should be an integer.
		scaled := v * 100
		if math.Abs(scaled-math.Round(scaled)) > 0.0001 {
			t.Fatalf("Float(1.0, 10.0, 2) = %v, not rounded to 2 decimal places", v)
		}
	}
}

func TestFloat_ZeroPrecision(t *testing.T) {
	for range 100 {
		v := Float(1.0, 10.0, 0)
		if v != math.Round(v) {
			t.Fatalf("Float with precision 0 = %v, expected integer", v)
		}
	}
}

func TestUniform(t *testing.T) {
	for range 1000 {
		v := Uniform(5.0, 15.0)
		if v < 5.0 || v >= 15.0 {
			t.Fatalf("Uniform(5.0, 15.0) = %v, out of range", v)
		}
	}
}

func TestZipf(t *testing.T) {
	bins := make([]int, 100)
	for range 10000 {
		v, err := Zipf(2.0, 1.0, 99)
		if err != nil {
			t.Fatal(err)
		}
		if v < 0 || v > 99 {
			t.Fatalf("Zipf(2.0, 1.0, 99) = %d, out of range [0, 99]", v)
		}
		bins[v]++
	}

	// Zipfian: lower values should be much more frequent.
	if bins[0] < bins[99] {
		t.Errorf("Zipf not skewed: bin[0]=%d, bin[99]=%d", bins[0], bins[99])
	}
}

func TestZipf_InvalidParams(t *testing.T) {
	// s <= 1 should return an error (NewZipf returns nil).
	_, err := Zipf(0.5, 1.0, 100)
	if err == nil {
		t.Error("Zipf with s <= 1 should return error")
	}
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
		if err != nil {
			t.Fatal(err)
		}
		if v < min || v > max {
			t.Fatalf("Norm value %v outside [%.0f, %.0f]", v, min, max)
		}
		// Default precision 0: should be a whole number.
		if v != math.Round(v) {
			t.Fatalf("Norm default precision: %v is not a whole number", v)
		}
		sum += v
	}

	observedMean := sum / n
	if observedMean < mean-2 || observedMean > mean+2 {
		t.Errorf("observed mean = %.1f, want ~%d", observedMean, mean)
	}
}

func TestNorm_WithPrecision(t *testing.T) {
	for range 1000 {
		v, err := Norm(50, 10, 1, 100, 2)
		if err != nil {
			t.Fatal(err)
		}
		if v < 1 || v > 100 {
			t.Fatalf("Norm with precision 2: %v outside [1, 100]", v)
		}

		// Verify 2 decimal places.
		scaled := v * 100
		if math.Abs(scaled-math.Round(scaled)) > 0.0001 {
			t.Fatalf("Norm with precision 2: %v not rounded to 2 decimal places", v)
		}
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
		if err != nil {
			t.Fatal(err)
		}
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

	if pct1 < 0.65 || pct1 > 0.71 {
		t.Errorf("within 1 stddev = %.1f%%, want ~68%%", pct1*100)
	}
	if pct2 < 0.93 || pct2 > 0.97 {
		t.Errorf("within 2 stddevs = %.1f%%, want ~95%%", pct2*100)
	}
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
		if err != nil {
			t.Fatal(err)
		}
		if v < min || v > max {
			t.Fatalf("Exp value %v outside [%.0f, %.0f]", v, min, max)
		}
		// Default precision 0: should be a whole number.
		if v != math.Round(v) {
			t.Fatalf("Exp default precision: %v is not a whole number", v)
		}
		sum += v
	}

	// Exponential with rate=1 has mean=1, but clamped to [0,100] shouldn't
	// shift the mean much. Observed mean should be close to 1.
	observedMean := sum / n
	if observedMean < 0.5 || observedMean > 1.5 {
		t.Errorf("observed mean = %.2f, want ~1.0", observedMean)
	}
}

func TestExp_WithPrecision(t *testing.T) {
	for range 1000 {
		v, err := Exp(0.5, 0, 100, 2)
		if err != nil {
			t.Fatal(err)
		}
		if v < 0 || v > 100 {
			t.Fatalf("Exp with precision 2: %v outside [0, 100]", v)
		}

		// Verify 2 decimal places.
		scaled := v * 100
		if math.Abs(scaled-math.Round(scaled)) > 0.0001 {
			t.Fatalf("Exp with precision 2: %v not rounded to 2 decimal places", v)
		}
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
	// so P ≈ 1 - e^(-0.5*2.5) ≈ 71.3%.
	belowMedian := 0
	for range n {
		v, err := Exp(rate, min, max)
		if err != nil {
			t.Fatal(err)
		}
		if v <= 2 {
			belowMedian++
		}
	}

	pct := float64(belowMedian) / n
	if pct < 0.67 || pct > 0.76 {
		t.Errorf("below mean = %.1f%%, want ~71%%", pct*100)
	}
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
		if err != nil {
			t.Fatal(err)
		}
		if v < min || v > max {
			t.Fatalf("LogNorm value %v outside [%.0f, %.0f]", v, min, max)
		}
		// Default precision 0: should be a whole number.
		if v != math.Round(v) {
			t.Fatalf("LogNorm default precision: %v is not a whole number", v)
		}
	}
}

func TestLogNorm_WithPrecision(t *testing.T) {
	for range 1000 {
		v, err := LogNorm(2.0, 0.5, 1, 100, 2)
		if err != nil {
			t.Fatal(err)
		}
		if v < 1 || v > 100 {
			t.Fatalf("LogNorm with precision 2: %v outside [1, 100]", v)
		}

		scaled := v * 100
		if math.Abs(scaled-math.Round(scaled)) > 0.0001 {
			t.Fatalf("LogNorm with precision 2: %v not rounded to 2 decimal places", v)
		}
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

	// For log-normal, the median is exp(mu) = exp(3) ≈ 20.09.
	// Roughly 50% of values should be below the median.
	expectedMedian := math.Exp(mu)
	belowMedian := 0
	sum := 0.0

	for range n {
		v, err := LogNorm(mu, sigma, min, max)
		if err != nil {
			t.Fatal(err)
		}
		sum += v
		if v <= expectedMedian {
			belowMedian++
		}
	}

	pct := float64(belowMedian) / n
	if pct < 0.45 || pct > 0.55 {
		t.Errorf("below median = %.1f%%, want ~50%%", pct*100)
	}

	// Mean of log-normal is exp(mu + sigma^2/2) ≈ exp(3.125) ≈ 22.76
	expectedMean := math.Exp(mu + sigma*sigma/2)
	observedMean := sum / n
	tolerance := expectedMean * 0.1
	if math.Abs(observedMean-expectedMean) > tolerance {
		t.Errorf("observed mean = %.1f, want ~%.1f", observedMean, expectedMean)
	}
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
			if err != nil {
				t.Fatalf("invalid regex: %v", err)
			}
			if !matched {
				t.Errorf("Regex(%q) = %q, does not match pattern", tt.pattern, result)
			}
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
		if dist > radiusKM+0.01 {
			t.Fatalf("Point(%.4f, %.4f, %.1f) = (%.6f, %.6f), distance %.4f km exceeds radius",
				centerLat, centerLon, radiusKM, lat, lon, dist)
		}
	}
}

func TestPoint_ZeroRadius(t *testing.T) {
	lat, lon := Point(40.0, -74.0, 0)
	if math.Abs(lat-40.0) > 0.0001 || math.Abs(lon-(-74.0)) > 0.0001 {
		t.Errorf("Point with zero radius = (%.6f, %.6f), want (40.0, -74.0)", lat, lon)
	}
}

func TestTimestamp(t *testing.T) {
	min := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	max := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for range 100 {
		ts := Timestamp(min, max)
		if ts.Before(min) || ts.After(max) {
			t.Fatalf("Timestamp not in range: %v", ts)
		}
	}
}

func TestTimestamp_Equal(t *testing.T) {
	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	got := Timestamp(ts, ts)
	if !got.Equal(ts) {
		t.Errorf("Timestamp(equal, equal) = %v, want %v", got, ts)
	}
}

func TestTimestamp_SwapsMinMax(t *testing.T) {
	min := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	max := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	ts := Timestamp(max, min)
	if ts.Before(min) || ts.After(max) {
		t.Fatalf("Timestamp with swapped args not in range: %v", ts)
	}
}

func TestDuration(t *testing.T) {
	min := 1 * time.Hour
	max := 24 * time.Hour

	for range 100 {
		d := Duration(min, max)
		if d < min || d > max {
			t.Fatalf("Duration not in range: %v", d)
		}
	}
}

func TestDuration_Equal(t *testing.T) {
	d := 5 * time.Minute
	got := Duration(d, d)
	if got != d {
		t.Errorf("Duration(equal, equal) = %v, want %v", got, d)
	}
}

func TestDuration_SwapsMinMax(t *testing.T) {
	min := 1 * time.Hour
	max := 24 * time.Hour

	d := Duration(max, min)
	if d < min || d > max {
		t.Fatalf("Duration with swapped args not in range: %v", d)
	}
}

func TestBytes(t *testing.T) {
	for range 100 {
		result := Bytes(16)
		if !strings.HasPrefix(result, `\x`) {
			t.Fatalf("Bytes(16) = %q, want \\x prefix", result)
		}
		// \x prefix + 32 hex chars = 34 total
		if len(result) != 34 {
			t.Fatalf("Bytes(16) length = %d, want 34", len(result))
		}
	}
}

func TestBytes_Zero(t *testing.T) {
	result := Bytes(0)
	if result != `\x` {
		t.Errorf("Bytes(0) = %q, want \\x", result)
	}
}

func TestBit(t *testing.T) {
	for range 100 {
		result := Bit(8)
		if len(result) != 8 {
			t.Fatalf("Bit(8) length = %d, want 8", len(result))
		}
		for _, c := range result {
			if c != '0' && c != '1' {
				t.Fatalf("Bit(8) = %q, contains non-bit character", result)
			}
		}
	}
}

func TestVarBit(t *testing.T) {
	for range 100 {
		result := VarBit(16)
		if len(result) < 1 || len(result) > 16 {
			t.Fatalf("VarBit(16) length = %d, want 1-16", len(result))
		}
		for _, c := range result {
			if c != '0' && c != '1' {
				t.Fatalf("VarBit(16) = %q, contains non-bit character", result)
			}
		}
	}
}

func TestVarBit_Zero(t *testing.T) {
	if result := VarBit(0); result != "" {
		t.Errorf("VarBit(0) = %q, want empty", result)
	}
}

func TestInet_IPv4(t *testing.T) {
	for range 100 {
		result, err := Inet("192.168.1.0/24")
		if err != nil {
			t.Fatalf("Inet error: %v", err)
		}
		ip := net.ParseIP(result)
		if ip == nil {
			t.Fatalf("Inet returned invalid IP: %q", result)
		}
		_, network, _ := net.ParseCIDR("192.168.1.0/24")
		if !network.Contains(ip) {
			t.Fatalf("Inet returned %v, not in 192.168.1.0/24", result)
		}
	}
}

func TestInet_IPv6(t *testing.T) {
	result, err := Inet("fd00::/64")
	if err != nil {
		t.Fatalf("Inet error: %v", err)
	}
	ip := net.ParseIP(result)
	if ip == nil {
		t.Fatalf("Inet returned invalid IP: %q", result)
	}
	_, network, _ := net.ParseCIDR("fd00::/64")
	if !network.Contains(ip) {
		t.Fatalf("Inet returned %v, not in fd00::/64", result)
	}
}

func TestInet_InvalidCIDR(t *testing.T) {
	_, err := Inet("not-a-cidr")
	if err == nil {
		t.Fatal("Inet with invalid CIDR should return error")
	}
}

func TestTimeOfDay(t *testing.T) {
	for range 100 {
		result, err := TimeOfDay("08:00:00", "17:30:00")
		if err != nil {
			t.Fatalf("TimeOfDay error: %v", err)
		}
		if len(result) != 8 {
			t.Fatalf("TimeOfDay = %q, want HH:MM:SS format", result)
		}
		// Verify it parses back.
		_, err = time.Parse("15:04:05", result)
		if err != nil {
			t.Fatalf("TimeOfDay returned invalid time %q: %v", result, err)
		}
		if result < "08:00:00" || result > "17:30:00" {
			t.Fatalf("TimeOfDay = %q, out of range [08:00:00, 17:30:00]", result)
		}
	}
}

func TestTimeOfDay_Equal(t *testing.T) {
	result, err := TimeOfDay("12:00:00", "12:00:00")
	if err != nil {
		t.Fatalf("TimeOfDay error: %v", err)
	}
	if result != "12:00:00" {
		t.Errorf("TimeOfDay(equal, equal) = %q, want 12:00:00", result)
	}
}

func TestTimeOfDay_SwapsMinMax(t *testing.T) {
	result, err := TimeOfDay("17:00:00", "08:00:00")
	if err != nil {
		t.Fatalf("TimeOfDay error: %v", err)
	}
	if result < "08:00:00" || result > "17:00:00" {
		t.Fatalf("TimeOfDay with swapped args = %q, out of range", result)
	}
}

func TestTimeOfDay_InvalidInput(t *testing.T) {
	_, err := TimeOfDay("bad", "17:00:00")
	if err == nil {
		t.Fatal("TimeOfDay with invalid min should return error")
	}
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
		if v != first[i] {
			t.Fatalf("Seed determinism: index %d got %v, want %v", i, v, first[i])
		}
	}
}

func TestSeed_DifferentSeeds(t *testing.T) {
	Seed(1)
	a := Uniform(0, 1000)

	Seed(2)
	b := Uniform(0, 1000)

	if a == b {
		t.Errorf("different seeds produced the same value: %v", a)
	}
}

func TestSeed_Float(t *testing.T) {
	Seed(100)
	a := Float(1.0, 10.0, 2)

	Seed(100)
	b := Float(1.0, 10.0, 2)

	if a != b {
		t.Errorf("Float not deterministic: %v != %v", a, b)
	}
}

func TestSeed_Norm(t *testing.T) {
	Seed(200)
	a, err := Norm(50, 10, 1, 100)
	if err != nil {
		t.Fatal(err)
	}

	Seed(200)
	b, err := Norm(50, 10, 1, 100)
	if err != nil {
		t.Fatal(err)
	}

	if a != b {
		t.Errorf("Norm not deterministic: %v != %v", a, b)
	}
}

func TestSeed_Zipf(t *testing.T) {
	Seed(300)
	a, err := Zipf(2.0, 1.0, 99)
	if err != nil {
		t.Fatal(err)
	}

	Seed(300)
	b, err := Zipf(2.0, 1.0, 99)
	if err != nil {
		t.Fatal(err)
	}

	if a != b {
		t.Errorf("Zipf not deterministic: %v != %v", a, b)
	}
}

func TestSeed_Timestamp(t *testing.T) {
	min := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	max := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	Seed(400)
	a := Timestamp(min, max)

	Seed(400)
	b := Timestamp(min, max)

	if !a.Equal(b) {
		t.Errorf("Timestamp not deterministic: %v != %v", a, b)
	}
}

func TestSeed_Regex(t *testing.T) {
	Seed(500)
	a := Regex("[A-Z]{5}")

	Seed(500)
	b := Regex("[A-Z]{5}")

	if a != b {
		t.Errorf("Regex not deterministic: %q != %q", a, b)
	}
}

func TestSeed_UUIDv4(t *testing.T) {
	Seed(600)
	a := UUIDv4()

	Seed(600)
	b := UUIDv4()

	if a != b {
		t.Errorf("UUIDv4 not deterministic: %q != %q", a, b)
	}
}

func TestSeed_Bytes(t *testing.T) {
	Seed(700)
	a := Bytes(16)

	Seed(700)
	b := Bytes(16)

	if a != b {
		t.Errorf("Bytes not deterministic: %q != %q", a, b)
	}
}

func TestSeed_Inet(t *testing.T) {
	Seed(800)
	a, err := Inet("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}

	Seed(800)
	b, err := Inet("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}

	if a != b {
		t.Errorf("Inet not deterministic: %q != %q", a, b)
	}
}

func TestRngReader(t *testing.T) {
	Seed(42)
	rr := &rngReader{Rng}
	buf := make([]byte, 32)
	n, err := rr.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != 32 {
		t.Fatalf("Read returned %d bytes, want 32", n)
	}

	// Re-seed and verify deterministic reads.
	Seed(42)
	rr2 := &rngReader{Rng}
	buf2 := make([]byte, 32)
	rr2.Read(buf2)

	for i := range buf {
		if buf[i] != buf2[i] {
			t.Fatalf("rngReader not deterministic at byte %d: %d != %d", i, buf[i], buf2[i])
		}
	}
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

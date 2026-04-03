package random

import (
	"math"
	"regexp"
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
		v := Zipf(2.0, 1.0, 99)
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
	// s <= 1 should return 0 (NewZipf returns nil).
	v := Zipf(0.5, 1.0, 100)
	if v != 0 {
		t.Errorf("Zipf with s <= 1 = %d, want 0", v)
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
		v := Norm(mean, stddev, min, max)
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
		v := Norm(50, 10, 1, 100, 2)
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
		v := Norm(mean, stddev, min, max)
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

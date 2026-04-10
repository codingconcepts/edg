package gen

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestGen(t *testing.T) {
	result, err := Gen("number:1,100")
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("Gen returned nil for valid pattern")
	}

	result, err = Gen("{number:1,100}")
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("Gen returned nil for already-wrapped pattern")
	}
}

func TestGenBatch(t *testing.T) {
	result, err := GenBatch(25, 10, "email")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("GenBatch(25, 10) returned %d batches, want 3", len(result))
	}

	// First two batches should have 10 emails each.
	for _, i := range []int{0, 1} {
		csv := result[i][0].(string)
		parts := strings.Split(csv, ",")
		if len(parts) != 10 {
			t.Errorf("batch %d has %d values, want 10", i, len(parts))
		}
	}

	// Last batch should have 5 emails (remainder).
	csv := result[2][0].(string)
	parts := strings.Split(csv, ",")
	if len(parts) != 5 {
		t.Errorf("last batch has %d values, want 5", len(parts))
	}

	// All emails across all batches should be unique.
	seen := map[string]bool{}
	for _, row := range result {
		for _, v := range strings.Split(row[0].(string), ",") {
			if seen[v] {
				t.Errorf("GenBatch produced duplicate: %s", v)
			}
			seen[v] = true
		}
	}
	if len(seen) != 25 {
		t.Errorf("GenBatch produced %d unique values, want 25", len(seen))
	}
}

func TestGenBatch_ExactMultiple(t *testing.T) {
	result, err := GenBatch(20, 10, "email")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("GenBatch(20, 10) returned %d batches, want 2", len(result))
	}
	for i, row := range result {
		parts := strings.Split(row[0].(string), ",")
		if len(parts) != 10 {
			t.Errorf("batch %d has %d values, want 10", i, len(parts))
		}
	}
}

func TestFloatRand(t *testing.T) {
	for range 1000 {
		v, err := FloatRand(1.0, 10.0, 2)
		if err != nil {
			t.Fatal(err)
		}
		if v < 1.0 || v > 10.0 {
			t.Fatalf("FloatRand(1.0, 10.0, 2) = %v, out of range", v)
		}
		scaled := v * 100
		if math.Abs(scaled-math.Round(scaled)) > 0.0001 {
			t.Fatalf("FloatRand precision 2: %v not rounded correctly", v)
		}
	}
}

func TestUniformRand(t *testing.T) {
	for range 1000 {
		v, err := UniformRand(5.0, 15.0)
		if err != nil {
			t.Fatal(err)
		}
		if v < 5.0 || v >= 15.0 {
			t.Fatalf("UniformRand(5.0, 15.0) = %v, out of range", v)
		}
	}
}

func TestZipfRand(t *testing.T) {
	bins := make([]int, 100)
	for range 10000 {
		v, err := ZipfRand(2.0, 1.0, 99)
		if err != nil {
			t.Fatal(err)
		}
		if v < 0 || v > 99 {
			t.Fatalf("ZipfRand out of range: %d", v)
		}
		bins[v]++
	}
	if bins[0] < bins[99] {
		t.Errorf("ZipfRand not skewed: bin[0]=%d, bin[99]=%d", bins[0], bins[99])
	}
}

func TestZipfRand_InvalidParams(t *testing.T) {
	v, err := ZipfRand(0.5, 1.0, 100)
	if err != nil {
		// Invalid params returning an error is acceptable.
		return
	}
	if v != 0 {
		t.Errorf("ZipfRand with s <= 1 = %d, want 0", v)
	}
}

func TestGenRegex(t *testing.T) {
	result := GenRegex("[A-Z]{3}-[0-9]{4}")
	matched, _ := regexp.MatchString(`^[A-Z]{3}-[0-9]{4}$`, result)
	if !matched {
		t.Errorf("GenRegex = %q, does not match pattern", result)
	}
}

func TestJsonObj(t *testing.T) {
	result, err := JsonObj("name", "alice", "age", 30)
	if err != nil {
		t.Fatalf("JsonObj error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("JsonObj produced invalid JSON: %v", err)
	}
	if m["name"] != "alice" {
		t.Errorf("name = %v, want alice", m["name"])
	}
	if m["age"] != float64(30) {
		t.Errorf("age = %v, want 30", m["age"])
	}
}

func TestJsonObj_OddArgs(t *testing.T) {
	_, err := JsonObj("key1", "val1", "key2")
	if err == nil {
		t.Fatal("JsonObj with odd args should return error")
	}
}

func TestJsonArr(t *testing.T) {
	result, err := JsonArr(3, 3, "email")
	if err != nil {
		t.Fatalf("JsonArr error: %v", err)
	}

	var arr []any
	if err := json.Unmarshal([]byte(result), &arr); err != nil {
		t.Fatalf("JsonArr produced invalid JSON: %v", err)
	}
	if len(arr) != 3 {
		t.Errorf("JsonArr length = %d, want 3", len(arr))
	}
}

func TestGenPoint(t *testing.T) {
	centerLat := 51.5074
	centerLon := -0.1278
	radiusKM := 10.0

	for range 100 {
		p, err := GenPoint(centerLat, centerLon, radiusKM)
		if err != nil {
			t.Fatal(err)
		}
		lat := p["lat"].(float64)
		lon := p["lon"].(float64)

		dLat := (lat - centerLat) * math.Pi / 180
		dLon := (lon - centerLon) * math.Pi / 180
		a := math.Sin(dLat/2)*math.Sin(dLat/2) +
			math.Cos(centerLat*math.Pi/180)*math.Cos(lat*math.Pi/180)*
				math.Sin(dLon/2)*math.Sin(dLon/2)
		dist := 6371.0 * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
		if dist > radiusKM+0.01 {
			t.Fatalf("GenPoint distance %.4f km exceeds radius", dist)
		}
	}
}

func TestGenPointWKT(t *testing.T) {
	centerLat := 51.5074
	centerLon := -0.1278
	radiusKM := 10.0

	for range 100 {
		wkt, err := GenPointWKT(centerLat, centerLon, radiusKM)
		if err != nil {
			t.Fatal(err)
		}

		var lon, lat float64
		_, err = fmt.Sscanf(wkt, "POINT(%f %f)", &lon, &lat)
		if err != nil {
			t.Fatalf("GenPointWKT returned invalid WKT %q: %v", wkt, err)
		}

		dLat := (lat - centerLat) * math.Pi / 180
		dLon := (lon - centerLon) * math.Pi / 180
		a := math.Sin(dLat/2)*math.Sin(dLat/2) +
			math.Cos(centerLat*math.Pi/180)*math.Cos(lat*math.Pi/180)*
				math.Sin(dLon/2)*math.Sin(dLon/2)
		dist := 6371.0 * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
		if dist > radiusKM+0.01 {
			t.Fatalf("GenPointWKT distance %.4f km exceeds radius", dist)
		}
	}
}

func TestRandTimestamp(t *testing.T) {
	min := "2020-01-01T00:00:00Z"
	max := "2025-01-01T00:00:00Z"
	result, err := RandTimestamp(min, max)
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("RandTimestamp returned empty string")
	}

	ts, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Fatalf("RandTimestamp produced invalid RFC3339: %v", err)
	}
	minT, _ := time.Parse(time.RFC3339, min)
	maxT, _ := time.Parse(time.RFC3339, max)
	if ts.Before(minT) || ts.After(maxT) {
		t.Errorf("RandTimestamp %v not in range [%v, %v]", ts, minT, maxT)
	}
}

func TestRandTimestamp_InvalidInput(t *testing.T) {
	_, err := RandTimestamp("bad", "2025-01-01T00:00:00Z")
	if err == nil {
		t.Fatal("RandTimestamp with bad min should return error")
	}
}

func TestRandDuration(t *testing.T) {
	result, err := RandDuration("1h", "24h")
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("RandDuration returned empty string")
	}

	d, err := time.ParseDuration(result)
	if err != nil {
		t.Fatalf("RandDuration produced invalid duration: %v", err)
	}
	if d < time.Hour || d > 24*time.Hour {
		t.Errorf("RandDuration %v not in range [1h, 24h]", d)
	}
}

func TestRandDuration_InvalidInput(t *testing.T) {
	_, err := RandDuration("bad", "24h")
	if err == nil {
		t.Fatal("RandDuration with bad min should return error")
	}
}

func TestDateRand(t *testing.T) {
	result, err := DateRand("2006-01-02", "2020-01-01T00:00:00Z", "2025-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("DateRand returned empty string")
	}

	ts, err := time.Parse("2006-01-02", result)
	if err != nil {
		t.Fatalf("DateRand produced invalid date: %v", err)
	}
	if ts.Year() < 2020 || ts.Year() > 2025 {
		t.Errorf("DateRand year %d not in range [2020, 2025]", ts.Year())
	}
}

func TestDateOffset(t *testing.T) {
	result, err := DateOffset("1h")
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("DateOffset returned empty string")
	}

	ts, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Fatalf("DateOffset produced invalid RFC3339: %v", err)
	}

	expected := time.Now().Add(time.Hour).UTC()
	diff := ts.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	if diff > 2*time.Second {
		t.Errorf("DateOffset('1h') = %v, expected ~%v", ts, expected)
	}
}

func TestDateOffset_InvalidInput(t *testing.T) {
	_, err := DateOffset("bad")
	if err == nil {
		t.Fatal("DateOffset with bad duration should return error")
	}
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
		if err != nil {
			t.Fatal(err)
		}
		if v < min || v > max {
			t.Fatalf("ExpRand value %v outside [%.0f, %.0f]", v, min, max)
		}
		sum += v
	}

	// Exponential with rate=1 has mean=1.
	observedMean := sum / n
	if observedMean < 0.5 || observedMean > 1.5 {
		t.Errorf("observed mean = %.2f, want ~1.0", observedMean)
	}
}

func TestExpRandF(t *testing.T) {
	for range 100 {
		v, err := ExpRandF(0.5, 0.0, 100.0, 2)
		if err != nil {
			t.Fatal(err)
		}
		if v < 0 || v > 100 {
			t.Fatalf("ExpRandF value %v outside [0, 100]", v)
		}
		scaled := v * 100
		if math.Abs(scaled-math.Round(scaled)) > 0.0001 {
			t.Fatalf("ExpRandF precision 2: %v not rounded correctly", v)
		}
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
		if err != nil {
			t.Fatal(err)
		}
		if v < min || v > max {
			t.Fatalf("LognormRand value %v outside [%.0f, %.0f]", v, min, max)
		}
	}
}

func TestLognormRandF(t *testing.T) {
	for range 100 {
		v, err := LognormRandF(2.0, 0.5, 1.0, 100.0, 2)
		if err != nil {
			t.Fatal(err)
		}
		if v < 1 || v > 100 {
			t.Fatalf("LognormRandF value %v outside [1, 100]", v)
		}
		scaled := v * 100
		if math.Abs(scaled-math.Round(scaled)) > 0.0001 {
			t.Fatalf("LognormRandF precision 2: %v not rounded correctly", v)
		}
	}
}

func TestGenBytes(t *testing.T) {
	result, err := GenBytes(8)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result, `\x`) {
		t.Fatalf("GenBytes(8) = %q, want \\x prefix", result)
	}
	if len(result) != 18 { // \x + 16 hex chars
		t.Fatalf("GenBytes(8) length = %d, want 18", len(result))
	}
}

func TestGenBit(t *testing.T) {
	result, err := GenBit(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 8 {
		t.Fatalf("GenBit(8) length = %d, want 8", len(result))
	}
	matched, _ := regexp.MatchString(`^[01]+$`, result)
	if !matched {
		t.Fatalf("GenBit(8) = %q, contains non-bit characters", result)
	}
}

func TestGenBool(t *testing.T) {
	trueCount := 0
	for range 1000 {
		if GenBool() {
			trueCount++
		}
	}
	if trueCount == 0 || trueCount == 1000 {
		t.Fatalf("GenBool returned the same value 1000 times (true=%d)", trueCount)
	}
}

func TestGenVarBit(t *testing.T) {
	for range 100 {
		result, err := GenVarBit(16)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) < 1 || len(result) > 16 {
			t.Fatalf("GenVarBit(16) length = %d, want 1-16", len(result))
		}
	}
}

func TestGenInet(t *testing.T) {
	result, err := GenInet("10.0.0.0/8")
	if err != nil {
		t.Fatalf("GenInet error: %v", err)
	}
	if !strings.HasPrefix(result, "10.") {
		t.Fatalf("GenInet('10.0.0.0/8') = %q, want 10.x.x.x", result)
	}
}

func TestGenInet_InvalidCIDR(t *testing.T) {
	_, err := GenInet("bad")
	if err == nil {
		t.Fatal("GenInet with invalid CIDR should return error")
	}
}

func TestGenArray(t *testing.T) {
	result, err := GenArray(3, 3, "number:1,100")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result, "{") || !strings.HasSuffix(result, "}") {
		t.Fatalf("GenArray = %q, want {}-wrapped", result)
	}
	inner := result[1 : len(result)-1]
	parts := strings.Split(inner, ",")
	if len(parts) != 3 {
		t.Errorf("GenArray(3,3) produced %d elements, want 3", len(parts))
	}
}

func TestGenArray_Range(t *testing.T) {
	for range 100 {
		result, err := GenArray(2, 5, "letter")
		if err != nil {
			t.Fatal(err)
		}
		inner := result[1 : len(result)-1]
		parts := strings.Split(inner, ",")
		if len(parts) < 2 || len(parts) > 5 {
			t.Fatalf("GenArray(2,5) produced %d elements, want 2-5", len(parts))
		}
	}
}

func TestGenTime(t *testing.T) {
	result, err := GenTime("08:00:00", "17:00:00")
	if err != nil {
		t.Fatalf("GenTime error: %v", err)
	}
	if result < "08:00:00" || result > "17:00:00" {
		t.Fatalf("GenTime = %q, out of range", result)
	}
}

func TestGenTime_InvalidInput(t *testing.T) {
	_, err := GenTime("bad", "17:00:00")
	if err == nil {
		t.Fatal("GenTime with invalid min should return error")
	}
}

func TestGenTimez(t *testing.T) {
	result, err := GenTimez("08:00:00", "17:00:00")
	if err != nil {
		t.Fatalf("GenTimez error: %v", err)
	}
	if !strings.HasSuffix(result, "+00:00") {
		t.Fatalf("GenTimez = %q, want +00:00 suffix", result)
	}
	timePart := strings.TrimSuffix(result, "+00:00")
	if timePart < "08:00:00" || timePart > "17:00:00" {
		t.Fatalf("GenTimez time part = %q, out of range", timePart)
	}
}

func TestGenTimez_InvalidInput(t *testing.T) {
	_, err := GenTimez("bad", "17:00:00")
	if err == nil {
		t.Fatal("GenTimez with invalid min should return error")
	}
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

package pkg

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
	result, err := gen("number:1,100")
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("gen returned nil for valid pattern")
	}

	result, err = gen("{number:1,100}")
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("gen returned nil for already-wrapped pattern")
	}
}

func TestGenBatch(t *testing.T) {
	result, err := genBatch(25, 10, "email")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("genBatch(25, 10) returned %d batches, want 3", len(result))
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
				t.Errorf("genBatch produced duplicate: %s", v)
			}
			seen[v] = true
		}
	}
	if len(seen) != 25 {
		t.Errorf("genBatch produced %d unique values, want 25", len(seen))
	}
}

func TestGenBatch_ExactMultiple(t *testing.T) {
	result, err := genBatch(20, 10, "email")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("genBatch(20, 10) returned %d batches, want 2", len(result))
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
		v, err := floatRand(1.0, 10.0, 2)
		if err != nil {
			t.Fatal(err)
		}
		if v < 1.0 || v > 10.0 {
			t.Fatalf("floatRand(1.0, 10.0, 2) = %v, out of range", v)
		}
		scaled := v * 100
		if math.Abs(scaled-math.Round(scaled)) > 0.0001 {
			t.Fatalf("floatRand precision 2: %v not rounded correctly", v)
		}
	}
}

func TestUniformRand(t *testing.T) {
	for range 1000 {
		v, err := uniformRand(5.0, 15.0)
		if err != nil {
			t.Fatal(err)
		}
		if v < 5.0 || v >= 15.0 {
			t.Fatalf("uniformRand(5.0, 15.0) = %v, out of range", v)
		}
	}
}

func TestZipfRand(t *testing.T) {
	bins := make([]int, 100)
	for range 10000 {
		v, err := zipfRand(2.0, 1.0, 99)
		if err != nil {
			t.Fatal(err)
		}
		if v < 0 || v > 99 {
			t.Fatalf("zipfRand out of range: %d", v)
		}
		bins[v]++
	}
	if bins[0] < bins[99] {
		t.Errorf("zipfRand not skewed: bin[0]=%d, bin[99]=%d", bins[0], bins[99])
	}
}

func TestZipfRand_InvalidParams(t *testing.T) {
	v, err := zipfRand(0.5, 1.0, 100)
	if err != nil {
		// Invalid params returning an error is acceptable.
		return
	}
	if v != 0 {
		t.Errorf("zipfRand with s <= 1 = %d, want 0", v)
	}
}

func TestGenRegex(t *testing.T) {
	result := genRegex("[A-Z]{3}-[0-9]{4}")
	matched, _ := regexp.MatchString(`^[A-Z]{3}-[0-9]{4}$`, result)
	if !matched {
		t.Errorf("genRegex = %q, does not match pattern", result)
	}
}

func TestJsonObj(t *testing.T) {
	result, err := jsonObj("name", "alice", "age", 30)
	if err != nil {
		t.Fatalf("jsonObj error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("jsonObj produced invalid JSON: %v", err)
	}
	if m["name"] != "alice" {
		t.Errorf("name = %v, want alice", m["name"])
	}
	if m["age"] != float64(30) {
		t.Errorf("age = %v, want 30", m["age"])
	}
}

func TestJsonObj_OddArgs(t *testing.T) {
	_, err := jsonObj("key1", "val1", "key2")
	if err == nil {
		t.Fatal("jsonObj with odd args should return error")
	}
}

func TestJsonArr(t *testing.T) {
	result, err := jsonArr(3, 3, "email")
	if err != nil {
		t.Fatalf("jsonArr error: %v", err)
	}

	var arr []any
	if err := json.Unmarshal([]byte(result), &arr); err != nil {
		t.Fatalf("jsonArr produced invalid JSON: %v", err)
	}
	if len(arr) != 3 {
		t.Errorf("jsonArr length = %d, want 3", len(arr))
	}
}

func TestGenPoint(t *testing.T) {
	centerLat := 51.5074
	centerLon := -0.1278
	radiusKM := 10.0

	for range 100 {
		p, err := genPoint(centerLat, centerLon, radiusKM)
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
			t.Fatalf("genPoint distance %.4f km exceeds radius", dist)
		}
	}
}

func TestGenPointWKT(t *testing.T) {
	centerLat := 51.5074
	centerLon := -0.1278
	radiusKM := 10.0

	for range 100 {
		wkt, err := genPointWKT(centerLat, centerLon, radiusKM)
		if err != nil {
			t.Fatal(err)
		}

		var lon, lat float64
		_, err = fmt.Sscanf(wkt, "POINT(%f %f)", &lon, &lat)
		if err != nil {
			t.Fatalf("genPointWKT returned invalid WKT %q: %v", wkt, err)
		}

		dLat := (lat - centerLat) * math.Pi / 180
		dLon := (lon - centerLon) * math.Pi / 180
		a := math.Sin(dLat/2)*math.Sin(dLat/2) +
			math.Cos(centerLat*math.Pi/180)*math.Cos(lat*math.Pi/180)*
				math.Sin(dLon/2)*math.Sin(dLon/2)
		dist := 6371.0 * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
		if dist > radiusKM+0.01 {
			t.Fatalf("genPointWKT distance %.4f km exceeds radius", dist)
		}
	}
}

func TestRandTimestamp(t *testing.T) {
	min := "2020-01-01T00:00:00Z"
	max := "2025-01-01T00:00:00Z"
	result, err := randTimestamp(min, max)
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("randTimestamp returned empty string")
	}

	ts, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Fatalf("randTimestamp produced invalid RFC3339: %v", err)
	}
	minT, _ := time.Parse(time.RFC3339, min)
	maxT, _ := time.Parse(time.RFC3339, max)
	if ts.Before(minT) || ts.After(maxT) {
		t.Errorf("randTimestamp %v not in range [%v, %v]", ts, minT, maxT)
	}
}

func TestRandTimestamp_InvalidInput(t *testing.T) {
	_, err := randTimestamp("bad", "2025-01-01T00:00:00Z")
	if err == nil {
		t.Fatal("randTimestamp with bad min should return error")
	}
}

func TestRandDuration(t *testing.T) {
	result, err := randDuration("1h", "24h")
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("randDuration returned empty string")
	}

	d, err := time.ParseDuration(result)
	if err != nil {
		t.Fatalf("randDuration produced invalid duration: %v", err)
	}
	if d < time.Hour || d > 24*time.Hour {
		t.Errorf("randDuration %v not in range [1h, 24h]", d)
	}
}

func TestRandDuration_InvalidInput(t *testing.T) {
	_, err := randDuration("bad", "24h")
	if err == nil {
		t.Fatal("randDuration with bad min should return error")
	}
}

func TestDateRand(t *testing.T) {
	result, err := dateRand("2006-01-02", "2020-01-01T00:00:00Z", "2025-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("dateRand returned empty string")
	}

	ts, err := time.Parse("2006-01-02", result)
	if err != nil {
		t.Fatalf("dateRand produced invalid date: %v", err)
	}
	if ts.Year() < 2020 || ts.Year() > 2025 {
		t.Errorf("dateRand year %d not in range [2020, 2025]", ts.Year())
	}
}

func TestDateOffset(t *testing.T) {
	result, err := dateOffset("1h")
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("dateOffset returned empty string")
	}

	ts, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Fatalf("dateOffset produced invalid RFC3339: %v", err)
	}

	expected := time.Now().Add(time.Hour).UTC()
	diff := ts.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	if diff > 2*time.Second {
		t.Errorf("dateOffset('1h') = %v, expected ~%v", ts, expected)
	}
}

func TestDateOffset_InvalidInput(t *testing.T) {
	_, err := dateOffset("bad")
	if err == nil {
		t.Fatal("dateOffset with bad duration should return error")
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
		v, err := expRand(rate, min, max)
		if err != nil {
			t.Fatal(err)
		}
		if v < min || v > max {
			t.Fatalf("expRand value %v outside [%.0f, %.0f]", v, min, max)
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
		v, err := expRandF(0.5, 0.0, 100.0, 2)
		if err != nil {
			t.Fatal(err)
		}
		if v < 0 || v > 100 {
			t.Fatalf("expRandF value %v outside [0, 100]", v)
		}
		scaled := v * 100
		if math.Abs(scaled-math.Round(scaled)) > 0.0001 {
			t.Fatalf("expRandF precision 2: %v not rounded correctly", v)
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
		v, err := lognormRand(mu, sigma, min, max)
		if err != nil {
			t.Fatal(err)
		}
		if v < min || v > max {
			t.Fatalf("lognormRand value %v outside [%.0f, %.0f]", v, min, max)
		}
	}
}

func TestLognormRandF(t *testing.T) {
	for range 100 {
		v, err := lognormRandF(2.0, 0.5, 1.0, 100.0, 2)
		if err != nil {
			t.Fatal(err)
		}
		if v < 1 || v > 100 {
			t.Fatalf("lognormRandF value %v outside [1, 100]", v)
		}
		scaled := v * 100
		if math.Abs(scaled-math.Round(scaled)) > 0.0001 {
			t.Fatalf("lognormRandF precision 2: %v not rounded correctly", v)
		}
	}
}

func TestGenBytes(t *testing.T) {
	result, err := genBytes(8)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result, `\x`) {
		t.Fatalf("genBytes(8) = %q, want \\x prefix", result)
	}
	if len(result) != 18 { // \x + 16 hex chars
		t.Fatalf("genBytes(8) length = %d, want 18", len(result))
	}
}

func TestGenBit(t *testing.T) {
	result, err := genBit(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 8 {
		t.Fatalf("genBit(8) length = %d, want 8", len(result))
	}
	matched, _ := regexp.MatchString(`^[01]+$`, result)
	if !matched {
		t.Fatalf("genBit(8) = %q, contains non-bit characters", result)
	}
}

func TestGenVarBit(t *testing.T) {
	for range 100 {
		result, err := genVarBit(16)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) < 1 || len(result) > 16 {
			t.Fatalf("genVarBit(16) length = %d, want 1-16", len(result))
		}
	}
}

func TestGenInet(t *testing.T) {
	result, err := genInet("10.0.0.0/8")
	if err != nil {
		t.Fatalf("genInet error: %v", err)
	}
	if !strings.HasPrefix(result, "10.") {
		t.Fatalf("genInet('10.0.0.0/8') = %q, want 10.x.x.x", result)
	}
}

func TestGenInet_InvalidCIDR(t *testing.T) {
	_, err := genInet("bad")
	if err == nil {
		t.Fatal("genInet with invalid CIDR should return error")
	}
}

func TestGenArray(t *testing.T) {
	result, err := genArray(3, 3, "number:1,100")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result, "{") || !strings.HasSuffix(result, "}") {
		t.Fatalf("genArray = %q, want {}-wrapped", result)
	}
	inner := result[1 : len(result)-1]
	parts := strings.Split(inner, ",")
	if len(parts) != 3 {
		t.Errorf("genArray(3,3) produced %d elements, want 3", len(parts))
	}
}

func TestGenArray_Range(t *testing.T) {
	for range 100 {
		result, err := genArray(2, 5, "letter")
		if err != nil {
			t.Fatal(err)
		}
		inner := result[1 : len(result)-1]
		parts := strings.Split(inner, ",")
		if len(parts) < 2 || len(parts) > 5 {
			t.Fatalf("genArray(2,5) produced %d elements, want 2-5", len(parts))
		}
	}
}

func TestGenTime(t *testing.T) {
	result, err := genTime("08:00:00", "17:00:00")
	if err != nil {
		t.Fatalf("genTime error: %v", err)
	}
	if result < "08:00:00" || result > "17:00:00" {
		t.Fatalf("genTime = %q, out of range", result)
	}
}

func TestGenTime_InvalidInput(t *testing.T) {
	_, err := genTime("bad", "17:00:00")
	if err == nil {
		t.Fatal("genTime with invalid min should return error")
	}
}

func TestGenTimez(t *testing.T) {
	result, err := genTimez("08:00:00", "17:00:00")
	if err != nil {
		t.Fatalf("genTimez error: %v", err)
	}
	if !strings.HasSuffix(result, "+00:00") {
		t.Fatalf("genTimez = %q, want +00:00 suffix", result)
	}
	timePart := strings.TrimSuffix(result, "+00:00")
	if timePart < "08:00:00" || timePart > "17:00:00" {
		t.Fatalf("genTimez time part = %q, out of range", timePart)
	}
}

func TestGenTimez_InvalidInput(t *testing.T) {
	_, err := genTimez("bad", "17:00:00")
	if err == nil {
		t.Fatal("genTimez with invalid min should return error")
	}
}

func TestNormRandF(t *testing.T) {
	env := testEnv(nil)

	for range 100 {
		v, err := env.normRandF(50, 10, 1, 100, 2)
		if err != nil {
			t.Fatal(err)
		}
		if v < 1 || v > 100 {
			t.Fatalf("normRandF value %v outside [1, 100]", v)
		}
		scaled := v * 100
		if math.Abs(scaled-math.Round(scaled)) > 0.0001 {
			t.Fatalf("normRandF precision 2: %v not rounded correctly", v)
		}
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
				genBatch(tc.total, tc.batch, "number:1,1000")			}
		})
	}
}

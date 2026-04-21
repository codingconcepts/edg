package seq

import (
	"sync"
	"testing"
)

func TestSequenceNext(t *testing.T) {
	mgr := NewManager([]Config{
		{Name: "orders", Start: 1, Step: 1},
		{Name: "even", Start: 0, Step: 2},
	})

	for i := range 5 {
		v, err := mgr.Next("orders")
		if err != nil {
			t.Fatal(err)
		}
		want := int64(1 + i)
		if v != want {
			t.Fatalf("orders: got %d, want %d", v, want)
		}
	}

	for i := range 3 {
		v, err := mgr.Next("even")
		if err != nil {
			t.Fatal(err)
		}
		want := int64(i * 2)
		if v != want {
			t.Fatalf("even: got %d, want %d", v, want)
		}
	}
}

func TestSequenceUnknown(t *testing.T) {
	mgr := NewManager(nil)

	fns := []struct {
		name string
		call func() error
	}{
		{"Next", func() error { _, err := mgr.Next("nope"); return err }},
		{"Rand", func() error { _, err := mgr.Rand("nope"); return err }},
		{"Zipf", func() error { _, err := mgr.Zipf("nope", 2, 1); return err }},
		{"Norm", func() error { _, err := mgr.Norm("nope", 50, 10); return err }},
		{"Exp", func() error { _, err := mgr.Exp("nope", 0.5); return err }},
		{"Lognorm", func() error { _, err := mgr.Lognorm("nope", 1, 0.5); return err }},
	}

	for _, fn := range fns {
		if fn.call() == nil {
			t.Fatalf("%s: expected error for unknown sequence", fn.name)
		}
	}
}

func TestSequenceEmptyErrors(t *testing.T) {
	mgr := NewManager([]Config{
		{Name: "empty", Start: 1, Step: 1},
	})

	fns := []struct {
		name string
		call func() error
	}{
		{"Rand", func() error { _, err := mgr.Rand("empty"); return err }},
		{"Zipf", func() error { _, err := mgr.Zipf("empty", 2, 1); return err }},
		{"Norm", func() error { _, err := mgr.Norm("empty", 50, 10); return err }},
		{"Exp", func() error { _, err := mgr.Exp("empty", 0.5); return err }},
		{"Lognorm", func() error { _, err := mgr.Lognorm("empty", 1, 0.5); return err }},
	}

	for _, fn := range fns {
		if fn.call() == nil {
			t.Fatalf("%s: expected error for empty sequence", fn.name)
		}
	}
}

func TestSequenceConcurrent(t *testing.T) {
	mgr := NewManager([]Config{
		{Name: "shared", Start: 0, Step: 1},
	})

	const goroutines = 10
	const perGoroutine = 1000

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				mgr.Next("shared")
			}
		}()
	}
	wg.Wait()

	v, _ := mgr.Next("shared")
	want := int64(goroutines * perGoroutine)
	if v != want {
		t.Fatalf("concurrent: got %d, want %d", v, want)
	}
}

func TestHasSequence(t *testing.T) {
	mgr := NewManager([]Config{
		{Name: "orders", Start: 1, Step: 1},
	})

	if !mgr.HasSequence("orders") {
		t.Fatal("expected HasSequence to return true for 'orders'")
	}
	if mgr.HasSequence("nope") {
		t.Fatal("expected HasSequence to return false for 'nope'")
	}
}

// seedSequence advances a sequence by n, producing values start, start+step, ...
func seedSequence(mgr *Manager, name string, n int) {
	for range n {
		mgr.Next(name)
	}
}

// validValues returns the set of valid sequence values for a given config and count.
func validValues(cfg Config, count int) map[int64]bool {
	m := make(map[int64]bool, count)
	for i := range count {
		m[cfg.Start+int64(i)*cfg.Step] = true
	}
	return m
}

func TestRand(t *testing.T) {
	cfg := Config{Name: "a", Start: 10, Step: 3}
	mgr := NewManager([]Config{cfg})
	seedSequence(mgr, "a", 100)
	valid := validValues(cfg, 100)

	for range 500 {
		v, err := mgr.Rand("a")
		if err != nil {
			t.Fatal(err)
		}
		if !valid[v] {
			t.Fatalf("Rand returned invalid value %d", v)
		}
	}
}

func TestRandStep(t *testing.T) {
	cfg := Config{Name: "odd", Start: 1, Step: 2}
	mgr := NewManager([]Config{cfg})
	seedSequence(mgr, "odd", 50)
	valid := validValues(cfg, 50)

	for range 500 {
		v, err := mgr.Rand("odd")
		if err != nil {
			t.Fatal(err)
		}
		if !valid[v] {
			t.Fatalf("Rand returned %d which is not a valid odd-step value", v)
		}
	}
}

func TestZipf(t *testing.T) {
	cfg := Config{Name: "a", Start: 10, Step: 3}
	mgr := NewManager([]Config{cfg})
	seedSequence(mgr, "a", 100)
	valid := validValues(cfg, 100)

	for range 500 {
		v, err := mgr.Zipf("a", 2.0, 1.0)
		if err != nil {
			t.Fatal(err)
		}
		if !valid[v] {
			t.Fatalf("Zipf returned invalid value %d", v)
		}
	}
}

func TestZipfInvalidParams(t *testing.T) {
	mgr := NewManager([]Config{{Name: "a", Start: 1, Step: 1}})
	seedSequence(mgr, "a", 10)

	_, err := mgr.Zipf("a", 0.5, 1.0) // s must be > 1
	if err == nil {
		t.Fatal("expected error for s <= 1")
	}
}

func TestZipfSkew(t *testing.T) {
	cfg := Config{Name: "hot", Start: 1, Step: 1}
	mgr := NewManager([]Config{cfg})
	seedSequence(mgr, "hot", 1000)

	counts := make(map[int64]int)
	for range 10000 {
		v, err := mgr.Zipf("hot", 2.0, 1.0)
		if err != nil {
			t.Fatal(err)
		}
		counts[v]++
	}

	if counts[1] < counts[500] {
		t.Fatalf("Zipf should skew toward start: count[1]=%d count[500]=%d", counts[1], counts[500])
	}
}

func TestNorm(t *testing.T) {
	cfg := Config{Name: "a", Start: 10, Step: 3}
	mgr := NewManager([]Config{cfg})
	seedSequence(mgr, "a", 100)
	valid := validValues(cfg, 100)

	for range 500 {
		v, err := mgr.Norm("a", 50, 10)
		if err != nil {
			t.Fatal(err)
		}
		if !valid[v] {
			t.Fatalf("Norm returned invalid value %d", v)
		}
	}
}

func TestNormCentering(t *testing.T) {
	cfg := Config{Name: "bell", Start: 1, Step: 1}
	mgr := NewManager([]Config{cfg})
	seedSequence(mgr, "bell", 1000)

	var sum int64
	n := 5000
	for range n {
		v, err := mgr.Norm("bell", 500, 50)
		if err != nil {
			t.Fatal(err)
		}
		sum += v
	}

	avg := float64(sum) / float64(n)
	// mean=500, index 500 maps to value 501 (start=1)
	if avg < 400 || avg > 600 {
		t.Fatalf("Norm average %f should be near 501", avg)
	}
}

func TestExp(t *testing.T) {
	cfg := Config{Name: "a", Start: 10, Step: 3}
	mgr := NewManager([]Config{cfg})
	seedSequence(mgr, "a", 100)
	valid := validValues(cfg, 100)

	for range 500 {
		v, err := mgr.Exp("a", 0.1)
		if err != nil {
			t.Fatal(err)
		}
		if !valid[v] {
			t.Fatalf("Exp returned invalid value %d", v)
		}
	}
}

func TestExpSkew(t *testing.T) {
	cfg := Config{Name: "exp", Start: 1, Step: 1}
	mgr := NewManager([]Config{cfg})
	seedSequence(mgr, "exp", 1000)

	lowCount := 0
	highCount := 0
	for range 5000 {
		v, err := mgr.Exp("exp", 0.5)
		if err != nil {
			t.Fatal(err)
		}
		if v <= 100 {
			lowCount++
		}
		if v > 900 {
			highCount++
		}
	}

	if lowCount < highCount {
		t.Fatalf("Exp should skew toward low values: low=%d high=%d", lowCount, highCount)
	}
}

func TestLognorm(t *testing.T) {
	cfg := Config{Name: "a", Start: 10, Step: 3}
	mgr := NewManager([]Config{cfg})
	seedSequence(mgr, "a", 100)
	valid := validValues(cfg, 100)

	for range 500 {
		v, err := mgr.Lognorm("a", 2, 0.5)
		if err != nil {
			t.Fatal(err)
		}
		if !valid[v] {
			t.Fatalf("Lognorm returned invalid value %d", v)
		}
	}
}

func TestLognormSkew(t *testing.T) {
	cfg := Config{Name: "ln", Start: 1, Step: 1}
	mgr := NewManager([]Config{cfg})
	seedSequence(mgr, "ln", 1000)

	lowCount := 0
	highCount := 0
	for range 5000 {
		v, err := mgr.Lognorm("ln", 1, 0.5)
		if err != nil {
			t.Fatal(err)
		}
		if v <= 100 {
			lowCount++
		}
		if v > 900 {
			highCount++
		}
	}

	if lowCount < highCount {
		t.Fatalf("Lognorm should skew toward low values: low=%d high=%d", lowCount, highCount)
	}
}

func TestAllDistributionsRespectStep(t *testing.T) {
	cfg := Config{Name: "step5", Start: 100, Step: 5}
	mgr := NewManager([]Config{cfg})
	seedSequence(mgr, "step5", 200)
	valid := validValues(cfg, 200)

	type distFn struct {
		name string
		call func() (int64, error)
	}

	fns := []distFn{
		{"Rand", func() (int64, error) { return mgr.Rand("step5") }},
		{"Zipf", func() (int64, error) { return mgr.Zipf("step5", 2.0, 1.0) }},
		{"Norm", func() (int64, error) { return mgr.Norm("step5", 100, 30) }},
		{"Exp", func() (int64, error) { return mgr.Exp("step5", 0.1) }},
		{"Lognorm", func() (int64, error) { return mgr.Lognorm("step5", 2, 0.5) }},
	}

	for _, fn := range fns {
		for range 200 {
			v, err := fn.call()
			if err != nil {
				t.Fatalf("%s: %v", fn.name, err)
			}
			if !valid[v] {
				t.Fatalf("%s returned %d which is not a valid step-5 value (start=100)", fn.name, v)
			}
		}
	}
}

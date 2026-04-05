package pkg

import (
	"math"
	"testing"
)

func testEnvWithDataset(name string, data []map[string]any) *Env {
	e := &Env{
		env: map[string]any{
			name: data,
		},
	}
	return e
}

var ordersData = []map[string]any{
	{"id": 1, "amount": 10.0, "status": "open"},
	{"id": 2, "amount": 20.0, "status": "closed"},
	{"id": 3, "amount": 30.0, "status": "open"},
	{"id": 4, "amount": 40.0, "status": "pending"},
}

func TestAggSum(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggSum("orders", "amount")
	if got != 100.0 {
		t.Errorf("aggSum = %v, want 100", got)
	}
}

func TestAggSum_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggSum("missing", "amount")
	if got != 0 {
		t.Errorf("aggSum(missing) = %v, want 0", got)
	}
}

func TestAggAvg(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggAvg("orders", "amount")
	if got != 25.0 {
		t.Errorf("aggAvg = %v, want 25", got)
	}
}

func TestAggAvg_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggAvg("missing", "amount")
	if got != 0 {
		t.Errorf("aggAvg(missing) = %v, want 0", got)
	}
}

func TestAggMin(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggMin("orders", "amount")
	if got != 10.0 {
		t.Errorf("aggMin = %v, want 10", got)
	}
}

func TestAggMin_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggMin("missing", "amount")
	if got != 0 {
		t.Errorf("aggMin(missing) = %v, want 0", got)
	}
}

func TestAggMax(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggMax("orders", "amount")
	if got != 40.0 {
		t.Errorf("aggMax = %v, want 40", got)
	}
}

func TestAggMax_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggMax("missing", "amount")
	if got != 0 {
		t.Errorf("aggMax(missing) = %v, want 0", got)
	}
}

func TestAggCount(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggCount("orders")
	if got != 4 {
		t.Errorf("aggCount = %v, want 4", got)
	}
}

func TestAggCount_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggCount("missing")
	if got != 0 {
		t.Errorf("aggCount(missing) = %v, want 0", got)
	}
}

func TestAggDistinct(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggDistinct("orders", "status")
	if got != 3 {
		t.Errorf("aggDistinct = %v, want 3 (open, closed, pending)", got)
	}
}

func TestAggDistinct_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggDistinct("missing", "status")
	if got != 0 {
		t.Errorf("aggDistinct(missing) = %v, want 0", got)
	}
}

func TestAggSum_IntValues(t *testing.T) {
	data := []map[string]any{
		{"v": 1},
		{"v": 2},
		{"v": 3},
	}
	e := testEnvWithDataset("d", data)
	got := e.aggSum("d", "v")
	if got != 6.0 {
		t.Errorf("aggSum(ints) = %v, want 6", got)
	}
}

func TestAggMin_SingleRow(t *testing.T) {
	data := []map[string]any{
		{"v": 42.0},
	}
	e := testEnvWithDataset("d", data)
	got := e.aggMin("d", "v")
	if got != 42.0 {
		t.Errorf("aggMin(single) = %v, want 42", got)
	}
}

func TestAggMax_SingleRow(t *testing.T) {
	data := []map[string]any{
		{"v": 42.0},
	}
	e := testEnvWithDataset("d", data)
	got := e.aggMax("d", "v")
	if got != 42.0 {
		t.Errorf("aggMax(single) = %v, want 42", got)
	}
}

func TestAggMin_NegativeValues(t *testing.T) {
	data := []map[string]any{
		{"v": -5.0},
		{"v": -10.0},
		{"v": 3.0},
	}
	e := testEnvWithDataset("d", data)
	got := e.aggMin("d", "v")
	if got != -10.0 {
		t.Errorf("aggMin(negative) = %v, want -10", got)
	}
}

func TestAggMax_NegativeValues(t *testing.T) {
	data := []map[string]any{
		{"v": -5.0},
		{"v": -10.0},
		{"v": -1.0},
	}
	e := testEnvWithDataset("d", data)
	got := e.aggMax("d", "v")
	if got != -1.0 {
		t.Errorf("aggMax(negative) = %v, want -1", got)
	}
}

// Verify empty datasets return 0 for all functions.
func TestAgg_EmptyDataset(t *testing.T) {
	e := &Env{env: map[string]any{
		"empty": []map[string]any{},
	}}

	if got := e.aggSum("empty", "v"); got != 0 {
		t.Errorf("sum(empty) = %v, want 0", got)
	}
	if got := e.aggAvg("empty", "v"); got != 0 {
		t.Errorf("avg(empty) = %v, want 0", got)
	}
	if got := e.aggMin("empty", "v"); got != 0 {
		t.Errorf("min(empty) = %v, want 0", got)
	}
	if got := e.aggMax("empty", "v"); got != 0 {
		t.Errorf("max(empty) = %v, want 0", got)
	}
	if got := e.aggCount("empty"); got != 0 {
		t.Errorf("count(empty) = %v, want 0", got)
	}
	if got := e.aggDistinct("empty", "v"); got != 0 {
		t.Errorf("distinct(empty) = %v, want 0", got)
	}
}

func BenchmarkAggSum(b *testing.B) {
	data := make([]map[string]any, 1000)
	for i := range data {
		data[i] = map[string]any{"v": float64(i)}
	}
	e := testEnvWithDataset("d", data)

	for b.Loop() {
		e.aggSum("d", "v")
	}
}

func BenchmarkAggDistinct(b *testing.B) {
	data := make([]map[string]any, 1000)
	for i := range data {
		data[i] = map[string]any{"v": i % 50}
	}
	e := testEnvWithDataset("d", data)

	for b.Loop() {
		e.aggDistinct("d", "v")
	}
}

// Ensure math is imported (used for test expectations if needed).
var _ = math.Inf

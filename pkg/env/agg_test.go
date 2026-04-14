package env

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, 100.0, got)
}

func TestAggSum_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggSum("missing", "amount")
	assert.Equal(t, 0.0, got)
}

func TestAggAvg(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggAvg("orders", "amount")
	assert.Equal(t, 25.0, got)
}

func TestAggAvg_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggAvg("missing", "amount")
	assert.Equal(t, 0.0, got)
}

func TestAggMin(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggMin("orders", "amount")
	assert.Equal(t, 10.0, got)
}

func TestAggMin_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggMin("missing", "amount")
	assert.Equal(t, 0.0, got)
}

func TestAggMax(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggMax("orders", "amount")
	assert.Equal(t, 40.0, got)
}

func TestAggMax_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggMax("missing", "amount")
	assert.Equal(t, 0.0, got)
}

func TestAggCount(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggCount("orders")
	assert.Equal(t, 4, got)
}

func TestAggCount_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggCount("missing")
	assert.Equal(t, 0, got)
}

func TestAggDistinct(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggDistinct("orders", "status")
	assert.Equal(t, 3, got)
}

func TestAggDistinct_MissingDataset(t *testing.T) {
	e := testEnvWithDataset("orders", ordersData)
	got := e.aggDistinct("missing", "status")
	assert.Equal(t, 0, got)
}

func TestAggSum_IntValues(t *testing.T) {
	data := []map[string]any{
		{"v": 1},
		{"v": 2},
		{"v": 3},
	}
	e := testEnvWithDataset("d", data)
	got := e.aggSum("d", "v")
	assert.Equal(t, 6.0, got)
}

func TestAggMin_SingleRow(t *testing.T) {
	data := []map[string]any{
		{"v": 42.0},
	}
	e := testEnvWithDataset("d", data)
	got := e.aggMin("d", "v")
	assert.Equal(t, 42.0, got)
}

func TestAggMax_SingleRow(t *testing.T) {
	data := []map[string]any{
		{"v": 42.0},
	}
	e := testEnvWithDataset("d", data)
	got := e.aggMax("d", "v")
	assert.Equal(t, 42.0, got)
}

func TestAggMin_NegativeValues(t *testing.T) {
	data := []map[string]any{
		{"v": -5.0},
		{"v": -10.0},
		{"v": 3.0},
	}
	e := testEnvWithDataset("d", data)
	got := e.aggMin("d", "v")
	assert.Equal(t, -10.0, got)
}

func TestAggMax_NegativeValues(t *testing.T) {
	data := []map[string]any{
		{"v": -5.0},
		{"v": -10.0},
		{"v": -1.0},
	}
	e := testEnvWithDataset("d", data)
	got := e.aggMax("d", "v")
	assert.Equal(t, -1.0, got)
}

// Verify empty datasets return 0 for all functions.
func TestAgg_EmptyDataset(t *testing.T) {
	e := &Env{env: map[string]any{
		"empty": []map[string]any{},
	}}

	assert.Equal(t, 0.0, e.aggSum("empty", "v"))
	assert.Equal(t, 0.0, e.aggAvg("empty", "v"))
	assert.Equal(t, 0.0, e.aggMin("empty", "v"))
	assert.Equal(t, 0.0, e.aggMax("empty", "v"))
	assert.Equal(t, 0, e.aggCount("empty"))
	assert.Equal(t, 0, e.aggDistinct("empty", "v"))
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

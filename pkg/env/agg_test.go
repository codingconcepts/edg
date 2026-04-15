package env

import (
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
	cases := []struct {
		name    string
		dataset string
		data    map[string][]map[string]any
		field   string
		want    float64
	}{
		{"orders", "orders", map[string][]map[string]any{"orders": ordersData}, "amount", 100.0},
		{"missing dataset", "missing", map[string][]map[string]any{"orders": ordersData}, "amount", 0.0},
		{"int values", "d", map[string][]map[string]any{"d": {{"v": 1}, {"v": 2}, {"v": 3}}}, "v", 6.0},
		{"empty dataset", "empty", map[string][]map[string]any{"empty": {}}, "v", 0.0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := &Env{env: map[string]any{}}
			for k, v := range c.data {
				e.env[k] = v
			}
			assert.Equal(t, c.want, e.aggSum(c.dataset, c.field))
		})
	}
}

func TestAggAvg(t *testing.T) {
	cases := []struct {
		name    string
		dataset string
		want    float64
	}{
		{"orders", "orders", 25.0},
		{"missing dataset", "missing", 0.0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := testEnvWithDataset("orders", ordersData)
			assert.Equal(t, c.want, e.aggAvg(c.dataset, "amount"))
		})
	}
}

func TestAggMin(t *testing.T) {
	cases := []struct {
		name    string
		dataset string
		data    map[string][]map[string]any
		field   string
		want    float64
	}{
		{"orders", "orders", map[string][]map[string]any{"orders": ordersData}, "amount", 10.0},
		{"missing dataset", "missing", map[string][]map[string]any{"orders": ordersData}, "amount", 0.0},
		{"single row", "d", map[string][]map[string]any{"d": {{"v": 42.0}}}, "v", 42.0},
		{"negative values", "d", map[string][]map[string]any{"d": {{"v": -5.0}, {"v": -10.0}, {"v": 3.0}}}, "v", -10.0},
		{"empty dataset", "empty", map[string][]map[string]any{"empty": {}}, "v", 0.0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := &Env{env: map[string]any{}}
			for k, v := range c.data {
				e.env[k] = v
			}
			assert.Equal(t, c.want, e.aggMin(c.dataset, c.field))
		})
	}
}

func TestAggMax(t *testing.T) {
	cases := []struct {
		name    string
		dataset string
		data    map[string][]map[string]any
		field   string
		want    float64
	}{
		{"orders", "orders", map[string][]map[string]any{"orders": ordersData}, "amount", 40.0},
		{"missing dataset", "missing", map[string][]map[string]any{"orders": ordersData}, "amount", 0.0},
		{"single row", "d", map[string][]map[string]any{"d": {{"v": 42.0}}}, "v", 42.0},
		{"negative values", "d", map[string][]map[string]any{"d": {{"v": -5.0}, {"v": -10.0}, {"v": -1.0}}}, "v", -1.0},
		{"empty dataset", "empty", map[string][]map[string]any{"empty": {}}, "v", 0.0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := &Env{env: map[string]any{}}
			for k, v := range c.data {
				e.env[k] = v
			}
			assert.Equal(t, c.want, e.aggMax(c.dataset, c.field))
		})
	}
}

func TestAggCount(t *testing.T) {
	cases := []struct {
		name    string
		dataset string
		want    int
	}{
		{"orders", "orders", 4},
		{"missing dataset", "missing", 0},
		{"empty dataset", "empty", 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := testEnvWithDataset("orders", ordersData)
			e.env["empty"] = []map[string]any{}
			assert.Equal(t, c.want, e.aggCount(c.dataset))
		})
	}
}

func TestAggDistinct(t *testing.T) {
	cases := []struct {
		name    string
		dataset string
		field   string
		want    int
	}{
		{"orders", "orders", "status", 3},
		{"missing dataset", "missing", "status", 0},
		{"empty dataset", "empty", "v", 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := testEnvWithDataset("orders", ordersData)
			e.env["empty"] = []map[string]any{}
			assert.Equal(t, c.want, e.aggDistinct(c.dataset, c.field))
		})
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

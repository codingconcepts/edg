package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckExpectations_NoExpectations(t *testing.T) {
	require.NoError(t, checkExpectations(nil, nil, time.Minute))
}

func TestCheckExpectations_AllPass(t *testing.T) {
	stats := map[string]*queryStats{
		"read_balance": {
			count:        1000,
			errors:       5,
			totalLatency: 50 * time.Second,
			latencies:    makeDurations(1000, 50*time.Millisecond),
		},
	}

	expectations := []string{
		"error_rate < 1",
		"error_rate < 0.5",
		"read_balance.error_rate < 1",
		"read_balance.error_count < 10",
		"read_balance.p99 < 100",
		"read_balance.qps > 10",
	}

	require.NoError(t, checkExpectations(expectations, stats, time.Minute))
}

func TestCheckExpectations_SomeFail(t *testing.T) {
	stats := map[string]*queryStats{
		"slow_query": {
			count:        100,
			errors:       20,
			totalLatency: 100 * time.Second,
			latencies:    makeDurations(100, time.Second),
		},
	}

	expectations := []string{
		"error_rate < 1",        // 20/(100+20)=16.7% -> FAIL
		"success_count > 0",     // 100 -> PASS
		"slow_query.avg < 100",  // 1000ms -> FAIL
	}

	err := checkExpectations(expectations, stats, time.Minute)
	require.Error(t, err)
	require.EqualError(t, err, "2 expectation(s) failed")
}

func TestCheckExpectations_InvalidExpression(t *testing.T) {
	stats := map[string]*queryStats{}

	expectations := []string{"??? invalid"}

	err := checkExpectations(expectations, stats, time.Minute)
	require.Error(t, err)
}

func TestCheckExpectations_PerQueryMetrics(t *testing.T) {
	stats := map[string]*queryStats{
		"fast_query": {
			count:        500,
			errors:       0,
			totalLatency: 5 * time.Second,
			latencies:    makeDurations(500, 10*time.Millisecond),
		},
		"slow_query": {
			count:        500,
			errors:       0,
			totalLatency: 500 * time.Second,
			latencies:    makeDurations(500, time.Second),
		},
	}

	expectations := []string{
		"fast_query.p99 < 50",
		"slow_query.p99 > 500",
	}

	require.NoError(t, checkExpectations(expectations, stats, time.Minute))
}

func TestCheckExpectations_TotalMetrics(t *testing.T) {
	stats := map[string]*queryStats{
		"q1": {
			count:        600,
			errors:       2,
			totalLatency: 30 * time.Second,
			latencies:    makeDurations(600, 50*time.Millisecond),
		},
		"q2": {
			count:        400,
			errors:       3,
			totalLatency: 20 * time.Second,
			latencies:    makeDurations(400, 50*time.Millisecond),
		},
	}

	expectations := []string{
		"success_count == 1000",
		"total_errors == 5",
		"tpm > 0",
	}

	require.NoError(t, checkExpectations(expectations, stats, time.Minute))
}

func TestErrorRate(t *testing.T) {
	tests := []struct {
		name          string
		errors        int64
		successfulOps int
		want          float64
	}{
		{"no ops", 0, 0, 0},
		{"no errors", 0, 100, 0},
		{"all errors", 10, 0, 100},
		{"half errors", 50, 50, 50},
		{"one percent", 1, 99, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errorRate(tt.errors, tt.successfulOps)
			assert.Equal(t, tt.want, got)
		})
	}
}

// makeDurations creates n identical durations for testing.
func makeDurations(n int, d time.Duration) []time.Duration {
	out := make([]time.Duration, n)
	for i := range out {
		out[i] = d
	}
	return out
}

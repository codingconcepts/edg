package main

import (
	"bytes"
	"errors"
	"io"
	"math"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestPrintResults_BasicAccumulation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		results := make(chan config.QueryResult)

		go func() {
			results <- config.QueryResult{Name: "read", Latency: 10 * time.Millisecond, Count: 1}
			results <- config.QueryResult{Name: "read", Latency: 20 * time.Millisecond, Count: 1}
			results <- config.QueryResult{Name: "write", Latency: 5 * time.Millisecond, Count: 3}
			close(results)
		}()

		stats := printResults(io.Discard, results, time.Second, time.Now(), 1, time.Minute, 0)

		assert.Equal(t, int64(2), stats["read"].count)
		assert.Equal(t, 30*time.Millisecond, stats["read"].totalLatency)
		assert.Equal(t, int64(3), stats["write"].count)
		assert.Equal(t, 5*time.Millisecond, stats["write"].totalLatency)
	})
}

func TestPrintResults_ErrorCounting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		results := make(chan config.QueryResult)

		go func() {
			results <- config.QueryResult{Name: "read", Latency: 10 * time.Millisecond, Count: 1}
			results <- config.QueryResult{Name: "read", Err: errors.New("connection refused")}
			results <- config.QueryResult{Name: "read", Err: errors.New("timeout")}
			close(results)
		}()

		stats := printResults(io.Discard, results, time.Second, time.Now(), 1, time.Minute, 0)

		assert.Equal(t, int64(1), stats["read"].count)
		assert.Equal(t, int64(2), stats["read"].errors)
	})
}

func TestPrintResults_TransactionTracking(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		results := make(chan config.QueryResult)

		go func() {
			results <- config.QueryResult{Name: "transfer", Latency: 10 * time.Millisecond, Count: 1, IsTransaction: true, Rollback: false}
			results <- config.QueryResult{Name: "transfer", Latency: 15 * time.Millisecond, Count: 1, IsTransaction: true, Rollback: false}
			results <- config.QueryResult{Name: "transfer", Latency: 8 * time.Millisecond, Count: 1, IsTransaction: true, Rollback: true}
			close(results)
		}()

		stats := printResults(io.Discard, results, time.Second, time.Now(), 1, time.Minute, 0)

		assert.True(t, stats["transfer"].isTransaction)
		assert.Equal(t, int64(2), stats["transfer"].commits)
		assert.Equal(t, int64(1), stats["transfer"].rollbacks)
		assert.Equal(t, int64(3), stats["transfer"].count)
	})
}

func TestPrintResults_WarmupFiltersResults(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		results := make(chan config.QueryResult)

		go func() {
			// Sent during warmup - should be discarded.
			results <- config.QueryResult{Name: "read", Latency: 10 * time.Millisecond, Count: 1}
			results <- config.QueryResult{Name: "read", Latency: 20 * time.Millisecond, Count: 1}

			// Wait past warmup to ensure deadline has fired.
			time.Sleep(6 * time.Second)

			// Sent after warmup - should be counted.
			results <- config.QueryResult{Name: "read", Latency: 30 * time.Millisecond, Count: 1}
			close(results)
		}()

		stats := printResults(io.Discard, results, time.Second, time.Now(), 1, time.Minute, 5*time.Second)

		assert.Equal(t, int64(1), stats["read"].count)
		assert.Equal(t, 30*time.Millisecond, stats["read"].totalLatency)
	})
}

func TestPrintResults_NoWarmupCountsAll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		results := make(chan config.QueryResult)

		go func() {
			results <- config.QueryResult{Name: "read", Latency: 10 * time.Millisecond, Count: 1}
			results <- config.QueryResult{Name: "read", Latency: 20 * time.Millisecond, Count: 1}
			results <- config.QueryResult{Name: "read", Latency: 30 * time.Millisecond, Count: 1}
			close(results)
		}()

		stats := printResults(io.Discard, results, time.Second, time.Now(), 1, time.Minute, 0)

		assert.Equal(t, int64(3), stats["read"].count)
		assert.Equal(t, 60*time.Millisecond, stats["read"].totalLatency)
	})
}

func TestPrintProgress_QueryStats(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var buf bytes.Buffer
		start := time.Now()
		time.Sleep(10 * time.Second)

		stats := map[string]*queryStats{
			"read": {
				count:        100,
				errors:       2,
				totalLatency: 500 * time.Millisecond,
				latencies:    uniformLatencies(100, 5*time.Millisecond),
			},
		}

		printProgress(&buf, stats, start, time.Minute)
		out := buf.String()

		assert.Contains(t, out, "10s / 1m0s")
		assert.Contains(t, out, "QUERY")
		assert.Contains(t, out, "read")
		assert.Contains(t, out, "100")
		assert.Contains(t, out, "5ms")
	})
}

func TestPrintProgress_TransactionStats(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var buf bytes.Buffer
		start := time.Now()
		time.Sleep(10 * time.Second)

		stats := map[string]*queryStats{
			"transfer": {
				count:         50,
				errors:        1,
				commits:       45,
				rollbacks:     4,
				totalLatency:  500 * time.Millisecond,
				latencies:     uniformLatencies(50, 10*time.Millisecond),
				isTransaction: true,
			},
		}

		printProgress(&buf, stats, start, time.Minute)
		out := buf.String()

		assert.Contains(t, out, "TRANSACTION")
		assert.Contains(t, out, "transfer")
		assert.Contains(t, out, "45")
		assert.Contains(t, out, "10ms")
	})
}

func TestPrintProgress_MixedQueryAndTransaction(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var buf bytes.Buffer
		start := time.Now()
		time.Sleep(10 * time.Second)

		stats := map[string]*queryStats{
			"read": {
				count:        100,
				totalLatency: 500 * time.Millisecond,
				latencies:    uniformLatencies(100, 5*time.Millisecond),
			},
			"transfer": {
				count:         50,
				commits:       50,
				totalLatency:  500 * time.Millisecond,
				latencies:     uniformLatencies(50, 10*time.Millisecond),
				isTransaction: true,
			},
		}

		printProgress(&buf, stats, start, time.Minute)
		out := buf.String()

		assert.Contains(t, out, "QUERY")
		assert.Contains(t, out, "TRANSACTION")
		assert.Contains(t, out, "read")
		assert.Contains(t, out, "transfer")
	})
}

func TestPrintProgress_CustomPrintSuppressesStats(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var buf bytes.Buffer
		start := time.Now()
		time.Sleep(10 * time.Second)

		stats := map[string]*queryStats{
			"insert": {
				count:        100,
				totalLatency: 500 * time.Millisecond,
				latencies:    uniformLatencies(100, 5*time.Millisecond),
				printAggExprs: []string{""},
				printAggs: []*printAgg{
					{
						freq:  map[string]int64{"us": 60, "eu": 40},
						count: 100,
						min:   math.MaxFloat64,
						max:   -math.MaxFloat64,
					},
				},
			},
		}

		printProgress(&buf, stats, start, time.Minute)
		out := buf.String()

		assert.NotContains(t, out, "QUERY\t")
		assert.NotContains(t, out, "TRANSACTION")
		assert.Contains(t, out, "PRINT")
		assert.Contains(t, out, "us=60")
		assert.Contains(t, out, "eu=40")
	})
}

func TestPrintProgress_EmptyStats(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var buf bytes.Buffer
		start := time.Now()
		time.Sleep(5 * time.Second)

		stats := map[string]*queryStats{}

		printProgress(&buf, stats, start, time.Minute)
		out := buf.String()

		assert.Contains(t, out, "5s / 1m0s")
		assert.Contains(t, out, "QUERY")
	})
}

func TestPrintProgress_CustomAggExpr(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var buf bytes.Buffer
		start := time.Now()
		time.Sleep(10 * time.Second)

		stats := map[string]*queryStats{
			"insert": {
				count:        200,
				totalLatency: 1 * time.Second,
				latencies:    uniformLatencies(200, 5*time.Millisecond),
				printAggExprs: []string{
					"string(int(min)) + '-' + string(int(max))",
				},
				printAggs: []*printAgg{
					{
						freq:     map[string]int64{"10": 100, "20": 100},
						sum:      3000,
						min:      10,
						max:      20,
						count:    200,
						numCount: 200,
					},
				},
			},
		}

		printProgress(&buf, stats, start, time.Minute)
		out := buf.String()

		assert.NotContains(t, out, "QUERY\t")
		assert.Contains(t, out, "PRINT")
		assert.True(t, strings.Contains(out, "10-20"))
	})
}

func uniformLatencies(n int, d time.Duration) []time.Duration {
	l := make([]time.Duration, n)
	for i := range l {
		l[i] = d
	}
	return l
}

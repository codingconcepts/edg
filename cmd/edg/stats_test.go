package main

import (
	"errors"
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

		stats := printResults(results, time.Second, time.Now(), 1, time.Minute, 0)

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

		stats := printResults(results, time.Second, time.Now(), 1, time.Minute, 0)

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

		stats := printResults(results, time.Second, time.Now(), 1, time.Minute, 0)

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

		stats := printResults(results, time.Second, time.Now(), 1, time.Minute, 5*time.Second)

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

		stats := printResults(results, time.Second, time.Now(), 1, time.Minute, 0)

		assert.Equal(t, int64(3), stats["read"].count)
		assert.Equal(t, 60*time.Millisecond, stats["read"].totalLatency)
	})
}

package main

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/codingconcepts/edg/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckExpectations_NoExpectations(t *testing.T) {
	require.NoError(t, checkExpectations(nil, nil, nil, nil, time.Minute))
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

	expectations := []config.Expectation{
		{Expr: "error_rate < 1"},
		{Expr: "error_rate < 0.5"},
		{Expr: "read_balance.error_rate < 1"},
		{Expr: "read_balance.error_count < 10"},
		{Expr: "read_balance.p99 < 100"},
		{Expr: "read_balance.qps > 10"},
	}

	require.NoError(t, checkExpectations(nil, expectations, nil, stats, time.Minute))
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

	expectations := []config.Expectation{
		{Expr: "error_rate < 1"},       // 20/(100+20)=16.7% -> FAIL
		{Expr: "success_count > 0"},    // 100 -> PASS
		{Expr: "slow_query.avg < 100"}, // 1000ms -> FAIL
	}

	err := checkExpectations(nil, expectations, nil, stats, time.Minute)
	require.Error(t, err)
	require.EqualError(t, err, "2 expectation(s) failed")
}

func TestCheckExpectations_InvalidExpression(t *testing.T) {
	err := checkExpectations(nil, []config.Expectation{{Expr: "??? invalid"}}, nil, nil, time.Minute)
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

	expectations := []config.Expectation{
		{Expr: "fast_query.p99 < 50"},
		{Expr: "slow_query.p99 > 500"},
	}

	require.NoError(t, checkExpectations(nil, expectations, nil, stats, time.Minute))
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

	expectations := []config.Expectation{
		{Expr: "success_count == 1000"},
		{Expr: "total_errors == 5"},
		{Expr: "tpm > 0"},
	}

	require.NoError(t, checkExpectations(nil, expectations, nil, stats, time.Minute))
}

func TestCheckExpectations_QueryExpectation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(int64(42)))

	expectations := []config.Expectation{
		{Query: "SELECT COUNT(*) AS cnt FROM account", Expr: "cnt == 42"},
	}

	require.NoError(t, checkExpectations(db, expectations, nil, nil, time.Minute))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckExpectations_QueryExpectationFail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(int64(0)))

	expectations := []config.Expectation{
		{Query: "SELECT COUNT(*) AS cnt FROM account", Expr: "cnt > 0"},
	}

	err = checkExpectations(db, expectations, nil, nil, time.Minute)
	require.Error(t, err)
	require.EqualError(t, err, "1 expectation(s) failed")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckExpectations_MixedExpectations(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(int64(100)))

	stats := map[string]*queryStats{
		"q1": {
			count:        500,
			errors:       0,
			totalLatency: 5 * time.Second,
			latencies:    makeDurations(500, 10*time.Millisecond),
		},
	}

	expectations := []config.Expectation{
		{Expr: "error_rate < 1"},
		{Query: "SELECT COUNT(*) AS cnt FROM account", Expr: "cnt == 100"},
	}

	require.NoError(t, checkExpectations(db, expectations, nil, stats, time.Minute))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckExpectations_GlobalVariables(t *testing.T) {
	globals := map[string]any{
		"account_count": 100,
		"max_error_pct": 5.0,
	}

	stats := map[string]*queryStats{
		"q1": {
			count:        100,
			errors:       2,
			totalLatency: 5 * time.Second,
			latencies:    makeDurations(100, 50*time.Millisecond),
		},
	}

	expectations := []config.Expectation{
		{Expr: "success_count == account_count"},
		{Expr: "error_rate < max_error_pct"},
	}

	require.NoError(t, checkExpectations(nil, expectations, globals, stats, time.Minute))
}

func TestCheckExpectations_GlobalVariablesWithQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(int64(42)))

	globals := map[string]any{
		"expected_rows": int64(42),
	}

	expectations := []config.Expectation{
		{Query: "SELECT COUNT(*) AS cnt FROM account", Expr: "cnt == expected_rows"},
	}

	require.NoError(t, checkExpectations(db, expectations, globals, nil, time.Minute))
	require.NoError(t, mock.ExpectationsWereMet())
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

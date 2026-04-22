package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	envpkg "github.com/codingconcepts/edg/pkg/env"
	"github.com/expr-lang/expr"
)

func checkExpectations(db *sql.DB, expectations []config.Expectation, globals map[string]any, stats map[string]*queryStats, elapsed time.Duration) error {
	if len(expectations) == 0 {
		return nil
	}

	// Build the expression environment from globals and collected stats.
	env := map[string]any{}
	maps.Copy(env, globals)

	var totalCount int64
	var totalErrors int64
	var totalOps int

	for name, s := range stats {
		totalCount += s.count
		totalErrors += s.errors
		totalOps += len(s.latencies)

		var avg float64
		if s.count > 0 {
			avg = float64(s.totalLatency) / float64(s.count) / float64(time.Millisecond)
		}
		p50, p95, p99 := percentiles(s.latencies)
		qps := float64(s.count) / elapsed.Seconds()

		env[name] = map[string]any{
			config.MetricSuccessCount: s.count,
			config.MetricErrorCount:   s.errors,
			config.MetricErrorRate:    errorRate(s.errors, len(s.latencies)),
			config.MetricAvg:          avg,
			config.MetricP50:          float64(p50) / float64(time.Millisecond),
			config.MetricP95:          float64(p95) / float64(time.Millisecond),
			config.MetricP99:          float64(p99) / float64(time.Millisecond),
			config.MetricQPS:          qps,
		}
	}

	env[config.MetricSuccessCount] = totalCount
	env[config.MetricTotalErrors] = totalErrors
	env[config.MetricErrorRate] = errorRate(totalErrors, totalOps)
	env[config.MetricTPM] = float64(totalCount) / elapsed.Minutes()

	var failures int
	slog.Info("expectations")
	for _, e := range expectations {
		evalEnv := env

		if e.Query != "" {
			if db == nil {
				slog.Error("expectation failed", "check", e.Expr, "error", "query expectation requires a database connection")
				failures++
				continue
			}
			rows, err := db.QueryContext(context.Background(), e.Query)
			if err != nil {
				slog.Error("expectation failed", "check", e.Expr, "error", err)
				failures++
				continue
			}
			data, err := envpkg.ReadRows(rows)
			if err != nil {
				slog.Error("expectation failed", "check", e.Expr, "error", err)
				failures++
				continue
			}
			if len(data) > 0 {
				evalEnv = make(map[string]any, len(env)+len(data[0]))
				for k, v := range env {
					evalEnv[k] = v
				}
				for k, v := range data[0] {
					evalEnv[k] = v
				}
			}
		}

		program, err := expr.Compile(e.Expr, expr.Env(evalEnv), expr.AsBool())
		if err != nil {
			slog.Error("expectation failed", "check", e.Expr, "error", err)
			failures++
			continue
		}

		result, err := expr.Run(program, evalEnv)
		if err != nil {
			slog.Error("expectation failed", "check", e.Expr, "error", err)
			failures++
			continue
		}

		passed, ok := result.(bool)
		if !ok {
			slog.Warn("non-bool result", "check", e.Expr, "result", result)
			continue
		}

		if passed {
			slog.Info("expectation passed", "check", e.Expr)
		} else {
			slog.Error("expectation failed", "check", e.Expr)
			failures++
		}
	}

	if failures > 0 {
		return fmt.Errorf("%d expectation(s) failed", failures)
	}
	return nil
}

func errorRate(errors int64, successfulOps int) float64 {
	total := int64(successfulOps) + errors
	if total == 0 {
		return 0
	}
	return float64(errors) / float64(total) * 100
}

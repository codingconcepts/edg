package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/expr-lang/expr"
)

func checkExpectations(expectations []string, stats map[string]*queryStats, elapsed time.Duration) error {
	if len(expectations) == 0 {
		return nil
	}

	// Build the expression environment from collected stats.
	env := map[string]any{}

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
			"success_count": s.count,
			"error_count":   s.errors,
			"error_rate":    errorRate(s.errors, len(s.latencies)),
			"avg":           avg,
			"p50":           float64(p50) / float64(time.Millisecond),
			"p95":           float64(p95) / float64(time.Millisecond),
			"p99":           float64(p99) / float64(time.Millisecond),
			"qps":           qps,
		}
	}

	env["success_count"] = totalCount
	env["total_errors"] = totalErrors
	env["error_rate"] = errorRate(totalErrors, totalOps)
	env["tpm"] = float64(totalCount) / elapsed.Minutes()

	var failures int
	slog.Info("expectations")
	for _, check := range expectations {
		program, err := expr.Compile(check, expr.Env(env), expr.AsBool())
		if err != nil {
			slog.Error("expectation failed", "check", check, "error", err)
			failures++
			continue
		}

		result, err := expr.Run(program, env)
		if err != nil {
			slog.Error("expectation failed", "check", check, "error", err)
			failures++
			continue
		}

		passed, ok := result.(bool)
		if !ok {
			slog.Warn("non-bool result", "check", check, "result", result)
			continue
		}

		if passed {
			slog.Info("expectation passed", "check", check)
		} else {
			slog.Error("expectation failed", "check", check)
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

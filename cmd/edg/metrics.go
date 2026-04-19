package main

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "edg_query_duration_seconds",
		Help:    "Latency of individual query executions.",
		Buckets: prometheus.DefBuckets,
	}, []string{"query"})

	metricQueryErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "edg_query_errors_total",
		Help: "Total number of query errors.",
	}, []string{"query"})

	metricTxCommits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "edg_transaction_commits_total",
		Help: "Total number of committed transactions.",
	}, []string{"query"})

	metricTxRollbacks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "edg_transaction_rollbacks_total",
		Help: "Total number of rolled-back transactions.",
	}, []string{"query"})

	metricWorkers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "edg_workers",
		Help: "Number of concurrent workers.",
	})
)

// startMetricsServer starts an HTTP server that exposes /metrics.
// It blocks, so call it in a goroutine.
func startMetricsServer(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	slog.Info("metrics server listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		slog.Error("metrics server error", "error", err)
	}
}

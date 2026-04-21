package main

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/codingconcepts/edg/pkg/seq"
)

func run(ctx context.Context, cancel context.CancelFunc, db *sql.DB, req *config.Request, duration time.Duration, workers int, printInterval time.Duration) error {
	if flagMetricsAddr != "" {
		go startMetricsServer(flagMetricsAddr)
	}

	var stats map[string]*queryStats
	var elapsed time.Duration
	var err error

	if len(req.Stages) > 0 {
		stats, elapsed, err = runStages(ctx, cancel, db, req, printInterval)
	} else {
		stats, elapsed, err = runStage(ctx, cancel, db, req, duration, workers, printInterval)
	}
	if err != nil {
		return err
	}

	return checkExpectations(req.Expectations, stats, elapsed)
}

func runStages(ctx context.Context, _ context.CancelFunc, db *sql.DB, req *config.Request, printInterval time.Duration) (map[string]*queryStats, time.Duration, error) {
	seqMgr := seq.NewManager(req.Seq)

	initEnv, err := env.NewEnv(db, flagDriver, req)
	if err != nil {
		return nil, 0, err
	}
	defer initEnv.Close()
	initEnv.SetSeqManager(seqMgr)
	if err := initEnv.Init(ctx); err != nil {
		return nil, 0, err
	}

	results := make(chan config.QueryResult, 1000)
	start := time.Now()

	go func() {
		defer close(results)

		bgCtx, bgCancel := context.WithCancel(ctx)
		defer bgCancel()
		bgWg := startBackgroundWorkers(bgCtx, db, req, initEnv, results, seqMgr)

		for _, stage := range req.Stages {
			if ctx.Err() != nil {
				break
			}

			dur := time.Duration(stage.Duration)
			workers := stage.Workers
			if workers <= 0 {
				workers = 1
			}

			slog.Info("stage", "name", stage.Name, "workers", workers, "duration", dur)
			metricWorkers.Set(float64(workers))

			stageCtx, stageCancel := context.WithTimeout(ctx, dur)
			wg := startWorkers(stageCtx, workers, db, req, initEnv, results, seqMgr)
			wg.Wait()
			stageCancel()
		}

		bgCancel()
		bgWg.Wait()
	}()

	totalWorkers := 0
	for _, s := range req.Stages {
		if s.Workers > totalWorkers {
			totalWorkers = s.Workers
		}
	}
	var totalDuration time.Duration
	for _, s := range req.Stages {
		totalDuration += time.Duration(s.Duration)
	}

	stats := printResults(results, printInterval, start, totalWorkers, totalDuration)

	return stats, time.Since(start), nil
}

func runStage(ctx context.Context, cancel context.CancelFunc, db *sql.DB, req *config.Request, duration time.Duration, workers int, printInterval time.Duration) (map[string]*queryStats, time.Duration, error) {
	seqMgr := seq.NewManager(req.Seq)

	initEnv, err := env.NewEnv(db, flagDriver, req)
	if err != nil {
		return nil, 0, err
	}
	defer initEnv.Close()
	initEnv.SetSeqManager(seqMgr)
	if err := initEnv.Init(ctx); err != nil {
		return nil, 0, err
	}

	results := make(chan config.QueryResult, workers*100)
	start := time.Now()

	go func() {
		select {
		case <-time.After(duration):
			cancel()
		case <-ctx.Done():
		}
	}()

	wg := startWorkers(ctx, workers, db, req, initEnv, results, seqMgr)
	bgWg := startBackgroundWorkers(ctx, db, req, initEnv, results, seqMgr)

	go func() {
		wg.Wait()
		bgWg.Wait()
		close(results)
	}()

	metricWorkers.Set(float64(workers))
	slog.Info("running", "workers", workers, "duration", duration)
	stats := printResults(results, printInterval, start, workers, duration)

	return stats, time.Since(start), nil
}

func startWorkers(ctx context.Context, numWorkers int, db *sql.DB, req *config.Request, initEnv *env.Env, results chan<- config.QueryResult, seqMgr *seq.Manager) *sync.WaitGroup {
	var wg sync.WaitGroup

	for i := range numWorkers {
		wg.Go(func() {
			workerEnv, err := env.NewEnv(db, flagDriver, req)
			if err != nil {
				slog.Error("env error", "worker", i, "error", err)
				return
			}
			defer workerEnv.Close()
			workerEnv.InitFrom(initEnv)
			workerEnv.SetSeqManager(seqMgr)
			workerEnv.Results = results

			for ctx.Err() == nil {
				if err := workerEnv.RunIteration(ctx); err != nil {
					if ctx.Err() != nil {
						return
					}
					slog.Error("run error", "worker", i, "error", err)
					continue
				}
			}
		})
	}
	return &wg
}

func startBackgroundWorkers(ctx context.Context, db *sql.DB, req *config.Request, initEnv *env.Env, results chan<- config.QueryResult, seqMgr *seq.Manager) *sync.WaitGroup {
	var wg sync.WaitGroup
	for _, w := range req.Workers {
		slog.Info("background worker", "name", w.Name, "rate", w.Rate)

		wg.Go(func() {
			workerEnv, err := env.NewEnv(db, flagDriver, req)
			if err != nil {
				slog.Error("worker env error", "worker", w.Name, "error", err)
				return
			}
			defer workerEnv.Close()

			workerEnv.InitFrom(initEnv)
			workerEnv.SetSeqManager(seqMgr)
			workerEnv.Results = results
			workerEnv.RunWorker(ctx, w)
		})
	}
	return &wg
}

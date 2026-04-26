package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/db"
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/codingconcepts/edg/pkg/seq"
)

type workerDeps struct {
	DB      db.DB
	Req     *config.Request
	InitEnv *env.Env
	Results chan<- config.QueryResult
	SeqMgr  *seq.Manager
}

func run(ctx context.Context, cancel context.CancelFunc, p workload.RunParams) error {
	if flagMetricsAddr != "" {
		go startMetricsServer(flagMetricsAddr)
	}

	var stats map[string]*queryStats
	var elapsed time.Duration
	var err error

	if len(p.Req.Stages) > 0 {
		stats, elapsed, err = runStages(ctx, cancel, p)
	} else {
		stats, elapsed, err = runStage(ctx, cancel, p)
	}
	if err != nil {
		return err
	}

	return checkExpectations(p.DB, p.Req.Expectations, p.Req.Globals, stats, elapsed)
}

func runStages(ctx context.Context, _ context.CancelFunc, p workload.RunParams) (map[string]*queryStats, time.Duration, error) {
	seqMgr := seq.NewManager(p.Req.Seq)

	initEnv, err := env.NewEnv(p.DB, flagDriver, p.Req, config.ConfigSectionsRun...)
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

	deps := workerDeps{DB: p.DB, Req: p.Req, InitEnv: initEnv, Results: results, SeqMgr: seqMgr}

	go func() {
		defer close(results)

		bgCtx, bgCancel := context.WithCancel(ctx)
		defer bgCancel()
		bgWg := startBackgroundWorkers(bgCtx, deps)

		for _, stage := range p.Req.Stages {
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
			wg := startWorkers(stageCtx, workers, deps)
			wg.Wait()
			stageCancel()
		}

		bgCancel()
		bgWg.Wait()
	}()

	totalWorkers := 0
	for _, s := range p.Req.Stages {
		if s.Workers > totalWorkers {
			totalWorkers = s.Workers
		}
	}
	var totalDuration time.Duration
	for _, s := range p.Req.Stages {
		totalDuration += time.Duration(s.Duration)
	}

	stats := printResults(os.Stdout, results, p.PrintInterval, start, totalWorkers, totalDuration, p.WarmupDuration)

	return stats, time.Since(start), nil
}

func runStage(ctx context.Context, cancel context.CancelFunc, p workload.RunParams) (map[string]*queryStats, time.Duration, error) {
	seqMgr := seq.NewManager(p.Req.Seq)

	initEnv, err := env.NewEnv(p.DB, flagDriver, p.Req, config.ConfigSectionsRun...)
	if err != nil {
		return nil, 0, err
	}
	defer initEnv.Close()
	initEnv.SetSeqManager(seqMgr)
	if err := initEnv.Init(ctx); err != nil {
		return nil, 0, err
	}

	results := make(chan config.QueryResult, p.Workers*100)
	start := time.Now()

	go func() {
		select {
		case <-time.After(p.WarmupDuration + p.Duration):
			cancel()
		case <-ctx.Done():
		}
	}()

	deps := workerDeps{DB: p.DB, Req: p.Req, InitEnv: initEnv, Results: results, SeqMgr: seqMgr}

	wg := startWorkers(ctx, p.Workers, deps)
	bgWg := startBackgroundWorkers(ctx, deps)

	go func() {
		wg.Wait()
		bgWg.Wait()
		close(results)
	}()

	metricWorkers.Set(float64(p.Workers))
	if p.WarmupDuration > 0 {
		slog.Info("warming up", "duration", p.WarmupDuration)
	}
	slog.Info("running", "workers", p.Workers, "duration", p.Duration)
	stats := printResults(os.Stdout, results, p.PrintInterval, start, p.Workers, p.Duration, p.WarmupDuration)

	return stats, time.Since(start), nil
}

func startWorkers(ctx context.Context, numWorkers int, d workerDeps) *sync.WaitGroup {
	var wg sync.WaitGroup

	for i := range numWorkers {
		wg.Go(func() {
			workerEnv, err := env.NewEnv(d.DB, flagDriver, d.Req, config.ConfigSectionRun)
			if err != nil {
				slog.Error("env error", "worker", i, "error", err)
				return
			}
			defer workerEnv.Close()
			workerEnv.InitFrom(d.InitEnv)
			workerEnv.SetSeqManager(d.SeqMgr)
			workerEnv.Retries = flagRetries
			workerEnv.Results = d.Results

			for ctx.Err() == nil {
				if err := workerEnv.RunIteration(ctx); err != nil {
					if ctx.Err() != nil {
						return
					}
					var failErr *env.ErrFail
					if errors.As(err, &failErr) {
						slog.Error("worker stopped", "worker", i, "error", err)
						return
					}
					if flagErrors {
						slog.Error("run error", "worker", i, "error", err)
					}
					continue
				}
			}
		})
	}
	return &wg
}

func startBackgroundWorkers(ctx context.Context, d workerDeps) *sync.WaitGroup {
	var wg sync.WaitGroup
	for _, w := range d.Req.Workers {
		slog.Info("background worker", "name", w.Name, "rate", w.Rate)

		wg.Go(func() {
			workerEnv, err := env.NewEnv(d.DB, flagDriver, d.Req, config.ConfigSectionWorker)
			if err != nil {
				slog.Error("worker env error", "worker", w.Name, "error", err)
				return
			}
			defer workerEnv.Close()

			workerEnv.InitFrom(d.InitEnv)
			workerEnv.SetSeqManager(d.SeqMgr)
			workerEnv.Results = d.Results
			workerEnv.RunWorker(ctx, w)
		})
	}
	return &wg
}

package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var (
		duration      time.Duration
		workers       int
		printInterval time.Duration
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the benchmark workload",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := connect()
			if err != nil {
				return err
			}
			defer db.Close()

			if cmd.Flags().Changed("duration") {
				req.Stages = nil
			}

			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer cancel()

			return run(ctx, cancel, db, req, duration, workers, printInterval)
		},
	}

	cmd.Flags().DurationVarP(&duration, "duration", "d", time.Minute, "benchmark duration")
	cmd.Flags().IntVarP(&workers, "workers", "w", 1, "number of concurrent workers")
	cmd.Flags().DurationVar(&printInterval, "print-interval", time.Second, "progress reporting interval")

	return cmd
}

func run(ctx context.Context, cancel context.CancelFunc, db *sql.DB, req *config.Request, duration time.Duration, workers int, printInterval time.Duration) error {
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
	initEnv, err := env.NewEnv(db, flagDriver, req)
	if err != nil {
		return nil, 0, err
	}
	defer initEnv.Close()
	if err := initEnv.Init(ctx); err != nil {
		return nil, 0, err
	}

	results := make(chan config.QueryResult, 1000)
	start := time.Now()

	go func() {
		defer close(results)

		for _, stage := range req.Stages {
			if ctx.Err() != nil {
				return
			}

			dur := time.Duration(stage.Duration)
			workers := stage.Workers
			if workers <= 0 {
				workers = 1
			}

			slog.Info("stage", "name", stage.Name, "workers", workers, "duration", dur)

			stageCtx, stageCancel := context.WithTimeout(ctx, dur)
			wg := startWorkers(stageCtx, workers, db, req, initEnv, results)
			wg.Wait()
			stageCancel()
		}
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
	initEnv, err := env.NewEnv(db, flagDriver, req)
	if err != nil {
		return nil, 0, err
	}
	defer initEnv.Close()
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

	wg := startWorkers(ctx, workers, db, req, initEnv, results)

	go func() {
		wg.Wait()
		close(results)
	}()

	slog.Info("running", "workers", workers, "duration", duration)
	stats := printResults(results, printInterval, start, workers, duration)

	return stats, time.Since(start), nil
}

func startWorkers(ctx context.Context, numWorkers int, db *sql.DB, req *config.Request, initEnv *env.Env, results chan<- config.QueryResult) *sync.WaitGroup {
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

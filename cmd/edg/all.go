package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/codingconcepts/edg/pkg/env"
	"github.com/spf13/cobra"
)

func allCmd() *cobra.Command {
	var (
		duration      time.Duration
		workers       int
		printInterval time.Duration
	)

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Run up, seed, run, deseed, and down in sequence",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := connect()
			if err != nil {
				return err
			}
			defer db.Close()

			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer cancel()

			env, err := env.NewEnv(db, req)
			if err != nil {
				return err
			}

			// Always run teardown, even if up/seed/run fails.
			defer func() {
				if len(req.Deseed) > 0 {
					_ = env.Deseed(ctx)
				}
				if len(req.Down) > 0 {
					_ = env.Down(ctx)
				}
			}()

			if len(req.Up) > 0 {
				if err := env.Up(ctx); err != nil {
					return err
				}
			}
			if len(req.Seed) > 0 {
				if err := env.Seed(ctx); err != nil {
					return err
				}
			}

			if cmd.Flags().Changed("duration") {
				req.Stages = nil
			}

			if len(req.Run) > 0 || len(req.Stages) > 0 {
				// Create a child context for run's duration timeout so the
				// parent context remains live for teardown.
				runCtx, runCancel := context.WithCancel(ctx)
				return run(runCtx, runCancel, db, req, duration, workers, printInterval)
			}

			return nil
		},
	}

	cmd.Flags().DurationVarP(&duration, "duration", "d", time.Minute, "benchmark duration")
	cmd.Flags().IntVarP(&workers, "workers", "w", 1, "number of concurrent workers")
	cmd.Flags().DurationVar(&printInterval, "print-interval", time.Second, "progress reporting interval")

	return cmd
}

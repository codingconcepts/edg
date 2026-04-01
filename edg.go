package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codingconcepts/edg/pkg"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/sijms/go-ora/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	flagURL    string
	configFile string
	flagDriver string
)

func main() {
	log.SetFlags(0)

	root := &cobra.Command{
		Use:   "edg",
		Short: "Expression-based Data Generator",
	}

	root.PersistentFlags().StringVar(&flagURL, "url", "", "database connection URL (env: URL)")
	root.PersistentFlags().StringVar(&configFile, "config", "", "workload YAML config file")
	root.PersistentFlags().StringVar(&flagDriver, "driver", "pgx", "database/sql driver name [pgx, oracle]")

	root.AddCommand(upCmd(), seedCmd(), deseedCmd(), downCmd(), runCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func openDB() (*sql.DB, *pkg.Request, error) {
	url := flagURL
	if url == "" {
		url = os.Getenv("URL")
	}
	if url == "" {
		return nil, nil, fmt.Errorf("--url flag or URL env var required")
	}

	raw, err := os.ReadFile(configFile)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", configFile, err)
	}

	var req pkg.Request
	if err := yaml.Unmarshal(raw, &req); err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", configFile, err)
	}

	db, err := sql.Open(flagDriver, url)
	if err != nil {
		return nil, nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("connecting to database: %w", err)
	}

	return db, &req, nil
}

func upCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Create schema and populate data",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			env, err := pkg.NewEnv(db, req)
			if err != nil {
				return err
			}

			return env.Up(cmd.Context())
		},
	}
}

func seedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Populate tables with data",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			env, err := pkg.NewEnv(db, req)
			if err != nil {
				return err
			}

			return env.Seed(cmd.Context())
		},
	}
}

func deseedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deseed",
		Short: "Delete seeded data from tables",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			env, err := pkg.NewEnv(db, req)
			if err != nil {
				return err
			}

			return env.Deseed(cmd.Context())
		},
	}
}

func downCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Tear down schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			env, err := pkg.NewEnv(db, req)
			if err != nil {
				return err
			}

			return env.Down(cmd.Context())
		},
	}
}

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
			db, req, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer cancel()

			// Run init queries once and share results across workers.
			initEnv, err := pkg.NewEnv(db, req)
			if err != nil {
				return err
			}
			if err := initEnv.Init(ctx); err != nil {
				return err
			}

			var (
				wg       sync.WaitGroup
				count    atomic.Int64
				errCount atomic.Int64
			)

			start := time.Now()

			// Timer cancels context after duration.
			go func() {
				select {
				case <-time.After(duration):
					cancel()
				case <-ctx.Done():
				}
			}()

			// Progress reporter.
			go func() {
				ticker := time.NewTicker(printInterval)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						c := count.Load()
						e := errCount.Load()
						elapsed := time.Since(start)
						tps := float64(c) / elapsed.Seconds()
						tpm := float64(c) / elapsed.Minutes()
						slog.Info("progress",
							"qps", fmt.Sprintf("%.1f", tps),
							"tpm", fmt.Sprintf("%.1f", tpm),
							"errors", e,
							"elapsed", elapsed.Round(time.Second))
					case <-ctx.Done():
						return
					}
				}
			}()

			for i := range workers {
				wg.Go(func() {
					workerEnv, err := pkg.NewEnv(db, req)
					if err != nil {
						slog.Error("env error", "worker", i, "error", err)
						return
					}
					workerEnv.InitFrom(initEnv)

					for ctx.Err() == nil {
						if err := workerEnv.RunOnce(ctx); err != nil {
							if ctx.Err() != nil {
								return
							}
							slog.Error("run error", "worker", i, "error", err)
							errCount.Add(1)
							continue
						}
						count.Add(1)
					}
				})
			}

			slog.Info("running", "workers", workers, "duration", duration)
			wg.Wait()

			elapsed := time.Since(start)
			total := count.Load()
			errors := errCount.Load()
			tpm := float64(total) / elapsed.Minutes()

			fmt.Println()
			fmt.Printf("Duration:     %s\n", elapsed.Round(time.Millisecond))
			fmt.Printf("Workers:      %d\n", workers)
			fmt.Printf("Transactions: %d\n", total)
			fmt.Printf("Errors:       %d\n", errors)
			fmt.Printf("tpm:          %.1f\n", tpm)

			return nil
		},
	}

	cmd.Flags().DurationVarP(&duration, "duration", "d", time.Minute, "benchmark duration")
	cmd.Flags().IntVarP(&workers, "workers", "w", 1, "number of concurrent workers")
	cmd.Flags().DurationVar(&printInterval, "print-interval", time.Second, "progress reporting interval")

	return cmd
}

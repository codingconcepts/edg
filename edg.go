package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/codingconcepts/edg/pkg"
	_ "github.com/go-sql-driver/mysql"
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
		Use:   "edg [expression]",
		Short: "Expression-based Data Generator",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			input := strings.Join(args, " ")

			var req pkg.Request
			env, err := pkg.NewEnv(nil, &req)
			if err != nil {
				return err
			}

			// Try the expression as-is first; if it fails, wrap it as
			// a gen() call so bare words like "email" just work.
			result, err := env.Eval(input)
			if err != nil {
				result, err = env.Eval(fmt.Sprintf("gen('%s')", input))
				if err != nil {
					return fmt.Errorf("invalid expression: %s", input)
				}
			}
			fmt.Println(result)
			return nil
		},
	}

	root.PersistentFlags().StringVar(&flagURL, "url", "", "database connection URL (env: URL)")
	root.PersistentFlags().StringVar(&configFile, "config", "", "workload YAML config file")
	root.PersistentFlags().StringVar(&flagDriver, "driver", "pgx", "database/sql driver name [pgx, oracle, mysql]")

	root.AddCommand(upCmd(), seedCmd(), deseedCmd(), downCmd(), runCmd(), allCmd(), replCmd(), validateCmd())
	root.SilenceUsage = true
	root.SilenceErrors = true

	if err := root.Execute(); err != nil {
		if ctx := root.Context(); ctx != nil && ctx.Err() != nil {
			fmt.Fprintln(os.Stderr, "cancelled")
		} else if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "cancelled")
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func connect() (*sql.DB, *pkg.Request, error) {
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
			db, req, err := connect()
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
			db, req, err := connect()
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
			db, req, err := connect()
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
			db, req, err := connect()
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

func run(ctx context.Context, cancel context.CancelFunc, db *sql.DB, req *pkg.Request, duration time.Duration, workers int, printInterval time.Duration) error {
	initEnv, err := pkg.NewEnv(db, req)
	if err != nil {
		return err
	}
	if err := initEnv.Init(ctx); err != nil {
		return err
	}

	results := make(chan pkg.QueryResult, workers*100)
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
	printResults(results, printInterval, start, workers)

	return nil
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
			db, req, err := connect()
			if err != nil {
				return err
			}
			defer db.Close()

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

func startWorkers(ctx context.Context, numWorkers int, db *sql.DB, req *pkg.Request, initEnv *pkg.Env, results chan<- pkg.QueryResult) *sync.WaitGroup {
	var wg sync.WaitGroup

	for i := range numWorkers {
		wg.Go(func() {
			workerEnv, err := pkg.NewEnv(db, req)
			if err != nil {
				slog.Error("env error", "worker", i, "error", err)
				return
			}
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

type queryStats struct {
	count        int64
	errors       int64
	totalLatency time.Duration
}

func printResults(results <-chan pkg.QueryResult, interval time.Duration, start time.Time, numWorkers int) {
	stats := map[string]*queryStats{}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case r, ok := <-results:
			if !ok {
				printSummary(stats, start, numWorkers)
				return
			}
			s := stats[r.Name]
			if s == nil {
				s = &queryStats{}
				stats[r.Name] = s
			}
			if r.Err != nil {
				s.errors++
			} else {
				s.count += int64(r.Count)
				s.totalLatency += r.Latency
			}
		case <-ticker.C:
			printProgress(stats, start)
		}
	}
}

func printProgress(stats map[string]*queryStats, start time.Time) {
	elapsed := time.Since(start)

	names := make([]string, 0, len(stats))
	for name := range stats {
		names = append(names, name)
	}
	slices.Sort(names)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "\n%s elapsed\n", elapsed.Round(time.Second))
	fmt.Fprintf(w, "QUERY\tCOUNT\tERRORS\tAVG LATENCY\tQPS\n")
	for _, name := range names {
		s := stats[name]
		var avg time.Duration
		if s.count > 0 {
			avg = s.totalLatency / time.Duration(s.count)
		}
		qps := float64(s.count) / elapsed.Seconds()
		fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%.1f\n", name, s.count, s.errors, avg.Round(time.Microsecond), qps)
	}
	w.Flush()
}

func printSummary(stats map[string]*queryStats, start time.Time, numWorkers int) {
	elapsed := time.Since(start)

	names := make([]string, 0, len(stats))
	for name := range stats {
		names = append(names, name)
	}
	slices.Sort(names)

	var totalCount, totalErrors int64
	for _, s := range stats {
		totalCount += s.count
		totalErrors += s.errors
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "\nsummary\n")
	fmt.Fprintf(w, "Duration:\t%s\n", elapsed.Round(time.Millisecond))
	fmt.Fprintf(w, "Workers:\t%d\n", numWorkers)
	fmt.Fprintf(w, "\nQUERY\tCOUNT\tERRORS\tAVG LATENCY\tQPS\n")
	for _, name := range names {
		s := stats[name]
		var avg time.Duration
		if s.count > 0 {
			avg = s.totalLatency / time.Duration(s.count)
		}
		qps := float64(s.count) / elapsed.Seconds()
		fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%.1f\n", name, s.count, s.errors, avg.Round(time.Microsecond), qps)
	}
	tpm := float64(totalCount) / elapsed.Minutes()
	fmt.Fprintf(w, "\nTransactions:\t%d\n", totalCount)
	fmt.Fprintf(w, "Errors:\t%d\n", totalErrors)
	fmt.Fprintf(w, "tpm:\t%.1f\n", tpm)
	w.Flush()
}

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

			env, err := pkg.NewEnv(db, req)
			if err != nil {
				return err
			}

			if err := env.Up(ctx); err != nil {
				return err
			}
			if err := env.Seed(ctx); err != nil {
				return err
			}

			// Create a child context for run's duration timeout so the
			// parent context remains live for teardown.
			runCtx, runCancel := context.WithCancel(ctx)
			if err := run(runCtx, runCancel, db, req, duration, workers, printInterval); err != nil {
				return err
			}

			if err := env.Deseed(ctx); err != nil {
				return err
			}
			return env.Down(ctx)
		},
	}

	cmd.Flags().DurationVarP(&duration, "duration", "d", time.Minute, "benchmark duration")
	cmd.Flags().IntVarP(&workers, "workers", "w", 1, "number of concurrent workers")
	cmd.Flags().DurationVar(&printInterval, "print-interval", time.Second, "progress reporting interval")

	return cmd
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate a config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configFile == "" {
				return fmt.Errorf("--config flag required")
			}

			raw, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("reading %s: %w", configFile, err)
			}

			var req pkg.Request
			if err := yaml.Unmarshal(raw, &req); err != nil {
				return fmt.Errorf("parsing %s: %w", configFile, err)
			}

			if _, err := pkg.NewEnv(nil, &req); err != nil {
				return err
			}

			fmt.Println("config is valid")
			return nil
		},
	}
}

func replCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repl",
		Short: "Interactive expression evaluator",
		RunE: func(cmd *cobra.Command, args []string) error {
			var req pkg.Request
			if configFile != "" {
				raw, err := os.ReadFile(configFile)
				if err != nil {
					return fmt.Errorf("reading %s: %w", configFile, err)
				}
				if err := yaml.Unmarshal(raw, &req); err != nil {
					return fmt.Errorf("parsing %s: %w", configFile, err)
				}
			}

			env, err := pkg.NewEnv(nil, &req)
			if err != nil {
				return err
			}

			fmt.Println("edg repl - type expressions to evaluate")

			scanner := bufio.NewScanner(os.Stdin)
			for {
				fmt.Print(">> ")
				if !scanner.Scan() {
					break
				}

				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}

				result, err := env.Eval(line)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					continue
				}
				fmt.Println(result)
			}

			fmt.Println()
			return scanner.Err()
		},
	}
}

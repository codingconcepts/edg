package workload

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/db"
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/codingconcepts/edg/pkg/seq"
	"github.com/spf13/cobra"
)

type RunParams struct {
	DB             db.DB
	Req            *config.Request
	Duration       time.Duration
	Workers        int
	PrintInterval  time.Duration
	WarmupDuration time.Duration
}

// RunFunc matches the signature of the top-level run function.
type RunFunc func(ctx context.Context, cancel context.CancelFunc, p RunParams) error

// Deps holds dependencies injected from the main package.
type Deps struct {
	Connect   func() (db.DB, *config.Request, error)
	ConnectDB func() (db.DB, error)
	Driver    func() string
	Run       RunFunc
}

// Cmd returns the parent workload command (no subcommands attached).
func Cmd() *cobra.Command {
	return &cobra.Command{
		Use:   "workload",
		Short: "Run a pre-baked workload",
	}
}

var driverFile = map[string]string{
	"pgx":       "crdb.yaml",
	"dsql":      "crdb.yaml",
	"mysql":     "mysql.yaml",
	"oracle":    "oracle.yaml",
	"mssql":     "mssql.yaml",
	"spanner":   "spanner.yaml",
	"mongodb":   "mongodb.yaml",
	"cassandra": "cassandra.yaml",
}

// LoadWorkload reads the correct embedded YAML for the given driver.
func LoadWorkload(fs embed.FS, name, driver string) (*config.Request, error) {
	file, ok := driverFile[driver]
	if !ok {
		return nil, fmt.Errorf("workload %q has no config for driver %q", name, driver)
	}

	data, err := fs.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("reading embedded workload: %w", err)
	}

	return config.ParseConfig(data)
}

// WithWorkload returns a copy of deps whose Connect loads config from
// the embedded FS instead of from a --config file.
func WithWorkload(deps Deps, fs embed.FS, name string) Deps {
	return Deps{
		Connect: func() (db.DB, *config.Request, error) {
			req, err := LoadWorkload(fs, name, deps.Driver())
			if err != nil {
				return nil, nil, err
			}
			db, err := deps.ConnectDB()
			if err != nil {
				return nil, nil, err
			}
			return db, req, nil
		},
		ConnectDB: deps.ConnectDB,
		Driver:    deps.Driver,
		Run:       deps.Run,
	}
}

// AddSubcommands adds up, seed, deseed, down, run, and all to cmd.
func AddSubcommands(cmd *cobra.Command, deps Deps) {
	cmd.AddCommand(
		UpCmd(deps),
		SeedCmd(deps),
		DeseedCmd(deps),
		DownCmd(deps),
		RunCmd(deps),
		AllCmd(deps),
	)
}

func UpCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Create schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := deps.Connect()
			if err != nil {
				return err
			}
			defer db.Close()

			e, err := env.NewEnv(db, deps.Driver(), req, config.ConfigSectionUp)
			if err != nil {
				return err
			}

			return e.Up(cmd.Context())
		},
	}
}

func SeedCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Populate tables with data",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := deps.Connect()
			if err != nil {
				return err
			}
			defer db.Close()

			e, err := env.NewEnv(db, deps.Driver(), req, config.ConfigSectionSeed)
			if err != nil {
				return err
			}
			e.SetSeqManager(seq.NewManager(req.Seq))

			return e.Seed(cmd.Context())
		},
	}
}

func DeseedCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "deseed",
		Short: "Delete seeded data from tables",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := deps.Connect()
			if err != nil {
				return err
			}
			defer db.Close()

			e, err := env.NewEnv(db, deps.Driver(), req, config.ConfigSectionDeseed)
			if err != nil {
				return err
			}

			return e.Deseed(cmd.Context())
		},
	}
}

func DownCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Tear down schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := deps.Connect()
			if err != nil {
				return err
			}
			defer db.Close()

			e, err := env.NewEnv(db, deps.Driver(), req, config.ConfigSectionDown)
			if err != nil {
				return err
			}

			return e.Down(cmd.Context())
		},
	}
}

func RunCmd(deps Deps) *cobra.Command {
	var (
		duration       time.Duration
		workers        int
		printInterval  time.Duration
		warmupDuration time.Duration
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the benchmark workload",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := deps.Connect()
			if err != nil {
				return err
			}
			defer db.Close()

			if cmd.Flags().Changed("duration") {
				req.Stages = nil
			}

			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer cancel()

			return deps.Run(ctx, cancel, RunParams{DB: db, Req: req, Duration: duration, Workers: workers, PrintInterval: printInterval, WarmupDuration: warmupDuration})
		},
	}

	cmd.Flags().DurationVarP(&duration, "duration", "d", time.Minute, "benchmark duration")
	cmd.Flags().IntVarP(&workers, "workers", "w", 1, "number of concurrent workers")
	cmd.Flags().DurationVar(&printInterval, "print-interval", time.Second, "progress reporting interval")
	cmd.Flags().DurationVar(&warmupDuration, "warmup-duration", 0, "warmup period before collecting metrics (e.g. 10s)")

	return cmd
}

func AllCmd(deps Deps) *cobra.Command {
	var (
		duration       time.Duration
		workers        int
		printInterval  time.Duration
		warmupDuration time.Duration
	)

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Run up, seed, run, deseed, and down in sequence",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, req, err := deps.Connect()
			if err != nil {
				return err
			}
			defer db.Close()

			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer cancel()

			e, err := env.NewEnv(db, deps.Driver(), req)
			if err != nil {
				return err
			}
			defer e.Close()
			e.SetSeqManager(seq.NewManager(req.Seq))

			defer func() {
				if len(req.Deseed) > 0 {
					_ = e.Deseed(ctx)
				}
				if len(req.Down) > 0 {
					_ = e.Down(ctx)
				}
			}()

			if len(req.Up) > 0 {
				if err := e.Up(ctx); err != nil {
					return err
				}
			}
			if len(req.Seed) > 0 {
				if err := e.Seed(ctx); err != nil {
					return err
				}
			}

			if cmd.Flags().Changed("duration") {
				req.Stages = nil
			}

			if len(req.Run) > 0 || len(req.Stages) > 0 {
				runCtx, runCancel := context.WithCancel(ctx)
				return deps.Run(runCtx, runCancel, RunParams{DB: db, Req: req, Duration: duration, Workers: workers, PrintInterval: printInterval, WarmupDuration: warmupDuration})
			}

			return nil
		},
	}

	cmd.Flags().DurationVarP(&duration, "duration", "d", time.Minute, "benchmark duration")
	cmd.Flags().IntVarP(&workers, "workers", "w", 1, "number of concurrent workers")
	cmd.Flags().DurationVar(&printInterval, "print-interval", time.Second, "progress reporting interval")
	cmd.Flags().DurationVar(&warmupDuration, "warmup-duration", 0, "warmup period before collecting metrics (e.g. 10s)")

	return cmd
}

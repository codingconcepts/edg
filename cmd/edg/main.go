package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/codingconcepts/edg/cmd/edg/workload"
	"github.com/codingconcepts/edg/cmd/edg/workload/bank"
	"github.com/codingconcepts/edg/cmd/edg/workload/ch_benchmark"
	"github.com/codingconcepts/edg/cmd/edg/workload/kv"
	"github.com/codingconcepts/edg/cmd/edg/workload/movr"
	"github.com/codingconcepts/edg/cmd/edg/workload/seats"
	"github.com/codingconcepts/edg/cmd/edg/workload/sysbench_insert"
	"github.com/codingconcepts/edg/cmd/edg/workload/sysbench_point_select"
	"github.com/codingconcepts/edg/cmd/edg/workload/sysbench_read_write"
	"github.com/codingconcepts/edg/cmd/edg/workload/sysbench_update_index"
	"github.com/codingconcepts/edg/cmd/edg/workload/tatp"
	"github.com/codingconcepts/edg/cmd/edg/workload/tpcc"
	"github.com/codingconcepts/edg/cmd/edg/workload/tpch"
	"github.com/codingconcepts/edg/cmd/edg/workload/ttlbench"
	"github.com/codingconcepts/edg/cmd/edg/workload/ttllogger"
	"github.com/codingconcepts/edg/cmd/edg/workload/ycsb"
	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/codingconcepts/edg/pkg/random"
	"github.com/spf13/cobra"
)

var (
	flagURL         string
	configFile      string
	flagDriver      string
	flagRngSeed     uint64
	flagLicense     string
	flagMetricsAddr string
	flagErrors      bool
	flagRetries     int
	flagPoolSize    int
	flagNoWait      bool

	version string
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

			var req config.Request
			env, err := env.NewEnv(nil, "", &req)
			if err != nil {
				return err
			}

			result, err := env.Eval(input)
			if err != nil {
				return fmt.Errorf("invalid expression: %s", input)
			}
			fmt.Println(result)
			return nil
		},
	}

	root.PersistentFlags().StringVar(&flagURL, "url", "", "database connection URL (env: EDG_URL)")
	root.PersistentFlags().StringVar(&configFile, "config", "", "workload YAML config file (env: EDG_CONFIG)")
	root.PersistentFlags().StringVar(&flagDriver, "driver", "pgx", "database/sql driver name [pgx, oracle, mysql, mssql, dsql, spanner, mongodb, cassandra] (env: EDG_DRIVER)")
	root.PersistentFlags().StringVar(&flagLicense, "license", "", "license key for enterprise drivers (env: EDG_LICENSE)")
	root.PersistentFlags().Uint64Var(&flagRngSeed, "rng-seed", 0, "PRNG seed for deterministic output (env: EDG_RNG_SEED)")
	root.PersistentFlags().StringVar(&flagMetricsAddr, "metrics-addr", "", "address for Prometheus metrics endpoint (e.g. :9090) (env: EDG_METRICS_ADDR)")
	root.PersistentFlags().BoolVar(&flagErrors, "errors", false, "print worker errors to stderr (env: EDG_ERRORS)")
	root.PersistentFlags().IntVar(&flagRetries, "retries", 0, "number of transaction retry attempts on error (env: EDG_RETRIES)")
	root.PersistentFlags().IntVar(&flagPoolSize, "pool-size", 0, "maximum number of open database connections (0 = driver default) (env: EDG_POOL_SIZE)")
	root.PersistentFlags().BoolVar(&flagNoWait, "no-wait", false, "ignore wait durations configured in workload queries (env: EDG_NO_WAIT)")

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		bindEnv(cmd, "url", "EDG_URL")
		bindEnv(cmd, "config", "EDG_CONFIG")
		bindEnv(cmd, "driver", "EDG_DRIVER")
		bindEnv(cmd, "license", "EDG_LICENSE")
		bindEnv(cmd, "rng-seed", "EDG_RNG_SEED")
		bindEnv(cmd, "metrics-addr", "EDG_METRICS_ADDR")
		bindEnv(cmd, "errors", "EDG_ERRORS")
		bindEnv(cmd, "retries", "EDG_RETRIES")
		bindEnv(cmd, "pool-size", "EDG_POOL_SIZE")
		bindEnv(cmd, "no-wait", "EDG_NO_WAIT")

		if cmd.Flags().Changed("rng-seed") {
			random.Seed(flagRngSeed)
		}
		return nil
	}

	deps := workload.Deps{
		Connect:   connect,
		ConnectDB: connectDB,
		Driver:    func() string { return flagDriver },
		Run:       run,
	}

	wCmd := workload.Cmd()
	wCmd.AddCommand(
		bank.Cmd(deps),
		ch_benchmark.Cmd(deps),
		kv.Cmd(deps),
		movr.Cmd(deps),
		seats.Cmd(deps),
		sysbench_insert.Cmd(deps),
		sysbench_point_select.Cmd(deps),
		sysbench_read_write.Cmd(deps),
		sysbench_update_index.Cmd(deps),
		tatp.Cmd(deps),
		tpcc.Cmd(deps),
		tpch.Cmd(deps),
		ttlbench.Cmd(deps),
		ttllogger.Cmd(deps),
		ycsb.Cmd(deps),
	)

	root.AddCommand(
		workload.UpCmd(deps),
		workload.SeedCmd(deps),
		workload.DeseedCmd(deps),
		workload.DownCmd(deps),
		workload.RunCmd(deps),
		workload.AllCmd(deps),
		stageCmd(),
		replCmd(),
		validateCmd(),
		versionCmd(),
		initCmd(),
		syncCmd(),
		wCmd,
	)
	root.SilenceUsage = true
	root.SilenceErrors = true

	if err := root.Execute(); err != nil {
		if ctx := root.Context(); ctx != nil && ctx.Err() != nil {
			slog.Info("cancelled")
		} else if errors.Is(err, context.Canceled) {
			slog.Info("cancelled")
		} else {
			slog.Error("fatal", "error", err)
		}
		os.Exit(1)
	}
}

func bindEnv(cmd *cobra.Command, flagName, envVar string) {
	if cmd.Flags().Changed(flagName) {
		return
	}
	if v, ok := os.LookupEnv(envVar); ok {
		cmd.Flags().Set(flagName, v)
	}
}

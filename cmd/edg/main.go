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
	"github.com/codingconcepts/edg/cmd/edg/workload/kv"
	"github.com/codingconcepts/edg/cmd/edg/workload/movr"
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
	flagURL     string
	configFile  string
	flagDriver  string
	flagRngSeed uint64
	flagLicense string

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

	root.PersistentFlags().StringVar(&flagURL, "url", "", "database connection URL (env: URL)")
	root.PersistentFlags().StringVar(&configFile, "config", "", "workload YAML config file")
	root.PersistentFlags().StringVar(&flagDriver, "driver", "pgx", "database/sql driver name [pgx, oracle, mysql, mssql, dsql]")
	root.PersistentFlags().StringVar(&flagLicense, "license", "", "license key for enterprise drivers (env: EDG_LICENSE)")
	root.PersistentFlags().Uint64Var(&flagRngSeed, "rng-seed", 0, "PRNG seed for deterministic output")

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
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
		kv.Cmd(deps),
		movr.Cmd(deps),
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
		replCmd(),
		validateCmd(),
		versionCmd(),
		initCmd(),
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

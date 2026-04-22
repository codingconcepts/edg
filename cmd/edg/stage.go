package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/codingconcepts/edg/pkg/output"
	"github.com/codingconcepts/edg/pkg/seq"
	"github.com/spf13/cobra"
)

func stageCmd() *cobra.Command {
	var (
		flagFormat    string
		flagOutputDir string
	)

	cmd := &cobra.Command{
		Use:   "stage",
		Short: "Generate data to files instead of a database",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := output.ParseFormat(flagFormat)
			if err != nil {
				return err
			}

			req, err := config.LoadConfig(configFile)
			if err != nil {
				return err
			}

			if f == output.FormatStdout {
				slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
			} else {
				if err := os.MkdirAll(flagOutputDir, 0755); err != nil {
					return fmt.Errorf("creating output directory: %w", err)
				}
			}

			w, err := output.New(f, flagDriver, flagOutputDir)
			if err != nil {
				return err
			}

			e, err := env.NewEnv(nil, flagDriver, req)
			if err != nil {
				return err
			}
			e.SetSeqManager(seq.NewManager(req.Seq))
			e.SetOutput(w)

			ctx := cmd.Context()

			if err := e.Up(ctx); err != nil {
				return err
			}
			if err := e.Seed(ctx); err != nil {
				return err
			}
			if err := e.Deseed(ctx); err != nil {
				return err
			}
			if err := e.Down(ctx); err != nil {
				return err
			}

			return w.Flush()
		},
	}

	cmd.Flags().StringVarP(&flagFormat, "format", "f", "sql", "output format (sql, json, csv, parquet, stdout)")
	cmd.Flags().StringVarP(&flagOutputDir, "output-dir", "o", ".", "directory for output files")

	return cmd
}

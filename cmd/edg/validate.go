package main

import (
	"fmt"
	"log/slog"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/spf13/cobra"
)

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate a config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configFile == "" {
				return fmt.Errorf("--config flag required")
			}

			req, err := config.LoadConfig(configFile)
			if err != nil {
				return err
			}

			if _, err := env.NewEnv(nil, req); err != nil {
				return err
			}

			slog.Info("config is valid")
			return nil
		},
	}
}

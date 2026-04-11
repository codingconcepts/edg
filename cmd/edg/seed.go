package main

import (
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/spf13/cobra"
)

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

			env, err := env.NewEnv(db, flagDriver, req)
			if err != nil {
				return err
			}

			return env.Seed(cmd.Context())
		},
	}
}

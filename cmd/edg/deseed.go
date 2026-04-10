package main

import (
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/spf13/cobra"
)

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

			env, err := env.NewEnv(db, req)
			if err != nil {
				return err
			}

			return env.Deseed(cmd.Context())
		},
	}
}

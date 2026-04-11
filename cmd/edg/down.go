package main

import (
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/spf13/cobra"
)

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

			env, err := env.NewEnv(db, flagDriver, req)
			if err != nil {
				return err
			}

			return env.Down(cmd.Context())
		},
	}
}

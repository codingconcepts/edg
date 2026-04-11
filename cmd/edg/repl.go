package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/spf13/cobra"
)

func replCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repl",
		Short: "Interactive expression evaluator",
		RunE: func(cmd *cobra.Command, args []string) error {
			var req config.Request
			if configFile != "" {
				r, err := config.LoadConfig(configFile)
				if err != nil {
					return err
				}
				req = *r
			}

			env, err := env.NewEnv(nil, "", &req)
			if err != nil {
				return err
			}

			slog.Info("edg repl - type expressions to evaluate")

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
					slog.Error("eval error", "error", err)
					continue
				}
				fmt.Println(result)
			}

			fmt.Println()
			return scanner.Err()
		},
	}
}

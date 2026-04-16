package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/codingconcepts/edg/pkg/license"
	"github.com/codingconcepts/edg/pkg/schema"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	var database, schemaName string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a starter config from an existing database schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := database
			if schemaName != "" {
				target = schemaName
			}
			if target == "" {
				return fmt.Errorf("--schema or --database flag required")
			}

			url := flagURL
			if url == "" {
				url = os.Getenv("URL")
			}
			if url == "" {
				return fmt.Errorf("--url flag or URL env var required")
			}

			db, err := openDB(cmd.Context(), flagDriver, url)
			if err != nil {
				return err
			}
			defer db.Close()

			tables, err := schema.Inspect(cmd.Context(), db, flagDriver, target)
			if err != nil {
				return fmt.Errorf("inspecting schema: %w", err)
			}

			if len(tables) == 0 {
				return fmt.Errorf("no tables found in %q", target)
			}

			sorted, err := schema.SortTables(tables)
			if err != nil {
				return err
			}

			fmt.Print(schema.Generate(sorted, flagDriver))
			return nil
		},
	}

	cmd.Flags().StringVar(&schemaName, "schema", "", "schema or database name to introspect")
	cmd.Flags().StringVar(&database, "database", "", "schema or database name to introspect (alias for --schema)")
	return cmd
}

func openDB(ctx context.Context, driver, url string) (*sql.DB, error) {
	if license.IsEnterprise(driver) {
		if err := checkLicense(driver); err != nil {
			return nil, err
		}
	}

	var db *sql.DB
	var err error
	switch driver {
	case "dsql":
		db, err = connectDSQL(ctx, url)
	case "mssql":
		db, err = sql.Open("sqlserver", url)
	default:
		db, err = sql.Open(driver, url)
	}
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return db, nil
}

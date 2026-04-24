package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/env"
	"github.com/codingconcepts/edg/pkg/random"
	"github.com/codingconcepts/edg/pkg/seq"
	"github.com/spf13/cobra"
)

var (
	syncSourceDriver string
	syncSourceURL    string
	syncSourceConfig string
	syncTargetDriver string
	syncTargetURL    string
	syncTargetConfig string
)

func syncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Write data to multiple databases and verify consistency",
	}

	cmd.PersistentFlags().StringVar(&syncSourceDriver, "source-driver", "pgx", "source database driver (env: EDG_SOURCE_DRIVER)")
	cmd.PersistentFlags().StringVar(&syncSourceURL, "source-url", "", "source database URL (env: EDG_SOURCE_URL)")
	cmd.PersistentFlags().StringVar(&syncSourceConfig, "source-config", "", "source edg config file (env: EDG_SOURCE_CONFIG)")
	cmd.PersistentFlags().StringVar(&syncTargetDriver, "target-driver", "pgx", "target database driver (env: EDG_TARGET_DRIVER)")
	cmd.PersistentFlags().StringVar(&syncTargetURL, "target-url", "", "target database URL (env: EDG_TARGET_URL)")
	cmd.PersistentFlags().StringVar(&syncTargetConfig, "target-config", "", "target edg config file (env: EDG_TARGET_CONFIG)")

	cmd.AddCommand(syncRunCmd(), syncVerifyCmd(), syncDownCmd())
	return cmd
}

func bindSyncEnv(cmd *cobra.Command) {
	bindEnv(cmd, "source-driver", "EDG_SOURCE_DRIVER")
	bindEnv(cmd, "source-url", "EDG_SOURCE_URL")
	bindEnv(cmd, "source-config", "EDG_SOURCE_CONFIG")
	bindEnv(cmd, "target-driver", "EDG_TARGET_DRIVER")
	bindEnv(cmd, "target-url", "EDG_TARGET_URL")
	bindEnv(cmd, "target-config", "EDG_TARGET_CONFIG")
}

func syncRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run up and seed on source (and optionally target)",
		RunE: func(cmd *cobra.Command, args []string) error {
			bindSyncEnv(cmd)

			if syncSourceURL == "" {
				return errors.New("--source-url required")
			}
			if syncSourceConfig == "" {
				return errors.New("--source-config required")
			}

			ctx := cmd.Context()

			slog.Info("running source", "driver", syncSourceDriver)
			if flagRngSeed != 0 {
				random.Seed(flagRngSeed)
			}
			if err := syncLifecycle(ctx, syncSourceDriver, syncSourceURL, syncSourceConfig, config.ConfigSectionUp, config.ConfigSectionSeed); err != nil {
				return fmt.Errorf("source: %w", err)
			}

			if syncTargetConfig != "" {
				if syncTargetURL == "" {
					return errors.New("--target-url required when --target-config is set")
				}
				slog.Info("running target", "driver", syncTargetDriver)
				if flagRngSeed != 0 {
					random.Seed(flagRngSeed)
				}
				if err := syncLifecycle(ctx, syncTargetDriver, syncTargetURL, syncTargetConfig, config.ConfigSectionUp, config.ConfigSectionSeed); err != nil {
					return fmt.Errorf("target: %w", err)
				}
			}

			slog.Info("sync run complete")
			return nil
		},
	}
}

func syncDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Run deseed and down on source (and optionally target)",
		RunE: func(cmd *cobra.Command, args []string) error {
			bindSyncEnv(cmd)

			if syncSourceURL == "" {
				return errors.New("--source-url required")
			}
			if syncSourceConfig == "" {
				return errors.New("--source-config required")
			}

			ctx := cmd.Context()

			slog.Info("tearing down source", "driver", syncSourceDriver)
			if err := syncLifecycle(ctx, syncSourceDriver, syncSourceURL, syncSourceConfig, config.ConfigSectionDeseed, config.ConfigSectionDown); err != nil {
				return fmt.Errorf("source: %w", err)
			}

			if syncTargetConfig != "" {
				if syncTargetURL == "" {
					return errors.New("--target-url required when --target-config is set")
				}
				slog.Info("tearing down target", "driver", syncTargetDriver)
				if err := syncLifecycle(ctx, syncTargetDriver, syncTargetURL, syncTargetConfig, config.ConfigSectionDeseed, config.ConfigSectionDown); err != nil {
					return fmt.Errorf("target: %w", err)
				}
			}

			slog.Info("sync down complete")
			return nil
		},
	}
}

func syncLifecycle(ctx context.Context, driver, url, configPath string, sections ...config.ConfigSection) error {
	req, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	db, err := connectDBWith(driver, url)
	if err != nil {
		return err
	}
	defer db.Close()

	e, err := env.NewEnv(db, driver, req, sections...)
	if err != nil {
		return err
	}
	defer e.Close()
	e.SetSeqManager(seq.NewManager(req.Seq))

	for _, section := range sections {
		switch section {
		case config.ConfigSectionUp:
			if err := e.Up(ctx); err != nil {
				return fmt.Errorf("up: %w", err)
			}
		case config.ConfigSectionSeed:
			if err := e.Seed(ctx); err != nil {
				return fmt.Errorf("seed: %w", err)
			}
		case config.ConfigSectionDeseed:
			if err := e.Deseed(ctx); err != nil {
				return fmt.Errorf("deseed: %w", err)
			}
		case config.ConfigSectionDown:
			if err := e.Down(ctx); err != nil {
				return fmt.Errorf("down: %w", err)
			}
		}
	}

	return nil
}

func syncVerifyCmd() *cobra.Command {
	var (
		tables        string
		orderBy       string
		ignoreColumns string
		wait          time.Duration
		batchSize     int
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify data consistency between source and target",
		RunE: func(cmd *cobra.Command, args []string) error {
			bindSyncEnv(cmd)

			if syncSourceURL == "" {
				return errors.New("--source-url required")
			}
			if syncTargetURL == "" {
				return errors.New("--target-url required")
			}
			if tables == "" {
				return errors.New("--tables required")
			}
			if orderBy == "" {
				return errors.New("--order-by required")
			}

			if wait > 0 {
				slog.Info("waiting for convergence", "duration", wait)
				time.Sleep(wait)
			}

			ctx := cmd.Context()

			sourceDB, err := connectDBWith(syncSourceDriver, syncSourceURL)
			if err != nil {
				return fmt.Errorf("source: %w", err)
			}
			defer sourceDB.Close()

			targetDB, err := connectDBWith(syncTargetDriver, syncTargetURL)
			if err != nil {
				return fmt.Errorf("target: %w", err)
			}
			defer targetDB.Close()

			tableList := strings.Split(tables, ",")
			ignoreSet := make(map[string]bool)
			if ignoreColumns != "" {
				for _, col := range strings.Split(ignoreColumns, ",") {
					ignoreSet[strings.TrimSpace(col)] = true
				}
			}

			var totalMismatches int
			for _, table := range tableList {
				table = strings.TrimSpace(table)
				n, err := verifyTable(ctx, sourceDB, syncSourceDriver, targetDB, syncTargetDriver, table, orderBy, ignoreSet, batchSize)
				if err != nil {
					return fmt.Errorf("table %s: %w", table, err)
				}
				totalMismatches += n
			}

			if totalMismatches > 0 {
				return fmt.Errorf("verification failed: %d mismatches", totalMismatches)
			}

			slog.Info("all tables verified")
			return nil
		},
	}

	cmd.Flags().StringVar(&tables, "tables", "", "comma-separated table names to verify")
	cmd.Flags().StringVar(&orderBy, "order-by", "", "column for deterministic row ordering")
	cmd.Flags().StringVar(&ignoreColumns, "ignore-columns", "", "comma-separated columns to skip")
	cmd.Flags().DurationVar(&wait, "wait", 0, "delay before verifying (for CDC lag)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 10000, "rows per verification batch")

	return cmd
}

func verifyTable(ctx context.Context, sourceDB *sql.DB, sourceDriver string, targetDB *sql.DB, targetDriver string, table, orderBy string, ignoreSet map[string]bool, batchSize int) (int, error) {
	src := &rowCursor{db: sourceDB, driver: sourceDriver, table: table, orderBy: orderBy, batchSize: batchSize}
	tgt := &rowCursor{db: targetDB, driver: targetDriver, table: table, orderBy: orderBy, batchSize: batchSize}

	var (
		mismatches int
		rowCount   int
	)

	srcRow, err := src.next(ctx)
	if err != nil {
		return 0, fmt.Errorf("source: %w", err)
	}
	tgtRow, err := tgt.next(ctx)
	if err != nil {
		return 0, fmt.Errorf("target: %w", err)
	}

	compareCols := buildCompareCols(src.columns, tgt.columns, ignoreSet)

	for srcRow != nil && tgtRow != nil {
		srcKey := src.keyValue(srcRow)
		tgtKey := tgt.keyValue(tgtRow)

		cmp := compareKeys(srcKey, tgtKey)
		switch {
		case cmp == 0:
			for _, col := range compareCols {
				sv := colValue(srcRow, src.colIdx, col)
				tv := colValue(tgtRow, tgt.colIdx, col)
				if sv.Valid != tv.Valid || sv.String != tv.String {
					fmt.Printf("MISMATCH table=%s %s=%s column=%s source=%q target=%q\n",
						table, orderBy, srcKey, col, fmtNull(sv), fmtNull(tv))
					mismatches++
				}
			}
			rowCount++

			srcRow, err = src.next(ctx)
			if err != nil {
				return 0, fmt.Errorf("source: %w", err)
			}
			tgtRow, err = tgt.next(ctx)
			if err != nil {
				return 0, fmt.Errorf("target: %w", err)
			}

		case cmp < 0:
			fmt.Printf("MISSING table=%s %s=%s side=target\n", table, orderBy, srcKey)
			mismatches++
			rowCount++
			srcRow, err = src.next(ctx)
			if err != nil {
				return 0, fmt.Errorf("source: %w", err)
			}

		case cmp > 0:
			fmt.Printf("EXTRA table=%s %s=%s side=target\n", table, orderBy, tgtKey)
			mismatches++
			tgtRow, err = tgt.next(ctx)
			if err != nil {
				return 0, fmt.Errorf("target: %w", err)
			}
		}
	}

	for srcRow != nil {
		fmt.Printf("MISSING table=%s %s=%s side=target\n", table, orderBy, src.keyValue(srcRow))
		mismatches++
		rowCount++
		srcRow, err = src.next(ctx)
		if err != nil {
			return 0, fmt.Errorf("source: %w", err)
		}
	}

	for tgtRow != nil {
		fmt.Printf("EXTRA table=%s %s=%s side=target\n", table, orderBy, tgt.keyValue(tgtRow))
		mismatches++
		tgtRow, err = tgt.next(ctx)
		if err != nil {
			return 0, fmt.Errorf("target: %w", err)
		}
	}

	if mismatches == 0 {
		slog.Info("table verified", "table", table, "rows", rowCount)
	} else {
		slog.Warn("table has mismatches", "table", table, "mismatches", mismatches)
	}

	return mismatches, nil
}

func buildCompareCols(srcCols, tgtCols []string, ignoreSet map[string]bool) []string {
	tgtSet := make(map[string]bool, len(tgtCols))
	for _, col := range tgtCols {
		tgtSet[col] = true
	}

	var cols []string
	for _, col := range srcCols {
		if ignoreSet[col] || !tgtSet[col] {
			continue
		}
		cols = append(cols, col)
	}
	return cols
}

func colValue(row []sql.NullString, colIdx map[string]int, col string) sql.NullString {
	return row[colIdx[col]]
}

func fmtNull(ns sql.NullString) string {
	if !ns.Valid {
		return "<NULL>"
	}
	return ns.String
}

func compareKeys(a, b string) int {
	af, aErr := strconv.ParseFloat(a, 64)
	bf, bErr := strconv.ParseFloat(b, 64)
	if aErr == nil && bErr == nil {
		switch {
		case af < bf:
			return -1
		case af > bf:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(a, b)
}

type rowCursor struct {
	db        *sql.DB
	driver    string
	table     string
	orderBy   string
	batchSize int

	columns   []string
	colIdx    map[string]int
	rows      [][]sql.NullString
	pos       int
	lastValue *string
	done      bool
}

func (c *rowCursor) next(ctx context.Context) ([]sql.NullString, error) {
	if c.done {
		return nil, nil
	}

	if c.rows == nil || c.pos >= len(c.rows) {
		if err := c.fetchBatch(ctx); err != nil {
			return nil, err
		}
		if len(c.rows) == 0 {
			c.done = true
			return nil, nil
		}
		c.pos = 0
	}

	row := c.rows[c.pos]
	c.pos++
	return row, nil
}

func (c *rowCursor) keyValue(row []sql.NullString) string {
	return row[c.colIdx[c.orderBy]].String
}

func (c *rowCursor) fetchBatch(ctx context.Context) error {
	query := paginatedQuery(c.driver, c.table, c.orderBy, c.batchSize, c.lastValue == nil)

	var (
		sqlRows *sql.Rows
		err     error
	)
	if c.lastValue == nil {
		sqlRows, err = c.db.QueryContext(ctx, query)
	} else {
		sqlRows, err = c.db.QueryContext(ctx, query, *c.lastValue)
	}
	if err != nil {
		return err
	}
	defer sqlRows.Close()

	if c.columns == nil {
		c.columns, err = sqlRows.Columns()
		if err != nil {
			return err
		}
		c.colIdx = make(map[string]int, len(c.columns))
		for i, col := range c.columns {
			c.colIdx[col] = i
		}
	}

	c.rows = c.rows[:0]
	for sqlRows.Next() {
		row := make([]sql.NullString, len(c.columns))
		ptrs := make([]any, len(c.columns))
		for i := range row {
			ptrs[i] = &row[i]
		}
		if err := sqlRows.Scan(ptrs...); err != nil {
			return err
		}
		c.rows = append(c.rows, row)
	}

	if len(c.rows) > 0 {
		last := c.rows[len(c.rows)-1][c.colIdx[c.orderBy]].String
		c.lastValue = &last
	}

	return sqlRows.Err()
}

func paginatedQuery(driver, table, orderBy string, batchSize int, first bool) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("SELECT * FROM %s", table))

	if !first {
		var placeholder string
		switch driver {
		case "mysql":
			placeholder = "?"
		case "oracle":
			placeholder = ":1"
		case "mssql":
			placeholder = "@p1"
		default:
			placeholder = "$1"
		}
		parts = append(parts, fmt.Sprintf("WHERE %s > %s", orderBy, placeholder))
	}

	parts = append(parts, fmt.Sprintf("ORDER BY %s", orderBy))

	switch driver {
	case "mssql":
		parts = append(parts, fmt.Sprintf("OFFSET 0 ROWS FETCH NEXT %d ROWS ONLY", batchSize))
	case "oracle":
		parts = append(parts, fmt.Sprintf("FETCH FIRST %d ROWS ONLY", batchSize))
	default:
		parts = append(parts, fmt.Sprintf("LIMIT %d", batchSize))
	}

	return strings.Join(parts, " ")
}

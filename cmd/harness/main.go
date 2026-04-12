package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
)

type dbConfig struct {
	driver   string
	url      string
	compose  string
	yamlFile string
	initCmds [][]string
	timeout  time.Duration
	cleanup  bool
}

var databases = map[string]dbConfig{
	"crdb": {
		driver:   "pgx",
		url:      "postgres://root@localhost:26257?sslmode=disable",
		compose:  "compose_crdb.yml",
		yamlFile: "crdb.yaml",
		timeout:  60 * time.Second,
		cleanup:  true,
	},
	"mysql": {
		driver:   "mysql",
		url:      "root:password@tcp(localhost:3306)/mysql?parseTime=true",
		compose:  "compose_mysql.yml",
		yamlFile: "mysql.yaml",
		timeout:  60 * time.Second,
		cleanup:  true,
		initCmds: [][]string{
			{"docker", "exec", "mysql", "mysql", "-uroot", "-ppassword", "-e", "SET GLOBAL cte_max_recursion_depth = 200000"},
		},
	},
	"oracle": {
		driver:   "oracle",
		url:      "oracle://system:password@localhost:1521/defaultdb",
		compose:  "compose_oracle.yml",
		yamlFile: "oracle.yaml",
		timeout:  10 * time.Minute,
		cleanup:  false,
	},
	"mssql": {
		driver:   "mssql",
		url:      "sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable",
		compose:  "compose_mssql.yml",
		yamlFile: "mssql.yaml",
		timeout:  60 * time.Second,
		cleanup:  true,
	},
}

func main() {
	log.SetFlags(0)

	dbType := flag.String("db", "", "database type: crdb, mysql, oracle, mssql")
	examplesDir := flag.String("examples", "_examples", "path to examples directory")
	startFrom := flag.String("start", "", "skip examples before this one (by directory name)")
	duration := flag.Duration("duration", 5*time.Second, "run duration per example")
	workers := flag.Int("workers", 1, "number of workers per example")
	flag.Parse()

	if *dbType == "" {
		fmt.Fprintln(os.Stderr, "usage: harness -db <crdb|mysql|oracle|mssql>")
		os.Exit(1)
	}

	cfg, ok := databases[*dbType]
	if !ok {
		fmt.Fprintf(os.Stderr, "unsupported database: %s\nvalid options: crdb, mysql, oracle, mssql\n", *dbType)
		os.Exit(1)
	}

	composeFile := filepath.Join(*examplesDir, cfg.compose)
	if err := composeUp(composeFile); err != nil {
		log.Fatalf("compose up: %v", err)
	}

	if cfg.cleanup {
		defer composeDown(composeFile)
	}

	for _, initCmd := range cfg.initCmds {
		slog.Info("running init command", "cmd", strings.Join(initCmd, " "))
		if err := runInitWithRetry(initCmd, 30); err != nil {
			slog.Warn("init command failed (may be already initialized)", "error", err)
		}
	}

	slog.Info("waiting for database to be ready")
	if err := waitForDB(cfg); err != nil {
		log.Fatalf("database not ready: %v", err)
	}

	examples, err := discoverExamples(*examplesDir, cfg.yamlFile)
	if err != nil {
		log.Fatalf("discovering examples: %v", err)
	}

	if *startFrom != "" {
		found := false
		for i, ex := range examples {
			if ex.name == *startFrom {
				examples = examples[i:]
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("example %q not found", *startFrom)
		}
	}

	slog.Info("discovered examples", "count", len(examples), "db", *dbType)

	for _, ex := range examples {
		slog.Info("running example", "name", ex.name)
		if err := runExample(cfg, ex.configPath, *duration, *workers); err != nil {
			log.Fatalf("example %s failed: %v", ex.name, err)
		}
		slog.Info("example passed", "name", ex.name)
	}
}

type example struct {
	name       string
	configPath string
}

func discoverExamples(examplesDir, yamlFile string) ([]example, error) {
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		return nil, err
	}

	var examples []example
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		configPath := filepath.Join(examplesDir, entry.Name(), yamlFile)
		if _, err := os.Stat(configPath); err != nil {
			continue
		}

		examples = append(examples, example{
			name:       entry.Name(),
			configPath: configPath,
		})
	}
	return examples, nil
}

func runExample(cfg dbConfig, configPath string, duration time.Duration, workers int) error {
	cmd := exec.Command("go", "run", "./cmd/edg", "all",
		"--driver", cfg.driver,
		"--url", cfg.url,
		"--config", configPath,
		"--duration", duration.String(),
		"--workers", fmt.Sprintf("%d", workers),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func composeUp(composeFile string) error {
	slog.Info("starting database", "compose", composeFile)
	cmd := exec.Command("docker", "compose", "-f", composeFile, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func composeDown(composeFile string) {
	slog.Info("stopping database", "compose", composeFile)
	cmd := exec.Command("docker", "compose", "-f", composeFile, "down", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		slog.Error("compose down failed", "error", err)
	}
}

func runInitWithRetry(args []string, maxRetries int) error {
	var lastErr error
	for range maxRetries {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err == nil {
			slog.Info("init command succeeded")
			return nil
		}
		lastErr = fmt.Errorf("%w: %s", err, string(out))
		time.Sleep(time.Second)
	}
	return lastErr
}

func waitForDB(cfg dbConfig) error {
	retries := int(cfg.timeout.Seconds())
	for range retries {
		db, err := sql.Open(cfg.driver, cfg.url)
		if err == nil {
			err = db.Ping()
			db.Close()
			if err == nil {
				slog.Info("database is ready")
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("database did not become ready within %s", cfg.timeout)
}

package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
)

//go:embed compose/*.yml
var composeFS embed.FS

var workloads = []string{
	"bank",
	"ch-benchmark",
	"kv",
	"movr",
	"seats",
	"sysbench-insert",
	"sysbench-point-select",
	"sysbench-read-write",
	"sysbench-update-index",
	"tatp",
	"tpcc",
	"tpch",
	"ttlbench",
	"ttllogger",
	"ycsb",
}

type dbConfig struct {
	driver    string
	url       string
	compose   string
	initCmds  [][]string
	cleanup   bool
	env       []string
	urlFunc   func(workload string) string
	checkFunc func() error
}

var databases = map[string]dbConfig{
	"crdb": {
		driver:    "pgx",
		url:       "postgres://root@localhost:26257?sslmode=disable",
		compose:   "compose_crdb.yml",
		cleanup:   true,
		checkFunc: checkSQL("pgx", "postgres://root@localhost:26257?sslmode=disable"),
	},
	"mysql": {
		driver:  "mysql",
		url:     "root:password@tcp(localhost:3306)/defaultdb?parseTime=true",
		compose: "compose_mysql.yml",
		cleanup: true,
		initCmds: [][]string{
			{"docker", "exec", "mysql", "mysql", "-uroot", "-ppassword", "-e", "SET GLOBAL cte_max_recursion_depth = 200000"},
		},
		checkFunc: checkSQL("mysql", "root:password@tcp(localhost:3306)/defaultdb?parseTime=true"),
	},
	"oracle": {
		driver:    "oracle",
		url:       "oracle://system:password@localhost:1521/defaultdb",
		compose:   "compose_oracle.yml",
		cleanup:   false,
		checkFunc: checkSQL("oracle", "oracle://system:password@localhost:1521/defaultdb"),
	},
	"mssql": {
		driver:  "mssql",
		url:     "sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable",
		compose: "compose_mssql.yml",
		cleanup: true,
		urlFunc: func(workload string) string {
			return fmt.Sprintf("sqlserver://sa:P4ssw0rd@localhost:1433?database=%s&encrypt=disable", workload)
		},
		checkFunc: checkSQL("mssql", "sqlserver://sa:P4ssw0rd@localhost:1433?database=master&encrypt=disable"),
	},
	"spanner": {
		driver:   "spanner",
		compose:  "compose_spanner.yml",
		cleanup:  true,
		env:      []string{"SPANNER_EMULATOR_HOST=localhost:9010"},
		initCmds: spannerInitCmds(),
		urlFunc: func(workload string) string {
			return fmt.Sprintf("projects/test-project/instances/test-instance/databases/%s", workload)
		},
		checkFunc: waitForTCP("localhost:9020"),
	},
	"cassandra": {
		driver:    "cassandra",
		url:       "cassandra://localhost:9042",
		compose:   "compose_cassandra.yml",
		cleanup:   false,
		checkFunc: checkExec("docker", "exec", "cassandra", "cqlsh", "-e", "DESCRIBE KEYSPACES"),
	},
	"mongodb": {
		driver:  "mongodb",
		url:     "mongodb://localhost:27017/test?directConnection=true",
		compose: "compose_mongo.yml",
		cleanup: false,
		initCmds: [][]string{
			{"docker", "exec", "mongo", "mongosh", "--eval", "rs.initiate()"},
		},
		checkFunc: checkExec("docker", "exec", "mongo", "mongosh", "--eval", "db.runCommand({ping:1})"),
	},
}

func spannerInitCmds() [][]string {
	cmds := [][]string{
		{"curl", "-s", "-X", "POST", "http://localhost:9020/v1/projects/test-project/instances",
			"--json", `{"instanceId":"test-instance","instance":{"config":"projects/test-project/instanceConfigs/emulator-config","displayName":"Test","nodeCount":1}}`},
	}
	for _, w := range workloads {
		cmds = append(cmds, []string{
			"curl", "-s", "-X", "POST",
			"http://localhost:9020/v1/projects/test-project/instances/test-instance/databases",
			"--json", fmt.Sprintf(`{"createStatement":"CREATE DATABASE %s"}`, "`"+w+"`"),
		})
	}
	return cmds
}

func main() {
	log.SetFlags(0)

	dbType := flag.String("db", "", "database type: crdb, mysql, oracle, mssql, spanner, cassandra, mongodb")
	startFrom := flag.String("start", "", "skip workloads before this one (by name)")
	duration := flag.Duration("duration", 5*time.Second, "run duration per workload")
	workers := flag.Int("workers", 1, "number of workers per workload")
	flag.Parse()

	if *dbType == "" {
		fmt.Fprintln(os.Stderr, "usage: harness -db <crdb|mysql|oracle|mssql|spanner|cassandra|mongodb>")
		os.Exit(1)
	}

	cfg, ok := databases[*dbType]
	if !ok {
		fmt.Fprintf(os.Stderr, "unsupported database: %s\nvalid options: crdb, mysql, oracle, mssql, spanner, cassandra, mongodb\n", *dbType)
		os.Exit(1)
	}

	composeFile, err := writeComposeFile(cfg.compose)
	if err != nil {
		log.Fatalf("writing compose file: %v", err)
	}
	defer os.Remove(composeFile)

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
	waitForDB(cfg)

	wl := workloads
	if *startFrom != "" {
		found := false
		for i, name := range wl {
			if name == *startFrom {
				wl = wl[i:]
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("workload %q not found", *startFrom)
		}
	}

	slog.Info("running workloads", "count", len(wl), "db", *dbType)

	for _, name := range wl {
		slog.Info("running workload", "name", name)

		for attempt := 1; ; attempt++ {
			if err := cfg.checkFunc(); err != nil {
				slog.Warn("database not reachable, retrying in 5s", "name", name, "attempt", attempt, "error", err)
				time.Sleep(5 * time.Second)
				continue
			}
			break
		}

		if err := runWorkload(cfg, name, *duration, *workers); err != nil {
			log.Fatalf("workload %s failed: %v", name, err)
		}
		slog.Info("workload passed", "name", name)
	}
}

func runWorkload(cfg dbConfig, name string, duration time.Duration, workers int) error {
	url := cfg.url
	if cfg.urlFunc != nil {
		url = cfg.urlFunc(name)
	}

	cmd := exec.Command("go", "run", "./cmd/edg", "workload", name, "all",
		"--driver", cfg.driver,
		"--url", url,
		"--duration", duration.String(),
		"--workers", fmt.Sprintf("%d", workers),
		"--no-wait",
		"--errors",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if len(cfg.env) > 0 {
		cmd.Env = append(os.Environ(), cfg.env...)
	}
	return cmd.Run()
}

func writeComposeFile(name string) (string, error) {
	data, err := composeFS.ReadFile("compose/" + name)
	if err != nil {
		return "", fmt.Errorf("reading embedded %s: %w", name, err)
	}
	f, err := os.CreateTemp("", "edg-compose-*.yml")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()
	return f.Name(), nil
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

func waitForDB(cfg dbConfig) {
	for {
		if err := cfg.checkFunc(); err != nil {
			slog.Warn("database not ready, retrying in 5s", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}
		slog.Info("database is ready")
		return
	}
}

func checkSQL(driver, url string) func() error {
	return func() error {
		db, err := sql.Open(driver, url)
		if err != nil {
			return err
		}
		defer db.Close()
		return db.Ping()
	}
}

func checkExec(args ...string) func() error {
	return func() error {
		return exec.Command(args[0], args[1:]...).Run()
	}
}

func waitForTCP(addr string) func() error {
	return func() error {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}
}

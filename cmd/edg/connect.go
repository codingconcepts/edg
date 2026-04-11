package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/codingconcepts/edg/pkg/config"
	dsqlconn "github.com/awslabs/aurora-dsql-connectors/go/pgx/dsql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
)

func connect() (*sql.DB, *config.Request, error) {
	url := flagURL
	if url == "" {
		url = os.Getenv("URL")
	}
	if url == "" {
		return nil, nil, fmt.Errorf("--url flag or URL env var required")
	}

	req, err := config.LoadConfig(configFile)
	if err != nil {
		return nil, nil, err
	}

	var db *sql.DB
	switch flagDriver {
	case "dsql":
		db, err = connectDSQL(context.Background(), url)
	case "mssql":
		db, err = sql.Open("sqlserver", url)
	default:
		db, err = sql.Open(flagDriver, url)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("connecting to database: %w", err)
	}

	return db, req, nil
}

func connectDSQL(ctx context.Context, rawURL string) (*sql.DB, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "dsql://" + rawURL
	}
	pool, err := dsqlconn.NewPool(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("creating DSQL pool: %w", err)
	}
	return stdlib.OpenDBFromPool(pool), nil
}

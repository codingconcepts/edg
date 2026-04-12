package main

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	dsqlconn "github.com/awslabs/aurora-dsql-connectors/go/pgx/dsql"
	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/license"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
)

//go:embed public.key
var publicKeyB64 string

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

	if license.IsEnterprise(flagDriver) {
		if err := checkLicense(flagDriver); err != nil {
			return nil, nil, err
		}
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

func checkLicense(driver string) error {
	licStr := flagLicense
	if licStr == "" {
		licStr = os.Getenv("EDG_LICENSE")
	}
	if licStr == "" {
		return fmt.Errorf("driver %q requires a license (set --license flag or EDG_LICENSE env var)", driver)
	}

	pubBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKeyB64))
	if err != nil {
		return fmt.Errorf("decoding public key: %w", err)
	}
	publicKey := ed25519.PublicKey(pubBytes)

	lic, err := license.Verify(publicKey, licStr)
	if err != nil {
		return fmt.Errorf("invalid license: %w", err)
	}

	if lic.IsExpired() {
		return fmt.Errorf("license expired on %s", lic.ExpiresAt.Format("2006-01-02"))
	}

	if !lic.HasDriver(driver) {
		return fmt.Errorf("license does not include driver %q (licensed: %v)", driver, lic.Drivers)
	}

	return nil
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

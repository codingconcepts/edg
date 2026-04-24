package main

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	awsdsql "github.com/awslabs/aurora-dsql-connectors/go/pgx/dsql"
	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/db"
	"github.com/codingconcepts/edg/pkg/license"
	"github.com/gocql/gocql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/googleapis/go-sql-spanner"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

//go:embed public.key
var publicKeyB64 string

func connect() (db.DB, *config.Request, error) {
	req, err := config.LoadConfig(configFile)
	if err != nil {
		return nil, nil, err
	}

	d, err := connectDB()
	if err != nil {
		return nil, nil, err
	}

	return d, req, nil
}

func connectDB() (db.DB, error) {
	connURL := flagURL
	if connURL == "" {
		connURL = os.Getenv("URL")
	}
	if connURL == "" {
		return nil, errors.New("--url flag or URL env var required")
	}

	return connectDBWith(flagDriver, connURL)
}

func connectDBWith(driver, connURL string) (db.DB, error) {
	switch driver {
	case "mongodb":
		return connectMongoDB(connURL)
	case "cassandra":
		return connectCassandra(connURL)
	default:
		sqlDB, err := connectSQLDB(driver, connURL)
		if err != nil {
			return nil, err
		}
		return db.NewSQDB(sqlDB), nil
	}
}

func connectSQLDB(driver, connURL string) (*sql.DB, error) {
	if license.IsEnterprise(driver) {
		if err := checkLicense(driver); err != nil {
			return nil, err
		}
	}

	var (
		d   *sql.DB
		err error
	)
	switch driver {
	case "dsql":
		d, err = connectDSQL(context.Background(), connURL)
	case "mssql":
		d, err = sql.Open("sqlserver", connURL)
	default:
		d, err = sql.Open(driver, connURL)
	}
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if flagPoolSize > 0 {
		d.SetMaxOpenConns(flagPoolSize)
		d.SetMaxIdleConns(flagPoolSize)
	}

	if err := d.Ping(); err != nil {
		d.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return d, nil
}

func connectMongoDB(connURL string) (db.DB, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(connURL))
	if err != nil {
		return nil, fmt.Errorf("connecting to MongoDB: %w", err)
	}

	if err := client.Ping(context.Background(), nil); err != nil {
		client.Disconnect(context.Background())
		return nil, fmt.Errorf("pinging MongoDB: %w", err)
	}

	parsed, err := url.Parse(connURL)
	if err != nil {
		client.Disconnect(context.Background())
		return nil, fmt.Errorf("parsing MongoDB URL: %w", err)
	}

	dbName := strings.TrimPrefix(parsed.Path, "/")
	if dbName == "" {
		dbName = "test"
	}

	return db.NewMongoDB(client.Database(dbName)), nil
}

func connectCassandra(connURL string) (db.DB, error) {
	parsed, err := url.Parse(connURL)
	if err != nil {
		return nil, fmt.Errorf("parsing Cassandra URL: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		host = connURL
	}

	cluster := gocql.NewCluster(host)
	if port := parsed.Port(); port != "" {
		p, err := fmt.Sscanf(port, "%d", &cluster.Port)
		if err != nil || p != 1 {
			return nil, fmt.Errorf("parsing Cassandra port: %s", port)
		}
	}

	dbName := strings.TrimPrefix(parsed.Path, "/")
	if dbName != "" {
		cluster.Keyspace = dbName
	}

	if flagPoolSize > 0 {
		cluster.NumConns = flagPoolSize
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("connecting to Cassandra: %w", err)
	}

	return db.NewCassandraDB(session), nil
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
	pool, err := awsdsql.NewPool(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("creating DSQL pool: %w", err)
	}
	return stdlib.OpenDBFromPool(pool), nil
}

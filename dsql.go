package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	dsqlconn "github.com/awslabs/aurora-dsql-connectors/go/pgx/dsql"
	"github.com/jackc/pgx/v5/stdlib"
)

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

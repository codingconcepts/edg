package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/gocql/gocql"
)

type CassandraDB struct {
	session *gocql.Session
}

func NewCassandraDB(session *gocql.Session) *CassandraDB {
	return &CassandraDB{session: session}
}

func (c *CassandraDB) QueryContext(ctx context.Context, query string, args ...any) (RowIterator, error) {
	iter := c.session.Query(query, args...).WithContext(ctx).Iter()
	return newCassandraRowIterator(iter)
}

func (c *CassandraDB) ExecContext(ctx context.Context, query string, args ...any) error {
	if strings.Contains(query, batchSep) {
		return c.execBatch(ctx, query)
	}
	return c.session.Query(query, args...).WithContext(ctx).Exec()
}

func (c *CassandraDB) execBatch(ctx context.Context, query string) error {
	queries := ExpandBatchQuery(query)
	batch := c.session.NewBatch(gocql.UnloggedBatch).WithContext(ctx)
	for _, q := range queries {
		batch.Query(q)
	}
	return c.session.ExecuteBatch(batch)
}

func (c *CassandraDB) Close() error {
	c.session.Close()
	return nil
}

type cassandraRowIterator struct {
	iter    *gocql.Iter
	columns []string
	current map[string]any
}

func newCassandraRowIterator(iter *gocql.Iter) (*cassandraRowIterator, error) {
	cols := iter.Columns()
	columns := make([]string, len(cols))
	for i, col := range cols {
		columns[i] = col.Name
	}
	return &cassandraRowIterator{
		iter:    iter,
		columns: columns,
	}, nil
}

func (c *cassandraRowIterator) Columns() ([]string, error) {
	return c.columns, nil
}

func (c *cassandraRowIterator) ColumnTypes() ([]ColumnType, error) {
	types := make([]ColumnType, len(c.columns))
	cols := c.iter.Columns()
	for i, col := range cols {
		types[i] = &GenericColumnType{Name: col.TypeInfo.Type().String()}
	}
	return types, nil
}

func (c *cassandraRowIterator) Next() bool {
	c.current = make(map[string]any)
	return c.iter.MapScan(c.current)
}

func (c *cassandraRowIterator) Scan(dest ...any) error {
	for i, col := range c.columns {
		if i >= len(dest) {
			break
		}
		ptr, ok := dest[i].(*any)
		if !ok {
			return fmt.Errorf("scan: dest[%d] must be *any", i)
		}
		v := c.current[col]
		if u, ok := v.(gocql.UUID); ok {
			v = u.String()
		}
		*ptr = v
	}
	return nil
}

func (c *cassandraRowIterator) Close() error { return c.iter.Close() }
func (c *cassandraRowIterator) Err() error   { return c.iter.Close() }

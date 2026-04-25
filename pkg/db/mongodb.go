package db

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoDB struct {
	database *mongo.Database
}

func NewMongoDB(database *mongo.Database) *MongoDB {
	return &MongoDB{database: database}
}

func (m *MongoDB) QueryContext(ctx context.Context, query string, args ...any) (RowIterator, error) {
	var cmd bson.D
	if err := bson.UnmarshalExtJSON([]byte(query), false, &cmd); err != nil {
		return nil, fmt.Errorf("parsing MongoDB command: %w", err)
	}

	result := m.database.RunCommand(ctx, cmd)
	if err := result.Err(); err != nil {
		return nil, err
	}

	var raw bson.M
	if err := result.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding MongoDB result: %w", err)
	}

	return newMongoRowIterator(raw), nil
}

func (m *MongoDB) ExecContext(ctx context.Context, query string, args ...any) error {
	if strings.Contains(query, batchSep) {
		return m.execBatch(ctx, query)
	}

	var cmd bson.D
	if err := bson.UnmarshalExtJSON([]byte(query), false, &cmd); err != nil {
		return fmt.Errorf("parsing MongoDB command: %w", err)
	}

	return m.database.RunCommand(ctx, cmd).Err()
}

func (m *MongoDB) execBatch(ctx context.Context, query string) error {
	for _, q := range ExpandBatchQuery(query) {
		var cmd bson.D
		if err := bson.UnmarshalExtJSON([]byte(q), false, &cmd); err != nil {
			return fmt.Errorf("parsing MongoDB command: %w", err)
		}
		if err := m.database.RunCommand(ctx, cmd).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (m *MongoDB) BeginTx(ctx context.Context) (Transaction, error) {
	session, err := m.database.Client().StartSession()
	if err != nil {
		return nil, fmt.Errorf("starting MongoDB session: %w", err)
	}
	if err := session.StartTransaction(options.Transaction()); err != nil {
		session.EndSession(ctx)
		return nil, fmt.Errorf("starting MongoDB transaction: %w", err)
	}
	return &mongoTransaction{database: m.database, session: session}, nil
}

func (m *MongoDB) Close() error {
	return m.database.Client().Disconnect(context.Background())
}

type mongoTransaction struct {
	database *mongo.Database
	session  *mongo.Session
}

func (t *mongoTransaction) QueryContext(ctx context.Context, query string, args ...any) (RowIterator, error) {
	ctx = mongo.NewSessionContext(ctx, t.session)
	var cmd bson.D
	if err := bson.UnmarshalExtJSON([]byte(query), false, &cmd); err != nil {
		return nil, fmt.Errorf("parsing MongoDB command: %w", err)
	}
	result := t.database.RunCommand(ctx, cmd)
	if err := result.Err(); err != nil {
		return nil, err
	}
	var raw bson.M
	if err := result.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding MongoDB result: %w", err)
	}
	return newMongoRowIterator(raw), nil
}

func (t *mongoTransaction) ExecContext(ctx context.Context, query string, args ...any) error {
	ctx = mongo.NewSessionContext(ctx, t.session)
	if strings.Contains(query, batchSep) {
		for _, q := range ExpandBatchQuery(query) {
			var cmd bson.D
			if err := bson.UnmarshalExtJSON([]byte(q), false, &cmd); err != nil {
				return fmt.Errorf("parsing MongoDB command: %w", err)
			}
			if err := t.database.RunCommand(ctx, cmd).Err(); err != nil {
				return err
			}
		}
		return nil
	}
	var cmd bson.D
	if err := bson.UnmarshalExtJSON([]byte(query), false, &cmd); err != nil {
		return fmt.Errorf("parsing MongoDB command: %w", err)
	}
	return t.database.RunCommand(ctx, cmd).Err()
}

func (t *mongoTransaction) Commit() error {
	defer t.session.EndSession(context.Background())
	return t.session.CommitTransaction(context.Background())
}

func (t *mongoTransaction) Rollback() error {
	defer t.session.EndSession(context.Background())
	return t.session.AbortTransaction(context.Background())
}

type mongoRowIterator struct {
	columns []string
	docs    []bson.M
	pos     int
}

func newMongoRowIterator(raw bson.M) *mongoRowIterator {
	var docs []bson.M

	if cursor, ok := raw["cursor"]; ok {
		if fb := bsonLookup(cursor, "firstBatch"); fb != nil {
			if arr, ok := fb.(bson.A); ok {
				for _, item := range arr {
					docs = append(docs, bsonToM(item))
				}
			}
		}
	}

	if len(docs) == 0 {
		if v, ok := raw["value"]; ok {
			if vm, ok := v.(bson.M); ok {
				docs = []bson.M{vm}
			}
		}
	}

	if len(docs) == 0 {
		docs = []bson.M{raw}
	}

	var columns []string
	if len(docs) > 0 {
		for k := range docs[0] {
			columns = append(columns, k)
		}
		sort.Strings(columns)
	}

	return &mongoRowIterator{columns: columns, docs: docs, pos: -1}
}

func (m *mongoRowIterator) Columns() ([]string, error) {
	return m.columns, nil
}

func (m *mongoRowIterator) ColumnTypes() ([]ColumnType, error) {
	types := make([]ColumnType, len(m.columns))
	for i := range types {
		types[i] = &GenericColumnType{}
	}
	return types, nil
}

func (m *mongoRowIterator) Next() bool {
	m.pos++
	return m.pos < len(m.docs)
}

func (m *mongoRowIterator) Scan(dest ...any) error {
	if m.pos >= len(m.docs) {
		return fmt.Errorf("scan: no current row")
	}
	doc := m.docs[m.pos]
	for i, col := range m.columns {
		if i >= len(dest) {
			break
		}
		ptr, ok := dest[i].(*any)
		if !ok {
			return fmt.Errorf("scan: dest[%d] must be *any", i)
		}
		*ptr = doc[col]
	}
	return nil
}

func (m *mongoRowIterator) Close() error { return nil }
func (m *mongoRowIterator) Err() error   { return nil }

func bsonLookup(v any, key string) any {
	switch doc := v.(type) {
	case bson.M:
		return doc[key]
	case bson.D:
		for _, e := range doc {
			if e.Key == key {
				return e.Value
			}
		}
	}
	return nil
}

func bsonToM(v any) bson.M {
	switch doc := v.(type) {
	case bson.M:
		return doc
	case bson.D:
		m := make(bson.M, len(doc))
		for _, e := range doc {
			m[e.Key] = e.Value
		}
		return m
	}
	return bson.M{}
}

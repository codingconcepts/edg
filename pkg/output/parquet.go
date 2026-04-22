package output

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/parquet-go/parquet-go"
)

type parquetWriter struct {
	dir   string
	mu    sync.Mutex
	files map[string]*parquetFile
}

type parquetFile struct {
	columns []string
	rows    [][]any
}

func newParquetWriter(dir string) *parquetWriter {
	return &parquetWriter{
		dir:   dir,
		files: make(map[string]*parquetFile),
	}
}

func (w *parquetWriter) Add(section, queryName, querySQL string, columns []string, args []any) error {
	if len(args) == 0 {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	key := section + "_" + queryName
	f, ok := w.files[key]
	if !ok {
		f = &parquetFile{columns: columns}
		w.files[key] = f
	}

	row := make([]any, len(columns))
	for i := range columns {
		if i < len(args) {
			row[i] = args[i]
		}
	}
	f.rows = append(f.rows, row)
	return nil
}

func (w *parquetWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for name, f := range w.files {
		if err := w.writeFile(name, f); err != nil {
			return err
		}
	}
	return nil
}

func (w *parquetWriter) writeFile(name string, f *parquetFile) error {
	filename := filepath.Join(w.dir, name+".parquet")

	group := make(parquet.Group)
	for _, col := range f.columns {
		group[col] = parquet.Optional(parquet.Leaf(parquet.ByteArrayType))
	}
	schema := parquet.NewSchema("output", group)

	// Map column names to their index in the schema so rows align
	// regardless of map iteration order in parquet.Group.
	schemaFields := schema.Fields()
	colIndex := make(map[string]int, len(schemaFields))
	for i, field := range schemaFields {
		colIndex[field.Name()] = i
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating %s: %w", filename, err)
	}
	defer file.Close()

	pw := parquet.NewWriter(file, schema)

	for _, row := range f.rows {
		pRow := make(parquet.Row, len(f.columns))
		for i, col := range f.columns {
			idx := colIndex[col]
			val := row[i]
			if val == nil {
				pRow[idx] = parquet.Value{}.Level(0, 0, idx)
			} else {
				s := fmt.Sprint(val)
				pRow[idx] = parquet.ByteArrayValue([]byte(s)).Level(0, 1, idx)
			}
		}
		if _, err := pw.WriteRows([]parquet.Row{pRow}); err != nil {
			return fmt.Errorf("writing row to %s: %w", filename, err)
		}
	}

	if err := pw.Close(); err != nil {
		return fmt.Errorf("closing parquet writer for %s: %w", filename, err)
	}

	slog.Info("wrote output", "file", filename, "rows", len(f.rows))
	return nil
}

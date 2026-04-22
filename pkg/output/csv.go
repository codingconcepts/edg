package output

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type csvWriter struct {
	dir   string
	mu    sync.Mutex
	files map[string]*csvFile
}

type csvFile struct {
	columns []string
	rows    [][]string
}

func newCSVWriter(dir string) *csvWriter {
	return &csvWriter{
		dir:   dir,
		files: make(map[string]*csvFile),
	}
}

func (w *csvWriter) Add(section, queryName, querySQL string, columns []string, args []any) error {
	if len(args) == 0 {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	key := section + "_" + queryName
	f, ok := w.files[key]
	if !ok {
		f = &csvFile{columns: columns}
		w.files[key] = f
	}

	row := make([]string, len(columns))
	for i := range columns {
		if i < len(args) {
			row[i] = formatCSVValue(args[i])
		}
	}
	f.rows = append(f.rows, row)
	return nil
}

func formatCSVValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case []byte:
		return fmt.Sprintf("%x", val)
	default:
		return fmt.Sprint(v)
	}
}

func (w *csvWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for name, f := range w.files {
		filename := filepath.Join(w.dir, name+".csv")
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("creating %s: %w", filename, err)
		}

		writer := csv.NewWriter(file)
		if err := writer.Write(f.columns); err != nil {
			file.Close()
			return fmt.Errorf("writing headers to %s: %w", filename, err)
		}
		for _, row := range f.rows {
			if err := writer.Write(row); err != nil {
				file.Close()
				return fmt.Errorf("writing row to %s: %w", filename, err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			file.Close()
			return fmt.Errorf("flushing %s: %w", filename, err)
		}
		file.Close()

		slog.Info("wrote output", "file", filename, "rows", len(f.rows))
	}
	return nil
}

package output

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type jsonWriter struct {
	dir  string
	mu   sync.Mutex
	data map[string]map[string][]map[string]any
}

func newJSONWriter(dir string) *jsonWriter {
	return &jsonWriter{
		dir:  dir,
		data: make(map[string]map[string][]map[string]any),
	}
}

func (w *jsonWriter) Add(section, queryName, querySQL string, columns []string, args []any) error {
	if len(args) == 0 {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.data[section] == nil {
		w.data[section] = make(map[string][]map[string]any)
	}

	row := make(map[string]any, len(columns))
	for i, col := range columns {
		if i < len(args) {
			row[col] = normalizeValue(args[i])
		}
	}
	w.data[section][queryName] = append(w.data[section][queryName], row)
	return nil
}

func (w *jsonWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for section, queries := range w.data {
		filename := filepath.Join(w.dir, section+".json")
		data, err := json.MarshalIndent(queries, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling %s: %w", filename, err)
		}
		if err := os.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}

		totalRows := 0
		for _, rows := range queries {
			totalRows += len(rows)
		}
		slog.Info("wrote output", "file", filename, "rows", totalRows)
	}
	return nil
}

func normalizeValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []byte:
		return fmt.Sprintf("%x", val)
	default:
		return v
	}
}

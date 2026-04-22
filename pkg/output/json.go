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

func (w *jsonWriter) Add(row WriteRow) error {
	if len(row.Args) == 0 {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.data[row.Section] == nil {
		w.data[row.Section] = make(map[string][]map[string]any)
	}

	m := make(map[string]any, len(row.Columns))
	for i, col := range row.Columns {
		if i < len(row.Args) {
			m[col] = normalizeValue(row.Args[i])
		}
	}
	w.data[row.Section][row.Name] = append(w.data[row.Section][row.Name], m)
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

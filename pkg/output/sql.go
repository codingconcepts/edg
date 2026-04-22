package output

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type sqlWriter struct {
	driver   string
	dir      string
	mu       sync.Mutex
	sections map[string][]string
}

func newSQLWriter(driver, dir string) *sqlWriter {
	return &sqlWriter{
		driver:   driver,
		dir:      dir,
		sections: make(map[string][]string),
	}
}

func (w *sqlWriter) Add(row WriteRow) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	stmt := row.SQL
	if len(row.Args) > 0 {
		stmt = resolveArgs(stmt, w.driver, row.Args)
	}

	w.sections[row.Section] = append(w.sections[row.Section], ensureSemicolon(stmt))
	return nil
}

func (w *sqlWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for section, statements := range w.sections {
		filename := filepath.Join(w.dir, section+".sql")
		content := strings.Join(statements, "\n") + "\n"
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
		slog.Info("wrote output", "file", filename, "statements", len(statements))
	}
	return nil
}

func ensureSemicolon(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasSuffix(s, ";") {
		s += ";"
	}
	return s
}

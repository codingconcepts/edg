package output

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/codingconcepts/edg/pkg/convert"
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

func (w *sqlWriter) Add(section, queryName, querySQL string, columns []string, args []any) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var stmt string
	switch {
	case len(args) == 0:
		stmt = ensureSemicolon(querySQL)
	default:
		table := extractTable(querySQL)
		if table == "" {
			table = queryName
		}
		stmt = w.buildInsert(table, columns, args)
	}

	w.sections[section] = append(w.sections[section], stmt)
	return nil
}

func (w *sqlWriter) buildInsert(table string, columns []string, args []any) string {
	values := make([]string, len(args))
	for i, a := range args {
		values[i] = convert.SQLFormatValue(a, w.driver)
	}

	colList := ""
	if len(columns) > 0 && columns[0] != "col_1" {
		colList = " (" + strings.Join(columns, ", ") + ")"
	}

	return fmt.Sprintf("INSERT INTO %s%s VALUES (%s);", table, colList, strings.Join(values, ", "))
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

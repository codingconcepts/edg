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

func (w *sqlWriter) Add(row WriteRow) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var stmt string
	switch {
	case len(row.Args) == 0:
		stmt = ensureSemicolon(row.SQL)
	default:
		table := extractTable(row.SQL)
		if table == "" {
			table = row.Name
		}
		stmt = w.buildInsert(table, row.Columns, row.Args)
	}

	w.sections[row.Section] = append(w.sections[row.Section], stmt)
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

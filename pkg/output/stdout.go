package output

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type stdoutWriter struct {
	driver string
	w      io.Writer
	mu     sync.Mutex
}

func newStdoutWriter(driver string) *stdoutWriter {
	return &stdoutWriter{
		driver: driver,
		w:      os.Stdout,
	}
}

func (w *stdoutWriter) Add(row WriteRow) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	stmt := row.SQL
	if len(row.Args) > 0 {
		stmt = resolveArgs(stmt, w.driver, row.Args)
	}

	_, err := fmt.Fprintln(w.w, ensureSemicolon(stmt))
	return err
}

func (w *stdoutWriter) Flush() error {
	return nil
}

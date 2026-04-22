package output

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
)

type Format string

const (
	FormatSQL     Format = "sql"
	FormatJSON    Format = "json"
	FormatCSV     Format = "csv"
	FormatParquet Format = "parquet"
	FormatStdout  Format = "stdout"
)

func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "sql":
		return FormatSQL, nil
	case "json":
		return FormatJSON, nil
	case "csv":
		return FormatCSV, nil
	case "parquet":
		return FormatParquet, nil
	case "stdout":
		return FormatStdout, nil
	default:
		return "", fmt.Errorf("unknown output format %q (valid: sql, json, csv, parquet, stdout)", s)
	}
}

type WriteRow struct {
	Section  string
	Name     string
	SQL      string
	Columns  []string
	Args     []any
}

type Writer interface {
	Add(row WriteRow) error
	Flush() error
}

func New(format Format, driver, dir string) (Writer, error) {
	switch format {
	case FormatSQL:
		return newSQLWriter(driver, dir), nil
	case FormatJSON:
		return newJSONWriter(dir), nil
	case FormatCSV:
		return newCSVWriter(dir), nil
	case FormatParquet:
		return newParquetWriter(dir), nil
	case FormatStdout:
		return newStdoutWriter(driver), nil
	default:
		return nil, fmt.Errorf("unknown output format %q", format)
	}
}

var insertColRe = regexp.MustCompile(`(?i)INSERT\s+INTO\s+\S+\s*\(([^)]+)\)`)

func ExtractColumns(q *config.Query) []string {
	if q.Args.IsNamed() {
		cols := make([]string, len(q.Args.Names))
		for name, idx := range q.Args.Names {
			cols[idx] = name
		}
		return cols
	}

	if m := insertColRe.FindStringSubmatch(q.Query); len(m) == 2 {
		parts := strings.Split(m[1], ",")
		cols := make([]string, len(parts))
		for i, p := range parts {
			cols[i] = strings.TrimSpace(p)
		}
		if len(cols) == len(q.CompiledArgs) {
			return cols
		}
	}

	cols := make([]string, len(q.CompiledArgs))
	for i := range cols {
		cols[i] = fmt.Sprintf("col_%d", i+1)
	}
	return cols
}

func resolveArgs(query, driver string, args []any) string {
	for i := len(args) - 1; i >= 0; i-- {
		placeholder := fmt.Sprintf("$%d", i+1)
		formatted := convert.SQLFormatValue(args[i], driver)
		quoted := "'" + placeholder + "'"
		query = strings.ReplaceAll(query, quoted, formatted)
		query = strings.ReplaceAll(query, placeholder, formatted)
	}
	return query
}

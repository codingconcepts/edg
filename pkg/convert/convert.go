package convert

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/codingconcepts/edg/pkg/random"
)

// Sep is the batch field separator (ASCII unit separator, char 31).
// Used to delimit values within a single batch-expanded SQL placeholder.
const Sep = "\x1f"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func Constant(v any) any {
	return v
}

// Batch returns sequential integers [0, n) as a [][]any batch set,
// driving batched query execution without requiring a SQL query.
func Batch(args ...any) ([][]any, error) {
	if len(args) == 0 {
		return nil, errors.New("batch: requires at least 1 argument")
	}
	count, err := ToInt(args[0])
	if err != nil {
		return nil, fmt.Errorf("batch: %w", err)
	}
	step := 1
	if len(args) >= 2 {
		step, err = ToInt(args[1])
		if err != nil {
			return nil, fmt.Errorf("batch step: %w", err)
		}
	}
	result := make([][]any, count)
	for i := range count {
		result[i] = []any{i * step}
	}
	return result, nil
}

func ToInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case float64:
		return int(n), nil
	case int64:
		return int(n), nil
	case []byte:
		i, err := strconv.Atoi(string(n))
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q to int: %w", n, err)
		}
		return i, nil
	case string:
		i, err := strconv.Atoi(n)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q to int: %w", n, err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("cannot convert %T (%v) to int", v, v)
	}
}

func ToFloat(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case []byte:
		f, err := strconv.ParseFloat(string(n), 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q to float: %w", n, err)
		}
		return f, nil
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q to float: %w", n, err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("cannot convert %T (%v) to float", v, v)
	}
}

func Wrap(s string) string {
	if strings.HasPrefix(s, "{") {
		return s
	}
	return "{" + s + "}"
}

// Cond returns trueVal if predicate is true, falseVal otherwise.
//
//	cond(predicate, trueVal, falseVal)
func Cond(predicate, trueVal, falseVal any) any {
	if b, ok := predicate.(bool); ok && b {
		return trueVal
	}
	return falseVal
}

// Nullable returns nil with the given probability, otherwise returns val.
//
//	nullable(gen('email'), 0.3)
func Nullable(val, rawProbability any) (any, error) {
	p, err := ToFloat(rawProbability)
	if err != nil {
		return nil, fmt.Errorf("nullable probability: %w", err)
	}
	if random.Rng.Float64() < p {
		return nil, nil
	}
	return val, nil
}

// Coalesce returns the first non-nil value from arguments.
//
//	coalesce(val1, val2, val3, ...)
func Coalesce(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

// Tmpl formats a string using fmt.Sprintf.
//
//	template('ORD-%05d-%s', seq(1, 1), ref_rand('w').id)
func Tmpl(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

func BatchFormatValue(v any, driver string) string {
	if v == nil {
		return "NULL"
	}
	if b, ok := v.([]byte); ok {
		switch driver {
		case "mongodb":
			return base64.StdEncoding.EncodeToString(b)
		default:
			return hex.EncodeToString(b)
		}
	}
	var s string
	if f, ok := v.(float64); ok {
		if f == float64(int64(f)) {
			s = fmt.Sprintf("%d", int64(f))
		} else {
			s = strconv.FormatFloat(f, 'f', -1, 64)
		}
	} else {
		s = fmt.Sprint(v)
	}
	return strings.ReplaceAll(s, "'", "''")
}

// BatchJoinJSON takes a slice of formatted batch values and returns a
// properly escaped JSON array string (e.g. `["a","b","c"]`). Nil/NULL
// values become JSON null. This is safe for use with SQL Server's
// OPENJSON and avoids the delimiter/quoting issues of CSV joining.
func BatchJoinJSON(parts []string) string {
	elems := make([]string, len(parts))
	for i, p := range parts {
		if p == "NULL" {
			elems[i] = "null"
		} else {
			b, _ := json.Marshal(p)
			elems[i] = string(b)
		}
	}
	return "[" + strings.Join(elems, ",") + "]"
}

// RawSQL is a string that is already formatted for SQL and should not
// be quoted again by SQLFormatValue.
type RawSQL string

// SQLFormatValue formats a value for safe inline substitution in SQL.
// Strings are single-quoted with embedded quotes escaped ('→”);
// numeric types are returned as-is; nil becomes NULL; RawSQL values
// are returned unchanged; []byte values are hex-encoded using the
// driver-appropriate literal: 0x... for mssql, X'...' (ANSI SQL hex
// string literal) for all others.
func SQLFormatValue(v any, driver string) string {
	if v == nil {
		switch driver {
		case "mongodb":
			return "null"
		default:
			return "NULL"
		}
	}
	switch n := v.(type) {
	case RawSQL:
		return string(n)
	case int, int8, int16, int32, int64:
		return fmt.Sprint(v)
	case float64:
		if n == float64(int64(n)) {
			return fmt.Sprintf("%d", int64(n))
		}
		return strconv.FormatFloat(n, 'f', -1, 64)
	case []byte:
		switch driver {
		case "mssql", "cassandra":
			return "0x" + hex.EncodeToString(n)
		case "mongodb":
			return `"` + base64.StdEncoding.EncodeToString(n) + `"`
		default:
			return "X'" + hex.EncodeToString(n) + "'"
		}
	default:
		s := fmt.Sprint(n)
		switch driver {
		case "mongodb":
			return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
		case "cassandra":
			if uuidPattern.MatchString(s) {
				return s
			}
			return "'" + strings.ReplaceAll(s, "'", "''") + "'"
		default:
			return "'" + strings.ReplaceAll(s, "'", "''") + "'"
		}
	}
}

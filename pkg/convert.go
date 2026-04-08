package pkg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/codingconcepts/edg/pkg/random"
)

func constant(v any) any {
	return v
}

// batch returns sequential integers [0, n) as a [][]any batch set,
// driving batched query execution without requiring a SQL query.
func batch(n any) ([][]any, error) {
	count, err := toInt(n)
	if err != nil {
		return nil, fmt.Errorf("batch: %w", err)
	}
	result := make([][]any, count)
	for i := range count {
		result[i] = []any{i}
	}
	return result, nil
}

func toInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case float64:
		return int(n), nil
	case int64:
		return int(n), nil
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

func toFloat(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
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

func wrap(s string) string {
	if strings.HasPrefix(s, "{") {
		return s
	}
	return "{" + s + "}"
}

// cond returns trueVal if predicate is true, falseVal otherwise.
//
//	cond(predicate, trueVal, falseVal)
func cond(predicate, trueVal, falseVal any) any {
	if b, ok := predicate.(bool); ok && b {
		return trueVal
	}
	return falseVal
}

// nullable returns nil with the given probability, otherwise returns val.
//
//	nullable(gen('email'), 0.3)
func nullable(val, rawProbability any) (any, error) {
	p, err := toFloat(rawProbability)
	if err != nil {
		return nil, fmt.Errorf("nullable probability: %w", err)
	}
	if random.Rng.Float64() < p {
		return nil, nil
	}
	return val, nil
}

// coalesce returns the first non-nil value from arguments.
//
//	coalesce(val1, val2, val3, ...)
func coalesce(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

// tmpl formats a string using fmt.Sprintf.
//
//	template('ORD-%05d-%s', seq(1, 1), ref_rand('w').id)
func tmpl(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

// sqlFormatValue formats a value for safe inline substitution in SQL.
// Strings are single-quoted with embedded quotes escaped ('→'');
// numeric types are returned as-is; nil becomes NULL. The escaping
// is the same across PostgreSQL, MySQL, and Oracle.
func sqlFormatValue(v any) string {
	if v == nil {
		return "NULL"
	}
	switch v.(type) {
	case int, int64, float64:
		return fmt.Sprint(v)
	default:
		s := fmt.Sprint(v)
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
}

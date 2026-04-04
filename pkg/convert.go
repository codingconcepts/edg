package pkg

import (
	"fmt"
	"strconv"
	"strings"
)

func constant(v any) any {
	return v
}

// batch returns sequential integers [0, n) as a [][]any batch set,
// driving batched query execution without requiring a SQL query.
func batch(n any) [][]any {
	count := toInt(n)
	result := make([][]any, count)
	for i := range count {
		result[i] = []any{i}
	}
	return result
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		return 0
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

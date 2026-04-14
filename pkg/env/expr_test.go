package env

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newExprEnv builds an Env with the same globals and expressions
// as the _examples/expression/crdb.yaml file.
func newExprEnv(t *testing.T) *Env {
	t.Helper()

	req := &config.Request{
		Globals: map[string]any{
			"prices":    []any{9.99, 19.99, 4.99, 29.99, 14.99},
			"tags":      []any{"electronics", "books", "clothing", "food", "toys"},
			"names":     []any{"Alice", "Bob", "Charlie", "Diana", "Eve"},
			"threshold": 10,
			"matrix":    []any{[]any{1, 2}, []any{3, 4}, []any{5, 6}},
			"products": []any{
				map[string]any{"name": "Widget", "price": 29.99, "category": "tools"},
				map[string]any{"name": "Gadget", "price": 9.99, "category": "electronics"},
				map[string]any{"name": "Gizmo", "price": 19.99, "category": "tools"},
				map[string]any{"name": "Doohickey", "price": 4.99, "category": "toys"},
				map[string]any{"name": "Thingamajig", "price": 49.99, "category": "electronics"},
			},
		},
		Expressions: map[string]string{
			"grade": `if args[0] >= 90 { "A" }
else if args[0] >= 70 { "B" }
else { "F" }`,
			"sum_of_squares": `let a = float(args[0]); let b = float(args[1]); a**2 + b**2`,
		},
	}

	env, err := NewEnv(nil, "", req)
	require.NoError(t, err)
	return env
}

func TestArrayPredicates(t *testing.T) {
	env := newExprEnv(t)

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{"all_gt_0", "all(prices, # > 0)", true},
		{"all_gt_10", "all(prices, # > 10)", false},
		{"any_gt_25", "any(prices, # > 25)", true},
		{"any_gt_100", "any(prices, # > 100)", false},
		{"one_lt_5", "one(prices, # < 5)", true},
		{"one_gt_0", "one(prices, # > 0)", false},
		{"none_lt_0", "none(prices, # < 0)", true},
		{"none_gt_0", "none(prices, # > 0)", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.Eval(tt.expr)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestArraySearch(t *testing.T) {
	env := newExprEnv(t)

	tests := []struct {
		name string
		expr string
		want any
	}{
		{"find_gt_20", "find(prices, # > 20)", 29.99},
		{"findIndex_gt_20", "findIndex(prices, # > 20)", 3},
		{"findLast_gt_10", "findLast(prices, # > 10)", 14.99},
		{"findLastIndex_gt_10", "findLastIndex(prices, # > 10)", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.Eval(tt.expr)
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprint(tt.want), fmt.Sprint(got))
		})
	}
}

func TestArrayTransform_FilterMap(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(filter(prices, # > threshold), string(#)), ",")`)
	require.NoError(t, err)
	s := got.(string)
	// prices > 10: 19.99, 29.99, 14.99
	assert.Equal(t, "19.99,29.99,14.99", s)
}

func TestArrayTransform_MapDouble(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(prices, string(# * 2)), ",")`)
	require.NoError(t, err)
	s := got.(string)
	assert.Equal(t, "19.98,39.98,9.98,59.98,29.98", s)
}

func TestArrayTransform_Sort(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(sort(prices), string(#)), ",")`)
	require.NoError(t, err)
	s := got.(string)
	assert.Equal(t, "4.99,9.99,14.99,19.99,29.99", s)
}

func TestArrayTransform_SortBy(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(sortBy(products, #.price), #.name), ",")`)
	require.NoError(t, err)
	s := got.(string)
	assert.Equal(t, "Doohickey,Gadget,Gizmo,Widget,Thingamajig", s)
}

func TestArrayTransform_Reverse(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(reverse(names), ",")`)
	require.NoError(t, err)
	s := got.(string)
	assert.Equal(t, "Eve,Diana,Charlie,Bob,Alice", s)
}

func TestArrayTransform_UniqConcat(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(uniq(concat(tags, ["books", "games"])), ",")`)
	require.NoError(t, err)
	s := got.(string)
	// "books" already in tags so should appear once; "games" is new
	parts := strings.Split(s, ",")
	seen := map[string]int{}
	for _, p := range parts {
		seen[p]++
	}
	assert.Equal(t, 1, seen["books"])
	assert.Equal(t, 1, seen["games"])
	assert.Equal(t, 6, len(parts))
}

func TestArrayTransform_Flatten(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(flatten(matrix), string(#)), ",")`)
	require.NoError(t, err)
	s := got.(string)
	assert.Equal(t, "1,2,3,4,5,6", s)
}

func TestArrayAggregate_Reduce(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`reduce(prices, #acc + #, 0)`)
	require.NoError(t, err)
	f := got.(float64)
	want := 9.99 + 19.99 + 4.99 + 29.99 + 14.99
	assert.Equal(t, fmt.Sprintf("%.2f", want), fmt.Sprintf("%.2f", f))
}

func TestArrayAggregate_Mean(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`mean(prices)`)
	require.NoError(t, err)
	f := got.(float64)
	want := (9.99 + 19.99 + 4.99 + 29.99 + 14.99) / 5
	assert.Equal(t, fmt.Sprintf("%.2f", want), fmt.Sprintf("%.2f", f))
}

func TestArrayAggregate_Median(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`median(prices)`)
	require.NoError(t, err)
	f := got.(float64)
	// sorted: 4.99, 9.99, 14.99, 19.99, 29.99 -> median is 14.99
	assert.Equal(t, 14.99, f)
}

func TestArrayAggregate_First(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`first(names)`)
	require.NoError(t, err)
	assert.Equal(t, "Alice", got)
}

func TestArrayAggregate_Last(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`last(names)`)
	require.NoError(t, err)
	assert.Equal(t, "Eve", got)
}

func TestArrayAggregate_Take(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(take(names, 3), ",")`)
	require.NoError(t, err)
	assert.Equal(t, "Alice,Bob,Charlie", got)
}

func TestArrayAggregate_Len(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`len(names)`)
	require.NoError(t, err)
	assert.Equal(t, 5, got)
}

func TestMapFunctions_Keys(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(keys({a: 1, b: 2, c: 3}), ",")`)
	require.NoError(t, err)
	s := got.(string)
	parts := strings.Split(s, ",")
	require.Equal(t, 3, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		seen[p] = true
	}
	for _, k := range []string{"a", "b", "c"} {
		assert.True(t, seen[k], "keys: missing %q in %q", k, s)
	}
}

func TestMapFunctions_Values(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(values({a: 1, b: 2, c: 3}), string(#)), ",")`)
	require.NoError(t, err)
	s := got.(string)
	parts := strings.Split(s, ",")
	require.Equal(t, 3, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		seen[p] = true
	}
	for _, v := range []string{"1", "2", "3"} {
		assert.True(t, seen[v], "values: missing %q in %q", v, s)
	}
}

func TestMapFunctions_GroupBy(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`len(groupBy(products, #.category))`)
	require.NoError(t, err)
	assert.Equal(t, 3, got)
}

func TestStringOperators(t *testing.T) {
	env := newExprEnv(t)

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{"contains", `"hello world" contains "world"`, true},
		{"contains_false", `"hello world" contains "xyz"`, false},
		{"startsWith", `"hello" startsWith "hel"`, true},
		{"startsWith_false", `"hello" startsWith "xyz"`, false},
		{"endsWith", `"world" endsWith "rld"`, true},
		{"endsWith_false", `"world" endsWith "xyz"`, false},
		{"matches", `"hello123" matches "^[a-z]+[0-9]+$"`, true},
		{"matches_false", `"hello" matches "^[0-9]+$"`, false},
		{"in_membership", `"books" in tags`, true},
		{"in_membership_false", `"missing" in tags`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.Eval(tt.expr)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStringFunctions(t *testing.T) {
	env := newExprEnv(t)

	tests := []struct {
		name string
		expr string
		want string
	}{
		{"trimPrefix", `trimPrefix("test_value", "test_")`, "value"},
		{"trimSuffix", `trimSuffix("image.png", ".png")`, "image"},
		{"splitAfter", `join(splitAfter("a.b.c", "."), "|")`, "a.|b.|c"},
		{"repeat", `repeat("ab", 3)`, "ababab"},
		{"hasPrefix", `string(hasPrefix("hello", "hel"))`, "true"},
		{"hasSuffix", `string(hasSuffix("hello", "llo"))`, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.Eval(tt.expr)
			require.NoError(t, err)
			assert.Equal(t, tt.want, fmt.Sprint(got))
		})
	}
}

func TestStringFunctions_IndexOf(t *testing.T) {
	env := newExprEnv(t)

	tests := []struct {
		name string
		expr string
		want int
	}{
		{"indexOf", `indexOf("hello world", "world")`, 6},
		{"lastIndexOf", `lastIndexOf("abcabc", "abc")`, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.Eval(tt.expr)
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprint(tt.want), fmt.Sprint(got))
		})
	}
}

func TestTypeConversion_Type(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`type(42)`)
	require.NoError(t, err)
	assert.Equal(t, "int", got)
}

func TestTypeConversion_ToJSON(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`toJSON({name: "test", value: 42})`)
	require.NoError(t, err)
	s := got.(string)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(s), &parsed))
	assert.Equal(t, "test", parsed["name"])
	assert.Equal(t, float64(42), parsed["value"])
}

func TestTypeConversion_FromJSON(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`string(fromJSON("{\"a\": 1}").a)`)
	require.NoError(t, err)
	assert.Equal(t, "1", fmt.Sprint(got))
}

func TestTypeConversion_Base64(t *testing.T) {
	env := newExprEnv(t)

	t.Run("toBase64", func(t *testing.T) {
		got, err := env.Eval(`toBase64("hello")`)
		require.NoError(t, err)
		assert.Equal(t, "aGVsbG8=", got)
	})

	t.Run("fromBase64", func(t *testing.T) {
		got, err := env.Eval(`fromBase64("aGVsbG8=")`)
		require.NoError(t, err)
		assert.Equal(t, "hello", got)
	})
}

func TestTypeConversion_Pairs(t *testing.T) {
	env := newExprEnv(t)

	t.Run("toPairs", func(t *testing.T) {
		got, err := env.Eval(`len(toPairs({x: 1, y: 2}))`)
		require.NoError(t, err)
		assert.Equal(t, 2, got)
	})

	t.Run("fromPairs", func(t *testing.T) {
		got, err := env.Eval(`len(fromPairs([["x", 1], ["y", 2]]))`)
		require.NoError(t, err)
		assert.Equal(t, 2, got)
	})
}

func TestOperators_Range(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`reduce(1..5, #acc + #, 0)`)
	require.NoError(t, err)
	// 1..5 = [1,2,3,4,5] (inclusive), sum = 15
	assert.Equal(t, "15", fmt.Sprint(got))
}

func TestOperators_Slice(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(names[1:3], ",")`)
	require.NoError(t, err)
	assert.Equal(t, "Bob,Charlie", got)
}

func TestOperators_OptionalChaining_NilCoalescing(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`{name: "test"}?.missing ?? "default"`)
	require.NoError(t, err)
	assert.Equal(t, "default", got)
}

func TestOperators_IfElse(t *testing.T) {
	env := newExprEnv(t)

	tests := []struct {
		name string
		expr string
		want string
	}{
		{"grade_A", "grade(95)", "A"},
		{"grade_B", "grade(85)", "B"},
		{"grade_F", "grade(50)", "F"},
		{"grade_boundary_90", "grade(90)", "A"},
		{"grade_boundary_70", "grade(70)", "B"},
		{"grade_boundary_69", "grade(69)", "F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.Eval(tt.expr)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOperators_LetBindings(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`sum_of_squares(3, 4)`)
	require.NoError(t, err)
	// 3^2 + 4^2 = 9 + 16 = 25
	f, ok := got.(float64)
	require.True(t, ok, "sum_of_squares = %v (%T), want float64", got, got)
	assert.Equal(t, float64(25), f)
}

func TestBitwise(t *testing.T) {
	env := newExprEnv(t)

	tests := []struct {
		name string
		expr string
		want int
	}{
		{"bitand", "bitand(0xFF, 0x0F)", 0x0F},
		{"bitor", "bitor(0xF0, 0x0F)", 0xFF},
		{"bitxor", "bitxor(0xFF, 0x0F)", 0xF0},
		{"bitnand", "bitnand(0xFF, 0x0F)", 0xF0},
		{"bitnot", "bitnot(0)", ^0},
		{"bitshl", "bitshl(1, 8)", 256},
		{"bitshr", "bitshr(256, 4)", 16},
		{"bitushr", "bitushr(256, 4)", 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.Eval(tt.expr)
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprint(tt.want), fmt.Sprint(got))
		})
	}
}

func TestMisc_LenString(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`len("hello")`)
	require.NoError(t, err)
	assert.Equal(t, 5, got)
}

func TestMisc_GetWithDefault(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`get({a: 1, b: 2}, "c") ?? "missing"`)
	require.NoError(t, err)
	assert.Equal(t, "missing", got)
}

func TestMisc_GetExistingKey(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`get({a: 1, b: 2}, "a")`)
	require.NoError(t, err)
	assert.Equal(t, "1", fmt.Sprint(got))
}

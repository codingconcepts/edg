package pkg

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// newExprEnv builds an Env with the same globals and expressions
// as the _examples/expression/crdb.yaml file.
func newExprEnv(t *testing.T) *Env {
	t.Helper()

	req := &Request{
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

	env, err := NewEnv(nil, req)
	if err != nil {
		t.Fatalf("NewEnv failed: %v", err)
	}
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
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tt.expr, err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
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
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tt.expr, err)
			}
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestArrayTransform_FilterMap(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(filter(prices, # > threshold), string(#)), ",")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	s := got.(string)
	// prices > 10: 19.99, 29.99, 14.99
	if s != "19.99,29.99,14.99" {
		t.Errorf("filter+map = %q, want %q", s, "19.99,29.99,14.99")
	}
}

func TestArrayTransform_MapDouble(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(prices, string(# * 2)), ",")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	s := got.(string)
	if s != "19.98,39.98,9.98,59.98,29.98" {
		t.Errorf("map(# * 2) = %q, want %q", s, "19.98,39.98,9.98,59.98,29.98")
	}
}

func TestArrayTransform_Sort(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(sort(prices), string(#)), ",")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	s := got.(string)
	if s != "4.99,9.99,14.99,19.99,29.99" {
		t.Errorf("sort = %q, want %q", s, "4.99,9.99,14.99,19.99,29.99")
	}
}

func TestArrayTransform_SortBy(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(sortBy(products, #.price), #.name), ",")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	s := got.(string)
	if s != "Doohickey,Gadget,Gizmo,Widget,Thingamajig" {
		t.Errorf("sortBy(#.price) = %q, want %q", s, "Doohickey,Gadget,Gizmo,Widget,Thingamajig")
	}
}

func TestArrayTransform_Reverse(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(reverse(names), ",")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	s := got.(string)
	if s != "Eve,Diana,Charlie,Bob,Alice" {
		t.Errorf("reverse = %q, want %q", s, "Eve,Diana,Charlie,Bob,Alice")
	}
}

func TestArrayTransform_UniqConcat(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(uniq(concat(tags, ["books", "games"])), ",")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	s := got.(string)
	// "books" already in tags so should appear once; "games" is new
	parts := strings.Split(s, ",")
	seen := map[string]int{}
	for _, p := range parts {
		seen[p]++
	}
	if seen["books"] != 1 {
		t.Errorf("uniq+concat: 'books' appears %d times, want 1", seen["books"])
	}
	if seen["games"] != 1 {
		t.Errorf("uniq+concat: 'games' not found in result %q", s)
	}
	if len(parts) != 6 {
		t.Errorf("uniq+concat: got %d elements, want 6", len(parts))
	}
}

func TestArrayTransform_Flatten(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(flatten(matrix), string(#)), ",")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	s := got.(string)
	if s != "1,2,3,4,5,6" {
		t.Errorf("flatten = %q, want %q", s, "1,2,3,4,5,6")
	}
}

func TestArrayAggregate_Reduce(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`reduce(prices, #acc + #, 0)`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	f := got.(float64)
	want := 9.99 + 19.99 + 4.99 + 29.99 + 14.99
	if fmt.Sprintf("%.2f", f) != fmt.Sprintf("%.2f", want) {
		t.Errorf("reduce = %v, want %v", f, want)
	}
}

func TestArrayAggregate_Mean(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`mean(prices)`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	f := got.(float64)
	want := (9.99 + 19.99 + 4.99 + 29.99 + 14.99) / 5
	if fmt.Sprintf("%.2f", f) != fmt.Sprintf("%.2f", want) {
		t.Errorf("mean = %v, want %v", f, want)
	}
}

func TestArrayAggregate_Median(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`median(prices)`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	f := got.(float64)
	// sorted: 4.99, 9.99, 14.99, 19.99, 29.99 -> median is 14.99
	if f != 14.99 {
		t.Errorf("median = %v, want 14.99", f)
	}
}

func TestArrayAggregate_First(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`first(names)`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != "Alice" {
		t.Errorf("first = %v, want Alice", got)
	}
}

func TestArrayAggregate_Last(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`last(names)`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != "Eve" {
		t.Errorf("last = %v, want Eve", got)
	}
}

func TestArrayAggregate_Take(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(take(names, 3), ",")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != "Alice,Bob,Charlie" {
		t.Errorf("take(3) = %v, want Alice,Bob,Charlie", got)
	}
}

func TestArrayAggregate_Len(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`len(names)`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != 5 {
		t.Errorf("len(names) = %v, want 5", got)
	}
}

func TestMapFunctions_Keys(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(keys({a: 1, b: 2, c: 3}), ",")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	s := got.(string)
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		t.Fatalf("keys: got %d elements, want 3", len(parts))
	}
	seen := map[string]bool{}
	for _, p := range parts {
		seen[p] = true
	}
	for _, k := range []string{"a", "b", "c"} {
		if !seen[k] {
			t.Errorf("keys: missing %q in %q", k, s)
		}
	}
}

func TestMapFunctions_Values(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(map(values({a: 1, b: 2, c: 3}), string(#)), ",")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	s := got.(string)
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		t.Fatalf("values: got %d elements, want 3", len(parts))
	}
	seen := map[string]bool{}
	for _, p := range parts {
		seen[p] = true
	}
	for _, v := range []string{"1", "2", "3"} {
		if !seen[v] {
			t.Errorf("values: missing %q in %q", v, s)
		}
	}
}

func TestMapFunctions_GroupBy(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`len(groupBy(products, #.category))`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != 3 {
		t.Errorf("len(groupBy) = %v, want 3", got)
	}
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
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tt.expr, err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
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
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tt.expr, err)
			}
			if fmt.Sprint(got) != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
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
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tt.expr, err)
			}
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestTypeConversion_Type(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`type(42)`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != "int" {
		t.Errorf("type(42) = %v, want int", got)
	}
}

func TestTypeConversion_ToJSON(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`toJSON({name: "test", value: 42})`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	s := got.(string)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		t.Fatalf("toJSON result is not valid JSON: %v", err)
	}
	if parsed["name"] != "test" {
		t.Errorf("toJSON name = %v, want test", parsed["name"])
	}
	if parsed["value"] != float64(42) {
		t.Errorf("toJSON value = %v, want 42", parsed["value"])
	}
}

func TestTypeConversion_FromJSON(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`string(fromJSON("{\"a\": 1}").a)`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if fmt.Sprint(got) != "1" {
		t.Errorf("fromJSON = %v, want 1", got)
	}
}

func TestTypeConversion_Base64(t *testing.T) {
	env := newExprEnv(t)

	t.Run("toBase64", func(t *testing.T) {
		got, err := env.Eval(`toBase64("hello")`)
		if err != nil {
			t.Fatalf("Eval error: %v", err)
		}
		if got != "aGVsbG8=" {
			t.Errorf("toBase64 = %v, want aGVsbG8=", got)
		}
	})

	t.Run("fromBase64", func(t *testing.T) {
		got, err := env.Eval(`fromBase64("aGVsbG8=")`)
		if err != nil {
			t.Fatalf("Eval error: %v", err)
		}
		if got != "hello" {
			t.Errorf("fromBase64 = %v, want hello", got)
		}
	})
}

func TestTypeConversion_Pairs(t *testing.T) {
	env := newExprEnv(t)

	t.Run("toPairs", func(t *testing.T) {
		got, err := env.Eval(`len(toPairs({x: 1, y: 2}))`)
		if err != nil {
			t.Fatalf("Eval error: %v", err)
		}
		if got != 2 {
			t.Errorf("len(toPairs) = %v, want 2", got)
		}
	})

	t.Run("fromPairs", func(t *testing.T) {
		got, err := env.Eval(`len(fromPairs([["x", 1], ["y", 2]]))`)
		if err != nil {
			t.Fatalf("Eval error: %v", err)
		}
		if got != 2 {
			t.Errorf("len(fromPairs) = %v, want 2", got)
		}
	})
}

func TestOperators_Range(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`reduce(1..5, #acc + #, 0)`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	// 1..5 = [1,2,3,4,5] (inclusive), sum = 15
	if fmt.Sprint(got) != "15" {
		t.Errorf("reduce(1..5) = %v, want 15", got)
	}
}

func TestOperators_Slice(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`join(names[1:3], ",")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != "Bob,Charlie" {
		t.Errorf("names[1:3] = %v, want Bob,Charlie", got)
	}
}

func TestOperators_OptionalChaining_NilCoalescing(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`{name: "test"}?.missing ?? "default"`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != "default" {
		t.Errorf("?. + ?? = %v, want default", got)
	}
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
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tt.expr, err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestOperators_LetBindings(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`sum_of_squares(3, 4)`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	// 3^2 + 4^2 = 9 + 16 = 25
	f, ok := got.(float64)
	if !ok {
		t.Fatalf("sum_of_squares = %v (%T), want float64", got, got)
	}
	if f != 25 {
		t.Errorf("sum_of_squares(3, 4) = %v, want 25", f)
	}
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
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tt.expr, err)
			}
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestMisc_LenString(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`len("hello")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != 5 {
		t.Errorf("len(\"hello\") = %v, want 5", got)
	}
}

func TestMisc_GetWithDefault(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`get({a: 1, b: 2}, "c") ?? "missing"`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if got != "missing" {
		t.Errorf("get + ?? = %v, want missing", got)
	}
}

func TestMisc_GetExistingKey(t *testing.T) {
	env := newExprEnv(t)

	got, err := env.Eval(`get({a: 1, b: 2}, "a")`)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if fmt.Sprint(got) != "1" {
		t.Errorf("get existing key = %v, want 1", got)
	}
}

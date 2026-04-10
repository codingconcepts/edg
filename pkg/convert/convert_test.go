package convert

import (
	"testing"
)

func TestConstant(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"string", "hello"},
		{"int", 42},
		{"nil", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Constant(tt.input)
			if got != tt.input {
				t.Errorf("Constant(%v) = %v, want %v", tt.input, got, tt.input)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"number:1,10", "{number:1,10}"},
		{"{number:1,10}", "{number:1,10}"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := Wrap(tt.input); got != tt.want {
				t.Errorf("Wrap(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    float64
		wantErr bool
	}{
		{"float64", 3.14, 3.14, false},
		{"int", 42, 42.0, false},
		{"int64", int64(99), 99.0, false},
		{"unsupported", "hello", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToFloat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ToFloat(%v) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ToFloat(%v) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ToFloat(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBatch(t *testing.T) {
	result, err := Batch(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("Batch(3) returned %d sets, want 3", len(result))
	}
	for i, row := range result {
		if len(row) != 1 {
			t.Fatalf("batch row %d has %d values, want 1", i, len(row))
		}
		if row[0] != i {
			t.Errorf("batch row %d = %v, want %d", i, row[0], i)
		}
	}
}

func TestBatch_Zero(t *testing.T) {
	result, err := Batch(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("Batch(0) returned %d sets, want 0", len(result))
	}
}

func TestCond(t *testing.T) {
	if got := Cond(true, "yes", "no"); got != "yes" {
		t.Errorf("Cond(true) = %v, want yes", got)
	}
	if got := Cond(false, "yes", "no"); got != "no" {
		t.Errorf("Cond(false) = %v, want no", got)
	}
}

func TestCond_NonBool(t *testing.T) {
	// Non-bool predicate should return falseVal.
	if got := Cond("truthy", "yes", "no"); got != "no" {
		t.Errorf("Cond(string) = %v, want no", got)
	}
}

func TestNullable_AlwaysNull(t *testing.T) {
	// probability=1.0 should always return nil.
	for range 100 {
		got, err := Nullable("value", 1.0)
		if err != nil {
			t.Fatalf("Nullable error: %v", err)
		}
		if got != nil {
			t.Fatalf("Nullable(val, 1.0) = %v, want nil", got)
		}
	}
}

func TestNullable_NeverNull(t *testing.T) {
	// probability=0.0 should always return the value.
	for range 100 {
		got, err := Nullable("value", 0.0)
		if err != nil {
			t.Fatalf("Nullable error: %v", err)
		}
		if got != "value" {
			t.Fatalf("Nullable(val, 0.0) = %v, want value", got)
		}
	}
}

func TestNullable_InvalidProbability(t *testing.T) {
	_, err := Nullable("value", "not_a_number")
	if err == nil {
		t.Fatal("expected error for invalid probability, got nil")
	}
}

func TestCoalesce(t *testing.T) {
	if got := Coalesce(nil, nil, "first", "second"); got != "first" {
		t.Errorf("Coalesce = %v, want first", got)
	}
}

func TestCoalesce_AllNil(t *testing.T) {
	if got := Coalesce(nil, nil); got != nil {
		t.Errorf("Coalesce(nil, nil) = %v, want nil", got)
	}
}

func TestTemplate(t *testing.T) {
	got := Tmpl("ORD-%05d-%s", 42, "abc")
	if got != "ORD-00042-abc" {
		t.Errorf("Tmpl = %q, want ORD-00042-abc", got)
	}
}

func BenchmarkToInt(b *testing.B) {
	cases := []struct {
		name  string
		input any
	}{
		{"int", 42},
		{"float64", 42.0},
		{"int64", int64(42)},
		{"unsupported", "42"},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				_, _ = ToInt(tc.input)
			}
		})
	}
}

func BenchmarkWrap(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{"needs_wrap", "number:1,100"},
		{"already_wrapped", "{number:1,100}"},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for range b.N {
				Wrap(tc.input)
			}
		})
	}
}

package pkg

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration for YAML unmarshaling from strings like "1s".
type Duration time.Duration

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	dur, err := time.ParseDuration(node.Value)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

type Request struct {
	Globals     map[string]any                `json:"globals" yaml:"globals"`
	Expressions map[string]string             `json:"expressions" yaml:"expressions"`
	Rows        map[string][]string           `json:"rows" yaml:"rows"`
	Reference   map[string][]map[string]any   `json:"reference" yaml:"reference"`
	Up          []*Query                      `json:"up" yaml:"up"`
	Seed        []*Query                      `json:"seed" yaml:"seed"`
	Deseed      []*Query                      `json:"deseed" yaml:"deseed"`
	Down        []*Query                      `json:"down" yaml:"down"`
	Init        []*Query                      `json:"init" yaml:"init"`
	RunWeights  map[string]int                `json:"run_weights" yaml:"run_weights"`
	Run         []*Query                      `json:"run" yaml:"run"`
}

type QueryType string
type ConfigSection string

const (
	QueryTypeQuery     QueryType = "query"
	QueryTypeExec      QueryType = "exec"
	QueryTypeQueryBatch QueryType = "query_batch"
	QueryTypeExecBatch  QueryType = "exec_batch"

	ConfigSectionUp     ConfigSection = "up"
	ConfigSectionSeed   ConfigSection = "seed"
	ConfigSectionDeseed ConfigSection = "deseed"
	ConfigSectionDown   ConfigSection = "down"
	ConfigSectionInit   ConfigSection = "init"
	ConfigSectionRun    ConfigSection = "run"
)

type QueryResult struct {
	Name    string
	Section ConfigSection
	Latency time.Duration
	Err     error
	Count   int
}

type Query struct {
	Name         string        `json:"name" yaml:"name"`
	Type         QueryType     `json:"type" yaml:"type"`
	Wait         Duration      `json:"wait" yaml:"wait"`
	Count        any           `json:"count" yaml:"count"`
	Size         any           `json:"size" yaml:"size"`
	Query        string        `json:"query" yaml:"query"`
	Row          string        `json:"row" yaml:"row"`
	Args         []string      `json:"args" yaml:"args"`
	CompiledArgs []*vm.Program `json:"-" yaml:"-"`

	compiledCount *vm.Program
	compiledSize  *vm.Program
}

func (q *Query) isBatch() bool {
	return q.Type == QueryTypeQueryBatch || q.Type == QueryTypeExecBatch
}

func (q *Query) CompileArgs(e *Env) error {
	compiledArgs := make([]*vm.Program, len(q.Args))

	for i, arg := range q.Args {
		program, err := expr.Compile(arg, expr.Env(e.env))
		if err != nil {
			return fmt.Errorf("failed to compile arg %d (%s): %w", i, arg, err)
		}
		compiledArgs[i] = program
	}

	q.CompiledArgs = compiledArgs

	if q.Count != nil {
		p, err := expr.Compile(fmt.Sprintf("%v", q.Count), expr.Env(e.env))
		if err != nil {
			return fmt.Errorf("failed to compile count (%v): %w", q.Count, err)
		}
		q.compiledCount = p
	}
	if q.Size != nil {
		p, err := expr.Compile(fmt.Sprintf("%v", q.Size), expr.Env(e.env))
		if err != nil {
			return fmt.Errorf("failed to compile size (%v): %w", q.Size, err)
		}
		q.compiledSize = p
	}

	return nil
}

// GenerateArgs evaluates compiled arg expressions and returns one or more
// arg sets. When a single arg evaluates to [][]any (from ref_each), each
// inner slice becomes a separate arg set, causing the query to run once
// per batch row. Otherwise a single arg set is returned.
//
// For batch queries (type: query_batch or exec_batch), args are evaluated
// repeatedly per row, with values collected into comma-separated strings
// per arg position.
func (q *Query) GenerateArgs(e *Env) ([][]any, error) {
	defer e.clearOneCache()
	defer e.resetUniqIndex()

	if q.Type == QueryTypeQueryBatch || q.Type == QueryTypeExecBatch {
		return q.generateBatchArgs(e)
	}

	if len(q.CompiledArgs) == 0 {
		return [][]any{nil}, nil
	}

	var completeArgs []any
	for _, cq := range q.CompiledArgs {
		compiledArg, err := expr.Run(cq, e.env)
		if err != nil {
			return nil, fmt.Errorf("error running expr: %w", err)
		}
		completeArgs = append(completeArgs, compiledArg)
	}

	// Find a ref_each batch arg ([][]any) and expand it into multiple
	// arg sets, merging with any scalar args at their positions.
	for i, arg := range completeArgs {
		batches, ok := arg.([][]any)
		if !ok {
			continue
		}

		result := make([][]any, len(batches))
		for b, batchRow := range batches {
			row := make([]any, 0, len(completeArgs)-1+len(batchRow))
			row = append(row, completeArgs[:i]...)
			row = append(row, batchRow...)
			row = append(row, completeArgs[i+1:]...)
			result[b] = row
		}
		return result, nil
	}

	return [][]any{completeArgs}, nil
}

// generateBatchArgs handles type: query_batch/exec_batch queries. It evaluates each arg
// expression repeatedly (once per row), collecting values into CSV strings.
// Count and Size control the total rows and batch grouping.
func (q *Query) generateBatchArgs(e *Env) ([][]any, error) {
	count := 1
	if q.compiledCount != nil {
		v, err := expr.Run(q.compiledCount, e.env)
		if err != nil {
			return nil, fmt.Errorf("error evaluating count: %w", err)
		}
		c, err := toInt(v)
		if err != nil {
			return nil, fmt.Errorf("count: %w", err)
		}
		count = c
	}

	size := count
	if q.compiledSize != nil {
		v, err := expr.Run(q.compiledSize, e.env)
		if err != nil {
			return nil, fmt.Errorf("error evaluating size: %w", err)
		}
		s, err := toInt(v)
		if err != nil {
			return nil, fmt.Errorf("size: %w", err)
		}
		size = s
	}
	if size <= 0 {
		size = count
	}

	batches := (count + size - 1) / size
	result := make([][]any, batches)

	for b := range batches {
		n := size
		if remaining := count - b*size; remaining < size {
			n = remaining
		}

		perArg := make([][]string, len(q.CompiledArgs))
		for i := range perArg {
			perArg[i] = make([]string, n)
		}

		for row := range n {
			e.clearOneCache()
			for i, cq := range q.CompiledArgs {
				v, err := expr.Run(cq, e.env)
				if err != nil {
					return nil, fmt.Errorf("error running batch arg %d row %d: %w", i, b*size+row, err)
				}
				perArg[i][row] = sqlFormatValue(v)
			}
		}

		args := make([]any, len(q.CompiledArgs))
		for i, parts := range perArg {
			args[i] = strings.Join(parts, ",")
		}
		result[b] = args
	}

	return result, nil
}

func (q *Query) Run(ctx context.Context, e *Env, args ...any) error {
	switch q.Type {
	case QueryTypeExec, QueryTypeExecBatch:
		if err := e.Exec(ctx, e.db, q, args...); err != nil {
			return fmt.Errorf("executing exec %s: %w", q.Name, err)
		}
	case QueryTypeQuery, QueryTypeQueryBatch, "":
		if err := e.Query(ctx, e.db, q, args...); err != nil {
			return fmt.Errorf("executing query %s: %w", q.Name, err)
		}
	}

	return nil
}

// Validate checks the Request for structural issues that would cause
// confusing runtime errors.
func (r *Request) Validate() error {
	// When run_weights is configured, every run query needs a name for selection.
	if len(r.RunWeights) > 0 {
		for i, q := range r.Run {
			if q.Name == "" {
				return fmt.Errorf("run query %d is missing a name (required when run_weights is set)", i)
			}
		}
	}

	// run_weights keys must match actual run query names.
	if len(r.RunWeights) > 0 {
		runNames := make(map[string]bool, len(r.Run))
		for _, q := range r.Run {
			runNames[q.Name] = true
		}
		for name := range r.RunWeights {
			if !runNames[name] {
				return fmt.Errorf("run_weights references unknown query %q", name)
			}
		}
	}

	// Validate row references: row name must exist, and row + args are mutually exclusive.
	for _, group := range []struct {
		name    string
		queries []*Query
	}{
		{"up", r.Up},
		{"seed", r.Seed},
		{"deseed", r.Deseed},
		{"down", r.Down},
		{"init", r.Init},
		{"run", r.Run},
	} {
		for i, q := range group.queries {
			if q.Row == "" {
				continue
			}
			if len(q.Args) > 0 {
				return fmt.Errorf("%s query %d (%s): row and args are mutually exclusive", group.name, i, q.Name)
			}
			if _, ok := r.Rows[q.Row]; !ok {
				return fmt.Errorf("%s query %d (%s): references unknown row %q", group.name, i, q.Name, q.Row)
			}
		}
	}

	// Check for duplicate query names within each section.
	for _, group := range []struct {
		name    string
		queries []*Query
	}{
		{"up", r.Up},
		{"seed", r.Seed},
		{"deseed", r.Deseed},
		{"down", r.Down},
		{"init", r.Init},
		{"run", r.Run},
	} {
		seen := make(map[string]bool)
		for i, q := range group.queries {
			if q.Name == "" {
				continue
			}
			if seen[q.Name] {
				return fmt.Errorf("duplicate query name %q in %s (query %d)", q.Name, group.name, i)
			}
			seen[q.Name] = true
		}
	}

	return nil
}

package pkg

import (
	"context"
	"fmt"
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
	Globals     map[string]any    `json:"globals" yaml:"globals"`
	Expressions map[string]string `json:"expressions" yaml:"expressions"`
	Up          []*Query          `json:"up" yaml:"up"`
	Seed        []*Query          `json:"seed" yaml:"seed"`
	Deseed      []*Query          `json:"deseed" yaml:"deseed"`
	Down        []*Query          `json:"down" yaml:"down"`
	Init        []*Query          `json:"init" yaml:"init"`
	RunWeights  map[string]int    `json:"run_weights" yaml:"run_weights"`
	Run         []*Query          `json:"run" yaml:"run"`
}

type QueryType string
type ConfigSection string

const (
	QueryTypeQuery QueryType = "query"
	QueryTypeExec  QueryType = "exec"

	ConfigSectionUp     ConfigSection = "up"
	ConfigSectionSeed   ConfigSection = "seed"
	ConfigSectionDeseed ConfigSection = "deseed"
	ConfigSectionDown   ConfigSection = "down"
	ConfigSectionInit   ConfigSection = "init"
	ConfigSectionRun    ConfigSection = "run"
)

type Query struct {
	Name         string        `json:"name" yaml:"name"`
	Type         QueryType     `json:"type" yaml:"type"`
	Wait         Duration      `json:"wait" yaml:"wait"`
	Query        string        `json:"query" yaml:"query"`
	Args         []string      `json:"args" yaml:"args"`
	CompiledArgs []*vm.Program `json:"-" yaml:"-"`
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
	return nil
}

// GenerateArgs evaluates compiled arg expressions and returns one or more
// arg sets. When a single arg evaluates to [][]any (from ref_each), each
// inner slice becomes a separate arg set, causing the query to run once
// per batch row. Otherwise a single arg set is returned.
func (q *Query) GenerateArgs(e *Env) ([][]any, error) {
	defer e.clearOneCache()
	defer e.resetUniqIndex()

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

func (q *Query) Run(ctx context.Context, e *Env, args ...any) error {
	switch q.Type {
	case QueryTypeExec:
		if err := e.Exec(ctx, e.db, q, args...); err != nil {
			return fmt.Errorf("executing exec %s: %w", q.Name, err)
		}
	case QueryTypeQuery, "":
		if err := e.Query(ctx, e.db, q, args...); err != nil {
			return fmt.Errorf("executing query %s: %w", q.Name, err)
		}
	}

	return nil
}

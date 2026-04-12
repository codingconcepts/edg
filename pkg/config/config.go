package config

import (
	"fmt"
	"regexp"
	"time"

	"github.com/codingconcepts/edg/pkg/gen"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"gopkg.in/yaml.v3"
)

var genCallRe = regexp.MustCompile(`gen\(\s*['"]([^'"]+)['"]\s*\)`)

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

type Stage struct {
	Name     string   `json:"name" yaml:"name"`
	Workers  int      `json:"workers" yaml:"workers"`
	Duration Duration `json:"duration" yaml:"duration"`
}

type Request struct {
	Globals      map[string]any              `json:"globals" yaml:"globals"`
	Expressions  map[string]string           `json:"expressions" yaml:"expressions"`
	Rows         map[string][]string         `json:"rows" yaml:"rows"`
	Reference    map[string][]map[string]any `json:"reference" yaml:"reference"`
	Stages       []Stage                     `json:"stages" yaml:"stages"`
	Up           []*Query                    `json:"up" yaml:"up"`
	Seed         []*Query                    `json:"seed" yaml:"seed"`
	Deseed       []*Query                    `json:"deseed" yaml:"deseed"`
	Down         []*Query                    `json:"down" yaml:"down"`
	Init         []*Query                    `json:"init" yaml:"init"`
	RunWeights   map[string]int              `json:"run_weights" yaml:"run_weights"`
	Run          []*RunItem                  `json:"run" yaml:"run"`
	Expectations []string                    `json:"expectations" yaml:"expectations"`
}

// RunAllQueries returns a flat list of all queries in the run section,
// including queries nested inside transactions.
func (r *Request) RunAllQueries() []*Query {
	var queries []*Query
	for _, item := range r.Run {
		queries = append(queries, item.AllQueries()...)
	}
	return queries
}

type QueryType string
type ConfigSection string

const (
	QueryTypeQuery      QueryType = "query"
	QueryTypeExec       QueryType = "exec"
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
	Name          string
	Section       ConfigSection
	Latency       time.Duration
	Err           error
	Count         int
	IsTransaction bool
	Rollback      bool
}

type Query struct {
	Name         string        `json:"name" yaml:"name"`
	Type         QueryType     `json:"type" yaml:"type"`
	Prepared     bool          `json:"prepared" yaml:"prepared"`
	Wait         Duration      `json:"wait" yaml:"wait"`
	Count        any           `json:"count" yaml:"count"`
	Size         any           `json:"size" yaml:"size"`
	Query        string        `json:"query" yaml:"query"`
	Row          string        `json:"row" yaml:"row"`
	Args         []string      `json:"args" yaml:"args"`
	BatchFormat  string        `json:"batch_format" yaml:"batch_format"`
	RollbackIf   string        `json:"rollback_if" yaml:"rollback_if"`
	CompiledArgs []*vm.Program `json:"-" yaml:"-"`

	CompiledCount      *vm.Program `json:"-" yaml:"-"`
	CompiledSize       *vm.Program `json:"-" yaml:"-"`
	CompiledRollbackIf *vm.Program `json:"-" yaml:"-"`
}

// IsRollbackIf returns true when this query element is a rollback_if
// condition check rather than a database query.
func (q *Query) IsRollbackIf() bool {
	return q.RollbackIf != ""
}

// CompileRollbackIf compiles the rollback_if expression (if set)
// against the given environment map.
func (q *Query) CompileRollbackIf(envMap map[string]any) error {
	if q.RollbackIf == "" {
		return nil
	}
	p, err := expr.Compile(q.RollbackIf, expr.Env(envMap), expr.AsBool())
	if err != nil {
		return fmt.Errorf("failed to compile rollback_if (%s): %w", q.RollbackIf, err)
	}
	q.CompiledRollbackIf = p
	return nil
}

// Transaction groups multiple queries to run inside an explicit
// BEGIN/COMMIT block.
type Transaction struct {
	Name    string            `json:"name" yaml:"transaction"`
	Locals  map[string]string `json:"locals" yaml:"locals"`
	Queries []*Query          `json:"queries" yaml:"queries"`

	CompiledLocals map[string]*vm.Program `json:"-" yaml:"-"`
}

// CompileLocals compiles each locals expression against the given
// environment map.
func (tx *Transaction) CompileLocals(envMap map[string]any) error {
	tx.CompiledLocals = make(map[string]*vm.Program, len(tx.Locals))
	for name, body := range tx.Locals {
		p, err := expr.Compile(body, expr.Env(envMap))
		if err != nil {
			return fmt.Errorf("failed to compile local %q (%s): %w", name, body, err)
		}
		tx.CompiledLocals[name] = p
	}
	return nil
}

// RunItem represents a single entry in the run section: either a
// standalone query or a transaction containing multiple queries.
type RunItem struct {
	Query       *Query
	Transaction *Transaction
}

func (ri *RunItem) UnmarshalYAML(node *yaml.Node) error {
	// Probe for "transaction" key to distinguish from a plain query.
	var probe struct {
		Transaction string `yaml:"transaction"`
	}
	if err := node.Decode(&probe); err == nil && probe.Transaction != "" {
		var tx Transaction
		if err := node.Decode(&tx); err != nil {
			return err
		}
		ri.Transaction = &tx
		return nil
	}

	var q Query
	if err := node.Decode(&q); err != nil {
		return err
	}
	ri.Query = &q
	return nil
}

// Name returns the name of the run item (query name or transaction name).
func (ri *RunItem) Name() string {
	if ri.Transaction != nil {
		return ri.Transaction.Name
	}
	if ri.Query != nil {
		return ri.Query.Name
	}
	return ""
}

// IsTransaction returns true when this run item is a transaction group.
func (ri *RunItem) IsTransaction() bool {
	return ri.Transaction != nil
}

// AllQueries returns all queries contained in this run item.
func (ri *RunItem) AllQueries() []*Query {
	if ri.Transaction != nil {
		return ri.Transaction.Queries
	}
	if ri.Query != nil {
		return []*Query{ri.Query}
	}
	return nil
}

func (q *Query) IsBatch() bool {
	return q.Type == QueryTypeQueryBatch || q.Type == QueryTypeExecBatch
}

func (q *Query) CompileArgs(envMap map[string]any) error {
	compiledArgs := make([]*vm.Program, len(q.Args))

	for i, arg := range q.Args {
		program, err := expr.Compile(arg, expr.Env(envMap))
		if err != nil {
			return fmt.Errorf("failed to compile arg %d (%s): %w", i, arg, err)
		}
		compiledArgs[i] = program
	}

	q.CompiledArgs = compiledArgs

	if q.Count != nil {
		p, err := expr.Compile(fmt.Sprintf("%v", q.Count), expr.Env(envMap))
		if err != nil {
			return fmt.Errorf("failed to compile count (%v): %w", q.Count, err)
		}
		q.CompiledCount = p
	}
	if q.Size != nil {
		p, err := expr.Compile(fmt.Sprintf("%v", q.Size), expr.Env(envMap))
		if err != nil {
			return fmt.Errorf("failed to compile size (%v): %w", q.Size, err)
		}
		q.CompiledSize = p
	}

	return nil
}

// Validate checks the Request for structural issues that would cause
// confusing runtime errors.
func (r *Request) Validate() error {
	// When run_weights is configured, every run item needs a name for selection.
	if len(r.RunWeights) > 0 {
		for i, item := range r.Run {
			if item.Name() == "" {
				return fmt.Errorf("run item %d is missing a name (required when run_weights is set)", i)
			}
		}
	}

	// run_weights keys must match actual run item names.
	if len(r.RunWeights) > 0 {
		runNames := make(map[string]bool, len(r.Run))
		for _, item := range r.Run {
			runNames[item.Name()] = true
		}
		for name := range r.RunWeights {
			if !runNames[name] {
				return fmt.Errorf("run_weights references unknown query %q", name)
			}
		}
	}

	// Validate row references: row name must exist, and row + args are mutually exclusive.
	runQueries := r.RunAllQueries()
	for _, group := range []struct {
		name    string
		queries []*Query
	}{
		{"up", r.Up},
		{"seed", r.Seed},
		{"deseed", r.Deseed},
		{"down", r.Down},
		{"init", r.Init},
		{"run", runQueries},
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
		{"run", runQueries},
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

	// Check for duplicate names at the run-item level (transaction names
	// and standalone query names share the same namespace for run_weights).
	{
		seen := make(map[string]bool)
		for i, item := range r.Run {
			name := item.Name()
			if name == "" {
				continue
			}
			if seen[name] {
				return fmt.Errorf("duplicate name %q in run (item %d)", name, i)
			}
			seen[name] = true
		}
	}

	// Validate transaction constraints.
	for i, item := range r.Run {
		if !item.IsTransaction() {
			continue
		}
		if len(item.Transaction.Queries) == 0 {
			return fmt.Errorf("run transaction %d (%s): must contain at least one query", i, item.Name())
		}
		for j, q := range item.Transaction.Queries {
			if q.IsRollbackIf() {
				if q.Query != "" || q.Name != "" || q.Type != "" || len(q.Args) > 0 {
					return fmt.Errorf("run transaction %d (%s): rollback_if element %d must not have name, type, args, or query", i, item.Name(), j)
				}
				continue
			}
			if q.IsBatch() {
				return fmt.Errorf("run transaction %d (%s): query %d (%s) cannot be a batch type inside a transaction", i, item.Name(), j, q.Name)
			}
			if q.Prepared {
				return fmt.Errorf("run transaction %d (%s): query %d (%s) cannot use prepared statements inside a transaction", i, item.Name(), j, q.Name)
			}
		}

		// Locals names must not collide with query names in the same transaction.
		for name := range item.Transaction.Locals {
			for _, q := range item.Transaction.Queries {
				if q.Name == name {
					return fmt.Errorf("run transaction %d (%s): local %q shadows query name", i, item.Name(), name)
				}
			}
		}
	}

	// Validate gen() patterns reference known gofakeit functions.
	{
		var exprs []string
		for _, v := range r.Expressions {
			exprs = append(exprs, v)
		}
		for _, row := range r.Rows {
			exprs = append(exprs, row...)
		}
		for _, group := range []struct {
			name    string
			queries []*Query
		}{
			{"up", r.Up},
			{"seed", r.Seed},
			{"deseed", r.Deseed},
			{"down", r.Down},
			{"init", r.Init},
			{"run", runQueries},
		} {
			for _, q := range group.queries {
				exprs = append(exprs, q.Args...)
			}
		}
		for _, item := range r.Run {
			if item.IsTransaction() {
				for _, v := range item.Transaction.Locals {
					exprs = append(exprs, v)
				}
			}
		}

		for _, e := range exprs {
			for _, m := range genCallRe.FindAllStringSubmatch(e, -1) {
				if err := gen.ValidatePattern(m[1]); err != nil {
					return fmt.Errorf("expression %q: %w", e, err)
				}
			}
		}
	}

	return nil
}

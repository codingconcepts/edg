package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/codingconcepts/edg/pkg/gen"
	"github.com/codingconcepts/edg/pkg/seq"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"gopkg.in/yaml.v3"
)

var (
	genCallRe = regexp.MustCompile(`gen\(\s*['"]([^'"]+)['"]\s*\)`)
	envCallRe = regexp.MustCompile(`env\(\s*(?:'([^']+)'|"([^"]+)")\s*\)`)
	seqCallRe = regexp.MustCompile(`seq_(?:global|rand|zipf|norm|exp|lognorm)\(\s*(?:'([^']+)'|"([^"]+)")`)
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

type Rate struct {
	Times          int
	Interval       time.Duration
	tickerInterval time.Duration
}

func (r *Rate) UnmarshalYAML(node *yaml.Node) error {
	parts := strings.Split(node.Value, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid rate format %q: expected times/interval (e.g. 1/10s)", node.Value)
	}

	var err error
	if r.Times, err = strconv.Atoi(parts[0]); err != nil {
		return fmt.Errorf("parsing times: %w", err)
	}

	if r.Interval, err = time.ParseDuration(parts[1]); err != nil {
		return fmt.Errorf("parsing interval: %w", err)
	}

	r.tickerInterval = r.Interval / time.Duration(r.Times)
	return nil
}

func (r Rate) TickerInterval() time.Duration {
	return r.tickerInterval
}

func (r Rate) String() string {
	return fmt.Sprintf("%d/%s", r.Times, r.Interval)
}

type Stage struct {
	Name     string   `json:"name" yaml:"name"`
	Workers  int      `json:"workers" yaml:"workers"`
	Duration Duration `json:"duration" yaml:"duration"`
}

type Request struct {
	Globals      map[string]any              `json:"globals" yaml:"globals"`
	GlobalsOrder []string                    `json:"-" yaml:"-"`
	Expressions  map[string]string           `json:"expressions" yaml:"expressions"`
	Rows         map[string][]string         `json:"rows" yaml:"rows"`
	Reference    map[string][]map[string]any `json:"reference" yaml:"reference"`
	Seq          []seq.Config                `json:"seq" yaml:"seq"`
	Stages       []Stage                     `json:"stages" yaml:"stages"`
	Up           []*Query                    `json:"up" yaml:"up"`
	Seed         []*Query                    `json:"seed" yaml:"seed"`
	Deseed       []*Query                    `json:"deseed" yaml:"deseed"`
	Down         []*Query                    `json:"down" yaml:"down"`
	Init         []*Query                    `json:"init" yaml:"init"`
	RunWeights   map[string]int              `json:"run_weights" yaml:"run_weights"`
	Run          []*RunItem                  `json:"run" yaml:"run"`
	Workers      []*Worker                   `json:"workers" yaml:"workers"`
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

func (r *Request) WorkerQueries() []*Query {
	queries := make([]*Query, len(r.Workers))
	for i, w := range r.Workers {
		queries[i] = &w.Query
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
	ConfigSectionWorker ConfigSection = "worker"
)

type QueryResult struct {
	Name          string
	Section       ConfigSection
	Latency       time.Duration
	Err           error
	Count         int
	IsTransaction bool
	Rollback    bool
	PrintAggs   []string
	PrintValues []string
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
	Args         QueryArgs     `json:"args" yaml:"args"`
	BatchFormat  string        `json:"batch_format" yaml:"batch_format"`
	RollbackIf    string        `json:"rollback_if" yaml:"rollback_if"`
	Print         []PrintExpr   `json:"print" yaml:"print"`
	CompiledArgs  []*vm.Program `json:"-" yaml:"-"`
	CompiledPrint []*vm.Program `json:"-" yaml:"-"`

	CompiledCount      *vm.Program `json:"-" yaml:"-"`
	CompiledSize       *vm.Program `json:"-" yaml:"-"`
	CompiledRollbackIf *vm.Program `json:"-" yaml:"-"`
}

// QueryArgs holds arg expressions in either positional (list) or named (map)
// form. Named args preserve YAML declaration order for $1/$2/... binding.
type QueryArgs struct {
	Exprs []string       // ordered expressions
	Names map[string]int // name → index; nil for positional args
}

func (qa QueryArgs) Len() int      { return len(qa.Exprs) }
func (qa QueryArgs) IsNamed() bool { return qa.Names != nil }

func PositionalArgs(exprs ...string) QueryArgs {
	return QueryArgs{Exprs: exprs}
}

func (qa *QueryArgs) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.SequenceNode:
		var list []string
		if err := node.Decode(&list); err != nil {
			return err
		}
		qa.Exprs = list
		return nil

	case yaml.MappingNode:
		qa.Names = make(map[string]int, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			name := node.Content[i].Value
			value := node.Content[i+1].Value
			qa.Names[name] = len(qa.Exprs)
			qa.Exprs = append(qa.Exprs, value)
		}
		return nil

	default:
		return fmt.Errorf("args must be a list or map, got %v", node.Kind)
	}
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

func (q *Query) CompilePrint(envMap map[string]any) error {
	if len(q.Print) == 0 {
		return nil
	}
	compiled := make([]*vm.Program, len(q.Print))
	for i, pe := range q.Print {
		program, err := expr.Compile(pe.Expr, expr.Env(envMap))
		if err != nil {
			return fmt.Errorf("failed to compile print %d (%s): %w", i, pe.Expr, err)
		}
		compiled[i] = program
	}
	q.CompiledPrint = compiled
	return nil
}

type PrintExpr struct {
	Expr string `json:"expr" yaml:"expr"`
	Agg  string `json:"agg" yaml:"agg"`
}

func (pe *PrintExpr) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		pe.Expr = node.Value
		return nil
	}
	type raw PrintExpr
	var r raw
	if err := node.Decode(&r); err != nil {
		return err
	}
	*pe = PrintExpr(r)
	return nil
}

type Worker struct {
	Query `yaml:",inline" json:",inline"`
	Rate  Rate `json:"rate" yaml:"rate"`
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
	compiledArgs := make([]*vm.Program, q.Args.Len())

	for i, arg := range q.Args.Exprs {
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

	// Validate seq definitions.
	{
		seen := make(map[string]bool, len(r.Seq))
		for i, s := range r.Seq {
			if s.Name == "" {
				return fmt.Errorf("seq %d is missing a name", i)
			}
			if seen[s.Name] {
				return fmt.Errorf("duplicate seq name %q", s.Name)
			}
			seen[s.Name] = true
		}
	}

	// Validate row references: row name must exist, and row + args are mutually exclusive.
	runQueries := r.RunAllQueries()
	workerQueries := r.WorkerQueries()
	groups := []struct {
		name    string
		queries []*Query
	}{
		{"up", r.Up},
		{"seed", r.Seed},
		{"deseed", r.Deseed},
		{"down", r.Down},
		{"init", r.Init},
		{"run", runQueries},
		{"workers", workerQueries},
	}

	for _, group := range groups {
		for i, q := range group.queries {
			if q.Row == "" {
				continue
			}
			if q.Args.Len() > 0 {
				return fmt.Errorf("%s query %d (%s): row and args are mutually exclusive", group.name, i, q.Name)
			}
			if _, ok := r.Rows[q.Row]; !ok {
				return fmt.Errorf("%s query %d (%s): references unknown row %q", group.name, i, q.Name, q.Row)
			}
		}
	}

	// Check for duplicate query names within each section.
	for _, group := range groups {
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
				if q.Query != "" || q.Name != "" || q.Type != "" || q.Args.Len() > 0 {
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

	for i, w := range r.Workers {
		if w.Name == "" {
			return fmt.Errorf("worker %d is missing a name", i)
		}
		if w.Rate.Times <= 0 || w.Rate.Interval <= 0 {
			return fmt.Errorf("worker %d (%s): rate must have positive times and interval", i, w.Name)
		}
	}

	seqNames := make(map[string]bool, len(r.Seq))
	for _, s := range r.Seq {
		seqNames[s.Name] = true
	}

	// Walk the config and validate gen(...), env(...), and seq_*(...) expressions inline.
	validateExpr := func(e string) error {
		for _, m := range genCallRe.FindAllStringSubmatch(e, -1) {
			if err := gen.ValidatePattern(m[1]); err != nil {
				return fmt.Errorf("expression %q: %w", e, err)
			}
		}
		for _, m := range envCallRe.FindAllStringSubmatch(e, -1) {
			name := m[1]
			if name == "" {
				name = m[2]
			}
			if _, ok := os.LookupEnv(name); !ok {
				return fmt.Errorf("missing environment variable: %q", name)
			}
		}
		for _, m := range seqCallRe.FindAllStringSubmatch(e, -1) {
			name := m[1]
			if name == "" {
				name = m[2]
			}
			if !seqNames[name] {
				return fmt.Errorf("expression %q: unknown sequence %q (not defined in seq section)", e, name)
			}
		}
		return nil
	}

	for _, v := range r.Expressions {
		if err := validateExpr(v); err != nil {
			return err
		}
	}
	for _, row := range r.Rows {
		for _, v := range row {
			if err := validateExpr(v); err != nil {
				return err
			}
		}
	}
	for _, group := range groups {
		for _, q := range group.queries {
			for _, arg := range q.Args.Exprs {
				if err := validateExpr(arg); err != nil {
					return err
				}
			}
			for _, pe := range q.Print {
				if err := validateExpr(pe.Expr); err != nil {
					return err
				}
			}
		}
	}
	for _, item := range r.Run {
		if item.IsTransaction() {
			for _, v := range item.Transaction.Locals {
				if err := validateExpr(v); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

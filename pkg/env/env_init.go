package env

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/expr-lang/expr"
)

func (e *Env) loadGlobals(r *config.Request) error {
	for name := range r.Globals {
		if _, exists := e.env[name]; exists {
			return fmt.Errorf("global %q shadows a built-in function", name)
		}
	}

	maps.Copy(e.env, r.Globals)

	for _, name := range r.GlobalsOrder {
		strVal, ok := r.Globals[name].(string)
		if !ok {
			continue
		}
		program, err := expr.Compile(strVal, expr.Env(e.env))
		if err != nil {
			continue
		}
		result, err := expr.Run(program, e.env)
		if err != nil {
			return fmt.Errorf("evaluating global %q: %w", name, err)
		}
		e.env[name] = result
		r.Globals[name] = result
	}
	return nil
}

func (e *Env) loadReferences(r *config.Request) error {
	for name, rows := range r.Reference {
		if _, exists := e.env[name]; exists {
			return fmt.Errorf("reference dataset %q shadows a built-in function or global", name)
		}
		e.SetEnv(name, slices.Clone(rows))
	}
	return nil
}

func (e *Env) registerExpressions(r *config.Request) error {
	for name := range r.Expressions {
		if _, exists := e.env[name]; exists {
			return fmt.Errorf("expression %q shadows a built-in function, global, or reference dataset", name)
		}
		e.env[name] = func(args ...any) (any, error) {
			return nil, fmt.Errorf("expression %q used before compilation", name)
		}
	}
	for name, body := range r.Expressions {
		compileEnv := maps.Clone(e.env)
		compileEnv["args"] = []any{}

		program, err := expr.Compile(body, expr.Env(compileEnv))
		if err != nil {
			return fmt.Errorf("compiling expression %q: %w", name, err)
		}

		p := program
		e.env[name] = func(args ...any) (any, error) {
			runEnv := maps.Clone(e.env)
			runEnv["args"] = args
			return expr.Run(p, runEnv)
		}
	}
	return nil
}

func (e *Env) compileQueries(r *config.Request) error {
	runQueries := r.RunAllQueries()
	workerQueries := r.WorkerQueries()
	allQueries := [][]*config.Query{r.Up, r.Seed, r.Deseed, r.Down, r.Init, runQueries, workerQueries}

	for _, queries := range allQueries {
		for _, query := range queries {
			if query.Row != "" {
				query.Args = config.QueryArgs{Exprs: slices.Clone(r.Rows[query.Row])}
			}
		}
	}

	sepSQL := string(e.sep())
	for _, queries := range allQueries {
		for _, query := range queries {
			query.Query = strings.ReplaceAll(query.Query, "__sep__", sepSQL)
		}
	}

	for _, group := range []struct {
		name    string
		queries []*config.Query
	}{
		{"up", r.Up},
		{"seed", r.Seed},
		{"deseed", r.Deseed},
		{"down", r.Down},
		{"init", r.Init},
		{"run", runQueries},
		{"workers", workerQueries},
	} {
		for i, query := range group.queries {
			if query.IsRollbackIf() {
				if err := query.CompileRollbackIf(e.env); err != nil {
					return fmt.Errorf("failed to compile %s rollback_if %d: %w", group.name, i, err)
				}
				continue
			}
			switch query.Type {
			case config.QueryTypeQuery, config.QueryTypeExec, config.QueryTypeQueryBatch, config.QueryTypeExecBatch, "":
			default:
				return fmt.Errorf("unknown query type %q in %s query %d (%s)", query.Type, group.name, i, query.Name)
			}
			if err := query.CompileArgs(e.env); err != nil {
				return fmt.Errorf("failed to compile %s query %d: %w", group.name, i, err)
			}
			if err := query.CompilePrint(e.env); err != nil {
				return fmt.Errorf("failed to compile %s query %d print: %w", group.name, i, err)
			}
		}
	}

	for _, item := range r.Run {
		if item.Transaction != nil && len(item.Transaction.Locals) > 0 {
			if err := item.Transaction.CompileLocals(e.env); err != nil {
				return fmt.Errorf("transaction %s: %w", item.Transaction.Name, err)
			}
		}
	}
	return nil
}

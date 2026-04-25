package env

import (
	"fmt"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/convert"
	"github.com/codingconcepts/edg/pkg/output"
	"github.com/expr-lang/expr"
)

func (e *Env) captureOutput(q *config.Query, section config.ConfigSection) error {
	columns := output.ExtractColumns(q)

	if q.IsBatch() {
		defer e.clearOneCache()
		defer e.resetUniqIndex()
		defer e.evalPrint(q)
		return e.captureOutputBatch(q, section, columns)
	}

	argSets, _, err := e.GenerateArgs(q)
	if err != nil {
		return fmt.Errorf("generating args for %s query %s: %w", section, q.Name, err)
	}

	var envRows []map[string]any
	for _, args := range argSets {
		resolvedSQL := q.Query
		if len(args) > 0 {
			resolvedSQL = inlineArgs(q.Query, args, e.driver)
		}
		if err := e.Output.Add(output.WriteRow{Section: string(section), Name: q.Name, SQL: resolvedSQL, Columns: columns, Args: args}); err != nil {
			return err
		}
		if len(columns) > 0 && len(args) > 0 {
			row := make(map[string]any, len(columns))
			for i, col := range columns {
				if i < len(args) {
					row[col] = args[i]
				}
			}
			envRows = append(envRows, row)
		}
	}

	if len(envRows) > 0 {
		e.SetEnv(q.Name, envRows)
	}

	e.sendResult(config.QueryResult{Name: q.Name, Section: section, Count: len(argSets)})
	return nil
}

func (e *Env) captureOutputBatch(q *config.Query, section config.ConfigSection, columns []string) error {
	count := 1
	if q.CompiledCount != nil {
		v, err := expr.Run(q.CompiledCount, e.env)
		if err != nil {
			return fmt.Errorf("evaluating count: %w", err)
		}
		c, err := convert.ToInt(v)
		if err != nil {
			return fmt.Errorf("count: %w", err)
		}
		count = c
	}

	e.computedArgNames = q.Args.Names

	envRows := make([]map[string]any, 0, count)
	for row := range count {
		e.clearOneCache()
		e.computedArgs = e.computedArgs[:0]
		args := make([]any, len(q.CompiledArgs))
		for i, cq := range q.CompiledArgs {
			v, err := expr.Run(cq, e.env)
			if err != nil {
				return fmt.Errorf("evaluating arg %d row %d: %w", i, row, err)
			}
			args[i] = v
			e.computedArgs = append(e.computedArgs, v)
		}

		if err := e.Output.Add(output.WriteRow{Section: string(section), Name: q.Name, SQL: q.Query, Columns: columns, Args: args}); err != nil {
			return err
		}

		r := make(map[string]any, len(columns))
		for i, col := range columns {
			if i < len(args) {
				r[col] = args[i]
			}
		}
		envRows = append(envRows, r)
	}

	if len(envRows) > 0 {
		e.SetEnv(q.Name, envRows)
	}

	e.sendResult(config.QueryResult{Name: q.Name, Section: section, Count: count})
	return nil
}

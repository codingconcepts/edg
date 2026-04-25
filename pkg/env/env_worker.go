package env

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/codingconcepts/edg/pkg/db"
	"github.com/codingconcepts/edg/pkg/random"
	"github.com/expr-lang/expr"
)

func (e *Env) RunWorker(ctx context.Context, w *config.Worker) {
	ticker := time.NewTicker(w.Rate.TickerInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.runSection(ctx, []*config.Query{&w.Query}, config.ConfigSectionWorker, e.db); err != nil {
				if ctx.Err() != nil {
					return
				}
				var failErr *ErrFail
				if errors.As(err, &failErr) {
					slog.Error("worker stopped", "worker", w.Name, "error", err)
					return
				}
				slog.Error("worker error", "worker", w.Name, "error", err)
			}
		}
	}
}

// runRunItems dispatches each run item as either a standalone query
// or a multi-statement transaction.
func (e *Env) runRunItems(ctx context.Context, items []*config.RunItem) error {
	for _, item := range items {
		switch {
		case item.IsTransaction():
			if err := e.runTransaction(ctx, item.Transaction); err != nil {
				return err
			}
		default:
			if err := e.runSection(ctx, []*config.Query{item.Query}, config.ConfigSectionRun, e.db); err != nil {
				return err
			}
		}
	}
	return nil
}

// runTransaction wraps the queries of a Transaction in an explicit
// BEGIN/COMMIT block. On error the transaction is rolled back.
// When a rollback_if condition is set, it is evaluated after each
// query; if it returns true the transaction is rolled back without
// being treated as an error.
//
// When Retries > 0 the transaction is retried on error up to that
// many additional times before reporting a failure.
func (e *Env) runTransaction(ctx context.Context, tx *config.Transaction) error {
	attempts := 1 + e.Retries
	start := time.Now()

	var lastErr error
	for attempt := range attempts {
		if attempt > 0 {
			backoff := time.Duration(1<<attempt) * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				break
			}
		}
		lastErr = e.tryTransaction(ctx, tx)
		if lastErr == nil {
			e.sendResult(config.QueryResult{Name: tx.Name, Section: config.ConfigSectionRun, Latency: time.Since(start), Count: 1, IsTransaction: true})
			return nil
		}
		if errors.Is(lastErr, ErrConditionalRollback) {
			e.sendResult(config.QueryResult{Name: tx.Name, Section: config.ConfigSectionRun, Latency: time.Since(start), Count: 1, IsTransaction: true, Rollback: true})
			return nil
		}
		if ctx.Err() != nil {
			break
		}
	}

	e.sendResult(config.QueryResult{Name: tx.Name, Section: config.ConfigSectionRun, Err: lastErr, IsTransaction: true})
	return lastErr
}

func (e *Env) tryTransaction(ctx context.Context, tx *config.Transaction) error {
	transactor, ok := e.db.(db.Transactor)
	if !ok {
		return fmt.Errorf("driver does not support transactions")
	}

	dbTx, err := transactor.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction %s: %w", tx.Name, err)
	}

	if len(tx.CompiledLocals) > 0 {
		if err := e.evalLocals(tx); err != nil {
			_ = dbTx.Rollback()
			return err
		}
		defer e.clearLocals()
	}

	if err := e.runTransactionQueries(ctx, tx, dbTx); err != nil {
		_ = dbTx.Rollback()
		return err
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("committing transaction %s: %w", tx.Name, err)
	}

	return nil
}

// runTransactionQueries executes each query in the transaction.
// When a rollback_if element is encountered, its condition is
// evaluated; if true, ErrConditionalRollback is returned.
func (e *Env) runTransactionQueries(ctx context.Context, tx *config.Transaction, dbTx db.Transaction) error {
	for _, q := range tx.Queries {
		if q.IsRollbackIf() {
			result, err := expr.Run(q.CompiledRollbackIf, e.env)
			if err != nil {
				return fmt.Errorf("evaluating rollback_if in transaction %s: %w", tx.Name, err)
			}
			if b, ok := result.(bool); ok && b {
				return ErrConditionalRollback
			}
			continue
		}

		if err := e.runSection(ctx, []*config.Query{q}, config.ConfigSectionRun, dbTx); err != nil {
			return err
		}
	}
	return nil
}

// pickWeighted selects a single run item using the cumulative
// weights from run_weights. Items not listed in run_weights
// are excluded.
func (e *Env) pickWeighted() *config.RunItem {
	type entry struct {
		item       *config.RunItem
		cumulative int
	}

	var entries []entry
	total := 0
	for _, item := range e.request.Run {
		w, ok := e.request.RunWeights[item.Name()]
		if !ok {
			continue
		}
		total += w
		entries = append(entries, entry{item: item, cumulative: total})
	}

	if total == 0 {
		return nil
	}

	r := random.Rng.IntN(total)
	for _, e := range entries {
		if r < e.cumulative {
			return e.item
		}
	}

	return entries[len(entries)-1].item
}

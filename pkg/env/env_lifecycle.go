package env

import (
	"context"

	"github.com/codingconcepts/edg/pkg/config"
)

// Up runs the schema-up queries to create tables.
func (e *Env) Up(ctx context.Context) error {
	return e.runSection(ctx, e.request.Up, config.ConfigSectionUp, e.db)
}

// Seed runs the seed queries to populate tables with data.
func (e *Env) Seed(ctx context.Context) error {
	return e.runSection(ctx, e.request.Seed, config.ConfigSectionSeed, e.db)
}

// Deseed runs the deseed queries to delete data from tables.
func (e *Env) Deseed(ctx context.Context) error {
	return e.runSection(ctx, e.request.Deseed, config.ConfigSectionDeseed, e.db)
}

// Down runs the schema-down queries to tear down tables.
func (e *Env) Down(ctx context.Context) error {
	return e.runSection(ctx, e.request.Down, config.ConfigSectionDown, e.db)
}

// Init runs the init queries once (e.g. to seed reference data).
func (e *Env) Init(ctx context.Context) error {
	return e.runSection(ctx, e.request.Init, config.ConfigSectionInit, e.db)
}

// InitFrom copies init query results from another Env, avoiding
// redundant database queries when multiple workers need the same
// initial dataset. Each query-type result gets its own slice copy
// so that refDiff's in-place swaps don't interfere across workers.
func (e *Env) InitFrom(source *Env) {
	for _, q := range e.request.Init {
		if q.Type != config.QueryTypeQuery {
			continue
		}
		source.envMutex.RLock()
		data, ok := source.env[q.Name].([]map[string]any)
		source.envMutex.RUnlock()
		if !ok {
			continue
		}
		copied := make([]map[string]any, len(data))
		for i, row := range data {
			copied[i] = make(map[string]any, len(row))
			for k, v := range row {
				copied[i][k] = v
			}
		}
		e.SetEnv(q.Name, copied)
	}
}

// RunIteration executes one iteration of the run queries. When run_weights
// is configured, a single item is chosen by weighted random selection.
// Otherwise all run items execute sequentially.
func (e *Env) RunIteration(ctx context.Context) error {
	if len(e.request.RunWeights) == 0 {
		return e.runRunItems(ctx, e.request.Run)
	}

	item := e.pickWeighted()
	if item == nil {
		return e.runRunItems(ctx, e.request.Run)
	}
	return e.runRunItems(ctx, []*config.RunItem{item})
}

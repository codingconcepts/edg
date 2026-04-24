package env

import (
	"fmt"
	"strings"
	"testing"

	edgdb "github.com/codingconcepts/edg/pkg/db"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefRand(t *testing.T) {
	rows := sampleRows()
	env := testEnv(map[string][]map[string]any{"items": rows})

	result := env.refRand("items")
	require.NotNil(t, result)

	assert.Contains(t, result, "id")
}

func TestRefRand_UnknownName(t *testing.T) {
	env := testEnv(nil)
	result := env.refRand("nonexistent")
	assert.Nil(t, result)
}

func TestRefRand_EmptyData(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"empty": {}})
	result := env.refRand("empty")
	assert.Nil(t, result)
}

func TestRefN(t *testing.T) {
	rows := sampleRows()
	env := testEnv(map[string][]map[string]any{"items": rows})

	result := env.refN("items", "id", 2, 3)
	require.NotEmpty(t, result)

	parts := strings.Split(result, ",")
	if len(parts) < 2 || len(parts) > 3 {
		t.Errorf("refN returned %d items, want 2-3", len(parts))
	}

	// All values should be unique.
	seen := map[string]bool{}
	for _, v := range parts {
		assert.False(t, seen[v], "refN returned duplicate value: %v", v)
		seen[v] = true
	}
}

func TestRefN_ClampsToDataSize(t *testing.T) {
	rows := sampleRows()
	env := testEnv(map[string][]map[string]any{"items": rows})

	result := env.refN("items", "id", 5, 10)
	parts := strings.Split(result, ",")
	assert.Equal(t, 3, len(parts), "refN returned %d items, want 3 (clamped to data size)", len(parts))
}

func TestRefN_UnknownName(t *testing.T) {
	env := testEnv(nil)
	result := env.refN("nonexistent", "id", 1, 3)
	assert.Empty(t, result)
}

func TestRefSame_ReturnsSameRow(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"users": sampleRows()})

	first := env.refSame("users")
	second := env.refSame("users")

	assert.Equal(t, first["id"], second["id"])
}

func TestRefSame_ClearedBetweenCycles(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"users": sampleRows()})

	first := env.refSame("users")
	env.clearOneCache()

	// After clearing, a new random row is picked. Run enough times to
	// confirm it doesn't always match (statistically near-certain with 3 rows).
	different := false
	for range 20 {
		second := env.refSame("users")
		if first["id"] != second["id"] {
			different = true
			break
		}
		env.clearOneCache()
	}
	assert.True(t, different, "refSame returned the same row 20 times after cache clears; expected variation")
}

func TestRefSame_UnknownName(t *testing.T) {
	env := testEnv(nil)
	result := env.refSame("nonexistent")
	assert.Nil(t, result)
}

func TestRefSame_EmptyData(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"empty": {}})
	result := env.refSame("empty")
	assert.Nil(t, result)
}

func TestRefPerm_ReturnsSameRowForever(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"warehouses": sampleRows()})

	first := env.refPerm("warehouses")
	require.NotNil(t, first)

	for range 10 {
		got := env.refPerm("warehouses")
		assert.Equal(t, first["id"], got["id"])
	}
}

func TestRefPerm_SurvivesCacheClear(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"warehouses": sampleRows()})

	first := env.refPerm("warehouses")

	// oneCache clear should NOT affect permCache
	env.clearOneCache()

	got := env.refPerm("warehouses")
	assert.Equal(t, first["id"], got["id"])
}

func TestRefPerm_UnknownName(t *testing.T) {
	env := testEnv(nil)
	result := env.refPerm("nonexistent")
	assert.Nil(t, result)
}

func TestRefDiff_ReturnsUniqueRows(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"items": sampleRows()})

	seen := map[any]bool{}
	for range 3 {
		row := env.refDiff("items")
		require.NotNil(t, row)
		id := row["id"]
		assert.False(t, seen[id], "refDiff returned duplicate id: %v", id)
		seen[id] = true
	}

	assert.Equal(t, 3, len(seen))
}

func TestRefDiff_ResetsAfterCycle(t *testing.T) {
	env := testEnv(map[string][]map[string]any{"items": sampleRows()})

	// Exhaust all 3 rows.
	for range 3 {
		env.refDiff("items")
	}

	// Reset and verify we can get rows again.
	env.resetUniqIndex()

	row := env.refDiff("items")
	require.NotNil(t, row)
}

func TestRefDiff_UnknownName(t *testing.T) {
	env := testEnv(nil)
	result := env.refDiff("nonexistent")
	assert.Nil(t, result)
}

func TestRefEach(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, "alice").
			AddRow(2, "bob").
			AddRow(3, "charlie"),
	)

	env := &Env{db: edgdb.NewSQDB(db)}
	got, err := env.refEach("SELECT id, name FROM items")
	require.NoError(t, err)

	require.Equal(t, 3, len(got))
	for i, row := range got {
		assert.Equal(t, 2, len(row), "row %d has %d columns, want 2", i, len(row))
	}
	assert.Equal(t, int64(1), got[0][0])
	assert.Equal(t, "charlie", got[2][1])
}

func TestRefEach_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("connection refused"))

	env := &Env{db: edgdb.NewSQDB(db)}
	got, err := env.refEach("SELECT 1")

	require.Error(t, err)
	assert.Nil(t, got)
}

func TestRefEach_NoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}),
	)

	env := &Env{db: edgdb.NewSQDB(db)}
	got, err := env.refEach("SELECT id, name FROM empty_table")
	require.NoError(t, err)

	assert.Equal(t, 0, len(got))
}

func TestRefEach_NilDB(t *testing.T) {
	env := &Env{db: nil}
	got, err := env.refEach("SELECT id FROM items")

	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "requires a database connection")
}

func BenchmarkRefRand(b *testing.B) {
	cases := []struct {
		name string
		rows int
	}{
		{"rows_10", 10},
		{"rows_100", 100},
		{"rows_1000", 1000},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			env := benchEnv(tc.rows)
			b.ResetTimer()
			for range b.N {
				env.refRand("items")
			}
		})
	}
}

func BenchmarkRefN(b *testing.B) {
	cases := []struct {
		name string
		rows int
		n    int
	}{
		{"rows_100/n_5", 100, 5},
		{"rows_100/n_15", 100, 15},
		{"rows_100/n_50", 100, 50},
		{"rows_1000/n_5", 1000, 5},
		{"rows_1000/n_15", 1000, 15},
		{"rows_1000/n_50", 1000, 50},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			env := benchEnv(tc.rows)
			b.ResetTimer()
			for range b.N {
				env.refN("items", "id", tc.n, tc.n)
			}
		})
	}
}

func BenchmarkRefSame(b *testing.B) {
	rows := make([]map[string]any, 100)
	for i := range rows {
		rows[i] = map[string]any{"id": i}
	}

	b.Run("cache_hit", func(b *testing.B) {
		env := testEnv(map[string][]map[string]any{"items": rows})
		env.refSame("items")
		b.ResetTimer()
		for range b.N {
			env.refSame("items")
		}
	})

	b.Run("cache_miss", func(b *testing.B) {
		env := testEnv(map[string][]map[string]any{"items": rows})
		b.ResetTimer()
		for range b.N {
			env.refSame("items")
			env.clearOneCache()
		}
	})

	b.Run("parallel", func(b *testing.B) {
		env := testEnv(map[string][]map[string]any{"items": rows})
		env.refSame("items")
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				env.refSame("items")
			}
		})
	})
}

func BenchmarkRefPerm(b *testing.B) {
	b.Run("cache_hit", func(b *testing.B) {
		env := benchEnv(100)
		env.refPerm("items")
		b.ResetTimer()
		for range b.N {
			env.refPerm("items")
		}
	})

	b.Run("cache_miss", func(b *testing.B) {
		env := benchEnv(100)
		b.ResetTimer()
		for range b.N {
			env.refPerm("items")
			env.permCacheMutex.Lock()
			clear(env.permCache)
			env.permCacheMutex.Unlock()
		}
	})

	b.Run("parallel", func(b *testing.B) {
		env := benchEnv(100)
		env.refPerm("items")
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				env.refPerm("items")
			}
		})
	})
}

func BenchmarkRefDiff(b *testing.B) {
	cases := []struct {
		name string
		rows int
	}{
		{"rows_100", 100},
		{"rows_1000", 1000},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			env := benchEnv(tc.rows)
			count := 0
			b.ResetTimer()
			for range b.N {
				if count >= tc.rows {
					env.resetUniqIndex()
					count = 0
				}
				env.refDiff("items")
				count++
			}
		})
	}
}

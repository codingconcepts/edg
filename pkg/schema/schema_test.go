package schema

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tableNames(tables []Table) []string {
	names := make([]string, len(tables))
	for i, t := range tables {
		names[i] = t.Name
	}
	return names
}

func orderIndex(tables []Table) map[string]int {
	idx := make(map[string]int, len(tables))
	for i, t := range tables {
		idx[t.Name] = i
	}
	return idx
}

var errCyclicDependency = errors.New("cyclic dependency detected")

func TestSortTables(t *testing.T) {
	tests := []struct {
		name       string
		tables     []Table
		wantErr    error
		wantNames  []string
		wantBefore [][2]string
	}{
		{
			name:      "empty",
			tables:    nil,
			wantNames: nil,
		},
		{
			name: "no dependencies",
			tables: []Table{
				{Name: "c"},
				{Name: "a"},
				{Name: "b"},
			},
			wantNames: []string{"a", "b", "c"},
		},
		{
			// c -> b -> a  (a is root)
			name: "linear chain",
			tables: []Table{
				{Name: "c", Columns: []Column{{Name: "b_id", Ref: "b.id"}}},
				{Name: "a"},
				{Name: "b", Columns: []Column{{Name: "a_id", Ref: "a.id"}}},
			},
			wantNames:  []string{"a", "b", "c"},
			wantBefore: [][2]string{{"a", "b"}, {"b", "c"}},
		},
		{
			name: "self reference",
			tables: []Table{
				{Name: "categories", Columns: []Column{
					{Name: "id"},
					{Name: "parent_id", Ref: "categories.id"},
				}},
			},
			wantNames: []string{"categories"},
		},
		{
			//   root
			//  /    \
			// left  right
			//  \    /
			//   leaf
			name: "diamond",
			tables: []Table{
				{Name: "leaf", Columns: []Column{
					{Name: "left_id", Ref: "left.id"},
					{Name: "right_id", Ref: "right.id"},
				}},
				{Name: "left", Columns: []Column{{Name: "root_id", Ref: "root.id"}}},
				{Name: "right", Columns: []Column{{Name: "root_id", Ref: "root.id"}}},
				{Name: "root"},
			},
			wantNames: []string{"root", "left", "right", "leaf"},
			wantBefore: [][2]string{
				{"root", "left"},
				{"root", "right"},
				{"left", "leaf"},
				{"right", "leaf"},
			},
		},
		{
			name: "cyclic dependency",
			tables: []Table{
				{Name: "a", Columns: []Column{{Name: "b_id", Ref: "b.id"}}},
				{Name: "b", Columns: []Column{{Name: "a_id", Ref: "a.id"}}},
			},
			wantErr: errCyclicDependency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted, err := SortTables(tt.tables)

			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
				return
			}

			require.NoError(t, err)
			assert.ElementsMatch(t, tt.wantNames, tableNames(sorted))

			idx := orderIndex(sorted)
			for _, pair := range tt.wantBefore {
				assert.Less(t, idx[pair[0]], idx[pair[1]], "%s before %s", pair[0], pair[1])
			}
		})
	}
}

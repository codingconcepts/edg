package schema

import (
	"fmt"
	"strings"
)

// Table represents a database table with its columns.
type Table struct {
	Name       string
	CreateStmt string // native DDL from the database (when available)
	Columns    []Column
}

// Column represents a single column in a database table.
type Column struct {
	Name        string
	DataType    string
	IsNullable  bool
	Default     string
	IsPK        bool
	IsGenerated bool   // auto-increment, identity, serial, etc.
	Ref         string // FK reference in "table.column" format
	CheckMin    *int64 // lower bound from CHECK BETWEEN constraint
	CheckMax    *int64 // upper bound from CHECK BETWEEN constraint
}

// SortTables returns tables sorted in dependency order using topological sort
// (Kahn's algorithm). Parent tables come before their dependents.
func SortTables(tables []Table) ([]Table, error) {
	dependencies := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize inDegree for all tables.
	for _, table := range tables {
		inDegree[table.Name] = 0
	}

	// Populate dependencies and in-degree map.
	for _, table := range tables {
		for _, col := range table.Columns {
			if col.Ref == "" {
				continue
			}

			refTable := strings.Split(col.Ref, ".")[0]

			// Skip self-references (e.g. tree structures).
			if refTable == table.Name {
				continue
			}

			dependencies[refTable] = append(dependencies[refTable], table.Name)
			inDegree[table.Name]++
		}
	}

	// Topological Sort using Kahn's Algorithm.
	var sortedTables []Table
	queue := []string{}

	// Start with tables that have no incoming edges (in-degree 0).
	for table, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, table)
		}
	}

	for len(queue) > 0 {
		// Dequeue a table.
		currentTable := queue[0]
		queue = queue[1:]

		// Add to the sorted list.
		for _, table := range tables {
			if table.Name == currentTable {
				sortedTables = append(sortedTables, table)
				break
			}
		}

		// Decrease in-degree of dependent tables.
		for _, dependentTable := range dependencies[currentTable] {
			inDegree[dependentTable]--
			if inDegree[dependentTable] == 0 {
				queue = append(queue, dependentTable)
			}
		}
	}

	// Check for cyclic dependencies and fail if found.
	if len(sortedTables) != len(tables) {
		return nil, fmt.Errorf("cyclic dependency detected")
	}

	return sortedTables, nil
}

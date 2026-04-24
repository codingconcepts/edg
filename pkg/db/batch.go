package db

import "strings"

const batchSep = "\x1f"

func isGroupDelim(b byte) bool {
	switch b {
	case ',', ':', '(', ')', '[', ']', '{', '}', ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

func findBatchGroups(query string) [][2]int {
	var groups [][2]int
	n := len(query)
	inSingle, inDouble := false, false
	tokenStart := -1
	groupStart := -1
	hasSep := false

	for i := 0; i <= n; i++ {
		if i < n {
			b := query[i]

			if b == '\'' && !inDouble {
				inSingle = !inSingle
				if tokenStart < 0 {
					tokenStart = i
				}
				continue
			}
			if b == '"' && !inSingle {
				inDouble = !inDouble
				if tokenStart < 0 {
					tokenStart = i
				}
				continue
			}
			if inSingle || inDouble {
				continue
			}

			if b == '\x1f' {
				if tokenStart < 0 {
					tokenStart = i
				}
				if groupStart < 0 {
					groupStart = tokenStart
				}
				hasSep = true
				tokenStart = -1
				continue
			}

			if isGroupDelim(b) {
				if hasSep {
					groups = append(groups, [2]int{groupStart, i})
				}
				tokenStart = -1
				groupStart = -1
				hasSep = false
				continue
			}

			if tokenStart < 0 {
				tokenStart = i
			}
			continue
		}

		// End of string - flush any open group.
		if hasSep {
			groups = append(groups, [2]int{groupStart, n})
		}
	}

	return groups
}

// ExpandBatchQuery splits a query containing \x1f-separated value groups
// into individual queries, one per row. Returns the original query in a
// single-element slice when no \x1f is present.
func ExpandBatchQuery(query string) []string {
	if !strings.Contains(query, batchSep) {
		return []string{query}
	}

	matches := findBatchGroups(query)
	if len(matches) == 0 {
		return []string{query}
	}

	groups := make([][]string, len(matches))
	rowCount := 0
	for i, m := range matches {
		groups[i] = strings.Split(query[m[0]:m[1]], batchSep)
		if i == 0 {
			rowCount = len(groups[i])
		}
	}

	queries := make([]string, rowCount)
	for row := range rowCount {
		q := query
		for i := len(matches) - 1; i >= 0; i-- {
			q = q[:matches[i][0]] + groups[i][row] + q[matches[i][1]:]
		}
		queries[row] = q
	}

	return queries
}

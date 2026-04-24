package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandBatchQuery_NoSep(t *testing.T) {
	got := ExpandBatchQuery("INSERT INTO t (a) VALUES (1)")
	assert.Equal(t, []string{"INSERT INTO t (a) VALUES (1)"}, got)
}

func TestExpandBatchQuery_CQL(t *testing.T) {
	query := "INSERT INTO t (id, email) VALUES (uuid1\x1fuuid2\x1fuuid3, 'e1'\x1f'e2'\x1f'e3')"
	got := ExpandBatchQuery(query)

	assert.Equal(t, 3, len(got))
	assert.Equal(t, "INSERT INTO t (id, email) VALUES (uuid1, 'e1')", got[0])
	assert.Equal(t, "INSERT INTO t (id, email) VALUES (uuid2, 'e2')", got[1])
	assert.Equal(t, "INSERT INTO t (id, email) VALUES (uuid3, 'e3')", got[2])
}

func TestExpandBatchQuery_JSON(t *testing.T) {
	query := `{"insert": "t", "documents": [{"_id": "a1"` + "\x1f" + `"a2", "v": 1` + "\x1f" + `2}]}`
	got := ExpandBatchQuery(query)

	assert.Equal(t, 2, len(got))
	assert.Equal(t, `{"insert": "t", "documents": [{"_id": "a1", "v": 1}]}`, got[0])
	assert.Equal(t, `{"insert": "t", "documents": [{"_id": "a2", "v": 2}]}`, got[1])
}

func TestExpandBatchQuery_SepInsideQuotes(t *testing.T) {
	query := "INSERT INTO t (v) VALUES ('a\x1fb')"
	got := ExpandBatchQuery(query)

	assert.Equal(t, []string{"INSERT INTO t (v) VALUES ('a\x1fb')"}, got,
		"\\x1f inside single quotes must not split")
}

func TestExpandBatchQuery_SepInsideDoubleQuotes(t *testing.T) {
	query := `{"k": "a` + "\x1f" + `b"}`
	got := ExpandBatchQuery(query)

	assert.Equal(t, []string{`{"k": "a` + "\x1f" + `b"}`}, got,
		"\\x1f inside double quotes must not split")
}

func TestExpandBatchQuery_DelimInsideQuotes(t *testing.T) {
	query := "INSERT INTO t (v) VALUES ('(a,b)'\x1f'(c,d)')"
	got := ExpandBatchQuery(query)

	assert.Equal(t, 2, len(got))
	assert.Equal(t, "INSERT INTO t (v) VALUES ('(a,b)')", got[0])
	assert.Equal(t, "INSERT INTO t (v) VALUES ('(c,d)')", got[1])
}

func TestExpandBatchQuery_MixedQuoteTypes(t *testing.T) {
	query := `{"name": 'alice'` + "\x1f" + `'bob', "id": "x1"` + "\x1f" + `"x2"}`
	got := ExpandBatchQuery(query)

	assert.Equal(t, 2, len(got))
	assert.Equal(t, `{"name": 'alice', "id": "x1"}`, got[0])
	assert.Equal(t, `{"name": 'bob', "id": "x2"}`, got[1])
}

func TestExpandBatchQuery_SingleGroup(t *testing.T) {
	query := "SELECT * FROM t WHERE id IN (a\x1fb\x1fc)"
	got := ExpandBatchQuery(query)

	assert.Equal(t, 3, len(got))
	assert.Equal(t, "SELECT * FROM t WHERE id IN (a)", got[0])
	assert.Equal(t, "SELECT * FROM t WHERE id IN (b)", got[1])
	assert.Equal(t, "SELECT * FROM t WHERE id IN (c)", got[2])
}

func TestFindBatchGroups_NoSep(t *testing.T) {
	groups := findBatchGroups("INSERT INTO t (a) VALUES (1)")
	assert.Empty(t, groups)
}

func TestFindBatchGroups_Basic(t *testing.T) {
	query := "(a\x1fb, 'x'\x1f'y')"
	groups := findBatchGroups(query)

	assert.Equal(t, 2, len(groups))
	assert.Equal(t, "a\x1fb", query[groups[0][0]:groups[0][1]])
	assert.Equal(t, "'x'\x1f'y'", query[groups[1][0]:groups[1][1]])
}

func TestFindBatchGroups_SepInsideQuotes(t *testing.T) {
	query := "('a\x1fb')"
	groups := findBatchGroups(query)
	assert.Empty(t, groups, "\\x1f inside quotes should not produce a group")
}

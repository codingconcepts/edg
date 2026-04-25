package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildResult(t *testing.T) {
	tableMap := map[string]*Table{
		"a": {Name: "a"},
		"b": {Name: "b"},
		"c": {Name: "c"},
	}
	order := []string{"c", "a", "b"}

	result := buildResult(tableMap, order)
	assert.Equal(t, []string{"c", "a", "b"}, []string{result[0].Name, result[1].Name, result[2].Name})
}

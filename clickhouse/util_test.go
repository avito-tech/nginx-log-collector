package clickhouse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeUrl(t *testing.T) {
	table := []struct {
		inputDSN   string
		inputTable string
		expected   string
	}{
		{"http://host:333", "db.table", "http://host:333/?input_format_skip_unknown_fields=1&query=INSERT+INTO+db.table+FORMAT+JSONEachRow"},
		{"http://host:333/", "db.table", "http://host:333/?input_format_skip_unknown_fields=1&query=INSERT+INTO+db.table+FORMAT+JSONEachRow"},
	}

	for _, p := range table {
		url, err := MakeUrl(p.inputDSN, p.inputTable, true, 0)
		assert.Nil(t, err)
		assert.Equal(t, p.expected, url)
	}
}

package backlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseName(t *testing.T) {
	table := []struct {
		input    string
		expected string
	}{
		{"file", "file"},
		{"file.1.xx", "file.1"},
		{"file.xx", "file"},
		{"/foo/bar/file.xx", "/foo/bar/file"},
	}

	for _, p := range table {
		assert.Equal(t, p.expected, baseFileName(p.input))
	}
}

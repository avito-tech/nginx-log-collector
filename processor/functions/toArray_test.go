package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToArray(t *testing.T) {
	table := []struct {
		input    string
		expected string
	}{
		{"200 3300  4000", "[200,3300,4000]"},
		{"299.33 ", "[299.33]"},
		{"20 2020 ", "[20,2020]"},
	}

	for _, p := range table {
		callable := &toArray{}
		assert.Equal(t, p.expected, string(callable.Call(p.input)[0].Value))
	}
}

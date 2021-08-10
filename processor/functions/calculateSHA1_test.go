package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateSHA1(t *testing.T) {
	type testCase struct {
		input    string
		expected string
	}
	table := []testCase{
		{
			input:    "The quick brown fox jumps over the lazy dog",
			expected: `"2fd4e1c67a2d28fced849ee1bb76e7391b93eb12"`,
		},
		{
			input:    "",
			expected: `"da39a3ee5e6b4b0d3255bfef95601890afd80709"`,
		},
	}

	for _, p := range table {
		callable := &calculateSHA1{}
		v := callable.Call(p.input)
		assert.Equal(t, len(v), 1)

		assert.Equal(t, p.expected, string(v[0].Value))
	}
}

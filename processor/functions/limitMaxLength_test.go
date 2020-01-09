package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateMid(t *testing.T) {
	table := []struct {
		input     string
		maxLength int
		expected  string
	}{
		{"foobar2", 5, `"<...>"`},
		{"foobar2", 6, `"<...>2"`},
		{"foobar22", 7, `"f<...>2"`},
		{"foobar22", 6, `"<...>2"`},
		{"xxxxxjfkljffeyyyyy", 15, `"xxxxx<...>yyyyy"`},
		{"ok_message", 100, `"ok_message"`},
	}

	for _, p := range table {
		callable := &limitMaxLength{maxLength: p.maxLength}
		v := callable.Call(p.input)
		assert.Equal(t, len(v), 1)

		assert.Equal(t, p.expected, string(v[0].Value))
		assert.True(t, len(v)-2 <= p.maxLength)
	}
}

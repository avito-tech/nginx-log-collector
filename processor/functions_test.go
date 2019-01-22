package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateMid(t *testing.T) {
	table := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"foobar2", 5, `"<...>"`},
		{"foobar2", 6, `"<...>2"`},
		{"foobar22", 7, `"f<...>2"`},
		{"foobar22", 6, `"<...>2"`},
		{"xxxxxjfkljffeyyyyy", 15, `"xxxxx<...>yyyyy"`},
		{"ok_message", 100, `"ok_message"`},
	}

	for _, p := range table {
		v := limitMaxLength(p.input, p.maxLen)
		assert.Equal(t, p.expected, string(v))
		assert.True(t, len(v)-2 <= p.maxLen)
	}
}

func TestIpToUint32(t *testing.T) {
	table := []struct {
		input    string
		expected string
	}{
		{"2001:0db8:0000:0042:0000:8a2e:0370:7334", `"0"`},
		{"not ip", `"0"`},
		{"", `"0"`},
		{"127.0.0.1", `"2130706433"`},
		{"0.0.0.1", `"1"`},
		{"255.255.255.255", `"4294967295"`},
	}

	for _, p := range table {
		assert.Equal(t, p.expected, string(ipToUint32(p.input)))
	}
}

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
		assert.Equal(t, p.expected, string(toArray(p.input)))
	}
}

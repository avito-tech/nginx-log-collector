package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		callable := &ipToUint32{}
		assert.Equal(t, p.expected, string(callable.Call(p.input)[0].Value))
	}
}

package parser

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsing(t *testing.T) {
	table := []struct {
		inputFile    string
		expectedHost string
	}{
		{"error1", "g.avito.ru"},
		{"errorphp", "www.avito.ru"},
	}

	for _, p := range table {
		data, err := ioutil.ReadFile("./testdata/" + p.inputFile)
		assert.Nil(t, err)

		out := make(map[string]interface{})
		err = NginxErrorLogMessage(data, out)
		assert.Nil(t, err)
		assert.Equal(t, p.expectedHost, out["host"])
	}
}

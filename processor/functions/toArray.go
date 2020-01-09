package functions

import (
	"bytes"
	"strconv"
	"strings"
)

type toArray struct{}

func (f *toArray) Call(value string) FunctionResult {
	b := bytes.Buffer{}
	needComma := false
	result := FunctionPartialResult{}

	b.WriteByte('[')
	for _, n := range strings.Split(value, " ") {
		if n != "" {
			if needComma {
				b.WriteByte(',')
			}
			if _, err := strconv.ParseFloat(n, 32); err == nil {
				b.WriteString(n)
				needComma = true
			}
		}
	}
	b.WriteByte(']')

	result.Value = b.Bytes()
	return FunctionResult{result}
}

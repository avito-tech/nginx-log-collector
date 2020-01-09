package functions

import (
	"bytes"
)

const MIDDLE = "<...>"

type limitMaxLength struct {
	maxLength int
}

func (f *limitMaxLength) Call(value string) FunctionResult {
	// len(MIDDLE) = 5
	b := bytes.Buffer{}
	result := FunctionPartialResult{}

	b.WriteByte('"')
	if f.maxLength < 5 || len(value) <= f.maxLength {
		// string is too short - no need to do anything
		b.WriteString(value)
	} else {
		rawMaxLen := f.maxLength - 5

		leftLen := rawMaxLen / 2
		rightLen := rawMaxLen - leftLen

		b.WriteString(value[:leftLen])
		b.WriteString(MIDDLE)
		b.WriteString(value[len(value)-rightLen:])
	}
	b.WriteByte('"')

	result.Value = b.Bytes()
	return FunctionResult{result}
}

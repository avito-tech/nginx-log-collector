package functions

import (
	"bytes"
	"crypto/sha1"
	"fmt"
)

type calculateSHA1 struct {
	StoreTo *string `yaml:"store_to,omitempty"`
}

func (f *calculateSHA1) Call(value string) FunctionResult {
	b := bytes.Buffer{}
	result := FunctionPartialResult{DstFieldName: f.StoreTo}

	b.WriteByte('"')
	hash := sha1.Sum([]byte(value))
	hashPrintable := fmt.Sprintf("%x", hash)
	b.WriteString(hashPrintable)
	b.WriteByte('"')

	result.Value = b.Bytes()
	return FunctionResult{result}
}

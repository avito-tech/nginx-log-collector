package functions

import (
	"bytes"
	"strings"
)

type splitAndStore struct {
	Delimiter string         `yaml:"delimiter"`
	StoreTo   map[string]int `yaml:"store_to"`
}

func (f *splitAndStore) Call(value string) FunctionResult {
	result := make(FunctionResult, 0, len(f.StoreTo))

	parts := strings.Split(value, f.Delimiter)
	partsMap := make(map[int]string, len(parts))
	for i, part := range parts {
		partsMap[i] = part
	}

	for fieldName, index := range f.StoreTo {
		b := bytes.Buffer{}
		dstFieldName := fieldName
		valuePart := partsMap[index] // empty string if there is no split part with such index

		b.WriteByte('"')
		b.WriteString(valuePart)
		b.WriteByte('"')

		result = append(result, FunctionPartialResult{
			Value:        b.Bytes(),
			DstFieldName: &dstFieldName,
		})
	}

	return result
}

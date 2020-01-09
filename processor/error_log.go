package processor

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"nginx-log-collector/parser"
	"nginx-log-collector/processor/functions"
)

type ErrorLogConverter struct {
	transformers []transformer
}

func NewErrorLogConverter(transformerMap functions.FunctionSignatureMap) (*ErrorLogConverter, error) {
	transformers, err := parseTransformersMap(transformerMap)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create error_log converter")
	}
	return &ErrorLogConverter{
		transformers: transformers,
	}, nil
}

func (e *ErrorLogConverter) Convert(msg []byte, hostname string) ([]byte, error) {
	now := time.Now()
	v := make(map[string]interface{})
	err := parser.NginxErrorLogMessage(msg, v)
	if err != nil {
		return nil, err
	}
	v["hostname"] = hostname
	v[dateField] = now.Format(dateFmt)
	v[dateTimeField] = now.Format(dateTimeFmt)

	e.transform(v)
	return json.Marshal(v)
}

func (e *ErrorLogConverter) transform(v map[string]interface{}) {
	for _, tr := range e.transformers {
		value, found := v[tr.fieldNameSrc]
		if !found {
			continue
		}
		strValue, ok := value.(string)
		if !ok {
			continue
		}

		callResult := tr.function.Call(strValue)
		for _, chunk := range callResult {
			var fieldName string
			if chunk.DstFieldName != nil {
				fieldName = *chunk.DstFieldName
			} else {
				fieldName = tr.fieldNameSrc
			}

			v[fieldName] = chunk.Value
		}
	}
}

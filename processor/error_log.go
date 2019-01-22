package processor

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"nginx-log-collector/parser"
)

type ErrorLogConverter struct {
	transformers []Transformer
}

func NewErrorLogConverter(transformerMap map[string]string) (*ErrorLogConverter, error) {
	transformers, err := NewTransformers(transformerMap)
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
		value, found := v[tr.FieldName]
		if !found {
			continue
		}
		strValue, ok := value.(string)
		if !ok {
			continue
		}
		v[tr.FieldName] = tr.Fn(strValue)
	}
}

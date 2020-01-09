package processor

import (
	"time"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	"github.com/valyala/fastjson"
	"nginx-log-collector/processor/functions"
)

type AccessLogConverter struct {
	transformers []transformer
}

func NewAccessLogConverter(transformerMap functions.FunctionSignatureMap) (*AccessLogConverter, error) {
	transformers, err := parseTransformersMap(transformerMap)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create access_log converter")
	}
	return &AccessLogConverter{
		transformers: transformers,
	}, nil
}

func (a *AccessLogConverter) Convert(msg []byte, _ string) ([]byte, error) {
	if err := fastjson.ValidateBytes(msg); err != nil {
		return nil, errors.Wrap(err, "invalid json")
	}
	val, err := jsonparser.GetUnsafeString(msg, dateTimeField)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get datetime field")
	}
	t, err := time.Parse(time.RFC3339, val) // TODO use ParseInLocation?
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse datetime field")
	}
	t = t.In(time.Local)

	msg, err = jsonparser.Set(msg, []byte(`"`+t.Format(dateTimeFmt)+`"`), dateTimeField)
	if err != nil {
		return nil, errors.Wrap(err, "unable to set datetime field")
	}
	msg, err = jsonparser.Set(msg, []byte(`"`+t.Format(dateFmt)+`"`), dateField)
	if err != nil {
		return nil, errors.Wrap(err, "unable to set date field")
	}

	return a.transform(msg)
}

func (a *AccessLogConverter) transform(msg []byte) ([]byte, error) {
	for _, tr := range a.transformers {
		val, err := jsonparser.GetUnsafeString(msg, tr.fieldNameSrc)
		if err != nil {
			continue
		}

		callResult := tr.function.Call(val)
		for _, chunk := range callResult {
			var fieldName string
			if chunk.DstFieldName != nil {
				fieldName = *chunk.DstFieldName
			} else {
				fieldName = tr.fieldNameSrc
			}

			msg, err = jsonparser.Set(msg, chunk.Value, fieldName)
			if err != nil {
				return nil, errors.Wrap(err, "unable to set field")
			}
		}
	}
	return msg, nil
}

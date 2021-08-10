package processor

import (
	"time"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	"github.com/valyala/fastjson"

	"nginx-log-collector/processor/functions"
	"nginx-log-collector/utils"
)

type AccessLogConverter struct {
	transformers []transformer
}

var datetimeTransformers = []*utils.DatetimeTransformer{
	{
		"2006-01-02T15:04:05.000000000Z07:00",
		"2006-01-02T15:04:05.000000000",
		time.Local,
	},
	{
		time.RFC3339,
		dateTimeFmt,
		time.Local,
	},
	{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05.999999999",
		time.UTC,
	},
	{
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05.999999",
		time.UTC,
	},
	{
		"2006-01-02T15:04:05.999",
		"2006-01-02T15:04:05.999",
		time.UTC,
	},
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

	// try to transform source datetime to naive datetime and date strings using several datetime formats
	parsed, transformer, err := utils.TryDatetimeFormats(val, datetimeTransformers)
	if err != nil {
		return nil, err
	}

	msg, err = jsonparser.Set(msg, []byte(`"`+parsed.Format(transformer.FormatDst)+`"`), dateTimeField)
	if err != nil {
		return nil, errors.Wrap(err, "unable to set datetime field")
	}
	msg, err = jsonparser.Set(msg, []byte(`"`+parsed.Format(dateFmt)+`"`), dateField)
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

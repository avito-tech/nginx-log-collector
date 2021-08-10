package utils

import (
	"strings"
	"time"

	"github.com/pkg/errors"
)

// DatetimeTransformer contains info needed to transform
// datetime string from one format to another
type DatetimeTransformer struct {
	FormatSrc string
	FormatDst string
	Location  *time.Location
}

func TryDatetimeFormats(datetime string, transformers []*DatetimeTransformer) (parsed time.Time, matched *DatetimeTransformer, err error) {
	errorMessages := make([]string, 0, len(transformers))
	for _, transformer := range transformers {
		t, err := time.Parse(transformer.FormatSrc, datetime)
		if err != nil {
			errorMessages = append(errorMessages, err.Error())
		} else {
			return t.In(transformer.Location), transformer, nil
		}
	}

	if len(errorMessages) > 0 {
		err = errors.Wrap(errors.New(strings.Join(errorMessages, "\n")), "unable to parse datetime field")
	} else {
		err = errors.New("Empty transformers list")
	}

	return time.Time{}, nil, err
}

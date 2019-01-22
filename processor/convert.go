package processor

import (
	"fmt"

	"nginx-log-collector/config"
)

const (
	dateTimeField = "event_datetime"
	dateField     = "event_date"
	dateTimeFmt   = "2006-01-02 15:04:05"
	dateFmt       = "2006-01-02"
)

type Converter interface {
	Convert([]byte, string) ([]byte, error)
}

func NewConverter(cfg config.CollectedLog) (Converter, error) {
	switch cfg.Format {
	case "access":
		return NewAccessLogConverter(cfg.Transformers)
	case "error":
		return NewErrorLogConverter(cfg.Transformers)
	default:
		return nil, fmt.Errorf("unknown log format: %s", cfg.Format)
	}
}

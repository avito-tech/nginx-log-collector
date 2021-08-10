package processor

import (
	"nginx-log-collector/processor/functions"

	"github.com/pkg/errors"
)

type transformer struct {
	fieldNameSrc string
	function     functions.Callable
}

func parseTransformersMap(transformersMap functions.FunctionSignatureMap) ([]transformer, error) {
	transformers := make([]transformer, 0, len(transformersMap))

	for fieldNameSrc, functionSignature := range transformersMap {
		if callable, err := functions.Dispatch(functionSignature); err != nil {
			return nil, errors.Wrapf(err, "unable to convert expression for field %s to function", fieldNameSrc)
		} else {
			transformers = append(transformers, transformer{
				fieldNameSrc: fieldNameSrc,
				function:     callable,
			})
		}
	}

	return transformers, nil
}

package processor

import "github.com/pkg/errors"

type TransformFunc func(string) []byte

type Transformer struct {
	FieldName string
	Fn        TransformFunc
}

func NewTransformers(transformerMap map[string]string) ([]Transformer, error) {
	transformers := make([]Transformer, 0)
	for fieldName, funcName := range transformerMap {
		fn, err := strToFn(funcName)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to convert %s to function", funcName)

		}
		transformers = append(transformers, Transformer{
			FieldName: fieldName,
			Fn:        fn,
		})
	}
	return transformers, nil

}

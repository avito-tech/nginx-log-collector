package functions

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

func Dispatch(signature FunctionSignature) (Callable, error) {
	if len(signature) != 1 {
		return nil, fmt.Errorf("function signature must have exactly one key, got %d instead", len(signature))
	}

	var (
		callable      Callable
		err           error
		functionName  string
		functionExtra interface{}
	)

	for key, value := range signature {
		functionName = key
		functionExtra = value
		break
	}

	if functionName == "ipToUint32" {
		callable, err = validateIpToUint32(functionExtra)
	} else if functionName == "limitMaxLength" {
		callable, err = validateLimitMaxLength(functionExtra)
	} else if functionName == "splitAndStore" {
		callable, err = validateSplitAndStore(functionExtra)
	} else if functionName == "toArray" {
		callable, err = validateToArray(functionExtra)
	} else {
		err = fmt.Errorf("unknown function name: %s", functionName)
	}

	return callable, err
}

func expectValueEmpty(data interface{}) error {
	if data == nil {
		return nil
	} else {
		switch dataType := data.(type) {
		case string:
			if value := data.(string); value != "" {
				return fmt.Errorf("expects empty value, got \"%s\" instead", value)
			} else {
				return nil
			}
		default:
			return fmt.Errorf("expects empty value, got type %T instead", dataType)
		}
	}
}

func expectValuePositiveInt(data interface{}) error {
	switch dataType := data.(type) {
	case int:
		if value := data.(int); value <= 0 {
			return fmt.Errorf("expects positive integer value, got %d instead", value)
		} else {
			return nil
		}
	default:
		return fmt.Errorf("expects positive integer value, got %T instead", dataType)
	}
}

func validateIpToUint32(data interface{}) (*ipToUint32, error) {
	if err := expectValueEmpty(data); err != nil {
		return nil, fmt.Errorf("ipToUint32 %s", err.Error())
	} else {
		return &ipToUint32{}, nil
	}
}

func validateLimitMaxLength(data interface{}) (*limitMaxLength, error) {
	if err := expectValuePositiveInt(data); err != nil {
		return nil, fmt.Errorf("limitMaxLength %s", err.Error())
	} else {
		return &limitMaxLength{maxLength: data.(int)}, nil
	}
}

func validateSplitAndStore(data interface{}) (*splitAndStore, error) {
	var result splitAndStore

	if out, err := yaml.Marshal(data); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(out, &result); err != nil {
		return nil, err
	} else {
		return &result, nil
	}
}

func validateToArray(data interface{}) (*toArray, error) {
	if err := expectValueEmpty(data); err != nil {
		return nil, fmt.Errorf("ipToUint32 %s", err.Error())
	} else {
		return &toArray{}, nil
	}
}

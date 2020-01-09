package functions

// Callable is basic interface of any function
type Callable interface {
	Call(string) FunctionResult
}

// functionPartialResult is the part of the result of a single function call; there can be several values returned
type FunctionPartialResult struct {
	Value        []byte  // returned value
	DstFieldName *string // name of the field returned value will be stored to (optional)
}

// functionResult represents all the values returned by function
type FunctionResult []FunctionPartialResult

// FunctionSignature is signature of a single function as it is represented in config
type FunctionSignature map[string]interface{}

// FunctionSignatureMap is configuration of all functions as it is represented in config
type FunctionSignatureMap map[string]FunctionSignature

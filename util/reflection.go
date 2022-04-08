package util

import (
	"reflect"
)

// InvokeFunc will magically invoke a function with the matching name
func InvokeFunc(any interface{}, name string, args ...interface{}) []reflect.Value {
	inputs := make([]reflect.Value, len(args))
	for i := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	return reflect.ValueOf(any).MethodByName(name).Call(inputs)
}

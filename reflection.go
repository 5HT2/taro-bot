package main

import (
	"log"
	"reflect"
)

// InvokeFunc will magically invoke a function
func InvokeFunc(any interface{}, name string, args ...interface{}) []reflect.Value {
	inputs := make([]reflect.Value, len(args))
	for i := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	return reflect.ValueOf(any).MethodByName(name).Call(inputs)
}

// CallStringFunc will invoke a function with a []string return value
func CallStringFunc(any interface{}, name string) []string {
	result := InvokeFunc(any, name)
	if len(result) > 0 {
		str, ok := result[0].Interface().([]string)
		if !ok {
			log.Printf("Error with CallStringFunc: %s %v", name, any)
			return []string{}
		}
		return str
	}
	return []string{}
}

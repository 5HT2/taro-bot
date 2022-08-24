package util

import (
	"encoding/json"
	"log"
	"reflect"
	"runtime/debug"
	"time"
)

//
// Additions to Go's std-lib's builtin
//

type retryFunction func() ([]byte, error)

func LogPanic() {
	if x := recover(); x != nil {
		// recovering from a panic; x contains whatever was passed to panic()
		log.Printf("panic: %s\n", debug.Stack())
	}
}

// RetryFunc will re-try fn by n number of times, in addition to one regular try
func RetryFunc(fn retryFunction, n int, delayMs time.Duration) ([]byte, error) {
	if n < 0 {
		n = 0
	}

	for n > 0 {
		b, err := fn()
		if err == nil {
			return b, err
		}
		n--

		// Wait before re-trying, if we have re-tries left.
		if n > 0 && delayMs > 0 {
			time.Sleep(delayMs * time.Millisecond)
		}
	}

	return fn()
}

func NewInterface(typ reflect.Type, data []byte) (interface{}, error) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		dst := reflect.New(typ).Elem()
		err := json.Unmarshal(data, dst.Addr().Interface())
		return dst.Addr().Interface(), err
	} else {
		dst := reflect.New(typ).Elem()
		err := json.Unmarshal(data, dst.Addr().Interface())
		return dst.Interface(), err
	}
}

// SliceContains will return if slice s contains e
func SliceContains[T comparable](s []T, e T) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// SliceRemove will remove m from s
func SliceRemove[T comparable](s []T, m T) []T {
	ns := make([]T, 0)
	for _, in := range s {
		if in != m {
			ns = append(ns, in)
		}
	}
	return ns
}

// SliceReverse will reverse the order of s
func SliceReverse[S ~[]T, T any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

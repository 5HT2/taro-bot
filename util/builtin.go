package util

import (
	"encoding/json"
	"log"
	"reflect"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
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

// SliceRemoveIndex will remove index i from s
func SliceRemoveIndex[T comparable](s []T, i int) []T {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

// SliceReverse will reverse the order of s
func SliceReverse[S ~[]T, T any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// SliceSortAlphanumeric will sort a string slice alphanumerically
func SliceSortAlphanumeric[S ~[]T, T string](s S) {
	sort.Slice(s, func(i, j int) bool {
		// check if we have numbers, sort them accordingly
		if z, err := strconv.Atoi(string(s[i])); err == nil {
			if y, err := strconv.Atoi(string(s[j])); err == nil {
				return y < z
			}
			// if we get only one number, always say its greater than letter
			return true
		}
		// compare letters normally
		return s[j] > s[i]
	})
}

// SlicesCondition will return if all values of []T match condition c
func SlicesCondition[T comparable](s []T, c func(s T) bool) bool {
	for _, v := range s {
		if !c(v) {
			return false
		}
	}
	return true
}

// SliceJoin will join any slice based on the property or value that c returns
func SliceJoin[T any](s []T, sep string, c func(s T) *string) string {
	ns := make([]string, 0)
	for _, v := range s {
		if n := c(v); n != nil {
			ns = append(ns, *n)
		}
	}
	return strings.Join(ns, sep)
}

// SliceEqual returns true if all bytes of a and b are the same
func SliceEqual[T comparable](a []T, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

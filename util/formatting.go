package util

import (
	"strconv"
	"strings"
)

// JoinInt64Slice will join i with sep
func JoinInt64Slice(i []int64, sep string, prefix string, suffix string) string {
	elems := make([]string, 0)
	for _, e := range i {
		elems = append(elems, prefix+strconv.FormatInt(e, 10)+suffix)
	}
	return strings.Join(elems, sep)
}

package util

//
// Additions to Go's std-lib's builtin
//

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

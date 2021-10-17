package main

import (
	"regexp"
	"strconv"
)

var (
	pingRegex, _      = regexp.Compile("<@!?[0-9]+>")
	parsePingRegex, _ = regexp.Compile("[<@!>]")
)

// ParseInt64Arg will return an int64 from s, or -1 and an error
func ParseInt64Arg(a []string, pos int) (int64, *TaroError) {
	s, argErr := checkArgExists(a, pos, "ParseInt64Arg")
	if argErr != nil {
		return -1, argErr
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return -1, GenericSyntaxError("ParseInt64Arg", s, "expected int64")
	}
	return i, nil
}

// ParseUserArg will return the ID of a mentioned user, or -1 and an error
func ParseUserArg(a []string, pos int) (int64, *TaroError) {
	s, argErr := checkArgExists(a, pos, "ParseUserArg")
	if argErr != nil {
		return -1, argErr
	}

	ok := pingRegex.MatchString(s)
	if ok {
		id := parsePingRegex.ReplaceAllString(s, "")
		i, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return -1, GenericSyntaxError("ParseUserArg", s, err.Error())
		}
		return i, nil
	}
	return -1, GenericSyntaxError("ParseUserArg", s, "expected user mention")
}

// ParseStringArg will return the selected string, or "" with an error
func ParseStringArg(a []string, pos int) (string, *TaroError) {
	s, argErr := checkArgExists(a, pos, "ParseStringArg")
	if argErr != nil {
		return "", argErr
	}
	return s, nil
}

func checkArgExists(a []string, pos int, fn string) (s string, err *TaroError) {
	pos -= 1 // we want to increment this so ParseGenericArg(c.args, 1) will return the first arg
	// prevent panic if dev made an error
	if pos < 0 {
		pos = 1
	}

	if len(a) > pos {
		return a[pos], nil
	}

	// the position in the command the user is giving
	pos += 1
	return "", GenericError(fn, "getting arg "+strconv.Itoa(pos), "arg is missing")
}

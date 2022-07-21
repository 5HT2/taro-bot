package util

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"strconv"
	"strings"
)

var (
	printer = message.NewPrinter(language.English)
)

// JoinInt64Slice will join i with sep
func JoinInt64Slice(i []int64, sep string, prefix string, suffix string) string {
	elems := make([]string, 0)
	for _, e := range i {
		elems = append(elems, prefix+strconv.FormatInt(e, 10)+suffix)
	}
	return strings.Join(elems, sep)
}

// GetUserMention will return a formatted user mention from an id
func GetUserMention(id int64) string {
	return "<@!" + strconv.FormatInt(id, 10) + ">"
}

// FormattedNum will insert commas as necessary in large numbers
func FormattedNum(num int64) string {
	return printer.Sprintf("%d", num)
}

package util

import (
	"fmt"
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

// FormattedTime will turn seconds into a pretty time representation
func FormattedTime(secondsIn int64) string {
	hours := secondsIn / 3600
	minutes := (secondsIn / 60) - (60 * hours)
	seconds := secondsIn % 60

	units := make([]string, 0)
	if hours != 0 {
		units = append(units, JoinInt64AndStr(hours, "hour"))
	}
	if minutes != 0 {
		units = append(units, JoinInt64AndStr(minutes, "minute"))
	}
	if seconds != 0 || (hours == 0 && minutes == 0) {
		units = append(units, JoinInt64AndStr(seconds, "second"))
	}

	return strings.Join(units, ", ")
}

// FormattedNum will insert commas as necessary in large numbers
func FormattedNum(num int64) string {
	return printer.Sprintf("%d", num)
}

// JoinInt64AndStr will join and add a plural s to the str if int is not 1, for example, "0 hours", "1 hour", "2 hours".
func JoinInt64AndStr(int int64, str string) string {
	plural := "s"
	if int == 1 {
		plural = ""
	}
	return fmt.Sprintf("%s %s%s", FormattedNum(int), str, plural)
}

// JoinIntAndStr is a wrapper for JoinInt64AndStr
func JoinIntAndStr(int int, str string) string {
	return JoinInt64AndStr(int64(int), str)
}

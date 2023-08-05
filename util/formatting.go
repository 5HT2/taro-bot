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

// HeadLinesLimit will take the first amount of lines that fit into the X char limit
// If the string does not consist of split lines, instead just fit the first amount of chars.
func HeadLinesLimit(s string, limit int) string {
	lines := strings.Split(s, "\n")

	// We don't have any lines to work with - just get the first chars in s that fit into limit
	if len(lines) <= 1 {
		if limit > len(s) { // Don't slice out of bounds
			limit = len(s)
		}

		return s[:limit]
	}

	reached := 0
	headedLines := make([]string, 0)
	for _, line := range lines {
		if len(line)+reached <= limit {
			reached += len(line)
			reached += 1 // for newline
			headedLines = append(headedLines, line)
		} else {
			break
		}
	}

	return strings.Join(headedLines, "\n")
}

// TailLinesLimit will take the last amount of lines that fit into the X char limit.
// If the string does not consist of split lines, instead just fit the last amount of chars.
func TailLinesLimit(s string, limit int) string {
	lines := strings.Split(s, "\n")

	// We don't have any lines to work with - just get the last chars in s that fit into limit
	if len(lines) <= 1 {
		last := len(s) - limit

		if last < 0 { // Don't slice out of bounds
			last = 0
		}

		return s[last:]
	}

	// Reverse the order of the lines, we want to Tail them
	SliceReverse(lines)

	reached := 0
	tailedLines := make([]string, 0)
	for _, line := range lines {
		if len(line)+reached <= limit {
			reached += len(line)
			reached += 1 // for newline
			tailedLines = append(tailedLines, line)
		} else {
			break
		}
	}

	// Undo the reverse sort
	SliceReverse(tailedLines)

	return strings.Join(tailedLines, "\n")
}

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

package util

import (
	"github.com/diamondburned/arikawa/v3/discord"
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

// GetUserMention will return a formatted user mention from an id
func GetUserMention(id int64) string {
	return "<@!" + strconv.FormatInt(id, 10) + ">"
}

// GuildIDStr will return a discord.GuildID as a string
func GuildIDStr(id discord.GuildID) string {
	return strconv.FormatUint(uint64(id), 10)
}

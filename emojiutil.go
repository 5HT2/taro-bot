package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"net/url"
	"strings"
)

var (
	escapedCheckmark = "%E2%9C%85"
	escapedStar      = "%E2%AD%90"
	checkmarkEmoji   = discord.APIEmoji(escapedCheckmark)
)

func GuildTopicVoteEmoji(guild GuildConfig) (string, error) {
	e := guild.TopicVoteEmoji

	if len(e) == 0 {
		guild.TopicVoteEmoji = escapedCheckmark
		SetGuildConfig(guild)
	} else {
		e = strings.TrimSuffix(e, "a:")
	}

	return FormatEncodedEmoji(e)
}

func GuildTopicVoteApiEmoji(guild GuildConfig) (discord.APIEmoji, error) {
	e := guild.TopicVoteEmoji

	if len(e) == 0 {
		guild.TopicVoteEmoji = escapedCheckmark
		SetGuildConfig(guild)
	}

	return ConfigEmojiAsApiEmoji(e)
}

func ConfigEmojiAsApiEmoji(e string) (discord.APIEmoji, error) {
	e = strings.TrimPrefix(e, "a:")

	str, err := url.QueryUnescape(e)
	if err != nil {
		return checkmarkEmoji, err
	}

	return discord.APIEmoji(str), nil
}

func ApiEmojiAsConfig(e *discord.APIEmoji, animated bool) string {
	if e == nil {
		return ApiEmojiAsConfig(&checkmarkEmoji, animated)
	}

	a := ":"
	if animated {
		a = "a:"
	}

	str := e.PathString()
	if strings.Contains(str, ":") {
		str = a + str
	}

	return str
}

func ApiEmojiAsFormatted(e *discord.APIEmoji, animated bool) (string, error) {
	return FormatEncodedEmoji(ApiEmojiAsConfig(e, animated))
}

func FormatEncodedEmoji(e string) (string, error) {
	split := strings.Split(e, ":")
	if len(split) > 1 {
		return "<" + e + ">", nil
	}

	return url.QueryUnescape(e)
}

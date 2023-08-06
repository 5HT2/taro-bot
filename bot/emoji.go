package bot

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"net/url"
	"strings"
)

var (
	escapedWarning = "%E2%9A%A0%EF%B8%8F"
	warningEmoji   = discord.APIEmoji(escapedWarning)
)

func ConfigEmojiAsApiEmoji(e string) (discord.APIEmoji, error) {
	e = strings.TrimPrefix(e, "a:")

	str, err := url.QueryUnescape(e)
	if err != nil {
		return warningEmoji, err
	}

	return discord.APIEmoji(str), nil
}

func ApiEmojiAsConfig(e *discord.APIEmoji, animated bool) string {
	if e == nil {
		return ApiEmojiAsConfig(&warningEmoji, animated)
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

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

// EmojiApiAsConfig will convert a discord.APIEmoji to our own config format, to preserve the animated attribute
func EmojiApiAsConfig(e *discord.APIEmoji, animated bool) string {
	if e == nil {
		return EmojiApiAsConfig(&warningEmoji, animated)
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

// EmojiConfigAsApi will convert our own emoji format back to a discord.APIEmoji
func EmojiConfigAsApi(e string) (discord.APIEmoji, error) {
	e = strings.TrimPrefix(e, "a:")

	str, err := url.QueryUnescape(e)
	if err != nil {
		return warningEmoji, err
	}

	return discord.APIEmoji(str), nil
}

// EmojiApiFormatted will format a discord.APIEmoji for display in a message
func EmojiApiFormatted(e *discord.APIEmoji, animated bool) (string, error) {
	return EmojiConfigFormatted(EmojiApiAsConfig(e, animated))
}

// EmojiConfigFormatted will format a taro config emoji for display in a message
func EmojiConfigFormatted(e string) (string, error) {
	split := strings.Split(e, ":")
	if len(split) > 1 {
		return "<" + e + ">", nil // Discord custom emojis
	}

	return url.QueryUnescape(e) // Unicode emojis
}

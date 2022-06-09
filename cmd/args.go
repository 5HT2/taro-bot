package cmd

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/forPelevin/gomoji"
	"regexp"
	"strconv"
	"strings"
)

var (
	UrlRegex          = regexp.MustCompile(`https?://(www\.)?[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&/=]*)`)
	emojiUrlRegex     = regexp.MustCompile(`^http(s)?://cdn\.discordapp\.com/emojis/([0-9]+)`)
	discordEmojiRegex = regexp.MustCompile("<(a|):([A-z0-9_]+):([0-9]+)>")
	pingRegex         = regexp.MustCompile("<@!?[0-9]+>")
	channelRegex      = regexp.MustCompile("<#[0-9]+>")
	mentionFormats    = regexp.MustCompile("[<@!#&>]")
)

// ParseAllArgs will return the combined existing args
func ParseAllArgs(a []string) (string, *bot.Error) {
	s := strings.Join(a, " ")
	if len(a) == 0 {
		return "", bot.GenericSyntaxError("ParseAllArgs", "nothing", "expected arguments!")
	}
	return s, nil
}

// ParseInt64Arg will return an int64 from s, or -1 and an error
func ParseInt64Arg(a []string, pos int) (int64, *bot.Error) {
	s, argErr := checkArgExists(a, pos, "ParseInt64Arg")
	if argErr != nil {
		return -1, argErr
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return -1, bot.GenericSyntaxError("ParseInt64Arg", s, "expected int64")
	}
	return i, nil
}

// ParseUserArg will return the ID of a mentioned user, or -1 and an error
func ParseUserArg(a []string, pos int) (int64, *bot.Error) {
	s, argErr := checkArgExists(a, pos, "ParseUserArg")
	if argErr != nil {
		return -1, argErr
	}

	ok := pingRegex.MatchString(s)
	if ok {
		id := mentionFormats.ReplaceAllString(s, "")
		i, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return -1, bot.GenericSyntaxError("ParseUserArg", s, err.Error())
		}
		return i, nil
	}
	return -1, bot.GenericSyntaxError("ParseUserArg", s, "expected user mention")
}

// ParseUrlArg will return a URL, or "" and an error
func ParseUrlArg(a []string, pos int) (string, *bot.Error) {
	s, argErr := checkArgExists(a, pos, "ParseUrlArg")
	if argErr != nil {
		return "", argErr
	}

	ok := UrlRegex.MatchString(s)
	if ok {
		return s, nil
	}
	return "", bot.GenericSyntaxError("ParseUrlArg", s, "expected http or https url")
}

// ParseEmojiArg will return a discord.APIEmoji and the animated status, or nil, false and an error
func ParseEmojiArg(a []string, pos int, allowOmit bool) (*discord.APIEmoji, bool, *bot.Error) {
	s, argErr := checkArgExists(a, pos, "ParseEmojiArg")
	if argErr != nil {
		if allowOmit {
			return nil, false, nil
		}
		return nil, false, argErr
	}

	if e := gomoji.CollectAll(s); len(e) == 1 {
		emoji := discord.APIEmoji(e[0].Character)
		return &emoji, false, nil
	}

	emoji := discordEmojiRegex.FindStringSubmatch(s)
	if len(emoji) < 4 {
		return nil, false, bot.GenericSyntaxError("ParseEmojiArg", s, "expected full emoji")
	}

	id, err := strconv.Atoi(emoji[3])
	if err != nil {
		return nil, false, bot.GenericSyntaxError("ParseEmojiArg", s, "expected int")
	}

	apiEmoji := discord.NewCustomEmoji(discord.EmojiID(id), emoji[2])
	animated := emoji[1] == "a"
	return &apiEmoji, animated, nil
}

// ParseEmojiIdArg will return an emoji ID, or -1 and an error
func ParseEmojiIdArg(a []string, pos int) (int64, *bot.Error) {
	s, argErr := checkArgExists(a, pos, "ParseEmojiArg")
	if argErr != nil {
		return -1, argErr
	}

	emoji := discordEmojiRegex.FindStringSubmatch(s)
	if len(emoji) < 4 {
		return -1, bot.GenericSyntaxError("ParseEmojiIdArg", s, "expected full emoji")
	}

	id, err := strconv.ParseInt(emoji[3], 10, 64)
	if err != nil {
		return -1, bot.GenericSyntaxError("ParseEmojiIdArg", s, "expected int")
	}

	return id, nil
}

// ParseEmojiUrlArg will return an emoji ID, or -1 and an error
func ParseEmojiUrlArg(a []string, pos int) (int64, *bot.Error) {
	s, argErr := checkArgExists(a, pos, "ParseEmojiUrlArg")
	if argErr != nil {
		return -1, argErr
	}

	emoji := emojiUrlRegex.FindStringSubmatch(s)
	if len(emoji) < 3 {
		return -1, bot.GenericSyntaxError("ParseEmojiUrlArg", s, "couldn't parse emoji url")
	}

	if id, err := strconv.ParseInt(emoji[2], 10, 64); err != nil {
		return -1, bot.GenericSyntaxError("ParseEmojiUrlArg", s, err.Error())
	} else {
		return id, nil
	}
}

// ParseChannelSliceArg will return the IDs of the mentioned channels, or nil and an error
func ParseChannelSliceArg(a []string, pos1 int, pos2 int) ([]int64, *bot.Error) {
	if pos2 == -1 {
		pos2 = len(a)
	}

	s, err := getArgRange(a, pos1, pos2, "ParseChannelSliceArg")
	if err != nil {
		return nil, err
	}

	elems := make([]int64, 0)
	for _, c := range s {
		if id, err := validateChannelArg(c, "ParseChannelStringArg"); err != nil {
			return nil, err
		} else {
			elems = append(elems, id)
		}
	}

	return elems, nil
}

// ParseChannelArg will return the ID of a mentioned channel, or -1 and an error
func ParseChannelArg(a []string, pos int) (int64, *bot.Error) {
	s, argErr := checkArgExists(a, pos, "ParseChannelArg")
	if argErr != nil {
		return -1, argErr
	}

	return validateChannelArg(s, "ParseChannelArg")
}

// ParseStringArg will return the selected string, or "" with an error
func ParseStringArg(a []string, pos int, toLower bool) (string, *bot.Error) {
	s, argErr := checkArgExists(a, pos, "ParseStringArg")
	if argErr != nil {
		return "", argErr
	}
	if toLower {
		return strings.ToLower(s), nil
	}
	return s, nil
}

// ParseBoolArg will return a bool (True / true / 1), or false with an error
func ParseBoolArg(a []string, pos int) (bool, *bot.Error) {
	s, argErr := checkArgExists(a, pos, "ParseStringArg")
	if argErr != nil {
		return false, argErr
	}

	switch strings.ToLower(s) {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, bot.GenericSyntaxError("ParseBoolArg", s, "expected boolean")
	}
}

// validateChannelArg will return a valid channel mention, or nil and an error if it is invalid
func validateChannelArg(s string, fn string) (int64, *bot.Error) {
	ok := channelRegex.MatchString(s)
	if ok {
		id := mentionFormats.ReplaceAllString(s, "")
		i, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return -1, bot.GenericSyntaxError(fn, s, err.Error())
		}
		return i, nil
	}
	return -1, bot.GenericSyntaxError(fn, s, "expected channel mention")
}

// getArgRange will return the elements in a from pos1 to pos2, or nil and an error if the range is invalid
func getArgRange(a []string, pos1 int, pos2 int, fn string) (s []string, err *bot.Error) {
	elems := make([]string, 0)

	for pos := pos1; pos <= pos2; pos++ {
		if e, argErr := checkArgExists(a, pos, fn); argErr != nil {
			return nil, argErr
		} else {
			elems = append(elems, e)
		}
	}

	return elems, nil
}

// checkArgExists will return a[pos - 1] if said index exists, otherwise it will return an error
func checkArgExists(a []string, pos int, fn string) (s string, err *bot.Error) {
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
	return "", bot.GenericError(fn, "getting arg "+strconv.Itoa(pos), "arg is missing")
}

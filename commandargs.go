package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"regexp"
	"strconv"
	"strings"
)

var (
	urlRegex          = regexp.MustCompile(`https?://(www\.)?[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&/=]*)`)
	emojiRegex        = regexp.MustCompile(`([\x{2000}-\x{3300}]|[\x{D83C}\x{D000}-\x{D83C}\x{DFFF}]|[\x{D83D}\x{D000}-\x{D83D}\x{DFFF}]|[\x{D83E}\x{D000}-\x{D83E}\x{DFFF}])+`)
	discordEmojiRegex = regexp.MustCompile("<(a|):([A-z0-9_]+):([0-9]+)>")
	pingRegex         = regexp.MustCompile("<@!?[0-9]+>")
	channelRegex      = regexp.MustCompile("<#[0-9]+>")
	mentionFormats    = regexp.MustCompile("[<@!#&>]")
)

// ParseAllArgs will return the combined existing args
func ParseAllArgs(a []string) (string, *TaroError) {
	s := strings.Join(a, " ")
	if len(a) == 0 {
		return "", GenericSyntaxError("ParseAllArgs", "nothing", "expected arguments!")
	}
	return s, nil
}

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

// ParseEmojiArg will return a discord.APIEmoji and the animated status, or nil, false and an error
func ParseEmojiArg(a []string, pos int, allowOmit bool) (*discord.APIEmoji, bool, *TaroError) {
	s, argErr := checkArgExists(a, pos, "ParseEmojiArg")
	if argErr != nil {
		if allowOmit {
			return nil, false, nil
		}
		return nil, false, argErr
	}

	if emojiRegex.MatchString(s) {
		emoji := discord.APIEmoji(s)
		return &emoji, false, nil
	}

	emoji := discordEmojiRegex.FindStringSubmatch(s)
	if len(emoji) < 3 {
		return nil, false, GenericSyntaxError("ParseEmojiArg", s, "expected full emoji")
	}

	id, err := strconv.Atoi(emoji[3])
	if err != nil {
		return nil, false, GenericSyntaxError("ParseEmojiArg", s, "expected int")
	}

	apiEmoji := discord.NewCustomEmoji(discord.EmojiID(id), emoji[2])
	animated := emoji[1] == "a"
	return &apiEmoji, animated, nil
}

// ParseUserArg will return the ID of a mentioned user, or -1 and an error
func ParseUserArg(a []string, pos int) (int64, *TaroError) {
	s, argErr := checkArgExists(a, pos, "ParseUserArg")
	if argErr != nil {
		return -1, argErr
	}

	ok := pingRegex.MatchString(s)
	if ok {
		id := mentionFormats.ReplaceAllString(s, "")
		i, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return -1, GenericSyntaxError("ParseUserArg", s, err.Error())
		}
		return i, nil
	}
	return -1, GenericSyntaxError("ParseUserArg", s, "expected user mention")
}

// ParseUrlArg will return a URL, or "" and an error
func ParseUrlArg(a []string, pos int) (string, *TaroError) {
	s, argErr := checkArgExists(a, pos, "ParseUrlArg")
	if argErr != nil {
		return "", argErr
	}

	ok := urlRegex.MatchString(s)
	if ok {
		return s, nil
	}
	return "", GenericSyntaxError("ParseUrlArg", s, "expected http or https url")
}

// ParseChannelSliceArg will return the IDs of the mentioned channels, or nil and an error
func ParseChannelSliceArg(a []string, pos1 int, pos2 int) ([]int64, *TaroError) {
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
func ParseChannelArg(a []string, pos int) (int64, *TaroError) {
	s, argErr := checkArgExists(a, pos, "ParseChannelArg")
	if argErr != nil {
		return -1, argErr
	}

	return validateChannelArg(s, "ParseChannelArg")
}

// ParseStringArg will return the selected string, or "" with an error
func ParseStringArg(a []string, pos int, toLower bool) (string, *TaroError) {
	s, argErr := checkArgExists(a, pos, "ParseStringArg")
	if argErr != nil {
		return "", argErr
	}
	if toLower {
		return strings.ToLower(s), nil
	}
	return s, nil
}

// validateChannelArg will return a valid channel mention, or nil and an error if it is invalid
func validateChannelArg(s string, fn string) (int64, *TaroError) {
	ok := channelRegex.MatchString(s)
	if ok {
		id := mentionFormats.ReplaceAllString(s, "")
		i, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return -1, GenericSyntaxError(fn, s, err.Error())
		}
		return i, nil
	}
	return -1, GenericSyntaxError(fn, s, "expected channel mention")
}

// getArgRange will return the elements in a from pos1 to pos2, or nil and an error if the range is invalid
func getArgRange(a []string, pos1 int, pos2 int, fn string) (s []string, err *TaroError) {
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

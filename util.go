package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"image/color"
	"io/ioutil"
	"net/http"
)

var (
	timeFormat = "Jan 02 2006 15:04:05 MST"
)

func PrintEmojiUpdate(emoji discord.Emoji) {
	// TODO: make dynamic, and make it only work on actual emoji updates
	var id discord.ChannelID = 893249218003750952
	embed := discord.Embed{
		Author:    CreateEmbedAuthor(emoji.User),
		Title:     "Emoji created/deleted",
		Thumbnail: &discord.EmbedThumbnail{URL: emoji.EmojiURL()},
	}
	_, _ = SendCustomEmbed(id, embed)
}

func RequestUrl(url string, method string) ([]byte, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func ConvertColorToInt32(c color.RGBA) int32 {
	return int32((uint32(c.R) << 16) | (uint32(c.G) << 8) | (uint32(c.B) << 0))
}

func ParseHexColorFast(s string) (c color.RGBA, err error) {
	c.A = 0xff

	if s[0] != '#' {
		return c, GenericError("ParseHexColorFast", "parsing \""+s+"\"", "missing #")
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = SyntaxError(s)
		return 0
	}

	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	case 4:
		c.R = hexToByte(s[1]) * 17
		c.G = hexToByte(s[2]) * 17
		c.B = hexToByte(s[3]) * 17
	default:
		err = SyntaxError(s)
	}
	return
}

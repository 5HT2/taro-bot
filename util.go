package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"strconv"
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

func CreateEmbedAuthor(user discord.User) *discord.EmbedAuthor {
	url := "https://cdn.discordapp.com/avatars/" + strconv.FormatUint(uint64(user.ID), 10) + "/" + user.Avatar + ".png?size=2048"
	return &discord.EmbedAuthor{Name: user.Username, Icon: url}
}

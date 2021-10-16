package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"log"
	"strconv"
)

var (
	successColor discord.Color = 0x3cde5a
	errorColor   discord.Color = 0xde413c
	defaultColor discord.Color = 0x493cde
)

func SendCustomEmbed(c discord.ChannelID, embed discord.Embed) (*discord.Message, error) {
	msg, err := discordClient.SendEmbeds(
		c,
		embed,
	)
	if err != nil {
		log.Printf("Error sending embed: %v", err)
	}
	return msg, err
}

func SendErrorEmbed(c Command, err error) {
	_, _ = SendEmbed(c, "Error running `"+c.name+"`", err.Error(), errorColor)
}

func SendEmbed(c Command, title string, description string, color discord.Color) (*discord.Message, error) {
	msg, err := discordClient.SendEmbeds(
		c.e.ChannelID,
		makeEmbed(title, description, color),
	)
	if err != nil {
		log.Printf("Error sending embed: %v", err)
	}
	return msg, err
}

func SendMessage(c Command, content string) (*discord.Message, error) {
	msg, err := discordClient.SendMessage(
		c.e.ChannelID,
		content,
	)
	if err != nil {
		log.Printf("Error sending embed: %v", err)
	}
	return msg, err
}

func CreateEmbedAuthor(user discord.User) *discord.EmbedAuthor {
	url := "https://cdn.discordapp.com/avatars/" + strconv.FormatUint(uint64(user.ID), 10) + "/" + user.Avatar + ".png?size=2048"
	return &discord.EmbedAuthor{Name: user.Username, Icon: url}
}

func makeEmbed(title string, description string, color discord.Color) discord.Embed {
	return discord.Embed{
		Title:       title,
		Description: description,
		Color:       color,
	}
}

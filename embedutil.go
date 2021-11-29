package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"log"
	"strconv"
)

var (
	successColor   discord.Color = 0x3cde5a
	errorColor     discord.Color = 0xde413c
	defaultColor   discord.Color = 0x493cde
	starboardColor discord.Color = 0xffac33
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

func SendExternalErrorEmbed(c discord.ChannelID, cmdName string, err error) (*discord.Message, error) {
	return SendCustomEmbed(c, makeEmbed("Error running `"+cmdName+"`", err.Error(), errorColor))
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

func CreateEmbedAuthor(member discord.Member) *discord.EmbedAuthor {
	name := member.Nick
	if len(name) == 0 {
		name = member.User.Username
	}

	url := "https://cdn.discordapp.com/avatars/" + strconv.FormatUint(uint64(member.User.ID), 10) + "/" + member.User.Avatar + ".png?size=2048"
	return &discord.EmbedAuthor{Name: name, Icon: url}
}

func CreateMessageLink(guild int64, message *discord.Message, jump bool) string {
	guildID := strconv.FormatInt(guild, 10)
	channel := strconv.FormatInt(int64(message.ChannelID), 10)
	messageID := strconv.FormatInt(int64(message.ID), 10)
	link := "https://discord.com/channels/" + guildID + "/" + channel + "/" + messageID

	if jump {
		return "[Jump!](" + link + ")"
	}
	return link
}

func makeEmbed(title string, description string, color discord.Color) discord.Embed {
	return discord.Embed{
		Title:       title,
		Description: description,
		Color:       color,
	}
}

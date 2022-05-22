package cmd

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"strconv"
)

func SendCustomEmbed(c discord.ChannelID, embed discord.Embed) (*discord.Message, error) {
	msg, err := bot.Client.SendEmbeds(
		c,
		embed,
	)
	if err != nil {
		log.Printf("Error sending embed: %v (%v)", err, embed)
	}
	return msg, err
}

func SendCustomMessage(c discord.ChannelID, content string) (*discord.Message, error) {
	msg, err := bot.Client.SendMessage(
		c,
		content,
	)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
	return msg, err
}

func SendExternalErrorEmbed(c discord.ChannelID, cmdName string, err error) (*discord.Message, error) {
	return SendCustomEmbed(c, MakeEmbed("Error running `"+cmdName+"`", err.Error(), bot.ErrorColor))
}

func SendErrorEmbed(c bot.Command, err error) {
	_, _ = SendEmbed(c.E, "Error running `"+c.Name+"`", err.Error(), bot.ErrorColor)
}

func SendEmbed(e *gateway.MessageCreateEvent, title string, description string, color discord.Color) (*discord.Message, error) {
	embed := MakeEmbed(title, description, color)
	msg, err := bot.Client.SendEmbeds(
		e.ChannelID,
		embed,
	)
	if err != nil {
		log.Printf("Error sending embed: %v (%v)", err, embed)
	}
	return msg, err
}

func SendMessage(e *gateway.MessageCreateEvent, content string) (*discord.Message, error) {
	msg, err := bot.Client.SendMessage(
		e.ChannelID,
		content,
	)
	if err != nil {
		log.Printf("Error sending message: %v", err)
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

func MakeEmbed(title string, description string, color discord.Color) discord.Embed {
	return discord.Embed{
		Title:       title,
		Description: description,
		Color:       color,
	}
}

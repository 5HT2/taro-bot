package cmd

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"strconv"
	"strings"
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

func SendMessageEmbedSafe(c discord.ChannelID, content string, embed *discord.Embed) (*discord.Message, error) {
	if embed != nil {
		return bot.Client.SendMessage(c, content, *embed)
	}

	return bot.Client.SendMessage(c, content)
}

func CreateEmbedAuthor(member discord.Member) *discord.EmbedAuthor {
	name := member.Nick
	if len(name) == 0 {
		name = member.User.Username
	}

	return &discord.EmbedAuthor{
		Name: name,
		Icon: fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png?size=2048", member.User.ID.String(), member.User.Avatar),
	}
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

func GetEmbedAttachmentAndContent(msg discord.Message) (string, *discord.EmbedImage) {
	// Try to find a URL in the message content
	description := msg.Content
	url := UrlRegex.MatchString(msg.Content)

	// Set the embed image to the URL and try to find the first attached image in the message attachments
	var image *discord.EmbedImage = nil
	for _, attachment := range msg.Attachments {
		if strings.HasPrefix(attachment.ContentType, "image/") {
			image = &discord.EmbedImage{URL: attachment.URL}
			url = false // Don't remove URL in embed if we found an image attachment (eg, twitter link + image attachment)
			break
		}
	}

	// If we found only a URL (no other text) in the message content, and the found URL has an image extension, and we didn't find an attached image
	// Set the description to nothing and set the image to the found URL
	if url && FileExtMatches(ImageExtensions, msg.Content) {
		description = ""
		image = &discord.EmbedImage{URL: msg.Content}
	}

	return description, image
}

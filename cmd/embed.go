package cmd

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
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

func SendEmbed(e *gateway.MessageCreateEvent, title, description string, color discord.Color) (*discord.Message, error) {
	return SendEmbedFooter(e, title, description, "", color)
}

func SendEmbedFooter(e *gateway.MessageCreateEvent, title, description, footer string, color discord.Color) (*discord.Message, error) {
	embed := MakeEmbed(title, description, color)
	embed.Footer = &discord.EmbedFooter{Text: footer}
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

func SendDirectMessageEmbedSafe(id discord.UserID, content string, embed *discord.Embed) (*discord.Message, error) {
	channel, err := bot.Client.CreatePrivateChannel(id)
	if err != nil {
		return nil, err
	}

	return SendMessageEmbedSafe(channel.ID, content, embed)
}

func SendDirectMessage(userID discord.UserID, contents string) (*discord.Message, error) {
	channel, err := bot.Client.CreatePrivateChannel(userID)
	if err != nil {
		return nil, err
	}

	message, err := SendCustomMessage(channel.ID, contents)

	return message, err
}

func CreateEmbedAuthor(member discord.Member) *discord.EmbedAuthor {
	name := member.Nick
	if len(name) == 0 {
		name = member.User.Username
	}

	return &discord.EmbedAuthor{
		Name: name,
		Icon: fmt.Sprintf("%s?size=2048", member.User.AvatarURL()),
	}
}

func CreateEmbedAuthorUser(user discord.User) *discord.EmbedAuthor {
	return &discord.EmbedAuthor{
		Name: util.FormattedUserTag(user),
		Icon: fmt.Sprintf("%s?size=2048", user.AvatarURL()),
	}
}

func CreateMessageLink(guild int64, message *discord.Message, jump, dm bool) string {
	link := fmt.Sprintf("https://discord.com/channels/%v/%v/%v", guild, message.ChannelID, message.ID)
	if dm {
		link = fmt.Sprintf("https://discord.com/channels/@me/%v/%v", message.ChannelID, message.ID)
	}

	if jump {
		return "[Jump!](" + link + ")"
	}
	return link
}

func CreateMessageLinkInt64(guild, message, channel int64, jump, dm bool) string {
	link := fmt.Sprintf("https://discord.com/channels/%v/%v/%v", guild, channel, message)
	if dm {
		link = fmt.Sprintf("https://discord.com/channels/@me/%v/%v", channel, message)
	}

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
	if url {
		urlMatch := UrlRegex.FindStringSubmatch(msg.Content)

		if len(urlMatch) > 0 && FileExtMatches(ImageExtensions, urlMatch[0]) { // extract URL and make sure we have one
			// remove the URL from the message content, keep other content
			description = strings.ReplaceAll(msg.Content, urlMatch[0], "")
			image = &discord.EmbedImage{URL: urlMatch[0]}
		}
	}

	return description, image
}

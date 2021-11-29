package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"strconv"
	"strings"
)

type StarboardConfig struct {
	ID          int64              `json:"id"`                     // guild ID
	Channel     int64              `json:"channel,omitempty"`      // channel post ID
	NsfwChannel int64              `json:"nsfw_channel,omitempty"` // nsfw post channel ID
	Messages    []StarboardMessage `json:"messages,omitempty"`
	Threshold   int64              `json:"threshold,omitempty"`
}

type StarboardMessage struct {
	ID     int64   `json:"id"`      // the original message ID
	Author int64   `json:"author"`  // the original author ID
	PostID int64   `json:"message"` // the starboard post message ID
	IsNsfw bool    `json:"nsfw"`    // if the original message was made in an NSFW channel
	Stars  []int64 `json:"stars"`   // list of added user IDs
}

var (
	stars3Emoji = "⭐"
	stars5Emoji = "🌟"
	stars6Emoji = "💫"
	stars9Emoji = "✨"
)

func StarboardReactionHandler(e *gateway.MessageReactionAddEvent) {
	guild := GetStarboardConfig(int64(e.GuildID))

	if guild.Threshold == 0 {
		guild.Threshold = 3
		SetStarboardConfig(guild)
	}

	// Not starred by a guild member
	if e.Member == nil {
		log.Printf("Not a guild member\n")
		return
	}

	// Not a star
	if e.Emoji.APIString().PathString() != escapedStar {
		log.Printf("Not a star emoji\n")
		return
	}

	msg, err := discordClient.Message(e.ChannelID, e.MessageID)
	if err != nil {
		if *debug {
			log.Printf("Error retrieving starred message: %v\n", err)
		}
		return
	}
	channel, err := discordClient.Channel(e.ChannelID)
	if err != nil {
		if *debug {
			log.Printf("Error retrieving starred message channel: %v\n", err)
		}
		return
	}

	var sMsg *StarboardMessage = nil
	newPost := true
	sMsgPos := -1

	for i, m := range guild.Messages {
		if m.ID == int64(msg.ID) {
			sMsg = &m
			newPost = false
			sMsgPos = i
			break
		}
	}

	if newPost {
		sMsg = &StarboardMessage{ID: int64(msg.ID), PostID: 0, Author: int64(msg.Author.ID), IsNsfw: channel.NSFW, Stars: make([]int64, 0)}
	}

	// Channel to send starboard message to
	cID := guild.Channel
	if sMsg.IsNsfw == true {
		cID = guild.NsfwChannel
	}

	// Channel hasn't been set
	if cID == 0 {
		log.Printf("Channel ID is 0\n")
		return
	}

	// Get post channel and ensure it exists
	postChannel, err := discordClient.Channel(discord.ChannelID(cID))
	if err != nil {
		log.Printf("Couldn't get post channel\n")
		return
	}

	// When adding a new star, ensure star user is not the same as author
	// And also check if they've already been added
	sUserID := int64(e.Member.User.ID)
	if sMsg.Author != sUserID && !Int64SliceContains(sMsg.Stars, sUserID) {
		sMsg.Stars = append(sMsg.Stars, sUserID)
	}
	log.Printf("sUserID: %v\nsMsg:%v\n", sUserID, sMsg)

	stars := len(sMsg.Stars)

	// Check if message reactions are larger than our cached ones
	for _, reaction := range msg.Reactions {
		if reaction.Emoji.APIString().PathString() == escapedStar {
			if reaction.Count > stars {
				stars = reaction.Count
			}
			break
		}
	}

	// Not enough stars in sMsg to make post
	if int64(stars) < guild.Threshold {
		log.Printf("Not enough stars: %v\n", sMsg.Stars)
		return
	}

	content := getEmoji(stars) + " **" + strconv.Itoa(stars) + "** <#" + strconv.FormatInt(int64(msg.ChannelID), 10) + ">"

	// Attempt to get existing message, and make a new one if it isn't there
	pMsg, err := discordClient.Message(postChannel.ID, discord.MessageID(sMsg.PostID))
	if err != nil {
		log.Printf("Couldn't get pMsg %v\n", err)

		// Construct new starboard post if it couldn't retrieve an existing one

		// Try to find a URL in the message content
		description := msg.Content
		url := urlRegex.MatchString(msg.Content)

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
		if url && FileExtMatches(imageExts, msg.Content) {
			description = ""
			image = &discord.EmbedImage{URL: msg.Content}
		}

		member, err := discordClient.Member(discord.GuildID(guild.ID), discord.UserID(sMsg.Author))
		if err != nil {
			log.Printf("Couldn't get member %v\n", err)
			return
		}

		author := CreateEmbedAuthor(*member)
		field := discord.EmbedField{Name: "Source", Value: CreateMessageLink(guild.ID, msg, true)}
		footer := discord.EmbedFooter{Text: strconv.FormatInt(sMsg.Author, 10)}
		embed := discord.Embed{
			Description: description,
			Author:      author,
			Fields:      []discord.EmbedField{field},
			Footer:      &footer,
			Timestamp:   msg.Timestamp,
			Color:       starboardColor,
			Image:       image,
		}

		log.Printf("Embed image: %v\n", embed.Image)
		log.Printf("Author: %v\n", author)
		log.Printf("Member: %v\n", member)

		msg, err = discordClient.SendMessage(postChannel.ID, content, embed)
		if err != nil {
			log.Printf("Error sending starboard post: %v\n", err)
		} else {
			sMsg.PostID = int64(msg.ID)
		}
	} else {
		// Edit the post if it exists
		_, err = discordClient.EditMessage(postChannel.ID, discord.MessageID(sMsg.PostID), content, pMsg.Embeds...)
		if err != nil {
			log.Printf("Error updating starboard post: %v\n", err)
		}
	}

	// Now that we have updated the stars and starboard post ID, save it in the config
	if sMsgPos >= 0 {
		guild.Messages[sMsgPos] = *sMsg
	} else {
		guild.Messages = append(guild.Messages, *sMsg)
	}
	SetStarboardConfig(guild)
}

func getEmoji(stars int) (emoji string) {
	switch stars {
	case 0, 1, 2, 3, 4:
		emoji = stars3Emoji
	case 5:
		emoji = stars5Emoji
	case 6, 7, 8:
		emoji = stars6Emoji
	default:
		emoji = stars9Emoji
	}

	return emoji
}

package main

import (
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"strconv"
)

type ActiveTopicVote struct {
	Message int64  `json:"message"`
	Author  int64  `json:"author"`
	Topic   string `json:"topic"`
}

func TopicReactionHandler(e *gateway.MessageReactionAddEvent) {
	guild := GetGuildConfig(int64(e.GuildID))

	if reactionMatchesActiveVote(e, guild) {
		if guild.TopicVoteThreshold == 0 {
			guild.TopicVoteThreshold = 4
			SetGuildConfig(guild)
		}

		message, err := discordClient.Message(e.ChannelID, e.MessageID)
		if err != nil {
			return
		}

		emoji, err := GuildTopicVoteApiEmoji(guild)
		if err != nil {
			return
		}

		for _, reaction := range message.Reactions {
			if reaction.Emoji.APIString() == emoji {
				if int64(reaction.Count) >= guild.TopicVoteThreshold {
					vote := removeActiveVote(e, guild)
					channel, err := discordClient.Channel(e.ChannelID)
					if err != nil {
						return
					}

					oldTopic := "No previous topic set"
					if len(channel.Topic) > 0 {
						oldTopic = "\nOld topic was \"" + channel.Topic + "\""
					}

					embed := discord.Embed{
						Title: "New channel topic!",
						Description: "The topic is now **" + vote.Topic + "**, suggested by <@" +
							strconv.FormatInt(vote.Author, 10) + ">!",
						Footer: &discord.EmbedFooter{Text: oldTopic},
						Color:  successColor,
					}

					data := api.ModifyChannelData{Topic: option.NewNullableString(vote.Topic)}
					if err = discordClient.ModifyChannel(e.ChannelID, data); err != nil {
						_, _ = SendExternalErrorEmbed(e.ChannelID, "TopicReactionHandler", err)
					} else {
						_, _ = SendCustomEmbed(e.ChannelID, embed)
					}
				}
				break
			}
		}
	}
}

func reactionMatchesActiveVote(e *gateway.MessageReactionAddEvent, guild GuildConfig) bool {
	found := false
	for _, vote := range guild.ActiveTopicVotes {
		if int64(e.MessageID) == vote.Message {
			found = true
			break
		}
	}
	return found
}

func removeActiveVote(e *gateway.MessageReactionAddEvent, guild GuildConfig) ActiveTopicVote {
	oldVotes := make([]ActiveTopicVote, 0)
	var removedVote ActiveTopicVote
	message := int64(e.MessageID)

	for _, vote := range guild.ActiveTopicVotes {
		if message != vote.Message {
			oldVotes = append(oldVotes, vote)
		} else {
			removedVote = vote
		}
	}

	guild.ActiveTopicVotes = oldVotes
	SetGuildConfig(guild)

	return removedVote
}

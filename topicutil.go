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
	reactionMatchesActiveVote := false
	GuildContext(e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
		// Find an activeTopicVote that matches `e`'s reaction
		for _, vote := range g.ActiveTopicVotes {
			if int64(e.MessageID) == vote.Message {
				reactionMatchesActiveVote = true
				break
			}
		}
		return g, "TopicReactionHandler: check reaction emoji"
	})

	if reactionMatchesActiveVote {
		// TODO: Honestly, why are we doing this?
		GuildContext(e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
			if g.TopicVoteThreshold == 0 {
				g.TopicVoteThreshold = 3
			}
			return g, "TopicReactionHandler: check TopicVoteThreshold"
		})

		message, err := discordClient.Message(e.ChannelID, e.MessageID)
		if err != nil {
			return
		}

		emoji, err := GuildTopicVoteApiEmoji(e.GuildID)
		if err != nil {
			return
		}

		for _, reaction := range message.Reactions {
			if reaction.Emoji.APIString() == emoji {
				offset := 0
				if reaction.Me {
					offset = 1
				}

				meetsThreshold := false
				GuildContext(e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
					meetsThreshold = int64(reaction.Count-offset) >= g.TopicVoteThreshold
					return g, "TopicReactionHandler: check meetsThreshold"
				})

				if meetsThreshold {
					vote := removeActiveVote(e)
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

func removeActiveVote(e *gateway.MessageReactionAddEvent) ActiveTopicVote {
	oldVotes := make([]ActiveTopicVote, 0)
	var removedVote ActiveTopicVote
	message := int64(e.MessageID)

	GuildContext(e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
		for _, vote := range g.ActiveTopicVotes {
			if message != vote.Message {
				oldVotes = append(oldVotes, vote)
			} else {
				removedVote = vote
			}
		}

		g.ActiveTopicVotes = oldVotes
		return g, "removeActiveVote"
	})

	return removedVote
}

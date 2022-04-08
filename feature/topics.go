package feature

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"strconv"
)

func TopicReactionHandler(e *gateway.MessageReactionAddEvent) {
	defer util.LogPanic()

	reactionMatchesActiveVote := false
	bot.GuildContext(e.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		// Find an activeTopicVote that matches `e`'s reaction
		for _, vote := range g.ActiveTopicVotes {
			if int64(e.MessageID) == vote.Message {
				reactionMatchesActiveVote = true
				break
			}
		}

		// While we're here, make sure the vote threshold isn't the default
		if g.TopicVoteThreshold == 0 {
			g.TopicVoteThreshold = 3
		}
		return g, "TopicReactionHandler: check reaction emoji"
	})

	if reactionMatchesActiveVote {
		message, err := bot.Client.Message(e.ChannelID, e.MessageID)
		if err != nil {
			return
		}

		emoji, err := util.GuildTopicVoteApiEmoji(e.GuildID)
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
				bot.GuildContext(e.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
					meetsThreshold = int64(reaction.Count-offset) >= g.TopicVoteThreshold
					return g, "TopicReactionHandler: check meetsThreshold"
				})

				if meetsThreshold {
					vote := removeActiveVote(e)
					channel, err := bot.Client.Channel(e.ChannelID)
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
						Color:  bot.SuccessColor,
					}

					data := api.ModifyChannelData{Topic: option.NewNullableString(vote.Topic)}
					if err = bot.Client.ModifyChannel(e.ChannelID, data); err != nil {
						_, _ = cmd.SendExternalErrorEmbed(e.ChannelID, "TopicReactionHandler", err)
					} else {
						_, _ = cmd.SendCustomEmbed(e.ChannelID, embed)
					}
				}
				break
			}
		}
	}
}

func removeActiveVote(e *gateway.MessageReactionAddEvent) bot.ActiveTopicVote {
	oldVotes := make([]bot.ActiveTopicVote, 0)
	var removedVote bot.ActiveTopicVote
	message := int64(e.MessageID)

	bot.GuildContext(e.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
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

package main

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"reflect"
	"strconv"
	"strings"
)

var (
	escapedCheckmark = "%E2%9C%85"
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	return &plugins.Plugin{
		Name:        "Suggest Topic",
		Description: "Allow suggesting a topic for the current channel",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          TopicConfigCommand,
			FnName:      "TopicConfigCommand",
			Name:        "topicconfig",
			Aliases:     []string{"topiccfg"},
			Description: "Configure allowed topic channels",
			GuildOnly:   true,
		}, {
			Fn:          TopicCommand,
			FnName:      "TopicCommand",
			Name:        "topic",
			Description: "Suggest a new topic for the current channel",
			GuildOnly:   true,
		}},
		Responses: []bot.ResponseInfo{},
		Handlers: []bot.HandlerInfo{{
			Fn:     TopicReactionHandler,
			FnName: "TopicReactionHandler",
			FnType: reflect.TypeOf(func(*gateway.MessageReactionAddEvent) {}),
		}},
	}
}

func TopicConfigCommand(c bot.Command) error {
	if err := cmd.HasPermission(c, cmd.PermChannels); err != nil {
		return err
	}

	channels := []int64{int64(c.E.ChannelID)}

	if argChannels, err := cmd.ParseChannelSliceArg(c.Args, 2, -1); err == nil && len(argChannels) != 0 {
		channels = argChannels
	}
	channelsStr := util.JoinInt64Slice(channels, ", ", "<#", ">")

	arg1, err := cmd.ParseStringArg(c.Args, 1, true)
	if err != nil {
		arg1 = "help"
		err = nil
	}

	switch arg1 {
	case "enable":
		bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
			for _, channel := range channels {
				if !util.SliceContains(g.EnabledTopicChannels, channel) {
					g.EnabledTopicChannels = append(g.EnabledTopicChannels, channel)
				}
			}
			return g, "TopicConfigCommand: topic enable"
		})
		_, err := cmd.SendEmbed(c.E, "Configure Topics", "✅ Added "+channelsStr+" to the allowed topic channels", bot.SuccessColor)
		return err
	case "disable":
		bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
			for _, channel := range channels {
				if util.SliceContains(g.EnabledTopicChannels, channel) {
					g.EnabledTopicChannels = util.SliceRemove(g.EnabledTopicChannels, channel)
				}
			}
			return g, "TopicConfigCommand: topic disable"
		})
		_, err := cmd.SendEmbed(c.E, "Configure Topics", "⛔ Removed "+channelsStr+" from the allowed topic channels", bot.ErrorColor)
		return err
	case "emoji":
		arg2, animated, err3 := cmd.ParseEmojiArg(c.Args, 2, true)
		if err3 != nil {
			return err3
		}

		if arg2 == nil {
			if emoji, err := topicVoteEmoji(c.E.GuildID); err != nil {
				return err
			} else {
				_, err = cmd.SendEmbed(c.E, "Current Topic Vote Emoji:", emoji, bot.DefaultColor)
				return err
			}
		} else {
			configEmoji := bot.EmojiApiAsConfig(arg2, animated)
			emoji, err := bot.EmojiConfigFormatted(configEmoji)
			if err != nil {
				return err
			}

			bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
				g.TopicVoteEmoji = configEmoji
				return g, "TopicConfigCommand: update TopicVoteEmoji"
			})

			_, err = cmd.SendEmbed(c.E, "Set Topic Vote Emoji To:", emoji, bot.SuccessColor)
			return err
		}
	case "threshold":
		arg2, err2 := cmd.ParseInt64Arg(c.Args, 2)
		if err2 != nil {
			return err2
		}

		bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
			if arg2 <= 0 {
				arg2 = 3
			}

			g.TopicVoteThreshold = arg2
			return g, "TopicConfigCommand: update topic vote threshold"
		})

		_, err := cmd.SendEmbed(c.E, "Set Topic Vote Threshold To:", strconv.FormatInt(arg2, 10), bot.SuccessColor)
		return err
	case "list":
		noTopicChan := false
		formattedChannels := ""

		bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
			formattedChannels = util.JoinInt64Slice(g.EnabledTopicChannels, "\n", "✅ <#", ">")
			noTopicChan = len(g.EnabledTopicChannels) == 0
			return g, "TopicConfigCommand: get enabled topic channels"
		})

		if noTopicChan {
			_, err := cmd.SendEmbed(c.E, "Configure Topics", "There are currently no allowed topic channels", bot.DefaultColor)
			return err
		}

		_, err := cmd.SendEmbed(c.E, "Configure Topics", "Allowed Topic Channels:\n\n"+formattedChannels, bot.DefaultColor)
		return err
	default:
		_, err := cmd.SendEmbed(c.E,
			"Configure Topics",
			"Available arguments are:\n- `list`\n- `threshold [threshold]`\n- `enable|disable [channel]`",
			bot.DefaultColor)
		return err
	}
}

func TopicCommand(c bot.Command) error {
	topic, argErr := cmd.ParseAllArgs(c.Args)
	if argErr != nil {
		_, err := cmd.SendEmbed(c.E,
			"Suggest Topic",
			"Available arguments are:\n- `[topic to suggest]`\nUse the `topicconfig` command to figure topic channels!",
			bot.DefaultColor)
		return err
	}

	topicsEnabled := false
	bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		topicsEnabled = util.SliceContains(g.EnabledTopicChannels, int64(c.E.ChannelID))
		return g, "TopicCommand: check topicsEnabled"
	})

	if !topicsEnabled {
		_, err := cmd.SendEmbed(c.E, "Topics are disabled in this channel!", "", bot.ErrorColor)
		return err
	}

	msg, err := cmd.SendEmbed(c.E, "New topic suggested!", c.E.Author.Mention()+" suggests: "+topic, bot.DefaultColor)
	if err != nil {
		return err
	}

	emoji, err := topicVoteApiEmoji(c.E.GuildID)
	if err != nil {
		return err
	}

	bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		g.ActiveTopicVotes = append(g.ActiveTopicVotes, bot.ActiveTopicVote{Message: int64(msg.ID), Author: int64(c.E.Author.ID), Topic: topic})
		return g, "TopicCommand: append ActiveTopicVotes"
	})

	if err := bot.Client.React(msg.ChannelID, msg.ID, emoji); err != nil {
		return err
	}

	return nil
}

func TopicReactionHandler(i interface{}) {
	defer util.LogPanic()
	e := i.(*gateway.MessageReactionAddEvent)

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

		emoji, err := topicVoteApiEmoji(e.GuildID)
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

func topicVoteEmoji(id discord.GuildID) (string, error) {
	e := ""
	bot.GuildContext(id, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		e = g.TopicVoteEmoji

		if len(e) == 0 {
			g.TopicVoteEmoji = escapedCheckmark
		} else {
			e = strings.TrimSuffix(e, "a:")
		}
		return g, "topicVoteEmoji"
	})

	return bot.EmojiConfigFormatted(e)
}

func topicVoteApiEmoji(id discord.GuildID) (discord.APIEmoji, error) {
	e := ""
	bot.GuildContext(id, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		e = g.TopicVoteEmoji

		if len(e) == 0 {
			g.TopicVoteEmoji = escapedCheckmark
		}
		return g, "topicVoteApiEmoji"
	})

	return bot.EmojiConfigAsApi(e)
}

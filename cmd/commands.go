package cmd

import (
	"encoding/json"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func RegisterCommands() {
	bot.Commands = append(bot.Commands,
		[]bot.CommandInfo{
			{Fn: ChannelCommand, FnName: "ChannelCommand", Name: "channel", Description: "Manage channels", GuildOnly: true},
			{Fn: StealEmojiCommand, FnName: "StealEmojiCommand", Name: "stealemoji", Aliases: []string{"se"}, Description: "Upload an emoji to the current guild", GuildOnly: true},
			{Fn: FrogCommand, FnName: "FrogCommand", Name: "frog", Description: "\\*hands you a random frog pic\\*"},
			{Fn: HelpCommand, FnName: "HelpCommand", Name: "help", Aliases: []string{"h"}},
			{Fn: KirbyCommand, FnName: "KirbyCommand", Name: "kirby"},
			{Fn: PermissionCommand, FnName: "PermissionCommand", Name: "permission", Aliases: []string{"perm"}, Description: "Manage user permissions", GuildOnly: true},
			{Fn: PingCommand, FnName: "PingCommand", Name: "ping", Description: "Returns the current API latency"},
			{Fn: PrefixCommand, FnName: "PrefixCommand", Name: "prefix", Description: "Set the bot prefix for your guild", GuildOnly: true},
			{Fn: TopicCommand, FnName: "TopicCommand", Name: "topic", Description: "Suggest a new topic for the current channel", GuildOnly: true},
		}...,
	)
}

func StealEmojiCommand(c bot.Command) error {
	// try to get emoji ID
	emojiID, argErr := ParseInt64Arg(c.Args, 1)
	// try to get emoji URL
	if argErr != nil {
		emojiID, argErr = ParseEmojiUrlArg(c.Args, 1)
	}
	// try to get sent emoji
	if argErr != nil {
		emojiID, argErr = ParseEmojiIdArg(c.Args, 1)
	}
	// no emoji found
	if argErr != nil {
		return argErr
	}

	//
	// we now have the emoji ID, get the name

	emojiName, argErr := ParseStringArg(c.Args, 2, false)
	if argErr != nil {
		return bot.GenericError("StealEmojiCommand", "getting emoji name", "expected emoji name")
	}

	//
	// we now have the emoji ID and name, get the bytes

	url := "https://cdn.discordapp.com/emojis/" + strconv.FormatInt(emojiID, 10)
	bytes, res, err := util.RequestUrl(url+".gif", http.MethodGet)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		bytes, res, err = util.RequestUrl(url+".png", http.MethodGet)
		if err != nil {
			return err
		}

		if res.StatusCode != 200 {
			return bot.GenericError("StealEmojiCommand", "getting emoji bytes", "status was "+res.Status)
		}
	}

	// now we try to upload it

	image := api.Image{ContentType: res.Header.Get("content-type"), Content: bytes}
	createEmojiData := api.CreateEmojiData{
		Name:  emojiName,
		Image: image,
		AuditLogReason: api.AuditLogReason(
			"emoji created by " + util.GetUserMention(int64(c.E.Author.ID)),
		),
	}

	if emoji, err := bot.Client.CreateEmoji(c.E.GuildID, createEmojiData); err != nil {
		// error with uploading
		return bot.GenericError("StealEmojiCommand", "uploading emoji", err.Error())
	} else {
		// uploaded successfully, send a nice embed
		_, err := bot.Client.SendMessage(
			c.E.ChannelID,
			emoji.String(),
			discord.Embed{Title: "Emoji stolen ;)", Color: bot.SuccessColor},
		)
		return err
	}
}

func TopicCommand(c bot.Command) error {
	topic, argErr := ParseAllArgs(c.Args)
	if argErr != nil {
		return argErr
	}

	topicsEnabled := false
	bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		topicsEnabled = util.SliceContains(g.EnabledTopicChannels, int64(c.E.ChannelID))
		return g, "TopicCommand: check topicsEnabled"
	})

	if !topicsEnabled {
		_, err := SendEmbed(c, "Topics are disabled in this channel!", "", bot.ErrorColor)
		return err
	}

	msg, err := SendEmbed(c, "New topic suggested!", c.E.Author.Mention()+" suggests: "+topic, bot.DefaultColor)
	if err != nil {
		return err
	}

	emoji, err := util.GuildTopicVoteApiEmoji(c.E.GuildID)
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

func ChannelCommand(c bot.Command) error {
	arg1, _ := ParseStringArg(c.Args, 1, true)

	switch arg1 {
	case "archive":
		err := HasPermission("channels", c)
		if err == nil {
			bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
				if g.ArchiveRole == 0 {
					err = bot.GenericError(c.FnName, "getting archive role", "`archive_role` not set in guild config")
				}
				if g.ArchiveCategory == 0 {
					err = bot.GenericError(c.FnName, "getting archive category", "`archive_category` not set in guild config")
				}
				return g, "ChannelCommand: check archive permission"
			})

			if err != nil {
				return err
			}

			channel, err := bot.Client.Channel(c.E.ChannelID)
			if err != nil {
				return err
			}

			overwrites := make([]discord.Overwrite, 0)
			var data api.ModifyChannelData

			bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
				// Copy everything except the archive and @everyone roles to overwrites
				for _, overwrite := range channel.Overwrites {
					id := int64(overwrite.ID)
					if id != int64(c.E.GuildID) && id != g.ArchiveRole {
						overwrites = append(overwrites, overwrite)
						break
					}
				}

				overwrites = append(
					overwrites,
					discord.Overwrite{
						ID:   discord.Snowflake(c.E.GuildID),
						Type: discord.OverwriteRole,
						Deny: discord.PermissionViewChannel,
					},
					discord.Overwrite{
						ID:    discord.Snowflake(g.ArchiveRole),
						Type:  discord.OverwriteRole,
						Allow: discord.PermissionViewChannel,
					},
				)
				data = api.ModifyChannelData{Overwrites: &overwrites, CategoryID: discord.ChannelID(g.ArchiveCategory)}

				return g, "ChannelCommand: create overwrites data"
			})

			err = bot.Client.ModifyChannel(c.E.ChannelID, data)
			if err != nil {
				return err
			} else {
				_, err = SendEmbed(c, "Channels", "Successfully archived channel", bot.SuccessColor)
				return err
			}
		} else {
			return err
		}
	case "topic": // TODO: This is terrible. Future me please re-write.
		err := HasPermission("channels", c)
		if err == nil {
			channels := []int64{int64(c.E.ChannelID)}

			if argChannels, err := ParseChannelSliceArg(c.Args, 3, -1); err == nil && len(argChannels) != 0 {
				channels = argChannels
			}
			channelsStr := util.JoinInt64Slice(channels, ", ", "<#", ">")

			if arg2, _ := ParseStringArg(c.Args, 2, true); err != nil {
				return err
			} else {
				switch arg2 {
				case "enable":
					bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
						for _, channel := range channels {
							if !util.SliceContains(g.EnabledTopicChannels, channel) {
								g.EnabledTopicChannels = append(g.EnabledTopicChannels, channel)
							}
						}
						return g, "ChannelCommand: topic enable"
					})
					_, err := SendEmbed(c, "Channel Topic", "✅ Added "+channelsStr+" to the allowed topic channels", bot.SuccessColor)
					return err
				case "disable":
					bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
						for _, channel := range channels {
							if util.SliceContains(g.EnabledTopicChannels, channel) {
								g.EnabledTopicChannels = util.SliceRemove(g.EnabledTopicChannels, channel)
							}
						}
						return g, "ChannelCommand: topic disable"
					})
					_, err := SendEmbed(c, "Channel Topic", "⛔ Removed "+channelsStr+" from the allowed topic channels", bot.ErrorColor)
					return err
				case "emoji":
					arg3, animated, err3 := ParseEmojiArg(c.Args, 3, true)
					if err3 != nil {
						return err3
					}

					if arg3 == nil {
						if emoji, err := util.GuildTopicVoteEmoji(c.E.GuildID); err != nil {
							return err
						} else {
							_, err = SendEmbed(c, "Current Topic Vote Emoji:", emoji, bot.DefaultColor)
							return err
						}
					} else {
						configEmoji := util.ApiEmojiAsConfig(arg3, animated)
						emoji, err := util.FormatEncodedEmoji(configEmoji)
						if err != nil {
							return err
						}

						bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
							g.TopicVoteEmoji = configEmoji
							return g, "ChannelCommand: update TopicVoteEmoji"
						})

						_, err = SendEmbed(c, "Set Topic Vote Emoji To:", emoji, bot.SuccessColor)
						return err
					}
				case "threshold":
					arg3, err3 := ParseInt64Arg(c.Args, 3)
					if err3 != nil {
						return err3
					}

					bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
						if arg3 <= 0 {
							arg3 = 3
						}

						g.TopicVoteThreshold = arg3
						return g, "ChannelCommand: update topic vote threshold"
					})

					_, err := SendEmbed(c, "Set Topic Vote Threshold To:", strconv.FormatInt(arg3, 10), bot.SuccessColor)
					return err
				default:
					noTopicChan := false
					formattedChannels := ""

					bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
						formattedChannels = util.JoinInt64Slice(g.EnabledTopicChannels, "\n", "✅ <#", ">")
						noTopicChan = len(g.EnabledTopicChannels) == 0
						return g, "ChannelCommand: get enabled topic channels"
					})

					if noTopicChan {
						_, err := SendEmbed(c, "Channel Topic", "There are currently no allowed topic channels", bot.DefaultColor)
						return err
					}

					_, err := SendEmbed(c, "Channel Topic", "Allowed Topic Channels:\n\n"+formattedChannels, bot.DefaultColor)
					return err
				}
			}
		} else {
			return err
		}
	case "starboard":
		err := HasPermission("channels", c)
		if err == nil {
			if arg2, _ := ParseStringArg(c.Args, 2, true); err != nil {
				return err
			} else {
				arg3, errParse := ParseChannelArg(c.Args, 3)
				var err error = nil

				bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
					switch arg2 {
					case "regular":
						if errParse != nil {
							g.Starboard.Channel = 0
							_, err = SendEmbed(c, "Starboard Channels", "⛔ Disabled regular starboard", bot.ErrorColor)
							return g, "ChannelCommand: enable regular starboard"
						} else {
							g.Starboard.Channel = arg3
							_, err = SendEmbed(c, "Starboard Channels", "✅ Enabled regular starboard", bot.SuccessColor)
							return g, "ChannelCommand: disable regular starboard"
						}
					case "nsfw":
						if errParse != nil {
							g.Starboard.NsfwChannel = 0
							_, err = SendEmbed(c, "Starboard Channels", "⛔ Disabled NSFW starboard", bot.ErrorColor)
							return g, "ChannelCommand: enable nsfw starboard"
						} else {
							g.Starboard.NsfwChannel = arg3
							_, err = SendEmbed(c, "Starboard Channels", "✅ Enabled NSFW starboard", bot.SuccessColor)
							return g, "ChannelCommand: disable nsfw starboard"
						}
					default:
						regularC := "✅ Regular Starboard <#" + strconv.FormatInt(g.Starboard.Channel, 10) + ">"
						nsfwC := "✅ NSFW Starboard <#" + strconv.FormatInt(g.Starboard.NsfwChannel, 10) + ">"
						if g.Starboard.Channel == 0 {
							regularC = "⛔ Regular Starboard"
						}
						if g.Starboard.NsfwChannel == 0 {
							nsfwC = "⛔ NSFW Starboard"
						}

						embed := discord.Embed{
							Title:       "Starboard Channels",
							Description: regularC + "\n" + nsfwC,
							Color:       bot.DefaultColor,
						}
						_, err = SendCustomEmbed(c.E.ChannelID, embed)
						return g, "ChannelCommand: format starboard channels"
					}
				})
				return err
			}
		} else {
			return err
		}
	default:
		_, err := SendEmbed(c,
			"Channel",
			"Available arguments are:\n- `archive`\n- `topic enable|disable|emoji|threshold`\n- `starboard set regular|nsfw [channel]`",
			bot.DefaultColor)
		return err
	}
}

func PermissionCommand(c bot.Command) error {
	arg1, _ := ParseStringArg(c.Args, 1, true)

	switch arg1 {
	case "give":
		err := HasPermission("permissions", c)
		if err == nil {
			permission, argErr := ParseStringArg(c.Args, 2, true)
			if argErr != nil {
				return argErr
			}
			id, argErr := ParseUserArg(c.Args, 3)
			if argErr != nil {
				return argErr
			}

			if err := GivePermission(permission, id, c); err != nil {
				return err
			} else {
				_, err = SendEmbed(c,
					"Permissions",
					"Successfully gave "+util.GetUserMention(id)+" permission to use \""+permission+"\"",
					bot.SuccessColor)
				return err
			}
		} else {
			return err
		}
	case "op":
		if bot.C.OperatorID == 0 || int64(c.E.Author.ID) != bot.C.OperatorID {
			return bot.GenericError("PermissionCommand", "granting operator access", "user is not the bot operator!")
		}

		for _, permission := range Permissions {
			if err := GivePermission(permission, bot.C.OperatorID, c); err != nil {
				SendErrorEmbed(c, err)
			} else {
				_, err = SendEmbed(c,
					"Permissions",
					"Successfully gave "+util.GetUserMention(bot.C.OperatorID)+" permission to use \""+permission+"\"",
					bot.SuccessColor)
				return err
			}
		}

		return nil
	default:
		_, err := SendEmbed(c,
			"Permissions",
			"Available arguments are:\n- `give` <permission> <user>\n- `op`",
			bot.DefaultColor)
		return err
	}
}

func PrefixCommand(c bot.Command) error {
	arg, argErr := ParseStringArg(c.Args, 1, false)
	if argErr != nil {
		return argErr
	}

	// Filter spaces
	arg = strings.ReplaceAll(arg, " ", "")
	if len(arg) == 0 {
		return bot.GenericError(c.FnName, "getting prefix", "prefix is empty")
	}

	// Prefix is okay, set it in the cache
	//

	bot.C.Run(func(config *bot.Config) {
		config.PrefixCache[int64(c.E.GuildID)] = arg
	})

	// Also set it in the guild
	//

	bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		g.Prefix = arg
		return g, "PrefixCommand"
	})

	embed := discord.Embed{
		Description: "Set prefix to `" + arg + "`.",
		Footer:      &discord.EmbedFooter{Text: "At any time you can ping the bot with the word \"prefix\" to get the current prefix"},
		Color:       bot.SuccessColor,
	}
	_, err := SendCustomEmbed(c.E.ChannelID, embed)
	return err
}

func HelpCommand(c bot.Command) error {
	fmtCmds := make([]string, 0)
	for _, cmd := range bot.Commands {
		fmtCmds = append(fmtCmds, cmd.MarkdownString())
	}

	_, err := SendEmbed(c,
		"Taro Help",
		strings.Join(fmtCmds, "\n\n"),
		bot.DefaultColor)
	return err
}

func PingCommand(c bot.Command) error {
	if msg, err := SendEmbed(c,
		"Ping!",
		"Waiting for API response...",
		bot.DefaultColor); err != nil {
		return err
	} else {
		curTime := time.Now().UnixMilli()
		msgTime := msg.Timestamp.Time().UnixMilli()

		embed := makeEmbed("Pong!", "Latency is "+strconv.FormatInt(curTime-msgTime, 10)+"ms", bot.SuccessColor)
		_, err = bot.Client.EditMessage(msg.ChannelID, msg.ID, "", embed)
		return err
	}
}

func FrogCommand(c bot.Command) error {
	frogData, _, err := util.RequestUrl("https://frog.pics/api/random", http.MethodGet)
	if err != nil {
		return err
	}

	type FrogPicture struct {
		ImageUrl    string `json:"image_url"`
		MedianColor string `json:"median_color"`
	}
	var frogPicture FrogPicture

	if err := json.Unmarshal(frogData, &frogPicture); err != nil {
		return err
	}

	color, err := util.ParseHexColorFast("#" + frogPicture.MedianColor)
	if err != nil {
		return err
	}

	embed := discord.Embed{
		Color: discord.Color(util.ConvertColorToInt32(color)),
		Image: &discord.EmbedImage{URL: frogPicture.ImageUrl},
	}

	_, err = SendCustomEmbed(c.E.ChannelID, embed)
	return err
}

func KirbyCommand(c bot.Command) error {
	content, _ := ParseAllArgs(c.Args)
	_, _ = SendMessage(c, "<:kirbyfeet:893291555744542730>")
	_, _ = SendMessage(c, content)
	return nil
}

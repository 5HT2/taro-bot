package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"strconv"
	"strings"
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	return &plugins.Plugin{
		Name:        "Taro Base Extra",
		Description: "The extra commands as included as part of the bot",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          ChannelCommand,
			FnName:      "ChannelCommand",
			Name:        "channel",
			Aliases:     []string{"c"},
			Description: "Manage channels",
			GuildOnly:   true,
		}, {
			Fn:          PermissionCommand,
			FnName:      "PermissionCommand",
			Name:        "permission",
			Aliases:     []string{"perm"},
			Description: "Manage user permissions",
			GuildOnly:   true,
		}, {
			Fn:          TopicCommand,
			FnName:      "TopicCommand",
			Name:        "topic",
			Description: "Suggest a new topic for the current channel",
			GuildOnly:   true,
		}},
		Responses: []bot.ResponseInfo{},
	}
}

func ChannelCommand(c bot.Command) error {
	arg1, _ := cmd.ParseStringArg(c.Args, 1, true)

	switch arg1 {
	case "archive":
		err := cmd.HasPermission("channels", c)
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
				_, err = cmd.SendEmbed(c.E, "Channels", "Successfully archived channel", bot.SuccessColor)
				return err
			}
		} else {
			return err
		}
	case "starboard":
		err := cmd.HasPermission("channels", c)
		if err == nil {
			if arg2, _ := cmd.ParseStringArg(c.Args, 2, true); err != nil {
				return err
			} else {
				arg3, errParse := cmd.ParseChannelArg(c.Args, 3)
				var err error = nil

				bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
					switch arg2 {
					case "regular":
						if errParse != nil {
							g.Starboard.Channel = 0
							_, err = cmd.SendEmbed(c.E, "Starboard Channels", "⛔ Disabled regular starboard", bot.ErrorColor)
							return g, "ChannelCommand: enable regular starboard"
						} else {
							g.Starboard.Channel = arg3
							_, err = cmd.SendEmbed(c.E, "Starboard Channels", "✅ Enabled regular starboard", bot.SuccessColor)
							return g, "ChannelCommand: disable regular starboard"
						}
					case "nsfw":
						if errParse != nil {
							g.Starboard.NsfwChannel = 0
							_, err = cmd.SendEmbed(c.E, "Starboard Channels", "⛔ Disabled NSFW starboard", bot.ErrorColor)
							return g, "ChannelCommand: enable nsfw starboard"
						} else {
							g.Starboard.NsfwChannel = arg3
							_, err = cmd.SendEmbed(c.E, "Starboard Channels", "✅ Enabled NSFW starboard", bot.SuccessColor)
							return g, "ChannelCommand: disable nsfw starboard"
						}
					case "threshold":
						if arg3, errParse := cmd.ParseInt64Arg(c.Args, 3); errParse != nil {
							err = errParse
						} else {
							if arg3 <= 0 {
								arg3 = 1
							}

							g.Starboard.Threshold = arg3
							_, err = cmd.SendEmbed(c.E, "Starboard Threshold", fmt.Sprintf("✅ Set threshold to: %v", arg3), bot.SuccessColor)
						}

						return g, "ChannelCommand: set threshold"

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
						_, err = cmd.SendCustomEmbed(c.E.ChannelID, embed)
						return g, "ChannelCommand: format starboard channels"
					}
				})
				return err
			}
		} else {
			return err
		}
	default:
		_, err := cmd.SendEmbed(c.E,
			"Channel",
			"Available arguments are:\n- `archive`\n- `starboard regular|nsfw [channel]`\n- `starboard threshold [threshold]`",
			bot.DefaultColor)
		return err
	}
}

func PermissionCommand(c bot.Command) error {
	arg1, _ := cmd.ParseStringArg(c.Args, 1, true)

	switch arg1 {
	case "give":
		err := cmd.HasPermission("permissions", c)
		if err == nil {
			permission, argErr := cmd.ParseStringArg(c.Args, 2, true)
			if argErr != nil {
				return argErr
			}
			id, argErr := cmd.ParseUserArg(c.Args, 3)
			if argErr != nil {
				return argErr
			}

			if err := cmd.GivePermission(permission, id, c); err != nil {
				return err
			} else {
				_, err = cmd.SendEmbed(c.E,
					"Permissions",
					"Successfully gave "+util.GetUserMention(id)+" permission to use \""+permission+"\"",
					bot.SuccessColor)
				return err
			}
		} else {
			return err
		}
	case "op":
		id := int64(c.E.Author.ID)
		if id != bot.C.OperatorID && !cmd.HasAdminCached(c.E.GuildID, c.E.Member.RoleIDs, c.E.Author) {
			return bot.GenericError("PermissionCommand", "granting operator access", "user is not the bot operator!")
		}

		color := bot.SuccessColor
		errs := 0
		responses := make([]string, 0)

		for _, permission := range cmd.Permissions {
			if err := cmd.GivePermission(permission, id, c); err != nil {
				responses = append(responses, fmt.Sprintf("⛔ Failed to give \"%s\" permission:%s\n", permission, err.Error()))
				errs += 1
			} else {
				responses = append(responses, fmt.Sprintf("✅ Granted \"%s\" permission\n", permission))
			}
		}

		if errs == len(cmd.Permissions) {
			color = bot.ErrorColor
		} else if errs > 0 {
			color = bot.WarnColor
		}

		_, err := cmd.SendEmbed(c.E,
			"Permissions",
			strings.Join(responses, "\n"),
			color)

		return err
	default:
		_, err := cmd.SendEmbed(c.E,
			"Permissions",
			"Available arguments are:\n- `give` <permission> <user>\n- `op`",
			bot.DefaultColor)
		return err
	}
}

func TopicCommand(c bot.Command) error {
	topic, argErr := cmd.ParseAllArgs(c.Args)
	if argErr != nil {
		return argErr
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

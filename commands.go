package main

import (
	"encoding/json"
	"fmt"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type CommandInfo struct {
	FnName      string
	Name        string
	Description string
	Aliases     []string
}

func (ci CommandInfo) String() string {
	aliases := ""
	if len(ci.Aliases) > 0 {
		aliases = "(" + strings.Join(ci.Aliases, ", ") + ")"
	}
	description := ci.Description
	if len(description) == 0 {
		description = "No Description"
	}

	return fmt.Sprintf("**%s** %s\n%s", ci.Name, aliases, description)
}

var (
	commands = []CommandInfo{
		{FnName: "ChannelCommand", Name: "channel", Description: "Manage channels"},
		{FnName: "FrogCommand", Name: "frog", Description: "\\*hands you a random frog pic\\*"},
		{FnName: "HelpCommand", Name: "help", Aliases: []string{"h"}},
		{FnName: "KirbyCommand", Name: "kirby"},
		{FnName: "PermissionCommand", Name: "permission", Aliases: []string{"perm"}, Description: "Manage user permissions"},
		{FnName: "PingCommand", Name: "ping", Description: "Returns the current API latency"},
		{FnName: "PrefixCommand", Name: "prefix", Description: "Set the bot prefix for your guild"},
		{FnName: "TopicCommand", Name: "topic", Description: "Suggest a new topic for the current channel"},
	}
)

func (c Command) TopicCommand() error {
	topic, argErr := ParseAllArgs(c.args)
	if argErr != nil {
		return argErr
	}

	guild := GetGuildConfig(int64(c.e.GuildID))

	if !Int64SliceContains(guild.EnabledTopicChannels, int64(c.e.ChannelID)) {
		_, err := SendEmbed(c, "Topics are disabled in this channel!", "", errorColor)
		return err
	}

	msg, err := SendEmbed(c, "New topic suggested!", c.e.Author.Mention()+" suggests: "+topic, defaultColor)
	if err != nil {
		return err
	}

	emoji, err := GuildTopicVoteApiEmoji(guild)
	if err != nil {
		return err
	}

	guild.ActiveTopicVotes = append(guild.ActiveTopicVotes, ActiveTopicVote{int64(msg.ID), int64(c.e.Author.ID), topic})
	SetGuildConfig(guild)

	if err := discordClient.React(msg.ChannelID, msg.ID, emoji); err != nil {
		return err
	}

	return nil
}

func (c Command) ChannelCommand() error {
	arg1, _ := ParseStringArg(c.args, 1, true)

	switch arg1 {
	case "archive":
		err := HasPermission("channels", c)
		if err == nil {
			guild := GetGuildConfig(int64(c.e.GuildID))
			if guild.ArchiveRole == 0 {
				return GenericError(c.fnName, "getting archive role", "`archive_role` not set in guild config")
			}

			channel, err := discordClient.Channel(c.e.ChannelID)
			if err != nil {
				return err
			}

			overwrites := make([]discord.Overwrite, 0)

			// Copy everything except the archive and @everyone roles to overwrites
			for _, overwrite := range channel.Overwrites {
				id := int64(overwrite.ID)
				if id != int64(c.e.GuildID) && id != guild.ArchiveRole {
					overwrites = append(overwrites, overwrite)
					break
				}
			}

			overwrites = append(
				overwrites,
				discord.Overwrite{
					ID:   discord.Snowflake(c.e.GuildID),
					Type: discord.OverwriteRole,
					Deny: discord.PermissionViewChannel,
				},
				discord.Overwrite{
					ID:    discord.Snowflake(guild.ArchiveRole),
					Type:  discord.OverwriteRole,
					Allow: discord.PermissionViewChannel,
				},
			)

			data := api.ModifyChannelData{Overwrites: &overwrites}
			err = discordClient.ModifyChannel(c.e.ChannelID, data)
			if err != nil {
				return err
			} else {
				_, err = SendEmbed(c, "Channels", "Successfully archived channel", successColor)
				return err
			}
		} else {
			return err
		}
	case "topic":
		err := HasPermission("channels", c)
		if err == nil {
			guild := GetGuildConfig(int64(c.e.GuildID))
			channels := []int64{int64(c.e.ChannelID)}

			if argChannels, err := ParseChannelSliceArg(c.args, 3, -1); err == nil && len(argChannels) != 0 {
				channels = argChannels
			}
			channelsStr := JoinInt64Slice(channels, ", ", "<#", ">")

			if arg2, _ := ParseStringArg(c.args, 2, true); err != nil {
				return err
			} else {
				switch arg2 {
				case "enable":
					for _, channel := range channels {
						if !Int64SliceContains(guild.EnabledTopicChannels, channel) {
							guild.EnabledTopicChannels = append(guild.EnabledTopicChannels, channel)
						}
					}
					SetGuildConfig(guild)
					_, err := SendEmbed(c, "Channel Topic", "✅ Added "+channelsStr+" to the allowed topic channels", successColor)
					return err
				case "disable":
					for _, channel := range channels {
						if Int64SliceContains(guild.EnabledTopicChannels, channel) {
							guild.EnabledTopicChannels = Int64SliceRemove(guild.EnabledTopicChannels, channel)
						}
					}
					SetGuildConfig(guild)
					_, err := SendEmbed(c, "Channel Topic", "⛔ Removed "+channelsStr+" from the allowed topic channels", errorColor)
					return err
				case "emoji":
					arg3, animated, err3 := ParseEmojiArg(c.args, 3, true)
					if err3 != nil {
						return err3
					}

					if arg3 == nil {
						if emoji, err := GuildTopicVoteEmoji(guild); err != nil {
							return err
						} else {
							_, err = SendEmbed(c, "Current Topic Vote Emoji:", emoji, defaultColor)
							return err
						}
					} else {
						configEmoji := ApiEmojiAsConfig(arg3, animated)
						emoji, err := FormatEncodedEmoji(configEmoji)
						if err != nil {
							return err
						}

						guild.TopicVoteEmoji = configEmoji
						SetGuildConfig(guild)

						_, err = SendEmbed(c, "Set Topic Vote Emoji To:", emoji, successColor)
						return err
					}
				case "threshold":
					arg3, err3 := ParseInt64Arg(c.args, 3)
					if err3 != nil {
						return err3
					}

					if arg3 <= 0 {
						arg3 = 3
					}
					guild.TopicVoteThreshold = arg3
					SetGuildConfig(guild)

					_, err := SendEmbed(c, "Set Topic Vote Threshold To:", strconv.FormatInt(arg3, 10), successColor)
					return err
				default:
					if len(guild.EnabledTopicChannels) == 0 {
						_, err := SendEmbed(c, "Channel Topic", "There are currently no allowed topic channels", defaultColor)
						return err
					}

					formattedChannels := JoinInt64Slice(guild.EnabledTopicChannels, "\n", "✅ <#", ">")
					_, err := SendEmbed(c, "Channel Topic", "Allowed Topic Channels:\n\n"+formattedChannels, defaultColor)
					return err
				}
			}
		} else {
			return err
		}
	case "starboard":
		err := HasPermission("channels", c)
		if err == nil {
			if arg2, _ := ParseStringArg(c.args, 2, true); err != nil {
				return err
			} else {
				guild := GetStarboardConfig(int64(c.e.GuildID))
				arg3, err := ParseChannelArg(c.args, 3)

				switch arg2 {
				case "regular":
					if err != nil {
						guild.Channel = arg3
						SetStarboardConfig(guild)
						_, err := SendEmbed(c, "Starboard Channels", "⛔ Disabled regular starboard", errorColor)
						return err
					} else {
						guild.Channel = 0
						SetStarboardConfig(guild)
						_, err := SendEmbed(c, "Starboard Channels", "✅ Enabled regular starboard", successColor)
						return err
					}
				case "nsfw":
					if err != nil {
						guild.NsfwChannel = arg3
						SetStarboardConfig(guild)
						_, err := SendEmbed(c, "Starboard Channels", "⛔ Disabled NSFW starboard", errorColor)
						return err
					} else {
						guild.NsfwChannel = 0
						SetStarboardConfig(guild)
						_, err := SendEmbed(c, "Starboard Channels", "✅ Enabled NSFW starboard", successColor)
						return err
					}
				default:
					regularC := "✅ Regular Starboard <#" + strconv.FormatInt(guild.Channel, 10) + ">"
					nsfwC := "✅ NSFW Starboard <#" + strconv.FormatInt(guild.NsfwChannel, 10) + ">"
					if guild.Channel == 0 {
						regularC = "⛔ Regular Starboard"
					}
					if guild.NsfwChannel == 0 {
						nsfwC = "⛔ NSFW Starboard"
					}

					embed := discord.Embed{
						Title:       "Starboard Channels",
						Description: regularC + "\n" + nsfwC,
						Color:       defaultColor,
					}
					_, err := SendCustomEmbed(c.e.ChannelID, embed)
					return err
				}
			}
		} else {
			return err
		}
	default:
		_, err := SendEmbed(c,
			"Channel",
			"Available arguments are:\n- `archive`\n- `topic enable|disable|emoji|threshold`\n- `starboard set regular|nsfw [channel]`",
			defaultColor)
		return err
	}
}

func (c Command) PermissionCommand() error {
	arg1, _ := ParseStringArg(c.args, 1, true)

	switch arg1 {
	case "give":
		err := HasPermission("permissions", c)
		if err == nil {
			permission, argErr := ParseStringArg(c.args, 2, true)
			if argErr != nil {
				return argErr
			}
			id, argErr := ParseUserArg(c.args, 3)
			if argErr != nil {
				return argErr
			}

			if err := GivePermission(permission, id, c); err != nil {
				return err
			} else {
				_, err = SendEmbed(c,
					"Permissions",
					"Successfully gave "+GetUserMention(id)+" permission to use \""+permission+"\"",
					successColor)
				return err
			}
		} else {
			return err
		}
	default:
		_, err := SendEmbed(c,
			"Permissions",
			"Available arguments are:\n- `give` <permission> <user>",
			defaultColor)
		return err
	}
}

func (c Command) PrefixCommand() error {
	arg, argErr := ParseStringArg(c.args, 1, false)
	if argErr != nil {
		return argErr
	}

	// Filter spaces
	arg = strings.ReplaceAll(arg, " ", "")
	if len(arg) == 0 {
		return GenericError(c.fnName, "getting prefix", "prefix is empty")
	}

	guild := GetGuildConfig(int64(c.e.GuildID))
	guild.Prefix = arg
	SetGuildConfig(guild)

	embed := discord.Embed{
		Description: "Set prefix to `" + arg + "`.",
		Footer:      &discord.EmbedFooter{Text: "At any time you can ping the bot with the word \"prefix\" to get the current prefix"},
		Color:       successColor,
	}
	_, err := SendCustomEmbed(c.e.ChannelID, embed)
	return err
}

func (c Command) HelpCommand() error {
	fmtCmds := make([]string, 0)
	for _, cmd := range commands {
		fmtCmds = append(fmtCmds, cmd.String())
	}

	_, err := SendEmbed(c,
		"Taro Help",
		strings.Join(fmtCmds, "\n\n"),
		defaultColor)
	return err
}

func (c Command) PingCommand() error {
	if msg, err := SendEmbed(c,
		"Ping!",
		"Waiting for API response...",
		defaultColor); err != nil {
		return err
	} else {
		curTime := time.Now().UnixMilli()
		msgTime := msg.Timestamp.Time().UnixMilli()

		embed := makeEmbed("Pong!", "Latency is `"+strconv.FormatInt(curTime-msgTime, 10)+"`ms", successColor)
		_, err = discordClient.EditMessage(msg.ChannelID, msg.ID, "", embed)
		return err
	}
}

func (c Command) FrogCommand() error {
	frogData, err := RequestUrl("https://frog.pics/api/random", http.MethodGet)
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

	color, err := ParseHexColorFast("#" + frogPicture.MedianColor)
	if err != nil {
		return err
	}

	embed := discord.Embed{
		Color: discord.Color(ConvertColorToInt32(color)),
		Image: &discord.EmbedImage{URL: frogPicture.ImageUrl},
	}

	_, err = SendCustomEmbed(c.e.ChannelID, embed)
	return err
}

func (c Command) KirbyCommand() {
	content, _ := ParseAllArgs(c.args)
	_, _ = SendMessage(c, "<:kirbyfeet:893291555744542730>")
	_, _ = SendMessage(c, content)
}

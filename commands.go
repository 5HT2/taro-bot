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
	GuildOnly   bool
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
		{FnName: "ChannelCommand", Name: "channel", Description: "Manage channels", GuildOnly: true},
		{FnName: "StealEmojiCommand", Name: "stealemoji", Aliases: []string{"se"}, Description: "Upload an emoji to the current guild", GuildOnly: true},
		{FnName: "FrogCommand", Name: "frog", Description: "\\*hands you a random frog pic\\*"},
		{FnName: "HelpCommand", Name: "help", Aliases: []string{"h"}},
		{FnName: "KirbyCommand", Name: "kirby"},
		{FnName: "PermissionCommand", Name: "permission", Aliases: []string{"perm"}, Description: "Manage user permissions", GuildOnly: true},
		{FnName: "PingCommand", Name: "ping", Description: "Returns the current API latency"},
		{FnName: "PrefixCommand", Name: "prefix", Description: "Set the bot prefix for your guild", GuildOnly: true},
		{FnName: "TopicCommand", Name: "topic", Description: "Suggest a new topic for the current channel", GuildOnly: true},
	}
)

func (c Command) StealEmojiCommand() error {
	// try to get emoji ID
	emojiID, argErr := ParseInt64Arg(c.args, 1)
	// try to get emoji URL
	if argErr != nil {
		emojiID, argErr = ParseEmojiUrlArg(c.args, 1)
	}
	// try to get sent emoji
	if argErr != nil {
		emojiID, argErr = ParseEmojiIdArg(c.args, 1)
	}
	// no emoji found
	if argErr != nil {
		return argErr
	}

	//
	// we now have the emoji ID, get the name

	emojiName, argErr := ParseStringArg(c.args, 2, false)
	if argErr != nil {
		return GenericError("StealEmojiCommand", "getting emoji name", "expected emoji name")
	}

	//
	// we now have the emoji ID and name, get the bytes

	url := "https://cdn.discordapp.com/emojis/" + strconv.FormatInt(emojiID, 10)
	bytes, res, err := RequestUrl(url+".gif", http.MethodGet)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		bytes, res, err = RequestUrl(url+".png", http.MethodGet)
		if err != nil {
			return err
		}

		if res.StatusCode != 200 {
			return GenericError("StealEmojiCommand", "getting emoji bytes", "status was "+res.Status)
		}
	}

	// now we try to upload it

	image := api.Image{ContentType: res.Header.Get("content-type"), Content: bytes}
	createEmojiData := api.CreateEmojiData{
		Name:  emojiName,
		Image: image,
		AuditLogReason: api.AuditLogReason(
			"emoji created by " + GetUserMention(int64(c.e.Author.ID)),
		),
	}

	if emoji, err := discordClient.CreateEmoji(c.e.GuildID, createEmojiData); err != nil {
		// error with uploading
		return GenericError("StealEmojiCommand", "uploading emoji", err.Error())
	} else {
		// uploaded successfully, send a nice embed
		_, err := discordClient.SendMessage(
			c.e.ChannelID,
			emoji.String(),
			discord.Embed{Title: "Emoji stolen ;)", Color: successColor},
		)
		return err
	}
}

func (c Command) TopicCommand() error {
	topic, argErr := ParseAllArgs(c.args)
	if argErr != nil {
		return argErr
	}

	topicsEnabled := false
	GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
		topicsEnabled = util.SliceContains(g.EnabledTopicChannels, int64(c.e.ChannelID))
		return g, "TopicCommand: check topicsEnabled"
	})

	if !topicsEnabled {
		_, err := SendEmbed(c, "Topics are disabled in this channel!", "", errorColor)
		return err
	}

	msg, err := SendEmbed(c, "New topic suggested!", c.e.Author.Mention()+" suggests: "+topic, defaultColor)
	if err != nil {
		return err
	}

	emoji, err := GuildTopicVoteApiEmoji(c.e.GuildID)
	if err != nil {
		return err
	}

	GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
		g.ActiveTopicVotes = append(g.ActiveTopicVotes, ActiveTopicVote{int64(msg.ID), int64(c.e.Author.ID), topic})
		return g, "TopicCommand: append ActiveTopicVotes"
	})

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
			GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
				if g.ArchiveRole == 0 {
					err = GenericError(c.fnName, "getting archive role", "`archive_role` not set in guild config")
				}
				if g.ArchiveCategory == 0 {
					err = GenericError(c.fnName, "getting archive category", "`archive_category` not set in guild config")
				}
				return g, "ChannelCommand: check archive permission"
			})

			if err != nil {
				return err
			}

			channel, err := discordClient.Channel(c.e.ChannelID)
			if err != nil {
				return err
			}

			overwrites := make([]discord.Overwrite, 0)
			var data api.ModifyChannelData

			GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
				// Copy everything except the archive and @everyone roles to overwrites
				for _, overwrite := range channel.Overwrites {
					id := int64(overwrite.ID)
					if id != int64(c.e.GuildID) && id != g.ArchiveRole {
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
						ID:    discord.Snowflake(g.ArchiveRole),
						Type:  discord.OverwriteRole,
						Allow: discord.PermissionViewChannel,
					},
				)
				data = api.ModifyChannelData{Overwrites: &overwrites, CategoryID: discord.ChannelID(g.ArchiveCategory)}

				return g, "ChannelCommand: create overwrites data"
			})

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
	case "topic": // TODO: This is terrible. Future me please re-write.
		err := HasPermission("channels", c)
		if err == nil {
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
					GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
						for _, channel := range channels {
							if !util.SliceContains(g.EnabledTopicChannels, channel) {
								g.EnabledTopicChannels = append(g.EnabledTopicChannels, channel)
							}
						}
						return g, "ChannelCommand: topic enable"
					})
					_, err := SendEmbed(c, "Channel Topic", "✅ Added "+channelsStr+" to the allowed topic channels", successColor)
					return err
				case "disable":
					GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
						for _, channel := range channels {
							if util.SliceContains(g.EnabledTopicChannels, channel) {
								g.EnabledTopicChannels = util.SliceRemove(g.EnabledTopicChannels, channel)
							}
						}
						return g, "ChannelCommand: topic disable"
					})
					_, err := SendEmbed(c, "Channel Topic", "⛔ Removed "+channelsStr+" from the allowed topic channels", errorColor)
					return err
				case "emoji":
					arg3, animated, err3 := ParseEmojiArg(c.args, 3, true)
					if err3 != nil {
						return err3
					}

					if arg3 == nil {
						if emoji, err := GuildTopicVoteEmoji(c.e.GuildID); err != nil {
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

						GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
							g.TopicVoteEmoji = configEmoji
							return g, "ChannelCommand: update TopicVoteEmoji"
						})

						_, err = SendEmbed(c, "Set Topic Vote Emoji To:", emoji, successColor)
						return err
					}
				case "threshold":
					arg3, err3 := ParseInt64Arg(c.args, 3)
					if err3 != nil {
						return err3
					}

					GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
						if arg3 <= 0 {
							arg3 = 3
						}

						g.TopicVoteThreshold = arg3
						return g, "ChannelCommand: update topic vote threshold"
					})

					_, err := SendEmbed(c, "Set Topic Vote Threshold To:", strconv.FormatInt(arg3, 10), successColor)
					return err
				default:
					noTopicChan := false
					formattedChannels := ""

					GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
						formattedChannels = JoinInt64Slice(g.EnabledTopicChannels, "\n", "✅ <#", ">")
						noTopicChan = len(g.EnabledTopicChannels) == 0
						return g, "ChannelCommand: get enabled topic channels"
					})

					if noTopicChan {
						_, err := SendEmbed(c, "Channel Topic", "There are currently no allowed topic channels", defaultColor)
						return err
					}

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
				arg3, errParse := ParseChannelArg(c.args, 3)
				var err error = nil

				GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
					switch arg2 {
					case "regular":
						if errParse != nil {
							g.Starboard.Channel = 0
							_, err = SendEmbed(c, "Starboard Channels", "⛔ Disabled regular starboard", errorColor)
							return g, "ChannelCommand: enable regular starboard"
						} else {
							g.Starboard.Channel = arg3
							_, err = SendEmbed(c, "Starboard Channels", "✅ Enabled regular starboard", successColor)
							return g, "ChannelCommand: disable regular starboard"
						}
					case "nsfw":
						if errParse != nil {
							g.Starboard.NsfwChannel = 0
							_, err = SendEmbed(c, "Starboard Channels", "⛔ Disabled NSFW starboard", errorColor)
							return g, "ChannelCommand: enable nsfw starboard"
						} else {
							g.Starboard.NsfwChannel = arg3
							_, err = SendEmbed(c, "Starboard Channels", "✅ Enabled NSFW starboard", successColor)
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
							Color:       defaultColor,
						}
						_, err = SendCustomEmbed(c.e.ChannelID, embed)
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

	// Prefix is okay, set it in the cache
	//

	config.run(func(config *Config) {
		config.PrefixCache[int64(c.e.GuildID)] = arg
	})

	// Also set it in the guild
	//

	GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
		g.Prefix = arg
		return g, "PrefixCommand"
	})

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

		embed := makeEmbed("Pong!", "Latency is "+strconv.FormatInt(curTime-msgTime, 10)+"ms", successColor)
		_, err = discordClient.EditMessage(msg.ChannelID, msg.ID, "", embed)
		return err
	}
}

func (c Command) FrogCommand() error {
	frogData, _, err := RequestUrl("https://frog.pics/api/random", http.MethodGet)
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

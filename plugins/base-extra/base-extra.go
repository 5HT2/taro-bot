package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
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
			Fn:          ProfilePicCommand,
			FnName:      "ProfilePicCommand",
			Name:        "profilepic",
			Aliases:     []string{"pfp"},
			Description: "Get the profile picture of someone",
		}},
		Responses: []bot.ResponseInfo{},
	}
}

func ChannelCommand(c bot.Command) error {
	arg1, _ := cmd.ParseStringArg(c.Args, 1, true)
	arg2, _ := cmd.ParseStringArg(c.Args, 2, true)

	defaultResponse := func() error {
		_, err := cmd.SendEmbed(c.E,
			"Channel",
			"Available arguments are:\n- `archive`\n- `archive role|category [role id|category id]`\n- `slow [seconds]`",
			bot.DefaultColor)
		return err
	}

	switch arg1 {
	case "archive":
		err := cmd.HasPermission("channels", c)
		if err == nil {
			switch arg2 {
			case "role":
				var errCtx error
				role, err := cmd.ParseInt64Arg(c.Args, 3)
				bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {

					if err != nil {
						set := fmt.Sprintf("currently set to <@&%v>!", g.ArchiveRole)
						setColor := bot.DefaultColor
						if g.ArchiveRole == 0 {
							set = "not set."
							setColor = bot.WarnColor
						}
						_, errCtx = cmd.SendEmbed(c.E, "Channel Archive Role", set, setColor)
					} else {
						g.ArchiveRole = role
						_, errCtx = cmd.SendEmbed(c.E, "Channel Archive Role", fmt.Sprintf("Set to <@&%v>!", role), bot.SuccessColor)
					}
					return g, "ChannelCommand: set guild role"
				})
				return errCtx
			case "category":
				var errCtx error
				category, err := cmd.ParseInt64Arg(c.Args, 3)
				bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
					if err != nil {
						set := fmt.Sprintf("currently set to <#%v>!", g.ArchiveCategory)
						setColor := bot.DefaultColor
						if g.ArchiveCategory == 0 {
							set = "not set."
							setColor = bot.WarnColor
						}
						_, errCtx = cmd.SendEmbed(c.E, "Channel Archive Category", set, setColor)
					} else {
						g.ArchiveCategory = category
						_, errCtx = cmd.SendEmbed(c.E, "Channel Archive Category", fmt.Sprintf("Set to <#%v>!", category), bot.SuccessColor)
					}
					return g, "ChannelCommand: set guild role"
				})
				return errCtx
			case "":
				bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
					if g.ArchiveCategory == 0 {
						err = bot.GenericError(c.FnName, "getting archive category", "`archive_category` not set, use `archive category [category id]`")
					}
					if g.ArchiveRole == 0 {
						err = bot.GenericError(c.FnName, "getting archive role", "`archive_role` not set, use `archive role [role id]`")
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
					_, err = cmd.SendEmbed(c.E, "Channel Archive", "Successfully archived channel", bot.SuccessColor)
					return err
				}
			default:
				return defaultResponse()
			}
		} else {
			return err
		}
	case "slow":
		seconds, _ := cmd.ParseInt64Arg(c.Args, 2)
		channelID := c.E.ChannelID

		if channel, err := cmd.ParseChannelArg(c.Args, 2); err == nil {
			channelID = discord.ChannelID(channel)
			seconds, _ = cmd.ParseInt64Arg(c.Args, 3)
		}

		if seconds < 0 { // normalize to 0-21600
			seconds = 0
		} else if seconds > 21600 {
			seconds = 21600
		}

		data := api.ModifyChannelData{UserRateLimit: option.NewNullableUint(uint(seconds))}
		if err := bot.Client.ModifyChannel(channelID, data); err != nil {
			return err
		} else {
			message := fmt.Sprintf("Set slowmode to %v!", util.FormattedTime(seconds))
			if seconds == 0 {
				message = "Cleared slowmode!"
			}
			if channelID != c.E.ChannelID {
				message = fmt.Sprintf("Set slowmode in <#%v> to %v!", channelID, util.FormattedTime(seconds))
				if seconds == 0 {
					message = fmt.Sprintf("Cleared slowmode in <#%v>!", channelID)
				}
			}
			_, err = cmd.SendEmbed(c.E, "Channel Slow", message, bot.SuccessColor)
			return err
		}
	default:
		return defaultResponse()
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

func ProfilePicCommand(c bot.Command) error {
	self := false
	id, argErr := cmd.ParseInt64Arg(c.Args, 1)
	if argErr != nil {
		id, argErr = cmd.ParseUserArg(c.Args, 1)
		if argErr != nil {
			self = true
			id = int64(c.E.Author.ID)
		}
	}

	url := ""
	name := c.E.Author.Username

	// if command is being run inside a DM
	if c.E.Member != nil {
		name = c.E.Member.Nick
	}

	if self {
		url = c.E.Author.AvatarURLWithType(discord.AutoImage)
	} else {
		user, err := bot.Client.User(discord.UserID(id))
		if err != nil {
			return err
		}
		url = user.AvatarURLWithType(discord.AutoImage)
		name = user.Username
	}

	url += "?size=2048"

	e := discord.Embed{
		Title: name,
		URL:   url,
		Image: &discord.EmbedImage{URL: url},
		Color: bot.WhiteColor,
	}
	_, err := cmd.SendCustomEmbed(c.E.ChannelID, e)
	return err
}

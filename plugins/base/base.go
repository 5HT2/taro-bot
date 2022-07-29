package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/diamondburned/arikawa/v3/discord"
	"strconv"
	"strings"
	"time"
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	return &plugins.Plugin{
		Name:        "Taro Base",
		Description: "The base commands and responses included as part of the bot",
		Version:     "1.0.1",
		Commands: []bot.CommandInfo{{
			Fn:          InviteCommand,
			FnName:      "InviteCommand",
			Name:        "invite",
			Description: "Invite the bot to your own server!",
		}, {
			Fn:          HelpCommand,
			FnName:      "HelpCommand",
			Name:        "help",
			Aliases:     []string{"h"},
			Description: "Print a list of available commands",
		}, {
			Fn:          OperatorConfigCommand,
			FnName:      "OperatorConfigCommand",
			Name:        "operatorconfig",
			Aliases:     []string{"opcfg"},
			Description: "Allows the bot operator to configure bot-level settings",
		}, {
			Fn:          PingCommand,
			FnName:      "PingCommand",
			Name:        "ping",
			Description: "Returns the current API latency",
		}, {
			Fn:          PrefixCommand,
			FnName:      "PrefixCommand",
			Name:        "prefix",
			Description: "Set the bot prefix for your guild",
			GuildOnly:   true,
		}, {
			Fn:          ResponseConfigCommand,
			FnName:      "ResponseConfigCommand",
			Name:        "responseconfig",
			Aliases:     []string{"rcfg"},
			Description: "Manage responses for your guild",
			GuildOnly:   true,
		}},
		Responses: []bot.ResponseInfo{{
			Fn:       PrefixResponse,
			Regexes:  []string{"<@!?DISCORD_BOT_ID>", "(prefix|help)"},
			MatchMin: 2,
		}},
	}
}

func ResponseConfigCommand(c bot.Command) error {

	return nil
}

func OperatorConfigCommand(c bot.Command) error {
	arg1, _ := cmd.ParseStringArg(c.Args, 1, true)
	args, _ := cmd.ParseStringSliceArg(c.Args, 2, -1)
	argInt, _ := cmd.ParseInt64Arg(c.Args, 2)
	joinedArgs := strings.Join(args, " ")

	t := "Operator Config "
	var err error

	var opID int64 = 0
	bot.C.Run(func(c *bot.Config) {
		opID = c.OperatorID
	})

	if c.E.Author.ID == 0 || int64(c.E.Author.ID) != opID {
		return bot.GenericError("OperatorConfigCommand", "running command", "user is not the bot operator")
	}

	switch arg1 {
	case "activity_name":
		bot.C.Run(func(co *bot.Config) {
			if len(args) == 0 {
				_, err = cmd.SendEmbed(c.E, t+"`activity_name`", fmt.Sprintf("The current `activity_name` is\n```\n%s\n```", co.ActivityName), bot.DefaultColor)
			} else {
				co.ActivityName = joinedArgs
				_, err = cmd.SendEmbed(c.E, t+"`activity_name`", fmt.Sprintf("Set `activity_name` to\n```\n%s\n```", co.ActivityName), bot.SuccessColor)
			}
		})
	case "activity_url":
		bot.C.Run(func(co *bot.Config) {
			if len(args) == 0 {
				_, err = cmd.SendEmbed(c.E, t+"`activity_url`", fmt.Sprintf("The current `activity_url` is\n```\n%s\n```", co.ActivityUrl), bot.DefaultColor)
			} else {
				co.ActivityUrl = joinedArgs
				_, err = cmd.SendEmbed(c.E, t+"`activity_url`", fmt.Sprintf("Set `activity_url` to\n```\n%s\n```", co.ActivityUrl), bot.SuccessColor)
			}
		})
	case "activity_type":
		bot.C.Run(func(co *bot.Config) {
			if argInt == -1 {
				_, err = cmd.SendEmbed(c.E, t+"`activity_type`", fmt.Sprintf("The current `activity_type` is `%v`", co.ActivityType), bot.DefaultColor)
			} else {
				co.ActivityType = uint8(argInt)
				_, err = cmd.SendEmbed(c.E, t+"`activity_type`", fmt.Sprintf("Set `activity_type` to `%v`", co.ActivityType), bot.SuccessColor)
			}
		})
	case "operator_channel":
		bot.C.Run(func(co *bot.Config) {
			if argInt == -1 {
				_, err = cmd.SendEmbed(c.E, t+"`operator_channel`", fmt.Sprintf("The current `operator_channel` is `%v`", co.OperatorChannel), bot.DefaultColor)
			} else {
				co.OperatorChannel = argInt
				_, err = cmd.SendEmbed(c.E, t+"`operator_channel`", fmt.Sprintf("Set `operator_channel` to `%v`", co.OperatorChannel), bot.SuccessColor)
			}
		})
	case "operator_id":
		bot.C.Run(func(co *bot.Config) {
			if argInt == -1 {
				_, err = cmd.SendEmbed(c.E, t+"`operator_id`", fmt.Sprintf("The current `operator_id` is `%v`", co.OperatorID), bot.DefaultColor)
			} else {
				co.OperatorID = argInt
				_, err = cmd.SendEmbed(c.E, t+"`operator_id`", fmt.Sprintf("Set `operator_id` to `%v`", co.OperatorID), bot.SuccessColor)
			}
		})
	default:
		_, err = cmd.SendEmbed(c.E,
			"Operator Config",
			"Available arguments are:\n- `activity_name [activity name]`\n- `activity_url [activity url]`\n- `activity_type [activity type]`\n- `operator_channel [operator channel id]`\n- `operator_id [operator id]`",
			bot.DefaultColor)
	}

	bot.LoadActivityStatus() // Apply changes to activity
	return err
}

func HelpCommand(c bot.Command) error {
	fmtCmds := make([]string, 0)
	for _, command := range bot.Commands {
		// Filter GuildOnly commands when not in a guild
		if !command.GuildOnly || c.E.GuildID.IsValid() {
			fmtCmds = append(fmtCmds, command.MarkdownString())
		}
	}

	_, err := cmd.SendEmbed(c.E,
		"Taro Help",
		strings.Join(fmtCmds, "\n\n"),
		bot.DefaultColor)
	return err
}

func InviteCommand(c bot.Command) error {
	_, err := cmd.SendEmbed(c.E,
		bot.User.Username+" invite", fmt.Sprintf("[Click to add me to your own server!](https://discord.com/oauth2/authorize?client_id=%v&permissions=%v&scope=bot)", bot.User.ID, bot.PermissionsHex),
		bot.SuccessColor,
	)
	return err
}

func PingCommand(c bot.Command) error {
	if msg, err := cmd.SendEmbed(c.E,
		"Ping!",
		"Waiting for API response...",
		bot.DefaultColor); err != nil {
		return err
	} else {
		curTime := time.Now().UnixMilli()
		msgTime := msg.Timestamp.Time().UnixMilli()

		embed := cmd.MakeEmbed("Pong!", "Latency is "+strconv.FormatInt(curTime-msgTime, 10)+"ms", bot.SuccessColor)
		_, err = bot.Client.EditMessage(msg.ChannelID, msg.ID, "", embed)
		return err
	}
}

func PrefixCommand(c bot.Command) error {
	arg, argErr := cmd.ParseStringArg(c.Args, 1, false)
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
		Description: "Set prefix to `" + arg + "`",
		Footer:      &discord.EmbedFooter{Text: "At any time you can ping the bot with the word \"prefix\" to get the current prefix"},
		Color:       bot.SuccessColor,
	}
	_, err := cmd.SendCustomEmbed(c.E.ChannelID, embed)
	return err
}

func PrefixResponse(r bot.Response) {
	if !r.E.GuildID.IsValid() {
		_, _ = cmd.SendEmbed(r.E, "", "Commands in DMs don't use a prefix!\nUse `help` for a list of commands.", bot.DefaultColor)
		return
	}

	prefix := bot.DefaultPrefix
	bot.GuildContext(r.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		prefix = g.Prefix
		return g, "PrefixResponse"
	})

	_, _ = cmd.SendEmbed(r.E, "", fmt.Sprintf("The current prefix is `%s`\nUse `%shelp` for a list of commands.", prefix, prefix), bot.DefaultColor)
}

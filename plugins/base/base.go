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
			Description: "Print a list of available commands",
			Aliases:     []string{"h"},
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
		}},
		Responses: []bot.ResponseInfo{{
			Fn:       PrefixResponse,
			Regexes:  []string{"<@!?DISCORD_BOT_ID>", "prefix"},
			MatchMin: 2,
		}},
	}
}

func HelpCommand(c bot.Command) error {
	fmtCmds := make([]string, 0)
	for _, command := range bot.Commands {
		fmtCmds = append(fmtCmds, command.MarkdownString())
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
		Description: "Set prefix to `" + arg + "`.",
		Footer:      &discord.EmbedFooter{Text: "At any time you can ping the bot with the word \"prefix\" to get the current prefix"},
		Color:       bot.SuccessColor,
	}
	_, err := cmd.SendCustomEmbed(c.E.ChannelID, embed)
	return err
}

func PrefixResponse(r bot.Response) {
	prefix := bot.DefaultPrefix
	bot.GuildContext(r.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		prefix = g.Prefix
		return g, "PrefixResponse"
	})

	_, _ = cmd.SendEmbed(r.E, "", fmt.Sprintf("The current prefix is `%s`", prefix), bot.DefaultColor)
}

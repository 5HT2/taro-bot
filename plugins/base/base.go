package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
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
		}},
		Responses: []bot.ResponseInfo{},
	}
}

func InviteCommand(c bot.Command) error {
	_, err := cmd.SendEmbed(c,
		bot.User.Username+" invite", fmt.Sprintf("[Click to add me to your own server!](https://discord.com/oauth2/authorize?client_id=%v&permissions=%v&scope=bot)", bot.User.ID, bot.PermissionsHex),
		bot.SuccessColor,
	)
	return err
}

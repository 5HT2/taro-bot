package main

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"log"
	"regexp"
)

var (
	tenorRegex = regexp.MustCompile(`http(s)?://t([ex])nor\.[A-z]+/view/.*`)
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	return &plugins.Plugin{
		Name:        "Tenor Delete",
		Description: "Automatically delete tenor gifs",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          TenorDeleteCommand,
			FnName:      "TenorDeleteCommand",
			Name:        "tenordelete",
			Description: "Toggle tenor deletion on or off"},
		},
		Responses: []bot.ResponseInfo{{
			Fn:       TenorDeleteResponse,
			Regexes:  []string{tenorRegex.String()},
			MatchMin: 1,
		}},
	}
}

func TenorDeleteResponse(r bot.Response) {
	bot.GuildContext(r.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		if util.SliceContains(g.EnabledTenorDelete, int64(r.E.GuildID)) {
			if err := bot.Client.DeleteMessage(r.E.ChannelID, r.E.Message.ID, "Matched Tenor gif"); err != nil {
				log.Printf("TenorDeleteResponse: %v\n", err)
			}
		}

		return g, "TenorDeleteResponse"
	})
}

func TenorDeleteCommand(c bot.Command) error {
	if err := cmd.HasPermission("moderate", c); err != nil {
		return err
	}

	var err error = nil

	bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		id := int64(c.E.GuildID)

		if util.SliceContains(g.EnabledTenorDelete, id) {
			g.EnabledTenorDelete = util.SliceRemove(g.EnabledTenorDelete, id)
			_, err = cmd.SendEmbed(c.E, "Tenor Delete", "⛔ Disabled Tenor Delete for this guild", bot.ErrorColor)
		} else {
			g.EnabledTenorDelete = append(g.EnabledTenorDelete, id)
			_, err = cmd.SendEmbed(c.E, "Tenor Delete", "✅ Enabled Tenor Delete for this guild", bot.SuccessColor)
		}

		return g, "TenorDeleteCommand"
	})

	return err
}

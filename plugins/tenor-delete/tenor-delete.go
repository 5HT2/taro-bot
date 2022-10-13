package main

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"log"
	"reflect"
	"regexp"
	"sync"
)

var (
	p          *plugins.Plugin
	mutex      sync.Mutex
	tenorRegex = regexp.MustCompile(`http(s)?://t([ex])nor\.[A-z]+/view/.*`)
)

type config struct {
	Guilds map[string]bool `json:"guilds,omitempty"` // [guild id]enabled
}

func InitPlugin(i *plugins.PluginInit) *plugins.Plugin {
	p = &plugins.Plugin{
		Name:        "Tenor Delete",
		Description: "Automatically delete tenor gifs",
		Version:     "1.0.0",
		ConfigType:  reflect.TypeOf(config{}),
		Commands: []bot.CommandInfo{{
			Fn:          TenorDeleteCommand,
			FnName:      "TenorDeleteCommand",
			Name:        "tenordelete",
			Description: "Toggle tenor deletion on or off",
			GuildOnly:   true,
		}},
		Responses: []bot.ResponseInfo{{
			Fn:       TenorDeleteResponse,
			Regexes:  []string{tenorRegex.String()},
			MatchMin: 1,
		}},
	}
	p.ConfigDir = i.ConfigDir
	p.Config = p.LoadConfig()
	return p
}

func TenorDeleteResponse(r bot.Response) {
	mutex.Lock()
	defer mutex.Unlock()

	if p.Config == nil {
		return
	}

	if enabled, ok := p.Config.(config).Guilds[r.E.GuildID.String()]; ok && enabled {
		if err := bot.Client.DeleteMessage(r.E.ChannelID, r.E.Message.ID, "Matched Tenor gif"); err != nil {
			log.Printf("TenorDeleteResponse: %v\n", err)
		}
	}
}

func TenorDeleteCommand(c bot.Command) error {
	if err := cmd.HasPermission(c, cmd.PermModerate); err != nil {
		return err
	}

	id := c.E.GuildID.String()
	var err error = nil

	mutex.Lock()
	defer mutex.Unlock()

	if p.Config == nil {
		p.Config = config{Guilds: map[string]bool{id: false}}
	}

	enabled, _ := p.Config.(config).Guilds[id]
	p.Config.(config).Guilds[id] = !enabled

	if !enabled {
		_, err = cmd.SendEmbed(c.E, "Tenor Delete", "✅ Enabled Tenor Delete for this guild", bot.SuccessColor)
	} else {
		_, err = cmd.SendEmbed(c.E, "Tenor Delete", "⛔ Disabled Tenor Delete for this guild", bot.ErrorColor)
	}

	return err
}

package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/go-co-op/gocron"
	"reflect"
)

var p *plugins.Plugin

type config struct {
	Guilds map[string]map[string]ChannelConfig `json:"guilds,omitempty"` // [guild id][channel id]ChannelConfig
}

type ChannelConfig struct {
	MaxMessages int64 `json:"max_messages,omitempty"`
	MaxHours    int64 `json:"max_time,omitempty"`
}

func InitPlugin(i *plugins.PluginInit) *plugins.Plugin {
	p = &plugins.Plugin{
		Name:        "AutoDelete",
		Description: "AutoDelete messages in a channel after a certain number of messages or time has passed",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          AutoDeleteCommand,
			FnName:      "AutoDeleteCommand",
			Name:        "autodelete",
			Description: "Configure AutoDelete",
		}},
		Jobs: []bot.JobInfo{{
			Fn: func() (*gocron.Job, error) {
				return bot.Scheduler.Every(10).Minutes().Do(purgeMessages)
			},
			Name: "autodelete-check-for-time-purge",
		}},
		ConfigType: reflect.TypeOf(config{}),
	}
	p.ConfigDir = i.ConfigDir
	p.Config = p.LoadConfig()
	return p
}

func AutoDeleteCommand(c bot.Command) error {
	if err := cmd.HasPermission(c, cmd.PermChannels); err != nil {
		return err
	}

	defaultHelp := func() error {
		_, err := cmd.SendEmbed(c.E,
			"Configure AutoDelete",
			"Available arguments are:\n- `toggle [channel]`\n- `hours [channel] [max hours]`\n- `messages [channel] [max messages]`",
			bot.DefaultColor)
		return err
	}

	sub, _ := cmd.ParseStringArg(c.Args, 1, true)
	if sub != "toggle" && sub != "hours" && sub != "messages" {
		return defaultHelp()
	}

	if channelArg, err := cmd.ParseChannelArg(c.Args, 2); err != nil {
		return err
	} else {
		if channel, err := bot.Client.Channel(discord.ChannelID(channelArg)); err != nil {
			return err
		} else if channel.GuildID != c.E.GuildID {
			return bot.GenericError(c.FnName, "getting channel", "channel not in current guild!")
		} else {
			var err error
			cfg := getConfig(c.E.GuildID.String(), channel.ID.String())

			switch sub {
			case "toggle":
				if cfg.MaxHours == 0 && cfg.MaxMessages == 0 {
					cfg.MaxHours = 24
					cfg.MaxMessages = 1000
					_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Set AutoDelete in <#%v> to after 24 hours or 1,000 messages!", channel.ID), bot.SuccessColor)
				} else {
					cfg.MaxHours = 0
					cfg.MaxMessages = 0
					_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Disabled Autodelete in <#%v>!", channel.ID), bot.ErrorColor)
				}
			case "hours":
				fallthrough // TODO
			case "messages":
				fallthrough // TODO
			default:
				return defaultHelp()
			}

			saveConfig(c.E.GuildID.String(), channel.ID.String(), cfg)
			return err
		}
	}
}

func purgeMessages() {}

func saveConfig(gID, cID string, cfg ChannelConfig) {
	if p.Config == nil {
		p.Config = config{Guilds: map[string]map[string]ChannelConfig{gID: {cID: {}}}}
	}

	gCfg, ok := p.Config.(*config).Guilds[gID]
	if !ok {
		gCfg = map[string]ChannelConfig{cID: cfg}
		p.Config.(*config).Guilds[gID] = gCfg
	} else {
		p.Config.(*config).Guilds[gID][cID] = cfg
	}
}

func getConfig(gID, cID string) ChannelConfig {
	if p.Config == nil {
		p.Config = config{Guilds: map[string]map[string]ChannelConfig{gID: {cID: {}}}}
	}

	gCfg, ok := p.Config.(*config).Guilds[gID]
	if !ok {
		gCfg = map[string]ChannelConfig{cID: {}}
	}
	cfg, ok := gCfg[cID]
	if !ok {
		cfg = ChannelConfig{}
	}

	p.Config.(*config).Guilds[gID][cID] = cfg

	return cfg
}

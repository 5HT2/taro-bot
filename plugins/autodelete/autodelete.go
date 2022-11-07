package main

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/go-co-op/gocron"
	"reflect"
)

var p *plugins.Plugin

type config struct {
	Guilds map[discord.GuildID]ChannelConfig `json:"guilds,omitempty"` // [guild id]ChannelConfig
}

type ChannelConfig struct {
	ChannelData map[int64]ChannelDataConfig `json:"channel,omitempty"` // [channel id]ChannelDataConfig
}

type ChannelDataConfig struct {
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

	if cID, err := cmd.ParseChannelArg(c.Args, 2); err != nil {
		return err
	} else {
		if channel, err := bot.Client.Channel(discord.ChannelID(cID)); err != nil {
			return err
		} else if channel.GuildID != c.E.GuildID {
			return bot.GenericError(c.FnName, "getting channel", "channel not in current guild!")
		} else {
			switch sub {
			case "toggle":
				fallthrough // TODO
			case "hours":
				fallthrough // TODO
			case "messages":
				fallthrough // TODO
			default:
				return defaultHelp()
			}
		}
	}
}

func purgeMessages() {}

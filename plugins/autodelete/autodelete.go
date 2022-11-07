package main

import (
	"fmt"
	"reflect"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/go-co-op/gocron"
)

var p *plugins.Plugin

type config struct {
	Guilds map[discord.GuildID]ChannelConfig `json:"guilds,omitempty"` // [guild id]ChannelConfig
}

type ChannelConfig struct {
	ChannelData map[int64]ChannelDataConfig `json:"channel,omitempty"` // [channel id]ChannelDataConfig
}

type ChannelDataConfig struct {
	MaxMessages     int64          `json:"max_messages,omitempty"`
	MaxTimeHours    int64          `json:"max_time,omitempty"`
	HoursTilPurge   int64		   `json:"hours_til_purge,omitempty"`
}

func InitPlugin(i *plugins.PluginInit) *plugins.Plugin {
	p = &plugins.Plugin{
		Name:        "Autodelete",
		Description: "Autodelete messages in a channel after a certain number of messages or time has passed",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          AutoDeleteCommand,
			FnName:      "AutoDeleteCommand",
			Name:        "autodelete",
			Description: "Configure channel autodeleting",
		}},
		Jobs: []bot.JobInfo{{
			Fn: func() (*gocron.Job, error) {
				return bot.Scheduler.Every(1).Hour().Do(DoPurges)
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

	if arg, err := cmd.ParseStringArg(c.Args, 1, true); err != nil {
		return err
	} else {
		arg2, errParse := cmd.ParseChannelArg(c.Args, 2)
		var err error = nil

		bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
			switch arg {
			case "toggle":
				if errParse != nil {

					if val, ok := p.Config.(*config).Guilds[c.E.GuildID].ChannelData[arg2]; !ok {
						p.Config.(*config).Guilds[c.E.GuildID].ChannelData[arg2] = ChannelDataConfig{
							MaxMessages: 100,
							MaxTimeHours: 24,
							HoursTilPurge: 24,
						}
						formatted := fmt.Sprintf("Enabled AutoDelete for Channel <#%d>.\nChannel will purge after %d Messages or %d hours, " + 
						"whichever comes first.", arg2, val.MaxMessages, val.MaxTimeHours)
						_, err = cmd.SendEmbed(c.E, c.Name, formatted, bot.DefaultColor)
						return g, "Autodelete: enable autodelete"
					} else {
						delete(p.Config.(*config).Guilds[c.E.GuildID].ChannelData, arg2)
						_, err = cmd.SendEmbed(c.E, c.Name, fmt.Sprintf("Disabled AutoDelete for Channel <#%d>", arg2), bot.DefaultColor)
						return g, "Autodelete: disable autodelete"
					}
				}
			case "sethours":
				if arg3, errParse := cmd.ParseInt64Arg(c.Args, 3); errParse != nil {
					if val, ok := p.Config.(*config).Guilds[c.E.GuildID].ChannelData[arg2]; !ok {
						val.HoursTilPurge = arg3
						p.Config.(*config).Guilds[c.E.GuildID].ChannelData[arg2] = val
						formatted := fmt.Sprintf("Set Hours until purge for Channel <#%d> to %d hours.", arg2, val.MaxTimeHours)
						_, err = cmd.SendEmbed(c.E, c.Name, formatted, bot.DefaultColor)
						return g, "Autodelete: set hours until purge"
					} else {
						_, err = cmd.SendEmbed(c.E, c.Name, fmt.Sprintf("AutoDelete is not enabled for Channel <#%d>", arg2), bot.ErrorColor)
						return g, "Autodelete: failed to set hours, not enabled for channel"
					}
				}
			case "setmaxmessages":
				if arg3, errParse := cmd.ParseInt64Arg(c.Args, 3); errParse != nil {
					if val, ok := p.Config.(*config).Guilds[c.E.GuildID].ChannelData[arg2]; !ok {
						val.MaxMessages = arg3
						p.Config.(*config).Guilds[c.E.GuildID].ChannelData[arg2] = val
						formatted := fmt.Sprintf("Set Max messages until purge for Channel <#%d> to %d messages ", arg2, val.MaxMessages)
						_, err = cmd.SendEmbed(c.E, c.Name, formatted, bot.DefaultColor)
						return g, "Autodelete: set max messages until purge"
					} else {
						_, err = cmd.SendEmbed(c.E, c.Name, fmt.Sprintf("AutoDelete is not enabled for Channel <#%d>", arg2), bot.ErrorColor)
						return g, "Autodelete: failed to set max messages, not enabled for channel"
					}
				}
			default:
				_, err = cmd.SendEmbed(c.E,
					"Autodelete",
					"TODO",
					bot.DefaultColor)
					return g, "Autodelete: showed help"
			}
			return g, "Theoretically impossible" // TODO: I looked at starboard, and i couldn't find why it didn't need this but mine does...
		})
		return err
	}
}

func DoPurges() {}
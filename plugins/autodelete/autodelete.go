package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/go-co-op/gocron"
	"reflect"
	"sync"
)

var (
	p     *plugins.Plugin
	mutex sync.Mutex

	additionQueue = make(chan QueuedMessage, 0) // For incoming messages, to be checked for deletion
	deletionQueue = make(chan QueuedMessage, 0) // For messages marked for deletion

	additionQueueProcessors = make(map[string]map[string]func(ChannelConfig)) // Processes messages in the additionQueue
	deletionQueueProcessors = make(map[string]map[string]func(ChannelConfig)) // Processes messages in the deletionQueue
)

type config struct {
	Guilds map[string]map[string]ChannelConfig `json:"guilds,omitempty"` // [guild id][channel id]ChannelConfig
}

type ChannelConfig struct {
	MaxMessages int64 `json:"max_messages,omitempty"`
	MaxHours    int64 `json:"max_time,omitempty"`
}

type QueuedMessage struct {
	Guild   int64  `json:"guild"`
	Channel int64  `json:"channel"`
	Message int64  `json:"message"`
	Reason  string `json:"reason,omitempty"`
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
			Aliases:     []string{"adl"},
			Description: "Configure AutoDelete",
		}},
		Responses: []bot.ResponseInfo{{
			Fn:       AutoDeleteResponse,
			Regexes:  []string{"."},
			MatchMin: 1,
		}},
		Jobs: []bot.JobInfo{{
			Fn: func() (*gocron.Job, error) {
				return bot.Scheduler.Every(10).Minutes().Do(purgeMessages)
			},
			Name: "autodelete-check-for-time-purge",
		}, {
			Fn: func() (*gocron.Job, error) {
				return bot.Scheduler.Every(5).Minutes().Do(saveQueue)
			},
			Name: "autodelete-save-message-queue",
		}},
		ConfigType: reflect.TypeOf(config{}),
	}
	p.ConfigDir = i.ConfigDir
	p.Config = p.LoadConfig()
	return p
}

func AutoDeleteResponse(r bot.Response) {
	additionQueue <- QueuedMessage{Guild: int64(r.E.GuildID), Channel: int64(r.E.ChannelID), Message: int64(r.E.ID)}
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
		if len(c.Args) < 3 { // Re-call same command with current channel if unspecified, and if we haven't already tried adding a channel arg
			c.Args = append(c.Args[:2], c.Args[1:]...)
			c.Args[1] = c.E.ChannelID.Mention()
			cmd.CommandHandlerWithCommand(c.E, c.Name, c.Args)
			return nil
		}

		return err
	} else {
		if channel, err := bot.Client.Channel(discord.ChannelID(channelArg)); err != nil {
			return err
		} else if channel.GuildID != c.E.GuildID {
			return bot.GenericError(c.FnName, "getting channel", "channel not in current guild!")
		} else {
			mutex.Lock()
			defer mutex.Unlock()

			var err error
			cfg := getConfig(c.E.GuildID.String(), channel.ID.String())
			arg3, argErr := cmd.ParseInt64Arg(c.Args, 3)
			if arg3 < 0 {
				arg3 = 0
			}

			switch sub {
			case "toggle":
				if cfg.MaxHours == 0 && cfg.MaxMessages == 0 {
					cfg.MaxHours = 24
					cfg.MaxMessages = 1000
					_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Set AutoDelete in %s to after 24 hours or 1,000 messages!", channel.Mention()), bot.SuccessColor)
				} else {
					cfg.MaxHours = 0
					cfg.MaxMessages = 0
					_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Disabled Autodelete in %s!", channel.Mention()), bot.ErrorColor)
				}

				saveConfig(c.E.GuildID.String(), channel.ID.String(), cfg)
			case "hours":
				cfg.MaxHours = arg3
			case "messages":
				cfg.MaxMessages = arg3
			default:
				return defaultHelp()
			}

			if sub == "hours" || sub == "messages" {
				if argErr != nil {
					cfg = getConfig(c.E.GuildID.String(), channel.ID.String())
				}

				embed := discord.Embed{
					Title:       p.Name,
					Description: fmt.Sprintf("to after %s hours or %s messages!", util.FormattedNum(cfg.MaxHours), util.FormattedNum(cfg.MaxMessages)),
					Color:       bot.SuccessColor,
					Footer:      &discord.EmbedFooter{Text: "AutoDelete is enabled!"},
				}

				if cfg.MaxHours == 0 && cfg.MaxMessages == 0 {
					embed.Color = bot.ErrorColor
					embed.Footer = &discord.EmbedFooter{Text: "AutoDelete is disabled!"}
				}

				if argErr != nil {
					embed.Description = fmt.Sprintf("AutoDelete in %s is set ", channel.Mention()) + embed.Description
				} else {
					embed.Description = fmt.Sprintf("Set AutoDelete in %s ", channel.Mention()) + embed.Description
					saveConfig(c.E.GuildID.String(), channel.ID.String(), cfg)
				}

				_, err = cmd.SendCustomEmbed(c.E.ChannelID, embed)
			}

			return err
		}
	}
}

func purgeMessages() {
	if p.Config == nil {
		return
	}

	for gID, gCfg := range p.Config.(config).Guilds { // range through each guild's config
		for cID, cfg := range gCfg { // range through each guild's channel configs
			go func(cfg ChannelConfig) {
				if cfg.MaxHours == 0 && cfg.MaxMessages == 0 {
					return
				}

			}(cfg)
		}
	}
}

func saveConfig(gID, cID string, cfg ChannelConfig) {
	if p.Config == nil {
		p.Config = config{Guilds: map[string]map[string]ChannelConfig{gID: {cID: {}}}}
	}

	gCfg, ok := p.Config.(config).Guilds[gID]
	if !ok {
		gCfg = map[string]ChannelConfig{cID: cfg}
		p.Config.(config).Guilds[gID] = gCfg
	} else {
		p.Config.(config).Guilds[gID][cID] = cfg
	}
}

func getConfig(gID, cID string) ChannelConfig {
	if p.Config == nil {
		p.Config = config{Guilds: map[string]map[string]ChannelConfig{gID: {cID: {}}}}
	}

	gCfg, ok := p.Config.(config).Guilds[gID]
	if !ok {
		gCfg = map[string]ChannelConfig{cID: {}}
	}
	cfg, ok := gCfg[cID]
	if !ok {
		cfg = ChannelConfig{}
	}

	p.Config.(config).Guilds[gID][cID] = cfg

	return cfg
}

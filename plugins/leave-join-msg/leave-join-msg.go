package main

import (
	"encoding/json"
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"reflect"
	"strings"
	"sync"
)

var (
	p     *plugins.Plugin
	mutex sync.Mutex
)

type config struct {
	Guilds map[string]MsgConfig `json:"guilds,omitempty"` // [guild id]MsgConfig
}

type MsgConfig struct {
	JoinMessage  Message `json:"join_message"`
	LeaveMessage Message `json:"leave_message"`
}

type Message struct {
	Enabled         bool           `json:"enabled,omitempty"`
	Channel         int64          `json:"channel,omitempty"`
	Content         string         `json:"content,omitempty"`
	Embed           *discord.Embed `json:"embed,omitempty"`
	CollapseMessage bool           `json:"collapse_message,omitempty"`
	LastMessage     int64          `json:"last_message,omitempty"`
}

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	p = &plugins.Plugin{
		Name:        "Leave & Join Msg",
		Description: "Send a message when a user leaves or joins. `USER_ID` and `USER_TAG` are allowed for use in messages",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          LeaveJoinMsgCfgCommand,
			FnName:      "LeaveJoinMsgCfgCommand",
			Name:        "leavejoinconfig",
			Aliases:     []string{"ljcfg"},
			Description: "Edit leave & join msg config",
			GuildOnly:   true,
		}},
		ConfigType: reflect.TypeOf(config{}),
		Handlers: []bot.HandlerInfo{{
			Fn:     LeaveJoinAddHandler,
			FnName: "LeaveJoinAddHandler",
			FnType: reflect.TypeOf(func(event *gateway.GuildMemberAddEvent) {}),
		}, {
			Fn:     LeaveJoinRemoveHandler,
			FnName: "LeaveJoinRemoveHandler",
			FnType: reflect.TypeOf(func(event *gateway.GuildMemberRemoveEvent) {}),
		}},
	}
	p.Config = p.LoadConfig()
	return p
}

func LeaveJoinAddHandler(i interface{}) {
	mutex.Lock()
	defer mutex.Unlock()
	defer util.LogPanic()
	e := i.(*gateway.GuildMemberAddEvent)

	if p.Config == nil {
		return
	}

	if cfg, ok := p.Config.(config).Guilds[e.GuildID.String()]; ok && cfg.JoinMessage.Enabled {
		message := strings.ReplaceAll(cfg.JoinMessage.Content, "USER_ID", e.User.ID.String())
		message = strings.ReplaceAll(message, "USER_TAG", e.User.Tag())

		if msg, err := cmd.SendMessageEmbedSafe(discord.ChannelID(cfg.JoinMessage.Channel), message, cfg.JoinMessage.Embed); err != nil {
			log.Printf("error sending join message: %v\n", err)
		} else {
			if cfg.JoinMessage.CollapseMessage && cfg.JoinMessage.LastMessage != 0 {
				_ = bot.Client.DeleteMessage(discord.ChannelID(cfg.JoinMessage.Channel), discord.MessageID(cfg.JoinMessage.LastMessage), "join message collapsed")
			}

			cfg.JoinMessage.LastMessage = int64(msg.ID)
			p.Config.(config).Guilds[e.GuildID.String()] = cfg
		}
	}
}

func LeaveJoinRemoveHandler(i interface{}) {
	mutex.Lock()
	defer mutex.Unlock()
	defer util.LogPanic()
	e := i.(*gateway.GuildMemberRemoveEvent)

	if p.Config == nil {
		return
	}

	if cfg, ok := p.Config.(config).Guilds[e.GuildID.String()]; ok && cfg.LeaveMessage.Enabled {
		message := strings.ReplaceAll(cfg.LeaveMessage.Content, "USER_ID", e.User.ID.String())
		message = strings.ReplaceAll(message, "USER_TAG", e.User.Tag())

		if msg, err := cmd.SendMessageEmbedSafe(discord.ChannelID(cfg.LeaveMessage.Channel), message, cfg.LeaveMessage.Embed); err != nil {
			log.Printf("error sending leave message: %v\n", err)
		} else {
			if cfg.LeaveMessage.CollapseMessage && cfg.LeaveMessage.LastMessage != 0 {
				_ = bot.Client.DeleteMessage(discord.ChannelID(cfg.LeaveMessage.Channel), discord.MessageID(cfg.LeaveMessage.LastMessage), "leave message collapsed")
			}

			cfg.LeaveMessage.LastMessage = int64(msg.ID)
			p.Config.(config).Guilds[e.GuildID.String()] = cfg
		}
	}
}

func LeaveJoinMsgCfgCommand(c bot.Command) error {
	if err := cmd.HasPermission("moderate", c); err != nil {
		return err
	}

	mutex.Lock()
	defer mutex.Unlock()

	msgConfig := MsgConfig{JoinMessage: Message{}, LeaveMessage: Message{}}

	if p.Config != nil {
		if guildMsgConfig, ok := p.Config.(config).Guilds[c.E.GuildID.String()]; ok {
			msgConfig = guildMsgConfig
		}
	}

	var err error = nil

	arg, _ := cmd.ParseStringArg(c.Args, 1, true)
	arg2, _ := cmd.ParseStringArg(c.Args, 2, true)
	arg3, argErr := cmd.ParseStringSliceArg(c.Args, 3, -1)
	argChannel, argChannelErr := cmd.ParseChannelArg(c.Args, 3)
	argEnabled, argEnabledErr := cmd.ParseBoolArg(c.Args, 3)
	argCollapse, argCollapseErr := cmd.ParseBoolArg(c.Args, 3)

	defaultResponse := func() error {
		_, err := cmd.SendEmbed(c.E, "Leave & Join Message", "Available arguments are:\n- `join|leave channel|message|embed|enabled|collapse <channel|message|embed json|enabled bool|collapse bool>`", bot.DefaultColor)
		return err
	}

	subArgs := func(s string, msg Message) Message {
		switch arg2 {
		case "channel":
			if argChannelErr != nil {
				if msg.Channel == 0 {
					_, err = cmd.SendEmbed(c.E, s+" Message Channel", s+" Message channel is not set!", bot.WarnColor)

				} else {
					_, err = cmd.SendEmbed(c.E, s+" Message Channel", fmt.Sprintf("%s Message channel is set to <#%v>!", s, msg.Channel), bot.DefaultColor)
				}
			} else {
				msg.Channel = argChannel
				_, err = cmd.SendEmbed(c.E, s+" Message Channel", fmt.Sprintf("Set %s Message channel to <#%v>!", s, argChannel), bot.SuccessColor)
			}
		case "message":
			if argErr != nil {
				_, err = cmd.SendEmbed(c.E, s+" Message Content", fmt.Sprintf("%s Message content is set to \n```\n%s\n```", s, msg.Content), bot.DefaultColor)
			} else {
				msg.Content = strings.Join(arg3, " ")
				_, err = cmd.SendEmbed(c.E, s+" Message Content", fmt.Sprintf("Set %s Message content to \n```\n%s\n```", s, msg.Content), bot.SuccessColor)
			}
		case "embed":
			if argErr != nil || len(arg3) == 0 {
				embed := cmd.MakeEmbed(s+" Message Embed", fmt.Sprintf("%s Message embed is set to:", s), bot.DefaultColor)

				if msg.Embed != nil {
					log.Println("here1")
					_, err = bot.Client.SendMessage(c.E.ChannelID, "", embed, *msg.Embed)
				} else {
					log.Println("here2")
					_, err = bot.Client.SendMessage(c.E.ChannelID, "", embed)
				}
			} else {
				if err == nil {
					var embed discord.Embed
					err = json.Unmarshal([]byte(strings.Join(arg3, " ")), &embed)

					if err == nil {
						msg.Embed = &embed
						_, err = bot.Client.SendMessage(c.E.ChannelID, "", cmd.MakeEmbed(s+" Message Embed", fmt.Sprintf("Set %s Message embed to:", s), bot.SuccessColor), embed)
					}
				}
			}
		case "enabled":
			if argEnabledErr != nil {
				if msg.Enabled {
					_, err = cmd.SendEmbed(c.E, s+" Message", s+" Message is enabled!", bot.SuccessColor)
				} else {
					_, err = cmd.SendEmbed(c.E, s+" Message", s+" Message is not enabled!", bot.WarnColor)
				}
			} else {
				msg.Enabled = argEnabled
				if argEnabled {
					_, err = cmd.SendEmbed(c.E, s+" Message", "✅ Enabled "+s+" Message!", bot.SuccessColor)
				} else {
					_, err = cmd.SendEmbed(c.E, s+" Message", "⛔ Disabled "+s+" Message!", bot.ErrorColor)
				}
			}
		case "collapse":
			if argCollapseErr != nil {
				if msg.CollapseMessage {
					_, err = cmd.SendEmbed(c.E, s+" Message Collapsing", s+" Message Collapsing is enabled!", bot.SuccessColor)
				} else {
					_, err = cmd.SendEmbed(c.E, s+" Message Collapsing", s+" Message Collapsing is not enabled!", bot.WarnColor)
				}
			} else {
				msg.CollapseMessage = argCollapse
				if argCollapse {
					_, err = cmd.SendEmbed(c.E, s+" Message Collapsing", "✅ Enabled collapsing for "+s+" Message!", bot.SuccessColor)
				} else {
					_, err = cmd.SendEmbed(c.E, s+" Message Collapsing", "⛔ Disabled collapsing for "+s+" Message!", bot.ErrorColor)
				}
			}
		default:
			err = defaultResponse()
		}

		return msg
	}

	switch arg {
	case "join":
		msgConfig.JoinMessage = subArgs("Join", msgConfig.JoinMessage)
	case "leave":
		msgConfig.LeaveMessage = subArgs("Leave", msgConfig.LeaveMessage)
	default:
		err = defaultResponse()
	}

	if p.Config == nil {
		guilds := make(map[string]MsgConfig, 0)
		guilds[c.E.GuildID.String()] = msgConfig
		cfg := config{Guilds: guilds}
		p.Config = cfg
	} else {
		p.Config.(config).Guilds[c.E.GuildID.String()] = msgConfig
	}

	return err
}

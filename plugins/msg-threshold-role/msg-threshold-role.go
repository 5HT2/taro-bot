package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"log"
	"reflect"
	"sync"
)

var (
	p     *plugins.Plugin
	mutex sync.Mutex
)

type config struct {
	GuildUsers      map[string]map[string]User `json:"guild_users,omitempty"`      // [guild id][user id]User
	GuildThresholds map[string]ThresholdRole   `json:"guild_thresholds,omitempty"` // [guild id]threshold
}

type User struct {
	Msgs      int64 `json:"msgs"`
	GivenRole bool  `json:"given_role"`
}

type ThresholdRole struct {
	Threshold       int64   `json:"threshold"`
	Role            int64   `json:"role"`
	ExcludeChannels []int64 `json:"exclude_channels,omitempty"`
}

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	p = &plugins.Plugin{
		Name:        "Message Threshold Roles",
		Description: "Assign a role once a message threshold has been reached",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          MsgThresholdCfgCommand,
			FnName:      "MsgThresholdCfgCommand",
			Name:        "msgthresholdcfg",
			Aliases:     []string{"mtcfg"},
			Description: "Edit message threshold role config",
		}},
		ConfigType: reflect.TypeOf(config{}),
		Responses: []bot.ResponseInfo{{
			Fn:       MsgThresholdMsgResponse,
			Regexes:  []string{"."},
			MatchMin: 1,
		}},
	}
	p.Config = p.LoadConfig()
	return p
}

func MsgThresholdMsgResponse(r bot.Response) {
	if p.Config == nil {
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	threshold, ok := p.Config.(config).GuildThresholds[r.E.GuildID.String()]
	if !ok {
		return
	}

	// This guild has ExcludeChannels enabled and the message is from one of them
	if len(threshold.ExcludeChannels) > 0 && util.SliceContains(threshold.ExcludeChannels, int64(r.E.ChannelID)) {
		return
	}

	checkThreshold := func(threshold ThresholdRole, user User) User {
		if !user.GivenRole && user.Msgs >= threshold.Threshold && threshold.Role != 0 && threshold.Threshold != 0 {
			// Assign role
			reason := fmt.Sprintf("user messages met threshold of %v", threshold.Threshold)
			data := api.AddRoleData{AuditLogReason: api.AuditLogReason(reason)}
			log.Printf("attempting to add threshold role: %v (%s)\n", threshold.Role, data)

			if err := bot.Client.AddRole(r.E.GuildID, r.E.Author.ID, discord.RoleID(threshold.Role), data); err != nil {
				log.Printf("failed to add threshold role: %v\n", err)
			} else {
				user.GivenRole = true
			}
		}

		return user
	}

	// Check if the guild has an existing config
	if cfg, ok := p.Config.(config).GuildUsers[r.E.GuildID.String()]; ok {
		// If the guild has an existing config, does this user exist in it yet?
		if user, ok := cfg[r.E.Author.ID.String()]; ok {
			// User exists, bump their messages and check the threshold
			user.Msgs += 1
			user = checkThreshold(threshold, user)

			// Update the config
			p.Config.(config).GuildUsers[r.E.GuildID.String()][r.E.Author.ID.String()] = user
		} else {
			// User not in this guild's config, add them to it.
			user := User{Msgs: 1, GivenRole: false}
			user = checkThreshold(threshold, user)

			// Update the config
			p.Config.(config).GuildUsers[r.E.GuildID.String()][r.E.Author.ID.String()] = user
		}
	} else {
		// Users map not found, create it
		users := make(map[string]User)
		users[r.E.Author.ID.String()] = User{Msgs: 1, GivenRole: false}

		if len(p.Config.(config).GuildUsers) == 0 {
			guilds := make(map[string]map[string]User, 0)
			guilds[r.E.GuildID.String()] = users

			cfg := p.Config.(config)
			cfg.GuildUsers = guilds

			p.Config = cfg
		}

		// Save users map in the config
		p.Config.(config).GuildUsers[r.E.GuildID.String()] = users
	}
}

func MsgThresholdCfgCommand(c bot.Command) error {
	mutex.Lock()
	defer mutex.Unlock()

	arg, _ := cmd.ParseStringArg(c.Args, 1, true)

	thresholds := ThresholdRole{Role: 0, Threshold: 15}

	if p.Config != nil {
		if guildThreshold, ok := p.Config.(config).GuildThresholds[c.E.GuildID.String()]; ok {
			thresholds = guildThreshold
		}
	}

	var err error = nil

	switch arg {
	case "threshold":
		if threshold, argErr := cmd.ParseInt64Arg(c.Args, 2); argErr != nil {
			_, err = cmd.SendEmbed(c.E, "Message Threshold", fmt.Sprintf("Currently set to %v", thresholds.Threshold), bot.DefaultColor)
		} else {
			thresholds.Threshold = threshold
			_, err = cmd.SendEmbed(c.E, "Message Threshold", fmt.Sprintf("Set to %v", thresholds.Threshold), bot.SuccessColor)
		}
	case "role":
		if role, argErr := cmd.ParseInt64Arg(c.Args, 2); argErr != nil {
			if thresholds.Role == 0 {
				_, err = cmd.SendEmbed(c.E, "Message Threshold Role", "Role not currently set! Use the `role [role id]` subcommand to set it.", bot.WarnColor)
			} else {
				_, err = cmd.SendEmbed(c.E, "Message Threshold Role", fmt.Sprintf("Currently set to <@&%v>", thresholds.Role), bot.DefaultColor)
			}
		} else {
			thresholds.Role = role
			_, err = cmd.SendEmbed(c.E, "Message Threshold Role", fmt.Sprintf("Set to <@&%v>", thresholds.Role), bot.SuccessColor)
		}
	case "exclude":
		arg, _ = cmd.ParseStringArg(c.Args, 2, true)
		channel, argErr := cmd.ParseChannelArg(c.Args, 3)

		switch arg {
		case "add":
			if argErr != nil {
				return argErr
			}

			if !util.SliceContains(thresholds.ExcludeChannels, channel) {
				thresholds.ExcludeChannels = append(thresholds.ExcludeChannels, channel)
			}

			_, err = cmd.SendEmbed(c.E, "Message Threshold Exclude Channels", fmt.Sprintf("✅ Added <#%v> to excluded channels!", channel), bot.SuccessColor)
		case "remove":
			if argErr != nil {
				return argErr
			}

			thresholds.ExcludeChannels = util.SliceRemove(thresholds.ExcludeChannels, channel)

			_, err = cmd.SendEmbed(c.E, "Message Threshold Exclude Channels", fmt.Sprintf("⛔ Removed <#%v> from excluded channels!", channel), bot.ErrorColor)
		default:
			formattedChannels := util.JoinInt64Slice(thresholds.ExcludeChannels, "\n", "⛔ <#", ">")
			if len(thresholds.ExcludeChannels) == 0 {
				formattedChannels = "No excluded channels!"
			}

			_, err = cmd.SendEmbed(c.E, "Message Threshold Exclude Channels", fmt.Sprintf("Excluded channels:\n\n%s", formattedChannels), bot.DefaultColor)
		}
	default:
		_, err = cmd.SendEmbed(c.E,
			"Configure Message Threshold Roles",
			"Available arguments are:\n- `threshold <threshold>`\n- `role [role id]`\n- `exclude add|remove <channel>`",
			bot.DefaultColor)
	}

	if p.Config == nil {
		guilds := make(map[string]ThresholdRole, 0)
		guilds[c.E.GuildID.String()] = thresholds
		cfg := config{GuildThresholds: guilds}
		p.Config = cfg
	} else {
		p.Config.(config).GuildThresholds[c.E.GuildID.String()] = thresholds
	}

	return err
}

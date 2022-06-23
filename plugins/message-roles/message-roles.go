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
	"strconv"
	"strings"
	"sync"
)

var (
	p     *plugins.Plugin
	mutex sync.Mutex
)

type config struct {
	GuildUsers map[string]map[string]User `json:"guild_users,omitempty"`   // [guild id][user id]User
	GuildRoles map[string][]Role          `json:"guild_configs,omitempty"` // [guild id][]Role
	// this could also be a [guild id][role id]Role for performance reasons, but it's only loop-searched in commands,
	// so it can stay like this for now.
}

type User struct {
	Msgs       map[string]int64 `json:"msgs"`        // [role id]number of messages
	GivenRoles map[string]bool  `json:"given_roles"` // [role id]given role
}

type Role struct {
	Threshold int64   `json:"threshold"`
	ID        int64   `json:"role"`
	Whitelist []int64 `json:"whitelist"`
	Blacklist []int64 `json:"blacklist"`
}

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	p = &plugins.Plugin{
		Name:        "Message Roles",
		Description: "Assign a role once a message threshold has been reached",
		Version:     "1.0.2",
		Commands: []bot.CommandInfo{{
			Fn:          MessageRolesConfigCommand,
			FnName:      "MessageRolesConfigCommand",
			Name:        "messagerolesconfig",
			Aliases:     []string{"mrcfg"},
			Description: "Edit message roles config",
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

	roles, ok := p.Config.(config).GuildRoles[r.E.GuildID.String()]
	if !ok {
		return
	}

	// This guild has ExcludeChannels enabled and the message is from one of them
	//if len(guildConfig.ExcludeChannels) > 0 && util.SliceContains(guildConfig.ExcludeChannels, int64(r.E.ChannelID)) {
	//	return
	//}

	// this will go and validate if the message channel is in the whitelist or blacklist, or neither, and bump the message count for said role
	bumpMessages := func(roles []Role, user User, channel discord.ChannelID) User {
		for _, role := range roles {
			roleID := strconv.FormatInt(role.ID, 10)

			if len(role.Whitelist) > 0 {
				// If a whitelist is enabled and the channel IS in the whitelist.
				// We can't collapse this with && because we don't want the else to happen if a whitelist is enabled
				// and the channel ISN'T in the whitelist.
				if util.SliceContains(role.Whitelist, int64(channel)) {
					user.Msgs[roleID] += 1
				}
			} else if len(role.Blacklist) > 0 {
				// If a blacklist is enabled and the channel IS NOT in the blacklist.
				// We can't collapse this with && because we don't want the else to happen if a blacklist is enabled
				// and the channel IS in the blacklist.
				if !util.SliceContains(role.Blacklist, int64(channel)) {
					user.Msgs[roleID] += 1
				}
			} else {
				user.Msgs[roleID] += 1
			}
		}

		return user
	}

	// this will check each role if the threshold is met or not, and assign it if so
	checkThreshold := func(roles []Role, user User) User {
		for _, role := range roles {
			roleID := strconv.FormatInt(role.ID, 10)
			givenRole, _ := user.GivenRoles[roleID]

			if !givenRole && user.Msgs[roleID] >= role.Threshold && role.ID != 0 && role.Threshold != 0 {
				// Assign role
				reason := fmt.Sprintf("user messages met threshold of %v for role <@&%v>", role.Threshold, role.ID)
				data := api.AddRoleData{AuditLogReason: api.AuditLogReason(reason)}
				log.Printf("attempting to add threshold role: %v (%s)\n", role.ID, data)

				if err := bot.Client.AddRole(r.E.GuildID, r.E.Author.ID, discord.RoleID(role.ID), data); err != nil {
					log.Printf("failed to add threshold role: %v\n", err)
				} else {
					user.GivenRoles[roleID] = true
				}
			}
		}
		return user
	}

	// Check if the guild has an existing config
	if cfg, ok := p.Config.(config).GuildUsers[r.E.GuildID.String()]; ok {
		user := User{Msgs: make(map[string]int64), GivenRoles: make(map[string]bool)}

		// If the guild has an existing config, does this user exist in it yet?
		if guildUser, ok := cfg[r.E.Author.ID.String()]; ok {
			user = guildUser
		}

		// User not in this guild's config, add them to it.
		user = bumpMessages(roles, user, r.E.ChannelID)
		user = checkThreshold(roles, user)

		// Update the config
		p.Config.(config).GuildUsers[r.E.GuildID.String()][r.E.Author.ID.String()] = user
	} else {
		// Make a new user, populate it
		user := User{Msgs: make(map[string]int64), GivenRoles: make(map[string]bool)}
		user = bumpMessages(roles, user, r.E.ChannelID)

		// Users map not found, create it
		users := make(map[string]User)
		users[r.E.Author.ID.String()] = user

		// If there are no guilds with users, create a new guild and replace it with the `users` map
		if len(p.Config.(config).GuildUsers) == 0 {
			guilds := make(map[string]map[string]User, 0)
			guilds[r.E.GuildID.String()] = users

			cfg := p.Config.(config)
			cfg.GuildUsers = guilds

			p.Config = cfg
		}

		// Save `users` map in the config
		p.Config.(config).GuildUsers[r.E.GuildID.String()] = users
	}
}

func MessageRolesConfigCommand(c bot.Command) error {
	if err := cmd.HasPermission("moderate", c); err != nil {
		return err
	}

	mutex.Lock()
	defer mutex.Unlock()

	arg, _ := cmd.ParseStringArg(c.Args, 1, true)

	roles := make([]Role, 0)

	if p.Config != nil {
		if guildRoles, ok := p.Config.(config).GuildRoles[c.E.GuildID.String()]; ok {
			roles = guildRoles
		}
	}

	var err error = nil

	switch arg {
	case "role":
		role, argErr1 := cmd.ParseInt64Arg(c.Args, 2)
		threshold, argErr2 := cmd.ParseInt64Arg(c.Args, 3)

		// For the future: some people might expect that setting a threshold to 0 will "auto-role" people, when in
		// reality it will only apply once the user sends any messages. This is probably fine for now.
		if threshold < 0 {
			threshold = 0
		}

		if argErr1 != nil {
			return argErr1
		}
		if argErr2 != nil {
			return argErr2
		}

		found := false
		for n, r := range roles {
			if r.ID == role {
				r.Threshold = threshold
				roles[n] = r
				_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Changed threshold for <@&%v> to %v!", r.ID, r.Threshold), bot.SuccessColor)

				found = true
				break
			}
		}

		if !found {
			newRole := Role{Threshold: threshold, ID: role}
			roles = append(roles, newRole)

			_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Created role <@&%v> with threshold %v!", role, threshold), bot.SuccessColor)
		}
	case "whitelist":
		role, argErr1 := cmd.ParseInt64Arg(c.Args, 2)
		channel, argErr2 := cmd.ParseChannelArg(c.Args, 3)

		if argErr1 != nil {
			return argErr1
		}
		if argErr2 != nil {
			return argErr2
		}

		found := false
		for n, r := range roles {
			if r.ID == role {
				if util.SliceContains(r.Whitelist, channel) {
					r.Whitelist = util.SliceRemove(r.Whitelist, channel)
					_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Removed <#%v> from <@&%v>'s whitelist", channel, r.ID), bot.ErrorColor)
				} else {
					r.Whitelist = append(r.Whitelist, channel)
					_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Added <#%v> to <@&%v>'s whitelist", channel, r.ID), bot.SuccessColor)
				}

				roles[n] = r
				found = true
				break
			}
		}

		if !found {
			_, err = cmd.SendEmbed(c.E, p.Name, "This role is not setup for Message Roles! Add it using the `role` argument.", bot.ErrorColor)
		}
	case "blacklist":
		role, argErr1 := cmd.ParseInt64Arg(c.Args, 2)
		channel, argErr2 := cmd.ParseChannelArg(c.Args, 3)

		if argErr1 != nil {
			return argErr1
		}
		if argErr2 != nil {
			return argErr2
		}

		found := false
		for n, r := range roles {
			if r.ID == role {
				if util.SliceContains(r.Blacklist, channel) {
					r.Blacklist = util.SliceRemove(r.Blacklist, channel)
					_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Removed <#%v> from <@&%v>'s blacklist", channel, r.ID), bot.ErrorColor)
				} else {
					r.Blacklist = append(r.Blacklist, channel)
					_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Added <#%v> to <@&%v>'s blacklist", channel, r.ID), bot.SuccessColor)
				}

				roles[n] = r
				found = true
				break
			}
		}

		if !found {
			_, err = cmd.SendEmbed(c.E, p.Name, "This role is not setup for Message Roles! Add it using the `role` argument.", bot.ErrorColor)
		}
	case "list":
		if len(roles) == 0 {
			_, err = cmd.SendEmbed(c.E, p.Name, "No message roles setup!", bot.WarnColor)
		} else {
			lines := make([]string, 0)
			for _, role := range roles {
				a1 := ""
				a2 := ""
				if len(role.Whitelist) > 0 {
					a1 = "\n✅ Whitelist: " + util.JoinInt64Slice(role.Whitelist, ", ", "<#", ">")
				}
				if len(role.Blacklist) > 0 {
					a2 = "\n⛔ Blacklist: " + util.JoinInt64Slice(role.Blacklist, ", ", "<#", ">")
				}

				lines = append(lines, fmt.Sprintf("<@&%v> (%v messages)%s%s", role.ID, role.Threshold, a1, a2))
			}
			_, err = cmd.SendEmbed(c.E, p.Name, strings.Join(lines, "\n"), bot.DefaultColor)
		}
	default:
		_, err = cmd.SendEmbed(c.E,
			"Configure Message Roles",
			"Available arguments are:\n- `role [role id] [threshold]`\n- `whitelist [role id] [channel]`\n- `blacklist [role id] [channel]`\n- `list`",
			bot.DefaultColor)
	}

	if p.Config == nil {
		guilds := make(map[string][]Role, 0)
		guilds[c.E.GuildID.String()] = roles
		cfg := config{GuildRoles: guilds}
		p.Config = cfg
	} else {
		p.Config.(config).GuildRoles[c.E.GuildID.String()] = roles
	}

	return err
}

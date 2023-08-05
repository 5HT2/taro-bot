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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	p     *plugins.Plugin
	mutex sync.Mutex
)

type config struct {
	StartDate  time.Time                  `json:"start_date"`              // Date bot started keeping track of User.TotalMsgs
	GuildUsers map[string]map[string]User `json:"guild_users,omitempty"`   // [guild id][user id]User
	GuildRoles map[string][]Role          `json:"guild_configs,omitempty"` // [guild id][]Role
	// this could also be a [guild id][role id]Role for performance reasons, but it's only loop-searched in commands,
	// so it can stay like this for now.
}

type User struct {
	TotalMsgs     int64            `json:"total_msgs"`      // number of messages sent while in that guild, ignoring any kind of whitelist / blacklist rules
	TotalRoleMsgs int64            `json:"total_role_msgs"` // number of messages sent, respecting whitelist / blacklist
	Msgs          map[string]int64 `json:"msgs"`            // [role id]number of messages
	GivenRoles    map[string]bool  `json:"given_roles"`     // [role id]given role
}

type Role struct {
	Threshold int64   `json:"threshold"`
	ID        int64   `json:"role"`
	Whitelist []int64 `json:"whitelist"`
	Blacklist []int64 `json:"blacklist"`
}

func InitPlugin(i *plugins.PluginInit) *plugins.Plugin {
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
			GuildOnly:   true,
		}, {
			Fn:          MessageTopCommand,
			FnName:      "MessageTopCommand",
			Name:        "messagetop",
			Aliases:     []string{"msgtop", "leaderboard"},
			Description: "Message Leaderboard",
			GuildOnly:   true,
		}},
		ConfigType: reflect.TypeOf(config{}),
		Responses: []bot.ResponseInfo{{
			Fn:       MsgThresholdMsgResponse,
			Regexes:  []string{"."},
			MatchMin: 1,
		}},
		StartupFn: func() {
			if cfg, ok := p.Config.(config); ok {
				if cfg.StartDate.IsZero() {
					cfg.StartDate = time.Now()
				}

				p.Config = cfg
			} else {
				p.Config = config{StartDate: time.Now()}
			}
		},
	}
	p.ConfigDir = i.ConfigDir
	p.Config = p.LoadConfig()
	return p
}

func MsgThresholdMsgResponse(r bot.Response) {
	mutex.Lock()
	defer mutex.Unlock()

	roles := make([]Role, 0)
	if guildRoles, ok := p.Config.(config).GuildRoles[r.E.GuildID.String()]; ok {
		roles = guildRoles
	}

	// this will go and validate if the message channel is in the whitelist or blacklist, or neither, and bump the message count for said role
	bumpMessages := func(roles []Role, user User, channel discord.ChannelID) User {
		user.TotalMsgs += 1

		for _, role := range roles {
			roleID := strconv.FormatInt(role.ID, 10)

			if len(role.Whitelist) > 0 {
				// If a whitelist is enabled and the channel IS in the whitelist.
				// We can't collapse this with && because we don't want the else to happen if a whitelist is enabled
				// and the channel ISN'T in the whitelist.
				if util.SliceContains(role.Whitelist, int64(channel)) {
					user.Msgs[roleID] += 1
					user.TotalRoleMsgs += 1
				}
			} else if len(role.Blacklist) > 0 {
				// If a blacklist is enabled and the channel IS NOT in the blacklist.
				// We can't collapse this with && because we don't want the else to happen if a blacklist is enabled
				// and the channel IS in the blacklist.
				if !util.SliceContains(role.Blacklist, int64(channel)) {
					user.Msgs[roleID] += 1
					user.TotalRoleMsgs += 1
				}
			} else {
				user.Msgs[roleID] += 1
				user.TotalRoleMsgs += 1
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
		// Make a new user, populate it
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
		user = checkThreshold(roles, user)

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
	if err := cmd.HasPermission(c, cmd.PermModerate); err != nil {
		return err
	}

	mutex.Lock()
	defer mutex.Unlock()

	roles := make([]Role, 0)
	if guildRoles, ok := p.Config.(config).GuildRoles[c.E.GuildID.String()]; ok {
		roles = guildRoles
	}

	arg, _ := cmd.ParseStringArg(c.Args, 1, true)
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
				_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Changed threshold for <@&%v> to %s!", r.ID, util.FormattedNum(r.Threshold)), bot.SuccessColor)

				found = true
				break
			}
		}

		if !found {
			newRole := Role{Threshold: threshold, ID: role}
			roles = append(roles, newRole)

			_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Created role <@&%v> with threshold %s!", role, util.FormattedNum(threshold)), bot.SuccessColor)
		}
	case "remove":
		role, argErr1 := cmd.ParseInt64Arg(c.Args, 2)

		if argErr1 != nil {
			return argErr1
		}

		orderedRoles := make([]Role, 0)

		for _, r := range roles {
			if r.ID != role {
				orderedRoles = append(orderedRoles, r)
			}
		}

		if len(orderedRoles) < len(roles) {
			_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Removed role <@&%v>!", role), bot.ErrorColor)
		} else {
			_, err = cmd.SendEmbed(c.E, p.Name, "This role is not setup for Message Roles! Add it using the `role` argument.", bot.ErrorColor)
		}

		roles = orderedRoles
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
		user0, argErr3 := cmd.ParseInt64Arg(c.Args, 3)
		user1, argErr4 := cmd.ParseUserArg(c.Args, 3)

		blacklistChannel := func() {
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
		}

		blacklistUser := func() error {
			user := user0       // try arg 3 as int64
			if argErr3 != nil { // arg 3 wasn't int64, try user mention
				user = user1
			}
			if argErr4 != nil { // arg3 wasn't user mention, exit
				return argErr4
			}

			if discordUser, err := bot.Client.User(discord.UserID(user)); err != nil {
				return err
			} else {
				roleStr := fmt.Sprintf("%v", role)

				// Check if the guild has an existing config
				if cfg, ok := p.Config.(config).GuildUsers[c.E.GuildID.String()]; ok {
					user := User{Msgs: make(map[string]int64), GivenRoles: make(map[string]bool)}

					// If the guild has an existing config, does this user exist in it yet?
					if guildUser, ok := cfg[discordUser.ID.String()]; ok {
						user = guildUser
					}

					// User not in this guild's config, add them to it.
					user.GivenRoles[roleStr] = true

					// Update the config
					p.Config.(config).GuildUsers[c.E.GuildID.String()][discordUser.ID.String()] = user
				} else {
					// Make a new user, populate it
					user := User{Msgs: make(map[string]int64), GivenRoles: make(map[string]bool)}
					user.GivenRoles[roleStr] = true

					// Users map not found, create it
					users := make(map[string]User)
					users[discordUser.ID.String()] = user

					// If there are no guilds with users, create a new guild and replace it with the `users` map
					if len(p.Config.(config).GuildUsers) == 0 {
						guilds := make(map[string]map[string]User, 0)
						guilds[c.E.GuildID.String()] = users

						cfg := p.Config.(config)
						cfg.GuildUsers = guilds

						p.Config = cfg
					}

					// Save `users` map in the config
					p.Config.(config).GuildUsers[c.E.GuildID.String()] = users
				}

				_, err = cmd.SendEmbed(c.E, p.Name, fmt.Sprintf("Succesfully blacklisted <@%v> from getting <@&%v>!", user, role), bot.SuccessColor)
				return err
			}
		}

		if argErr1 != nil {
			return argErr1
		}
		if argErr2 == nil { // parsed a channel mention successfully
			blacklistChannel()
		} else { // didn't parse a channel, blacklistUser will check if the new arg is a user or not
			return blacklistUser()
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

				lines = append(lines, fmt.Sprintf("<@&%v> (%s messages)%s%s", role.ID, util.FormattedNum(role.Threshold), a1, a2))
			}
			_, err = cmd.SendEmbed(c.E, p.Name, strings.Join(lines, "\n\n"), bot.DefaultColor)
		}
	default:
		_, err = cmd.SendEmbed(c.E,
			"Configure Message Roles",
			"Available arguments are:\n- `role [role id] [threshold]`\n- remove [role id]`\n- `whitelist [role id] [channel]`\n- `blacklist [role id] [channel]`\n- `blacklist [role id] [user]`\n- `list`",
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

func MessageTopCommand(c bot.Command) error {
	if cfg, ok := p.Config.(config).GuildUsers[c.E.GuildID.String()]; ok {
		topUsers := make([]string, 0)

		for k := range cfg {
			if len(k) == 0 {
				continue
			}

			topUsers = append(topUsers, k)
		}

		sort.SliceStable(topUsers, func(i, j int) bool {
			return cfg[topUsers[i]].TotalMsgs > cfg[topUsers[j]].TotalMsgs
		})

		lines := make([]string, 0)
		fields := make([]discord.EmbedField, 0)

		id := c.E.Author.ID.String()
		selfPos := 0
		selfNum := ""

		for n, u := range topUsers {
			if u == id {
				selfPos = n + 1
				selfNum = util.FormattedNum(cfg[u].TotalMsgs)
			}

			if n < 3 {
				emoji := ""
				switch n {
				case 0:
					emoji = "🥇"
				case 1:
					emoji = "🥈"
				case 2:
					emoji = "🥉"
				}

				fields = append(fields, discord.EmbedField{
					Name:  emoji,
					Value: fmt.Sprintf("<@%s>: %s", u, util.FormattedNum(cfg[u].TotalMsgs)),
				})
			} else {
				lines = append(lines, fmt.Sprintf("#%v <@%s>: %s", n+1, u, util.FormattedNum(cfg[u].TotalMsgs)))
			}
		}

		if len(lines) > 0 {
			fields = append(fields, discord.EmbedField{
				Name: "​", Value: util.HeadLinesLimit(strings.Join(lines, "\n"), 1024),
			})
		}

		author := cmd.CreateEmbedAuthor(*c.E.Member)
		if selfPos != 0 {
			author.Name += fmt.Sprintf(" (#%v: %s)", selfPos, selfNum)
		}

		_, err := cmd.SendCustomEmbed(c.E.ChannelID, discord.Embed{
			Title:     "Message Leaderboard",
			Author:    author,
			Fields:    fields,
			Footer:    &discord.EmbedFooter{Text: "Messages sent since"},
			Timestamp: discord.Timestamp(p.Config.(config).StartDate),
			Color:     bot.DefaultColor,
		})

		return err
	}

	_, err := cmd.SendEmbed(c.E, p.Name, "GuildUsers config for this guild is missing! Contact a developer for help, this shouldn't ever happen.", bot.ErrorColor)
	return err
}

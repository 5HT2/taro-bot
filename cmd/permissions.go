package cmd

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"strings"
	"sync"
	"time"
)

var (
	Permissions     = []string{"channels", "permissions", "moderate"}
	PermissionCache = permissionCache{}
)

type permissionCache struct {
	guilds []guildAdmins
	mutex  sync.Mutex
}

type guildAdmins struct {
	id     discord.GuildID
	admins []guildUser
}

type guildUser struct {
	lastCheck int64
	id        discord.UserID
	member    discord.Member
	admin     bool
	roles     []discord.RoleID
}

// HasPermission will return if the author of a command has said permission
func HasPermission(permission string, c bot.Command) *bot.Error {
	id := int64(c.E.Author.ID)

	if UserHasPermission(permission, c, id) {
		return nil
	} else {
		return bot.GenericError(c.FnName, "running command", util.GetUserMention(id)+" is missing the \""+permission+"\" permission")
	}
}

// UserHasPermission will return if the user with id has said permission
func UserHasPermission(permission string, c bot.Command, id int64) bool {
	if hasAdminCached(c.E.GuildID, c.E.Member.RoleIDs, c.E.Author) {
		return true
	}

	users := make([]int64, 0)
	bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		users = getPermissionSlice(permission, g)
		return g, "UserHasPermission: " + c.FnName
	})

	return util.SliceContains(users, id)
}

// GivePermission will return nil if the permission was successfully given to the user with a matching id
func GivePermission(permission string, id int64, c bot.Command) error {
	var err error = nil

	bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {

		users := getPermissionSlice(permission, g)
		mention := util.GetUserMention(id)

		if !util.SliceContains(users, id) {
			users = append(users, id)
		} else {
			err = bot.GenericError("GivePermission",
				"giving permission to "+mention,
				"user already has permission \""+permission+"\"")
		}

		switch permission {
		case "channels":
			g.Permissions.ManageChannels = users
		case "permissions":
			g.Permissions.ManagePermissions = users
		case "moderate":
			g.Permissions.Moderation = users
		default:
			err = bot.GenericError("GivePermission",
				"giving permission to "+mention,
				"couldn't find permission type \""+permission+"\"")
		}

		return g, "GivePermission: " + c.FnName
	})

	return err
}

// UpdateMemberCache will forcibly update the member cache
func UpdateMemberCache(e *gateway.GuildMemberUpdateEvent) {
	log.Printf("updating member cache\n")
	hasAdmin(e.GuildID, e.RoleIDs, e.User)
}

func getPermissionSlice(permission string, guild *bot.GuildConfig) []int64 {
	permission = strings.ToLower(permission)

	switch permission {
	case "channels":
		return guild.Permissions.ManageChannels
	case "permissions":
		return guild.Permissions.ManagePermissions
	case "moderate":
		return guild.Permissions.Moderation
	default:
		return make([]int64, 0)
	}
}

func hasAdminCached(id discord.GuildID, memberRoles []discord.RoleID, user discord.User) bool {
	PermissionCache.mutex.Lock()
	defer PermissionCache.mutex.Unlock()

	for _, g := range PermissionCache.guilds {
		if g.id == id {
			for _, u := range g.admins {
				// If ID matches and the last check was more recent than 10 minutes ago
				if u.id == user.ID && time.Now().Unix()-u.lastCheck < 600 {
					log.Printf("hasAdminCached: found %v\n", u.id)
					return u.admin
				}
			}
		}
	}

	log.Printf("hasAdminCached: didn't find anyone\n")
	return hasAdmin(id, memberRoles, user)
}

func hasAdmin(id discord.GuildID, memberRoles []discord.RoleID, user discord.User) bool {
	if PermissionCache.mutex.TryLock() {
		defer PermissionCache.mutex.Unlock()
	}

	roles, err := bot.Client.Roles(id)
	if err != nil {
		return false
	}
	guild, err := bot.Client.Guild(id)
	if err != nil {
		return false
	}

	admin := false
	if guild.OwnerID != user.ID {
		for _, r := range roles {
			if r.Permissions.Has(discord.PermissionAdministrator) && util.SliceContains(memberRoles, r.ID) {
				admin = true
				break
			}
		}
	} else {
		admin = true
	}

	found := false

	// Look through guilds
	for n, g := range PermissionCache.guilds {
		// If guild ID matches
		if g.id == id {
			foundUser := false
			found = true

			// Look through cached admins
			for n, u := range g.admins {
				if u.id == user.ID {
					foundUser = true
					u.admin = admin
					u.lastCheck = time.Now().Unix()
					g.admins[n] = u

					log.Printf("permission cache: found existing cache %v, setting to %v\n", user.ID, admin)
					break
				}
			}

			// If didn't find cached admin
			if !foundUser {
				u := guildUser{lastCheck: time.Now().Unix(), id: user.ID, admin: admin}
				g.admins = append(g.admins, u)
				log.Printf("permission cache: didn't find existing cache %v, setting to %v\n", user.ID, admin)
			}

			PermissionCache.guilds[n] = g
			break
		}
	}

	if !found {
		PermissionCache.guilds = append(PermissionCache.guilds, guildAdmins{id, []guildUser{{lastCheck: time.Now().Unix(), id: user.ID, admin: admin}}})
	}

	return admin
}

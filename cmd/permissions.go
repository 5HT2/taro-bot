package cmd

import (
	"fmt"
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
	Permissions     = []Permission{PermUndefined, PermChannels, PermPermissions, PermModerate, PermOperator}
	PermissionCache = permissionCache{}
)

type Permission int64

const (
	PermUndefined Permission = iota
	PermChannels
	PermPermissions
	PermModerate
	PermOperator // "operator" is a special permission, managed by bot.C.OperatorIDs
)

func (p Permission) String() string {
	switch p {
	case PermChannels:
		return "channels"
	case PermPermissions:
		return "permissions"
	case PermModerate:
		return "moderate"
	case PermOperator:
		return "operator"
	default:
		return "undefined"
	}
}

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
func HasPermission(c bot.Command, p Permission) *bot.Error {
	id := int64(c.E.Author.ID)

	if id == 0 {
		return bot.GenericError(c.FnName, "checking permission", "id is `0`")
	}

	if p == PermOperator {
		opIDs := make([]int64, 0)
		bot.C.Run(func(c *bot.Config) {
			opIDs = c.OperatorIDs
		})

		if !util.SliceContains(opIDs, int64(c.E.Author.ID)) {
			return bot.GenericError(c.FnName, "running command", util.GetUserMention(id)+" is not a bot operator")
		}

		return nil
	}

	if !c.E.GuildID.IsValid() {
		return bot.GenericError(c.FnName, "running command", "command run in a non-guild, permissions are not supported here")
	}

	if !UserHasPermission(c, p, id) {
		return bot.GenericError(c.FnName, "running command", fmt.Sprintf("%s is missing the \"%s\" permission", util.GetUserMention(id), p))
	}

	return nil
}

// UserHasPermission will return if the user with id has said permission
func UserHasPermission(c bot.Command, p Permission, id int64) bool {
	if HasAdminCached(c.E.GuildID, c.E.Member.RoleIDs, c.E.Author) {
		return true
	}

	users := make([]int64, 0)
	bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		users = getPermissionSlice(p, g)
		return g, "UserHasPermission: " + c.FnName
	})

	return util.SliceContains(users, id)
}

// GivePermission will return nil if the permission was successfully given to the user with a matching id
func GivePermission(c bot.Command, pStr string, id int64) error {
	var err error = nil

	bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		p := GetPermission(pStr)
		users := getPermissionSlice(p, g)

		if !util.SliceContains(users, id) {
			users = append(users, id)
		} else {
			err = bot.GenericError("GivePermission",
				"giving permission to "+util.GetUserMention(id),
				fmt.Sprintf("user already has permission \"%s\"", p))
		}

		if err == nil {
			switch p {
			case PermChannels:
				g.Permissions.ManageChannels = users
			case PermPermissions:
				g.Permissions.ManagePermissions = users
			case PermModerate:
				g.Permissions.Moderation = users
			default:
				err = bot.GenericError("GivePermission",
					"giving permission to "+util.GetUserMention(id),
					fmt.Sprintf("couldn't find permission type \"%s\"", pStr))
			}
		}

		return g, "GivePermission: " + c.FnName
	})

	return err
}

// GetPermission will return a valid Permission type from a string
func GetPermission(pStr string) Permission {
	pStr = strings.ToLower(pStr)

	for _, p := range Permissions {
		if p.String() == pStr {
			return p
		}
	}

	return PermUndefined
}

// UpdateMemberCache will forcibly update the member cache
func UpdateMemberCache(e *gateway.GuildMemberUpdateEvent) {
	log.Printf("updating member cache\n")
	hasAdmin(e.GuildID, e.RoleIDs, e.User)
}

func getPermissionSlice(p Permission, guild *bot.GuildConfig) []int64 {
	switch p {
	case PermChannels:
		return guild.Permissions.ManageChannels
	case PermPermissions:
		return guild.Permissions.ManagePermissions
	case PermModerate:
		return guild.Permissions.Moderation
	default:
		return make([]int64, 0)
	}
}

func HasAdminCached(id discord.GuildID, memberRoles []discord.RoleID, user discord.User) bool {
	PermissionCache.mutex.Lock()
	defer PermissionCache.mutex.Unlock()

	for _, g := range PermissionCache.guilds {
		if g.id == id {
			for _, u := range g.admins {
				// If ID matches and the last check was more recent than 10 minutes ago
				if u.id == user.ID && time.Now().Unix()-u.lastCheck < 600 {
					log.Printf("HasAdminCached: found %v\n", u.id)
					return u.admin
				}
			}
		}
	}

	log.Printf("HasAdminCached: didn't find anyone\n")
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

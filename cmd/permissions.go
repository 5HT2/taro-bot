package cmd

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/util"
	"strings"
)

// HasPermission will return if the author of a command has said permission
func HasPermission(permission string, c Command) *bot.Error {
	id := int64(c.e.Author.ID)

	if UserHasPermission(permission, c, id) {
		return nil
	} else {
		return bot.GenericError(c.fnName, "running command", util.GetUserMention(id)+" is missing the \""+permission+"\" permission")
	}
}

// UserHasPermission will return if the user with id has said permission
func UserHasPermission(permission string, c Command, id int64) bool {
	users := make([]int64, 0)
	bot.GuildContext(c.e.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		users = getPermissionSlice(permission, g)
		return g, "UserHasPermission: " + c.fnName
	})

	return util.SliceContains(users, id)
}

// GivePermission will return nil if the permission was successfully given to the user with a matching id
func GivePermission(permission string, id int64, c Command) error {
	var err error = nil

	bot.GuildContext(c.e.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {

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
		default:
			err = bot.GenericError("GivePermission",
				"giving permission to "+mention,
				"couldn't find permission type \""+permission+"\"")
		}

		return g, "GivePermission: " + c.fnName
	})

	return err
}

func getPermissionSlice(permission string, guild *bot.GuildConfig) []int64 {
	permission = strings.ToLower(permission)

	switch permission {
	case "channels":
		return guild.Permissions.ManageChannels
	case "permissions":
		return guild.Permissions.ManagePermissions
	default:
		return make([]int64, 0)
	}
}

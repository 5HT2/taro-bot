package main

import (
	"strings"
)

// PermissionGroups is collection of "permissions". Each permission is a list of user IDs that have said permission.
// Switching this to a list of {Name, Users} would maybe be better code-wise.
type PermissionGroups struct {
	ManageChannels    []int64 `json:"manage_channels,omitempty"`
	ManagePermissions []int64 `json:"manage_permissions,omitempty"`
}

// HasPermission will return if the author of a command has said permission
func HasPermission(permission string, c Command) *TaroError {
	id := int64(c.e.Author.ID)

	if UserHasPermission(permission, c, id) {
		return nil
	} else {
		return GenericError(c.fnName, "running command", GetUserMention(id)+" is missing the \""+permission+"\" permission")
	}
}

// UserHasPermission will return if the user with id has said permission
func UserHasPermission(permission string, c Command, id int64) bool {
	users := make([]int64, 0)
	GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
		users = getPermissionSlice(permission, g)
		return g, "UserHasPermission: " + c.fnName
	})

	return SliceContains(users, id)
}

// GivePermission will return nil if the permission was successfully given to the user with a matching id
func GivePermission(permission string, id int64, c Command) error {
	var err error = nil

	GuildContext(c.e.GuildID, func(g *GuildConfig) (*GuildConfig, string) {

		users := getPermissionSlice(permission, g)
		mention := GetUserMention(id)

		if !SliceContains(users, id) {
			users = append(users, id)
		} else {
			err = GenericError("GivePermission",
				"giving permission to "+mention,
				"user already has permission \""+permission+"\"")
		}

		switch permission {
		case "channels":
			g.Permissions.ManageChannels = users
		case "permissions":
			g.Permissions.ManagePermissions = users
		default:
			err = GenericError("GivePermission",
				"giving permission to "+mention,
				"couldn't find permission type \""+permission+"\"")
		}

		return g, "GivePermission: " + c.fnName
	})

	return err
}

func getPermissionSlice(permission string, guild *GuildConfig) []int64 {
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

package main

import (
	"encoding/json"
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"reflect"
	"strings"
	"time"
)

var p *plugins.Plugin

type config struct {
	Menus map[string]map[string]Menu `json:"menus,omitempty"` // [guild id][message id]Menu
}

type Menu struct {
	Channel int64           `json:"channel,omitempty"`
	Roles   map[string]Role `json:"roles"` // [api emoji]Role
}

type Role struct {
	RoleID int64 `json:"role_id"`
}

type RoleConfig struct {
	Emoji  string `json:"emoji"`
	RoleID int64  `json:"id"`
}

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	p = &plugins.Plugin{
		Name:        "Role Menu",
		Description: "Create menus to assign roles with reactions!",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          RoleMenuCommand,
			FnName:      "RoleMenuCommand",
			Name:        "rolemenu",
			Aliases:     []string{"rmcfg"},
			Description: "Create a role Menu",
		}},
		ConfigType: reflect.TypeOf(config{}),
		Handlers: []bot.HandlerInfo{{
			Fn:     ReactionAddHandler,
			FnName: "ReactionAddHandler",
			FnType: reflect.TypeOf(func(*gateway.MessageReactionAddEvent) {}),
		}, {
			Fn:     ReactionRemoveHandler,
			FnName: "ReactionRemoveHandler",
			FnType: reflect.TypeOf(func(event *gateway.MessageReactionRemoveEvent) {}),
		}},
	}
	p.Config = p.LoadConfig()
	return p
}

func RoleMenuCommand(c bot.Command) error {
	if err := cmd.HasPermission("moderate", c); err != nil {
		return err
	}

	args, _ := cmd.ParseAllArgs(c.Args)
	var roleConfigs []RoleConfig

	example := []RoleConfig{{"<:astolfo:880936523644669962>", 881205936818122754}, {"<:trans_sunglasses:880628887481102336>", 881206354658885673}}
	if args == "example" {
		bytes, _ := json.MarshalIndent(example, "", "    ")
		_, _ = cmd.SendMessage(c.E, "```json\n"+string(bytes)+"\n```")
		return nil
	}

	if err := json.Unmarshal([]byte(args), &roleConfigs); err != nil {
		return err
	}

	if msgOriginal, err := cmd.SendEmbed(c.E, "Role Menu", "Creating role menu...", bot.WarnColor); err != nil {
		return err
	} else {
		roles := make(map[string]Role, 0)

		// Parse the command args into an actual config now
		for _, rc := range roleConfigs {
			emoji, animated, argErr := cmd.ParseEmojiArg([]string{rc.Emoji}, 1, false)
			if argErr != nil {
				return argErr
			}

			parsedEmoji := util.ApiEmojiAsConfig(emoji, animated)
			roles[parsedEmoji] = Role{rc.RoleID}
		}

		lines := make([]string, 0) // formatted role menu message

		for parsedEmoji, role := range roles {
			apiEmoji, _ := util.ConfigEmojiAsApiEmoji(parsedEmoji)
			lines = append(lines, fmt.Sprintf("<%s> <@&%v>", apiEmoji, role.RoleID))
		}

		if msg, err := bot.Client.SendMessage(c.E.ChannelID, strings.Join(lines, "\n")); err != nil {
			return err
		} else {
			//
			// Save final menu in config

			finalMenu := Menu{Channel: int64(c.E.ChannelID), Roles: roles}
			messageMenu := make(map[string]Menu)
			messageMenu[msg.ID.String()] = finalMenu

			menus := make(map[string]map[string]Menu, 0)
			menus[c.E.GuildID.String()] = messageMenu
			p.Config = config{Menus: menus}

			// Add reactions to menu
			for parsedEmoji, _ := range roles {
				apiEmoji, _ := util.ConfigEmojiAsApiEmoji(parsedEmoji)
				if err := bot.Client.React(msg.ChannelID, msg.ID, apiEmoji); err != nil {
					log.Printf("failed to react when creating role menu: %v\n", err)
				}
				time.Sleep(750 * time.Millisecond) // We want to wait for the actual rate-limit, but Arikawa does not handle that for you
			}

			msg, _ = bot.Client.EditMessage(
				msgOriginal.ChannelID,
				msgOriginal.ID,
				"",
				discord.Embed{
					Title:       "Role Menu",
					Description: "Successfully created role menu!",
					Color:       bot.SuccessColor,
				},
			)
			time.Sleep(5 * time.Second)

			err = bot.Client.DeleteMessage(msg.ChannelID, msg.ID, "cleaning up log msg")
			return err
		}
	}
}

func ReactionAddHandler(i interface{}) {
	defer util.LogPanic()
	e := i.(*gateway.MessageReactionAddEvent)

	roleID, auditLogReason := getRoleFromEvent(e.GuildID, e.MessageID, e.ChannelID, e.Emoji, true)
	log.Printf("trying to add role: %v (%s)\n", roleID, auditLogReason)

	if err := bot.Client.AddRole(e.GuildID, e.UserID, discord.RoleID(roleID), api.AddRoleData{AuditLogReason: auditLogReason}); err != nil {
		log.Printf("failed to add reaction role: %v\n", err)
	}
}

func ReactionRemoveHandler(i interface{}) {
	defer util.LogPanic()
	e := i.(*gateway.MessageReactionRemoveEvent)

	roleID, auditLogReason := getRoleFromEvent(e.GuildID, e.MessageID, e.ChannelID, e.Emoji, false)
	log.Printf("trying to remove role: %v (%s)\n", roleID, auditLogReason)

	if err := bot.Client.RemoveRole(e.GuildID, e.UserID, discord.RoleID(roleID), auditLogReason); err != nil {
		log.Printf("failed to remove reaction role: %v\n", err)
	}
}

func getRoleFromEvent(id discord.GuildID, messageID discord.MessageID, channelID discord.ChannelID, emoji discord.Emoji, add bool) (int64, api.AuditLogReason) {
	if p.Config == nil {
		return -1, "" // Not configured
	}

	roleMenus, ok := p.Config.(config).Menus[id.String()]
	if !ok {
		return -1, "" // No Menu configured
	}

	menu, ok := roleMenus[messageID.String()]
	if !ok {
		return -1, "" // Reacted message does not have a Menu
	}

	apiEmoji := emoji.APIString()
	role, ok := menu.Roles[util.ApiEmojiAsConfig(&apiEmoji, emoji.Animated)]

	textReacted := "reacted"
	textTo := "to"
	if !add {
		textReacted = "removed"
		textTo = "from"
	}

	auditLogReason := api.AuditLogReason(fmt.Sprintf("user %s %s %s %v/%v", textReacted, emoji, textTo, channelID, messageID))
	return role.RoleID, auditLogReason
}

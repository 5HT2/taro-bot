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
	"strconv"
	"strings"
	"time"
)

var p *plugins.Plugin

type config struct {
	Menus map[string]map[string]Menu `json:"menus"` // [guild id][message id]Menu
}

// Menu stores the information needed to operate a role menu
type Menu struct {
	Channel int64           `json:"channel,omitempty"` // channel id
	Roles   map[string]Role `json:"roles"`             // [api emoji]Role
}

// Role is used to assign roles from a Menu
type Role struct {
	RoleID int64 `json:"role_id"`
}

// RoleConfig is used when setting up a role menu and is parsed into a Menu format
type RoleConfig struct {
	ID        string           `json:"-"` // MessageID as a string
	MessageID int64            `json:"message_id,omitempty"`
	ChannelID int64            `json:"channel_id,omitempty"`
	Roles     []RoleConfigRole `json:"roles"`
}

// RoleConfigRole is used when setting up a role menu and is parsed into a Menu and Role format
type RoleConfigRole struct {
	Emoji  string `json:"emoji"`
	RoleID int64  `json:"id"`
}

func InitPlugin(i *plugins.PluginInit) *plugins.Plugin {
	p = &plugins.Plugin{
		Name:        "Role Menu",
		Description: "Create menus to assign roles with reactions!",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          RoleMenuCommand,
			FnName:      "RoleMenuCommand",
			Name:        "rolemenu",
			Aliases:     []string{"rmcfg"},
			Description: "Create a role menu",
			GuildOnly:   true,
		}},
		ConfigType: reflect.TypeOf(config{}),
		Handlers: []bot.HandlerInfo{{
			Fn:     RoleMenuReactionAddHandler,
			FnName: "RoleMenuReactionAddHandler",
			FnType: reflect.TypeOf(func(*gateway.MessageReactionAddEvent) {}),
		}, {
			Fn:     RoleMenuReactionRemoveHandler,
			FnName: "RoleMenuReactionRemoveHandler",
			FnType: reflect.TypeOf(func(event *gateway.MessageReactionRemoveEvent) {}),
		}},
	}
	p.ConfigDir = i.ConfigDir
	p.Config = p.LoadConfig()
	return p
}

// Example usage:
// .rolemenu create {"roles": [{"emoji": "<:astolfo:880936523644669962>", "id": 881205936818122754 }, {"emoji": "<:trans_sunglasses:880628887481102336>", "id": 881206354658885673 }, {"emoji": "<:painedsmug:880628887871160350>", "id": 881206111536025620 }, {"emoji": "<:hewwo:880928545256394792>", "id": 881206521235644447 }]}

func RoleMenuCommand(c bot.Command) error {
	if err := cmd.HasPermission("moderate", c); err != nil {
		return err
	}

	firstArg, argErr := cmd.ParseStringArg(c.Args, 1, true)

	defaultHelp := func() error {
		_, err := cmd.SendEmbed(c.E, p.Name,
			"Available arguments are:\n- `create|add|remove [role json]`\n\n`create` a new role menu\n`add` roles\n`remove` existing roles",
			bot.DefaultColor)
		return err
	}

	if argErr != nil {
		return defaultHelp()
	}

	rolesJson, _ := cmd.ParseAllArgs(c.Args[1:])
	var roleConfig RoleConfig
	if err := json.Unmarshal([]byte(rolesJson), &roleConfig); err != nil {
		return err
	}

	//
	// Begin role message parsing section

	roles := make(map[string]Role, 0)

	// Parse the command args into an actual config now, and validate them as actual emojis
	for n, rc := range roleConfig.Roles {
		emoji, animated, argErr := cmd.ParseEmojiArg([]string{rc.Emoji}, 1, false)
		if argErr != nil {
			return argErr
		}

		parsedEmoji := util.ApiEmojiAsConfig(emoji, animated)
		roles[parsedEmoji] = Role{rc.RoleID}    // use config emoji, roles is used for the menu creation
		roleConfig.Roles[n].Emoji = parsedEmoji // use config emoji. roleConfig is only used for add and remove later on
	}

	//
	// End role message parsing section

	getLines := func() string {
		lines := make([]string, 0) // formatted role menu message

		for parsedEmoji, role := range roles {
			if strings.Contains(parsedEmoji, ":") {
				parsedEmoji = "<" + parsedEmoji + ">" // embed
			} else {
				apiEmoji, _ := util.ConfigEmojiAsApiEmoji(parsedEmoji)
				parsedEmoji = string(apiEmoji)
			}
			lines = append(lines, fmt.Sprintf("%s <@&%v>", parsedEmoji, role.RoleID))
		}
		return strings.Join(lines, "\n")
	}

	messageIDCheck := func(rc RoleConfig) (RoleConfig, *discord.Message, error) {
		if rc.MessageID == 0 {
			msg, _ := cmd.SendEmbed(c.E, p.Name, "`message_id` must be set to add to an existing role menu!", bot.ErrorColor)
			return rc, msg, bot.GenericError("RoleMenuCommand", "modifying role menu", "`message_id` not set")
		}
		if rc.ChannelID == 0 {
			rc.ChannelID = int64(c.E.ChannelID)
			msg, err := cmd.SendEmbed(c.E, p.Name, "`channel_id` not set, defaulting to existing channel. Editing menu...", bot.WarnColor)
			return rc, msg, err
		}

		msg, err := cmd.SendEmbed(c.E, p.Name, "`message_id` and `channel_id` set, editing menu...", bot.SuccessColor)
		return rc, msg, err
	}

	getMenu := func(c bot.Command, rc RoleConfig) (*Menu, error) {
		var menu *Menu
		if p.Config != nil {
			if guild, ok := p.Config.(config).Menus[c.E.GuildID.String()]; ok {
				if m, ok := guild[rc.ID]; ok {
					menu = &m
				}
			}
		}

		if menu == nil {
			return nil, bot.GenericError("RoleMenuCommand", "getting existing role menu", "none found")
		}
		return menu, nil
	}

	setMenu := func(c bot.Command, rc RoleConfig, m Menu) {
		if p.Config != nil {
			p.Config.(config).Menus[c.E.GuildID.String()][rc.ID] = m
		} else {
			menus := make(map[string]map[string]Menu, 0)
			msgMenu := make(map[string]Menu)
			msgMenu[rc.ID] = m
			menus[c.E.GuildID.String()] = msgMenu
			p.Config = config{Menus: menus}
		}
	}

	switch firstArg {
	case "add":
		roleConfig, msg, err := messageIDCheck(roleConfig)
		if err != nil {
			if msg != nil { // we're not handling the original message's error in this case, so we should check this
				time.Sleep(5 * time.Second)
				_ = bot.Client.DeleteMessage(msg.ChannelID, msg.ID, "cleaning up log msg")
			}
			return err
		}

		roleConfig.ID = strconv.FormatInt(roleConfig.MessageID, 10)

		existingMenu, err := getMenu(c, roleConfig)
		if err != nil {
			return err
		}

		newRoles := make(map[string]Role)

		// Create new roles to add
		for _, role := range roleConfig.Roles {
			newRoles[role.Emoji] = Role{RoleID: role.RoleID}
		}

		// Add old roles if they haven't been added yet
		for emoji, role := range existingMenu.Roles {
			if _, ok := newRoles[emoji]; !ok { // doesn't exist in new roles to add
				newRoles[emoji] = role
			} // else // does exist in the new roles, that means our old emoji now has a new role ID. the current `role.RoleID` is the old role ID
		}

		// Save menu in global config
		existingMenu.Roles = newRoles
		setMenu(c, roleConfig, *existingMenu)

		// Edit the menu message
		roles = newRoles

		if _, err := bot.Client.EditMessage(discord.ChannelID(roleConfig.ChannelID), discord.MessageID(roleConfig.MessageID), getLines()); err != nil {
			return err
		}

		// Add the emojis for the new roles
		for n, parsedEmoji := range roleConfig.Roles {
			apiEmoji, _ := util.ConfigEmojiAsApiEmoji(parsedEmoji.Emoji)
			if err := bot.Client.React(discord.ChannelID(roleConfig.ChannelID), discord.MessageID(roleConfig.MessageID), apiEmoji); err != nil {
				log.Printf("failed to react when creating role menu: %v\n", err)
			}

			if n < len(roleConfig.Roles)-1 {
				time.Sleep(750 * time.Millisecond) // We want to wait for the actual rate-limit, but Arikawa does not handle that for you
			}
		}

		msg, err = cmd.SendEmbed(c.E, p.Name, "Edited role menu!", bot.SuccessColor)
		if err != nil {
			return err
		}

		time.Sleep(5 * time.Second)
		err = bot.Client.DeleteMessage(msg.ChannelID, msg.ID, "cleaning up log msg")
		return err
	case "remove":
		roleConfig, msg, err := messageIDCheck(roleConfig)
		if err != nil {
			if msg != nil { // we're not handling the original message's error in this case, so we should check this
				time.Sleep(5 * time.Second)
				_ = bot.Client.DeleteMessage(msg.ChannelID, msg.ID, "cleaning up log msg")
			}
			return err
		}

		roleConfig.ID = strconv.FormatInt(roleConfig.MessageID, 10)

		existingMenu, err := getMenu(c, roleConfig)
		if err != nil {
			return err
		}

		oldRoles := make(map[string]Role)

		// Only add roles from the existingMenu that aren't in the roleConfig (set by the user in their message)
		for emoji, role := range existingMenu.Roles {
			if !util.SliceContains(roleConfig.Roles, RoleConfigRole{RoleID: role.RoleID, Emoji: emoji}) {
				oldRoles[emoji] = role
			}
		}

		// Save menu in global config
		existingMenu.Roles = oldRoles
		setMenu(c, roleConfig, *existingMenu)

		// Edit the menu message
		roles = oldRoles

		if _, err := bot.Client.EditMessage(discord.ChannelID(roleConfig.ChannelID), discord.MessageID(roleConfig.MessageID), getLines()); err != nil {
			return err
		}

		// Remove the emojis for the removed roles
		for n, parsedEmoji := range roleConfig.Roles {
			apiEmoji, _ := util.ConfigEmojiAsApiEmoji(parsedEmoji.Emoji)
			if err := bot.Client.Unreact(discord.ChannelID(roleConfig.ChannelID), discord.MessageID(roleConfig.MessageID), apiEmoji); err != nil {
				log.Printf("failed to unreact when creating role menu: %v\n", err)
			}

			if n < len(roleConfig.Roles)-1 {
				time.Sleep(750 * time.Millisecond) // We want to wait for the actual rate-limit, but Arikawa does not handle that for you
			}
		}

		msg, err = cmd.SendEmbed(c.E, p.Name, "Edited role menu!", bot.SuccessColor)
		if err != nil {
			return err
		}

		time.Sleep(5 * time.Second)
		err = bot.Client.DeleteMessage(msg.ChannelID, msg.ID, "cleaning up log msg")
		return err
	case "create":
		if msgOriginal, err := cmd.SendEmbed(c.E, "Role Menu", "Creating role menu...", bot.WarnColor); err != nil {
			return err
		} else {
			if msg, err := bot.Client.SendMessage(c.E.ChannelID, "Creating role menu..."); err != nil {
				return err
			} else {
				// Edit role menu text into existing message
				msg, err = bot.Client.EditMessage(msg.ChannelID, msg.ID, getLines())

				//
				// Save final menu in config

				menus := make(map[string]map[string]Menu, 0)
				if p.Config != nil {
					menus = p.Config.(config).Menus // copy over the menus for other builds and our current guild
				}

				createdMenu := Menu{Channel: int64(c.E.ChannelID), Roles: roles}

				if _, ok := menus[c.E.GuildID.String()]; ok {
					menus[c.E.GuildID.String()][msg.ID.String()] = createdMenu
				} else {
					messageMenu := make(map[string]Menu)
					messageMenu[msg.ID.String()] = createdMenu
					menus[c.E.GuildID.String()] = messageMenu
				}

				p.Config = config{Menus: menus}

				// Add reactions to menu
				for parsedEmoji := range roles {
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
	default:
		return defaultHelp()
	}
}

func RoleMenuReactionAddHandler(i interface{}) {
	defer util.LogPanic()
	e := i.(*gateway.MessageReactionAddEvent)

	// Don't modify bots / self
	if e.Member.User.Bot {
		return
	}

	roleID, auditLogReason := getRoleFromEvent(e.GuildID, e.MessageID, e.ChannelID, e.Emoji, true)
	if roleID == 0 || roleID == -1 {
		return
	}

	log.Printf("trying to add role: %v (%s)\n", roleID, auditLogReason)

	if err := bot.Client.AddRole(e.GuildID, e.UserID, discord.RoleID(roleID), api.AddRoleData{AuditLogReason: auditLogReason}); err != nil {
		log.Printf("failed to add reaction role: %v\n", err)
	}
}

func RoleMenuReactionRemoveHandler(i interface{}) {
	defer util.LogPanic()
	e := i.(*gateway.MessageReactionRemoveEvent)

	// Can't return for non-bot here, too lazy to lookup user. This shouldn't be possible, anyways.

	roleID, auditLogReason := getRoleFromEvent(e.GuildID, e.MessageID, e.ChannelID, e.Emoji, false)
	if roleID == 0 || roleID == -1 {
		return
	}

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

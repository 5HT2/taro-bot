package main

import (
	"encoding/json"
	"fmt"
	"github.com/diamondburned/arikawa/v3/discord"
	"net/http"
	"strings"
)

type CommandInfo struct {
	FnName      string
	Name        string
	Description string
	Aliases     []string
}

func (ci CommandInfo) String() string {
	aliases := ""
	if len(ci.Aliases) > 0 {
		aliases = "(" + strings.Join(ci.Aliases, ", ") + ")"
	}
	description := ci.Description
	if len(description) == 0 {
		description = "No Description"
	}

	return fmt.Sprintf("**%s** %s\n%s", ci.Name, aliases, description)
}

var (
	commands = []CommandInfo{
		{FnName: "FrogCommand", Name: "frog", Description: "\\*hands you a random frog pic\\*"},
		{FnName: "HelpCommand", Name: "help", Aliases: []string{"h"}},
		{FnName: "KirbyCommand", Name: "kirby"},
		{FnName: "PermissionCommand", Name: "permission", Aliases: []string{"perm"}, Description: "Manage user permissions"},
		{FnName: "PingCommand", Name: "ping", Description: "Returns the current API latency"},
		{FnName: "PrefixCommand", Name: "prefix", Description: "Set the bot prefix for your guild"},
	}
)

func (c Command) PermissionCommand() error {
	arg1, _ := ParseStringArg(c.args, 1, true)

	switch arg1 {
	case "give":
		if HasPermission("permissions", c) {
			permission, argErr := ParseStringArg(c.args, 2, true)
			if argErr != nil {
				return argErr
			}
			id, argErr := ParseUserArg(c.args, 3)
			if argErr != nil {
				return argErr
			}

			if err := GivePermission(permission, id, c); err != nil {
				return err
			} else {
				_, err = SendEmbed(c,
					"Permissions",
					"Successfully gave "+GetUserMention(id)+" permission to use \""+permission+"\"",
					successColor)
				return err
			}
		} else {
			_, err := SendEmbed(c, "Permissions", "You do not have permission to manage permissions!", errorColor)
			return err
		}
	default:
		_, err := SendEmbed(c,
			"Permissions",
			"Available arguments are:\n- `give` <permission> <user>",
			defaultColor)
		return err
	}
}

func (c Command) PrefixCommand() error {
	arg, argErr := ParseStringArg(c.args, 1, false)
	if argErr != nil {
		return argErr
	}

	// Filter spaces
	arg = strings.ReplaceAll(arg, " ", "")
	if len(arg) == 0 {
		return GenericError("PrefixCommand", "getting prefix", "prefix is empty")
	}

	guild := GetGuildConfig(int64(c.e.GuildID))
	guild.Prefix = arg
	SetGuildConfig(guild)

	embed := discord.Embed{
		Description: "Set prefix to `" + arg + "`.",
		Footer:      &discord.EmbedFooter{Text: "At any time you can ping the bot with the word \"prefix\" to get the current prefix"},
		Color:       successColor,
	}
	_, err := SendCustomEmbed(c.e.ChannelID, embed)
	return err
}

func (c Command) HelpCommand() error {
	fmtCmds := make([]string, 0)
	for _, cmd := range commands {
		fmtCmds = append(fmtCmds, cmd.String())
	}

	_, err := SendEmbed(c,
		"Taro Help",
		strings.Join(fmtCmds, "\n\n"),
		defaultColor)
	return err
}

func (c Command) PingCommand() error {
	msg, err := SendEmbed(c,
		"Ping!",
		"Unfinished", // TODO do
		defaultColor)
	if err != nil {
		_, err = SendEmbed(c, "Pong!", msg.Timestamp.Format(timeFormat), defaultColor)
		return err
	}
	return err
}

func (c Command) FrogCommand() error {
	frogData, err := RequestUrl("https://frog.pics/api/random", http.MethodGet)
	if err != nil {
		return err
	}

	type FrogPicture struct {
		ImageUrl    string `json:"image_url"`
		MedianColor string `json:"median_color"`
	}
	var frogPicture FrogPicture

	if err := json.Unmarshal(frogData, &frogPicture); err != nil {
		return err
	}

	color, err := ParseHexColorFast("#" + frogPicture.MedianColor)
	if err != nil {
		return err
	}

	embed := discord.Embed{
		Color: discord.Color(ConvertColorToInt32(color)),
		Image: &discord.EmbedImage{URL: frogPicture.ImageUrl},
	}

	_, err = SendCustomEmbed(c.e.ChannelID, embed)
	return err
}

func (c Command) KirbyCommand() {
	content := strings.Join(c.args, " ")
	_, _ = SendMessage(c, "<:kirbyfeet:893291555744542730>")
	_, _ = SendMessage(c, content)
}

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
		{FnName: "HelpCommand", Name: "help", Aliases: []string{"h"}},
		{FnName: "PingCommand", Name: "ping", Description: "Returns the current API latency"},
		{FnName: "FrogCommand", Name: "frog", Description: "\\*hands you a random frog pic\\*"},
		{FnName: "KirbyCommand", Name: "kirby"},
	}
)

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
	content := strings.Join(strings.Split(c.e.Content, " ")[1:], " ")
	_, _ = SendMessage(c, "<:kirbyfeet:893291555744542730>")
	_, _ = SendMessage(c, content)
}

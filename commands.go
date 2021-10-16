package main

import (
	"encoding/json"
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

var (
	commands = []CommandInfo{
		{FnName: "PingCommand", Name: "ping"},
		{FnName: "FrogCommand", Name: "frog"},
		{FnName: "KirbyCommand", Name: "kirby"},
	}
)

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

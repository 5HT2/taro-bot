package main

import (
	"encoding/json"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"net/http"
	"strconv"
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	return &plugins.Plugin{
		Name:        "Taro Base Fun",
		Description: "The fun commands as included as part of the bot",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          FrogCommand,
			FnName:      "FrogCommand",
			Name:        "frog",
			Description: "\\*hands you a random frog pic\\*",
		}, {
			Fn:          StealEmojiCommand,
			FnName:      "StealEmojiCommand",
			Name:        "stealemoji",
			Aliases:     []string{"se"},
			Description: "Upload an emoji to the current guild",
			GuildOnly:   true,
		}},
		Responses: []bot.ResponseInfo{},
	}
}

func FrogCommand(c bot.Command) error {
	frogData, _, err := util.RequestUrl("https://frog.pics/api/random", http.MethodGet)
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

	color, err := util.ParseHexColorFast("#" + frogPicture.MedianColor)
	if err != nil {
		return err
	}

	embed := discord.Embed{
		Color: discord.Color(util.ConvertColorToInt32(color)),
		Image: &discord.EmbedImage{URL: frogPicture.ImageUrl},
	}

	_, err = cmd.SendCustomEmbed(c.E.ChannelID, embed)
	return err
}

func StealEmojiCommand(c bot.Command) error {
	// try to get emoji ID
	emojiID, argErr := cmd.ParseInt64Arg(c.Args, 1)
	// try to get emoji URL
	if argErr != nil {
		emojiID, argErr = cmd.ParseEmojiUrlArg(c.Args, 1)
	}
	// try to get sent emoji
	if argErr != nil {
		emojiID, argErr = cmd.ParseEmojiIdArg(c.Args, 1)
	}
	// no emoji found
	if argErr != nil {
		return argErr
	}

	//
	// we now have the emoji ID, get the name

	emojiName, argErr := cmd.ParseStringArg(c.Args, 2, false)
	if argErr != nil {
		return bot.GenericError("StealEmojiCommand", "getting emoji name", "expected emoji name")
	}

	//
	// we now have the emoji ID and name, get the bytes

	url := "https://cdn.discordapp.com/emojis/" + strconv.FormatInt(emojiID, 10)
	bytes, res, err := util.RequestUrl(url+".gif", http.MethodGet)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		bytes, res, err = util.RequestUrl(url+".png", http.MethodGet)
		if err != nil {
			return err
		}

		if res.StatusCode != 200 {
			return bot.GenericError("StealEmojiCommand", "getting emoji bytes", "status was "+res.Status)
		}
	}

	// now we try to upload it

	image := api.Image{ContentType: res.Header.Get("content-type"), Content: bytes}
	createEmojiData := api.CreateEmojiData{
		Name:  emojiName,
		Image: image,
		AuditLogReason: api.AuditLogReason(
			"emoji created by " + util.GetUserMention(int64(c.E.Author.ID)),
		),
	}

	if emoji, err := bot.Client.CreateEmoji(c.E.GuildID, createEmojiData); err != nil {
		// error with uploading
		return bot.GenericError("StealEmojiCommand", "uploading emoji", err.Error())
	} else {
		// uploaded successfully, send a nice embed
		_, err := bot.Client.SendMessage(
			c.E.ChannelID,
			emoji.String(),
			discord.Embed{Title: "Emoji stolen ;)", Color: bot.SuccessColor},
		)
		return err
	}
}

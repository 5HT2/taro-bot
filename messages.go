package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"log"
)

var (
	successColor discord.Color = 0x3cde5a
	errorColor   discord.Color = 0xde413c
	defaultColor discord.Color = 0x493cde
)

func SendEmbed(channel discord.ChannelID, title string, description string, color discord.Color) (*discord.Message, error) {
	msg, err := client.SendEmbeds(
		channel,
		embed(title, description, color),
	)
	if err != nil {
		log.Printf("Error sending embed: %v", err)
	}
	return msg, err
}

func embed(title string, description string, color discord.Color) discord.Embed {
	return discord.Embed{
		Title:       title,
		Description: description,
		Color:       color,
	}
}

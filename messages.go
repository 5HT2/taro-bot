package main

import "github.com/diamondburned/arikawa/v3/discord"

var (
	successColor discord.Color = 0x3cde5a
	errorColor   discord.Color = 0xde413c
	defaultColor discord.Color = 0x493cde
)

func embed(title string, description string, color discord.Color) discord.Embed {
	return discord.Embed{
		Title:       title,
		Description: description,
		Color:       color,
	}
}

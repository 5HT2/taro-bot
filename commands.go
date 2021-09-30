package main

import (
	"github.com/diamondburned/arikawa/v3/gateway"
)

var (
	commands []func(c Command)
)

type Command struct{ e *gateway.MessageCreateEvent }

func RegisterCommands() {
	commands = make([]func(c Command), 0)
	commands[0] = PingCommand
}

func PingCommand(c Command) {
	id := c.e.Message.ChannelID

	msg, err := SendEmbed(id,
		"Ping!",
		"Unfinished", // TODO do
		defaultColor)
	if err != nil {
		_, _ = SendEmbed(id, "Pong!", msg.Timestamp.Format(timeFormat), defaultColor)
	}
}

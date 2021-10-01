package main

import (
	"github.com/diamondburned/arikawa/v3/gateway"
)

func GuildEmojisUpdateEvent(e *gateway.GuildEmojisUpdateEvent) {
	//for _, emoji := range e.Emojis {
	//
	//}
}

func MessageCreateEvent(e *gateway.MessageCreateEvent) {
	CommandHandler(e)
}

package main

import (
	"github.com/diamondburned/arikawa/v3/gateway"
)

func GuildEmojisUpdateEvent(e *gateway.GuildEmojisUpdateEvent) {
	//for _, emoji := range e.Emojis {
	// PrintEmojiUpdate(emoji) // TODO: fix this
	//}
}

func MessageCreateEvent(e *gateway.MessageCreateEvent) {
	CommandHandler(e)
	ResponseHandler(e)
}

package main

import (
	"github.com/diamondburned/arikawa/v3/gateway"
)

func MessageReactionAddEvent(e *gateway.MessageReactionAddEvent) {
	go StarboardReactionHandler(e)
	go TopicReactionHandler(e)
}

func MessageCreateEvent(e *gateway.MessageCreateEvent) {
	go CommandHandler(e)
	go ResponseHandler(e)
}

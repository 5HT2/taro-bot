package main

import (
	"github.com/diamondburned/arikawa/v3/gateway"
)

func MessageReactionAddEvent(e *gateway.MessageReactionAddEvent) {
	TopicReactionHandler(e)
}

func MessageCreateEvent(e *gateway.MessageCreateEvent) {
	CommandHandler(e)
	ResponseHandler(e)
}

package main

import (
	"context"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"log"
)

var client *session.Session

func main() {
	loadConfig()
	var token = config.BotToken
	if token == "" {
		log.Fatalln("No bot_token given")
	}

	client, err := session.New("Bot " + token)
	if err != nil {
		log.Fatalln("Session failed:", err)
	}

	// Add handlers
	client.AddHandler(guildEmojisUpdateEvent)

	// Add the needed Gateway intents.
	client.AddIntents(gateway.IntentGuildMessages)
	client.AddIntents(gateway.IntentGuildEmojis)
	client.AddIntents(gateway.IntentDirectMessages)

	if err := client.Open(context.Background()); err != nil {
		log.Fatalln("Failed to connect:", err)
	}
	defer client.Close()

	u, err := client.Me()
	if err != nil {
		log.Fatalln("Failed to get bot user:", err)
	}

	log.Printf("Started as %v (%s#%s)\n", u.ID, u.Username, u.Discriminator)

	// Block forever.
	select {}
}

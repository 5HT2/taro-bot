package main

import (
	"context"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"log"
	"net/http"
	"runtime"
	"time"
)

var (
	discordClient session.Session
	httpClient    = http.Client{Timeout: 5 * time.Second}
)

func main() {
	log.Printf("Running on Go version: %s\n", runtime.Version())

	LoadConfig()
	var token = config.BotToken
	if token == "" {
		log.Fatalln("No bot_token given")
	}

	c, err := session.New("Bot " + token)
	discordClient = *c

	if err != nil {
		log.Fatalln("Session failed:", err)
	}

	// Add handlers
	c.AddHandler(GuildEmojisUpdateEvent)
	c.AddHandler(MessageCreateEvent)

	// Add the needed Gateway intents.
	c.AddIntents(gateway.IntentGuildMessages)
	c.AddIntents(gateway.IntentGuildEmojis)
	c.AddIntents(gateway.IntentDirectMessages)

	if err := c.Open(context.Background()); err != nil {
		log.Fatalln("Failed to connect:", err)
	}
	defer c.Close()

	u, err := c.Me()
	if err != nil {
		log.Fatalln("Failed to get bot user:", err)
	}

	log.Printf("Started as %v (%s#%s)\n", u.ID, u.Username, u.Discriminator)

	// Block forever.
	select {}
}

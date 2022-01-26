package main

import (
	"context"
	"flag"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"log"
	"net/http"
	"runtime"
	"time"
)

var (
	discordClient session.Session
	discordBot    *discord.User
	httpClient    = http.Client{Timeout: 5 * time.Second}
	debug         = flag.Bool("debug", false, "Debug messages and faster config saving")
)

func main() {
	flag.Parse()
	log.Printf("Running on Go version: %s\n", runtime.Version())

	LoadConfig()
	var token = config.BotToken
	if token == "" {
		log.Fatalln("No bot_token given")
	}

	// TODO: Migrate to NewWithIntents
	c := session.New("Bot " + token)
	discordClient = *c

	if c == nil {
		log.Fatalln("Session failed: is nil")
	}

	me, err := c.Me()
	if err != nil {
		log.Fatalln(err)
	}
	discordBot = me

	// Add handlers
	c.AddHandler(MessageReactionAddEvent)
	c.AddHandler(MessageCreateEvent)

	// Add the needed Gateway intents.
	c.AddIntents(gateway.IntentGuildMessages)
	c.AddIntents(gateway.IntentGuildEmojis)
	c.AddIntents(gateway.IntentGuildMessageReactions)
	c.AddIntents(gateway.IntentDirectMessages)

	if err := c.Open(context.Background()); err != nil {
		log.Fatalln("Failed to connect:", err)
	}
	defer c.Close()

	u, err := c.Me()
	if err != nil {
		log.Fatalln("Failed to get bot user:", err)
	}

	go SetupConfigSaving()
	log.Printf("Started as %v (%s#%s)\n", u.ID, u.Username, u.Discriminator)

	// Block forever.
	select {}
}

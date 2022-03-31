package main

import (
	"bufio"
	"context"
	"flag"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/go-co-op/gocron"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	discordClient session.Session
	httpClient    = http.Client{Timeout: 5 * time.Second}
	scheduler     = gocron.NewScheduler(getTimeZone())

	lastExitCode = flag.Int64("exited", 0, "Called by Dockerfile")
	debugLog     = flag.Bool("debug", false, "Debug messages and faster config saving")
	debugLogFile = "/tmp/taro-bot.log"
)

func main() {
	flag.Parse()
	log.Printf("Running on Go version: %s\n", runtime.Version())

	LoadConfig()
	var token = config.BotToken
	if token == "" {
		log.Fatalln("No bot_token given")
	}

	c := session.NewWithIntents("Bot "+token,
		gateway.IntentGuildMessages,
		gateway.IntentGuildEmojis,
		gateway.IntentGuildMessageReactions,
		gateway.IntentDirectMessages,
	)
	discordClient = *c

	if c == nil {
		log.Fatalln("Session failed: is nil")
	}

	// Add handlers
	c.AddHandler(MessageReactionAddEvent)
	c.AddHandler(MessageCreateEvent)

	if err := c.Open(context.Background()); err != nil {
		log.Fatalln("Failed to connect:", err)
	}
	defer c.Close()

	u, err := c.Me()
	if err != nil {
		log.Fatalln("Failed to get bot user:", err)
	}

	go SetupConfigSaving()
	go SetupPlugins()

	// program has been called with -exited, upload the logs and don't run the bot
	if lastExitCode != nil && *lastExitCode > 0 {
		checkExited()
		os.Exit(int(*lastExitCode))
		return
	}

	log.Printf("Started as %v (%s#%s)\n", u.ID, u.Username, u.Discriminator)

	// Block forever.
	select {}
}

func checkExited() {
	log.Printf("Last exit code was %v\n", *lastExitCode)
	if config.OperatorChannel == 0 || config.OperatorID == 0 {
		log.Printf("Not uploading logs, OperatorChannel or OperatorID were not set\n")
		return
	}

	file, err := os.Open(debugLogFile)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatalln(err)
	}

	// Format stacktrace
	stack := "<@" + strconv.FormatInt(config.OperatorID, 10) + ">\n```\n" + strings.Join(lines, "\n")
	if len(stack) > 1996 {
		stack = stack[:1996]
	}
	stack += "\n```"

	if _, err = discordClient.SendMessage(discord.ChannelID(config.OperatorChannel), stack); err != nil {
		log.Fatalln(err)
	}
}

func getTimeZone() *time.Location {
	tzEnv := os.Getenv("TZ")
	if len(tzEnv) == 0 {
		tzEnv = "Local"
	}

	l, err := time.LoadLocation(tzEnv)
	if err != nil {
		log.Printf("error loading timezone, defaulting to UTC: %v\n", err)
		return time.UTC
	}

	log.Printf("using location %s for timezone\n", l)
	return l
}

package main

import (
	"bufio"
	"context"
	"flag"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"io"
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
	httpClient    = http.Client{Timeout: 10 * time.Second}
	debugLog      = flag.Bool("debug", false, "Debug messages and faster config saving")
	lastExitCode  = flag.Int64("exited", 0, "Called by Dockerfile")
	logFile       = "/tmp/taro-bot.log"
)

func main() {
	setupLogging()

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

	// program has been called with -exited, upload the logs and don't run the bot
	if *lastExitCode != 0 {
		checkExited()
		os.Exit(int(*lastExitCode))
		return
	} else { // clear old logs
		if err := os.Remove(logFile); err != nil {
			log.Printf("error removing logFile: %v\n", err)
		}
	}

	log.Printf("Started as %v (%s#%s)\n", u.ID, u.Username, u.Discriminator)

	// Block forever.
	select {}
}

func setupLogging() {
	logFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func checkExited() {
	log.Printf("Last exit code was %v\n", *lastExitCode)
	if config.OperatorChannel == 0 || config.OperatorID == 0 {
		log.Printf("Not uploading logs, OperatorChannel or OperatorID were not set\n")
		return
	}

	file, err := os.Open(logFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	found := false
	lines := make([]string, 0)
	for scanner.Scan() {
		if found || strings.HasPrefix(scanner.Text()[20:], "panic:") {
			found = true
			lines = append(lines, scanner.Text())
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	// Format stacktrace
	stack := "<@" + strconv.FormatInt(config.OperatorID, 10) + ">\n```\n" + strings.Join(lines, "\n")
	if len(stack) > 1996 {
		stack = stack[:1996]
	}
	stack += "\n```"

	_, err = discordClient.SendMessage(discord.ChannelID(config.OperatorChannel), stack)
	if err != nil {
		log.Fatalln(err)
	}
}

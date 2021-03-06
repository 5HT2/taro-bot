package main

import (
	"bufio"
	"flag"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

var (
	defaultPlugins = "base base-extra base-fun bookmarker leave-join-msg message-roles role-menu spotifytoyoutube " +
		"starboard remindme sys-stats suggest-topic tenor-delete"
	pluginDir    = flag.String("plugindir", "bin", "Default dir to search for plugins")
	pluginList   = flag.String("plugins", defaultPlugins, "List of plugins to load")
	lastExitCode = flag.Int64("exited", 0, "Called by Dockerfile")
	debugLog     = flag.Bool("debug", false, "Debug messages and faster config saving")
	debugLogFile = "/tmp/taro-bot.log"
)

func main() {
	flag.Parse()
	log.Printf("Running on Go version: %s\n", runtime.Version())

	// Load config before anything else, as it will be needed
	bot.LoadConfig()
	var token = bot.C.BotToken
	if token == "" {
		log.Fatalln("No bot_token given")
	}

	c := state.NewWithIntents("Bot "+token,
		gateway.IntentGuildMessages,
		gateway.IntentGuildEmojis,
		gateway.IntentGuildMessageReactions,
		gateway.IntentDirectMessages,
		gateway.IntentGuildMembers,
	)
	bot.Client = *c

	if c == nil {
		log.Fatalln("Session failed: is nil")
	}

	// Add handlers
	c.AddHandler(func(e *gateway.MessageCreateEvent) {
		go cmd.CommandHandler(e)
		go cmd.ResponseHandler(e)
	})
	c.AddHandler(func(e *gateway.GuildMemberUpdateEvent) {
		go cmd.UpdateMemberCache(e)
	})

	if err := c.Open(bot.Ctx); err != nil {
		log.Fatalln("Failed to connect:", err)
	}

	// Cancel context when SIGINT / SIGKILL / SIGTERM. SIGTERM is used by `docker stop`
	ctx, cancel := signal.NotifyContext(bot.Ctx, os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()

	if err := c.Open(ctx); err != nil {
		log.Println("cannot open:", err)
	}

	u, err := c.Me()
	if err != nil {
		log.Fatalln("Failed to get bot user:", err)
	}
	bot.User = u

	// program has been called with -exited, upload the logs and don't run the bot
	if lastExitCode != nil && *lastExitCode > 0 {
		checkExited()
		os.Exit(int(*lastExitCode))
		return
	}

	// We want http bash requests immediately accessible just in case something needs them.
	// Though, this shouldn't really ever happen, it doesn't hurt.
	util.RegisterHttpBashRequests()

	// Call plugins after logging in with the bot, but before doing anything else at all
	go plugins.RegisterAll(*pluginDir, *pluginList)

	// Set up the bots status
	go bot.LoadActivityStatus()

	// Now we can start the routine-based tasks
	go bot.SetupConfigSaving()
	go bot.Scheduler.StartAsync()

	log.Printf("Started as %v (%s#%s)\n", u.ID, u.Username, u.Discriminator)

	<-ctx.Done() // block until Ctrl+C / SIGINT / SIGTERM

	log.Println("received signal, shutting down")

	bot.SaveConfig()
	plugins.SaveConfig()

	if err := c.Close(); err != nil {
		log.Println("cannot close:", err)
	}

	log.Println("closed connection")
}

func checkExited() {
	log.Printf("Last exit code was %v\n", *lastExitCode)
	if bot.C.OperatorChannel == 0 || bot.C.OperatorID == 0 {
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

	if *debugLog {
		log.Printf("%v\n", bot.C.OperatorID)
	}

	// Format stacktrace
	stack := "<@" + strconv.FormatInt(bot.C.OperatorID, 10) + ">\n```\n" + strings.Join(lines, "\n")
	if len(stack) > 1996 {
		stack = stack[:1996]
	}
	stack += "\n```"

	if _, err = bot.Client.SendMessage(discord.ChannelID(bot.C.OperatorChannel), stack); err != nil {
		log.Fatalln(err)
	}
}

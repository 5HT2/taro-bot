package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/go-co-op/gocron"
	"log"
	"math/rand"
	"reflect"
)

var p *plugins.Plugin

type config struct {
	Fn string `json:"fn"`
}

// InitPlugin is called when a plugin is registered, and is used to register commands, responses, jobs and handlers.
func InitPlugin(i *plugins.PluginInit) *plugins.Plugin {
	// All the `FeatureNameInfo` fields are optional, and can be omitted.
	p = &plugins.Plugin{
		Name:        "Example plugin",
		Description: "This is an example plugin",
		Version:     "1.0.0",
		// Commands are called explicitly, with a prefix. For example, `.example` or `.err`.
		Commands: []bot.CommandInfo{{
			Fn:          ExampleCommand,
			FnName:      "ExampleCommand",
			Name:        "example",
			Description: "This command is an example",
		}, {
			Fn:          ErrorCommand,
			FnName:      "ErrorCommand",
			Name:        "error",
			Aliases:     []string{"err", "e"},
			Description: "This command will only return errors",
		}},
		// This is used to ensure type safety when loading the Config
		// If you forget to declare this and use p.LoadConfig(), you will get a safe panic when loading
		ConfigType: reflect.TypeOf(config{}),
		// Responses are called based on regex matching the message.
		// DISCORD_BOT_ID is replaced in the regex matching, and this response will be called by pinging the bot with the word test or help.
		// MatchMin means that a minimum of two of the Regexes need to match.
		// Using a MatchMin of 1 means you would only need to match a ping OR the test|help sequence.
		Responses: []bot.ResponseInfo{{
			Fn:       TestResponse,
			Regexes:  []string{"<@!?DISCORD_BOT_ID>", "(test|help)"},
			MatchMin: 2,
		}},
		// Jobs are called when they are scheduled. This job is scheduled for every minute.
		// Other examples include `bot.Scheduler.Every(1).Day().At("04:00")` (running a job every day at 4am)
		// as well as `bot.Scheduler.Cron("15 4 * * SUN")` (running a job every sunday at 4:15am, see https://crontab.guru/ for more).
		// The Name is used to identify the job when registering, and should be unique. The name itself doesn't actually matter as long as it is unique.
		Jobs: []bot.JobInfo{{
			Fn: func() (*gocron.Job, error) {
				return bot.Scheduler.Every(1).Minute().Do(EveryMinuteJob)
			},
			Name: "example-plugin-every-minute",
		}},
		// Handlers are functions that are registered to discord's event gateway. The documentation can be found at https://discord.com/developers/docs/topics/gateway
		// FnType is used to ensure type safety and simplify registration syntax.
		Handlers: []bot.HandlerInfo{{
			Fn:     ReactionHandler,
			FnName: "ReactionHandler",
			FnType: reflect.TypeOf(func(*gateway.MessageReactionAddEvent) {}),
		}},
		// ShutdownFn optionally allows you to register a function that will run when the bot has been killed / stopped.
		ShutdownFn: Shutdown,
		StartupFn:  Startup,
	}
	// This is required to set the config directory initially.
	p.ConfigDir = i.ConfigDir
	// When loading a config, you should cast not cast it, as it will be nil by default.
	// Instead, check if it is nil before doing .(config) in order to use it.
	p.Config = p.LoadConfig()
	return p
}

// Startup will run after all plugins have been loaded and before schedulers are started
func Startup() {
	log.Println("hello from the example plugin!")
}

// Shutdown will run when the bot is killed / stopped, and says goodbye to the console.
func Shutdown() {
	log.Println("goodbye from the example plugin!")
}

// ExampleCommand (.example) is a basic example of returning just a message with a command.
func ExampleCommand(c bot.Command) error {
	_, err := cmd.SendEmbed(c.E, "Example Command", "This command is an example", bot.DefaultColor)
	return err // error here is an error received by discord, it's usually nil, but we want to handle it anyways
}

// ErrorCommand (.err) will return only errors when called, as an example of how errors are handled in the bot.
func ErrorCommand(c bot.Command) error {
	// Errors are not usually defined in the command, and instead you use the return function to handle an error
	// mid-command when you want to stop.
	errors := []bot.Error{{
		Func:   "ErrorCommand",
		Action: "doing something",
		Err:    "expected something else",
	}, {
		Func:   "ErrorCommand",
		Action: "doing another thing",
		Err:    "expected not an error",
	}, {
		Func:   "ErrorCommand",
		Action: "one last thing",
		Err:    "big oops",
	}}

	// Choose a random error to return
	randomIndex := rand.Intn(len(errors))
	err := errors[randomIndex]

	_, _ = cmd.SendMessage(c.E, "We're doing something here, doesn't matter")
	return &err // return random error as an example
}

// TestResponse will send an embed when a message contains the @mention (ping) of the bot and the word help or the word test.
func TestResponse(r bot.Response) {
	_, _ = cmd.SendEmbed(r.E, "Test Response", "This response was called auto-magically", bot.DefaultColor)
}

// EveryMinuteJob will print something to the console every minute.
func EveryMinuteJob() {
	log.Printf("This was called from the example plugin, and is called every minute\n")
}

// ReactionHandler will send a message whenever someone adds a reaction to a message, as well as info about the reaction.
func ReactionHandler(i interface{}) {
	defer util.LogPanic()                     // handle panics and log them. panics are safe even without this, but aren't logged.
	e := i.(*gateway.MessageReactionAddEvent) // this is necessary to access the event. FnType ensures that this is safe.

	_, _ = cmd.SendCustomMessage(e.ChannelID, fmt.Sprintf("This is in response to a reaction added by <@%v>, the emoji name is `%s`", e.UserID, e.Emoji.Name))
}

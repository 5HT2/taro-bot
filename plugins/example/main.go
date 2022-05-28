package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"math/rand"
	"reflect"
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	// All the `FeatureNameInfo` fields are optional, and can be omitted.
	return &plugins.Plugin{
		Name:        "Example plugin",
		Description: "This is an example plugin",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          ExampleCommand,
			FnName:      "ExampleCommand",
			Name:        "example",
			Description: "This command is an example",
		}, {
			Fn:          ErrorCommand,
			FnName:      "ErrorCommand",
			Name:        "error",
			Description: "This command will only return errors",
			Aliases:     []string{"error", "e"},
		}},
		Responses: []bot.ResponseInfo{{
			Fn:       TestResponse,
			Regexes:  []string{"<@!?DISCORD_BOT_ID>", "(test|help)"},
			MatchMin: 2,
		}},
		Jobs: []bot.JobInfo{{
			Fn:        EveryMinuteJob,
			Tag:       "example-plugin-every-minute", // This is used to identify the job when de-registering, and should be unique
			Scheduler: bot.Scheduler.Every(1).Minute(),
		}},
		Handlers: []bot.HandlerInfo{{
			Fn:     ReactionHandler,
			FnName: "ReactionHandler",
			FnType: reflect.TypeOf(func(*gateway.MessageReactionAddEvent) {}),
		}},
	}
}

func ExampleCommand(c bot.Command) error {
	_, err := cmd.SendEmbed(c.E, "Example Command", "This command is an example", bot.DefaultColor)
	return err // error here is an error received by discord, it's usually nil, but we want to handle it anyways
}

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

func TestResponse(r bot.Response) {
	_, _ = cmd.SendEmbed(r.E, "Test Response", "This response was called auto-magically", bot.DefaultColor)
}

func EveryMinuteJob() {
	log.Printf("This was called from the example plugin, and is called every minute\n")
}

func ReactionHandler(i interface{}) {
	defer util.LogPanic()                     // handle panics and log them. panics are safe even without this, but aren't logged.
	e := i.(*gateway.MessageReactionAddEvent) // this is necessary to access the event. FnType ensures that this is safe.

	_, _ = cmd.SendCustomMessage(e.ChannelID, fmt.Sprintf("This is in response to a reaction added by <@%v>, the emoji name is `%s`", e.UserID, e.Emoji.Name))
}

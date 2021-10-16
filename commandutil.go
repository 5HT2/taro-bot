package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"reflect"
	"strings"
)

type Command struct {
	e    *gateway.MessageCreateEvent
	name string
}

// CommandHandler will parse commands and run the appropriate command func
func CommandHandler(e *gateway.MessageCreateEvent) {
	cmdName := extractCommandName(e.Message)
	if len(cmdName) == 0 {
		return
	}

	funcName, exists := commands[cmdName]
	if exists {
		command := Command{e, cmdName}
		result := invokeFunc(command, funcName)
		if len(result) > 0 {
			err, _ := result[0].Interface().(error)
			if err != nil {
				log.Printf("Error with \"%s\" command: %v\n", cmdName, err)
				SendErrorEmbed(command, err)
			}
		}
	}
}

// invokeFunc will magically invoke a function
func invokeFunc(any interface{}, name string, args ...interface{}) []reflect.Value {
	inputs := make([]reflect.Value, len(args))
	for i := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	return reflect.ValueOf(any).MethodByName(name).Call(inputs)
}

// extractCommandName will extract a command name from a message with a prefix
// TODO: add custom prefix support
func extractCommandName(message discord.Message) string {
	content := message.Content

	// If command doesn't start with a dot, or it's just a dot
	if !strings.HasPrefix(content, ".") || len(content) < 2 {
		return ""
	}

	// Remove prefix
	content = content[1:]
	// Split by space to remove
	contentArr := strings.Split(content, " ")
	return contentArr[0]
}

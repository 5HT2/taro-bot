package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"reflect"
	"strings"
)

var (
	successColor discord.Color = 0x3cde5a
	errorColor   discord.Color = 0xde413c
	defaultColor discord.Color = 0x493cde
)

type Command struct {
	e *gateway.MessageCreateEvent
}

// CommandHandler will parse commands and run the appropriate command func
func CommandHandler(e *gateway.MessageCreateEvent) {
	cmdName := extractCommandName(e.Message)
	if len(cmdName) == 0 {
		return
	}

	funcName, exists := commands[cmdName]
	if exists {
		invokeFunc(Command{e}, funcName)
	}
}

// invokeFunc will magically invoke a function
func invokeFunc(any interface{}, name string, args ...interface{}) {
	inputs := make([]reflect.Value, len(args))
	for i := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	reflect.ValueOf(any).MethodByName(name).Call(inputs)
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

func SendEmbed(c Command, title string, description string, color discord.Color) (*discord.Message, error) {
	msg, err := client.SendEmbeds(
		c.e.ChannelID,
		embed(title, description, color),
	)
	if err != nil {
		log.Printf("Error sending embed: %v", err)
	}
	return msg, err
}

func SendMessage(c Command, content string) (*discord.Message, error) {
	msg, err := client.SendMessage(
		c.e.ChannelID,
		content,
	)
	if err != nil {
		log.Printf("Error sending embed: %v", err)
	}
	return msg, err
}

func embed(title string, description string, color discord.Color) discord.Embed {
	return discord.Embed{
		Title:       title,
		Description: description,
		Color:       color,
	}
}

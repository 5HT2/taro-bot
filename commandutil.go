package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"strings"
)

type Command struct {
	e      *gateway.MessageCreateEvent
	name   string
	fnName string
	args   []string
}

// CommandHandler will parse commands and run the appropriate command func
func CommandHandler(e *gateway.MessageCreateEvent) {
	// Don't respond to bot messages.
	if e.Author.Bot {
		return
	}

	cmdName, cmdArgs := extractCommand(e.Message)
	if len(cmdName) == 0 {
		return
	}

	cmdInfo := getCommandWithName(cmdName)
	if cmdInfo != nil {
		command := Command{e, cmdName, cmdInfo.FnName, cmdArgs}
		result := InvokeFunc(command, cmdInfo.FnName)
		if len(result) > 0 {
			err, _ := result[0].Interface().(error)
			if err != nil {
				log.Printf("Error with \"%s\" command: %v\n", cmdName, err)
				SendErrorEmbed(command, err)
			}
		}
	}
}

// extractCommand will extract a command name and args from a message with a prefix
func extractCommand(message discord.Message) (string, []string) {
	content := message.Content
	cfg := GetGuildConfig(int64(message.GuildID))

	// If command doesn't start with a dot, or it's just a dot
	if !strings.HasPrefix(content, cfg.Prefix) || len(content) < (1+len(cfg.Prefix)) {
		return "", []string{}
	}

	// Remove prefix
	content = content[1*len(cfg.Prefix):]
	// Split by space to remove everything after the prefix
	contentArr := strings.Split(content, " ")
	// Get first element of slice (the command name)
	contentLower := strings.ToLower(contentArr[0])
	// Remove first element of slice (the command name)
	contentArr = append(contentArr[:0], contentArr[1:]...)
	return contentLower, contentArr
}

// getCommandWithName will return the found CommandInfo with a matching name or alias
func getCommandWithName(name string) *CommandInfo {
	for _, cmd := range commands {
		if cmd.Name == name || StringSliceContains(cmd.Aliases, name) {
			return &cmd
		}
	}
	return nil
}

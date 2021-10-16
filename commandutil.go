package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
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

	cmdInfo := getCommandWithName(cmdName)
	if cmdInfo != nil {
		command := Command{e, cmdName}
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

// extractCommandName will extract a command name from a message with a prefix
func extractCommandName(message discord.Message) string {
	content := message.Content
	cfg := GetGuildConfig(int64(message.GuildID))

	// If command doesn't start with a dot, or it's just a dot
	if !strings.HasPrefix(content, cfg.Prefix) || len(content) < (1+len(cfg.Prefix)) {
		return ""
	}

	// Remove prefix
	content = content[1*len(cfg.Prefix):]
	// Split by space to remove everything after the prefix
	contentArr := strings.Split(content, " ")
	contentLower := strings.ToLower(contentArr[0])
	return contentLower
}

func getCommandWithName(name string) *CommandInfo {
	for _, cmd := range commands {
		if cmd.Name == name || StringSliceContains(cmd.Aliases, name) {
			return &cmd
		}
	}
	return nil
}

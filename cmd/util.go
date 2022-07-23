package cmd

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"path/filepath"
	"strings"
)

var (
	ImageExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".gifv"}
)

// CommandHandler will parse commands and run the appropriate command func
func CommandHandler(e *gateway.MessageCreateEvent) {
	defer util.LogPanic()

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
		command := bot.Command{E: e, FnName: cmdInfo.FnName, Name: cmdName, Args: cmdArgs}

		if cmdInfo.GuildOnly && !e.GuildID.IsValid() {
			_, err := SendEmbed(e, "Error", "The `"+cmdInfo.Name+"` command only works in guilds!", bot.ErrorColor)
			if err != nil {
				log.Printf("Error with \"%s\" command (Cancelled): %v\n", cmdName, err)
			}
			return
		}

		if err := cmdInfo.Fn(command); err != nil {
			log.Printf("Error with \"%s\" command: %v\n", cmdName, err)
			SendErrorEmbed(command, err)
		}
	}
}

// extractCommand will extract a command name and args from a message with a prefix
func extractCommand(message discord.Message) (string, []string) {
	content := message.Content
	prefix := bot.DefaultPrefix
	ok := true

	if !message.GuildID.IsValid() {
		prefix = ""
	} else {
		bot.C.Run(func(c *bot.Config) {
			prefix, ok = c.PrefixCache[int64(message.GuildID)]
		})

		// If the PrefixCache somehow doesn't have a prefix, set a default one and log it.
		// This is most likely when the bot has joined a new guild without accessing GuildContext
		if !ok {
			log.Printf("expected prefix to be in prefix cache: %s (%s)\n",
				message.GuildID, CreateMessageLink(int64(message.GuildID), &message, false, false))

			bot.GuildContext(message.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
				g.Prefix = bot.DefaultPrefix
				return g, "extractCommand: reset prefix"
			})

			prefix = bot.DefaultPrefix
		}
	}

	// If command doesn't start with a dot, or it's just a dot
	if !strings.HasPrefix(content, prefix) || len(content) < (1+len(prefix)) {
		return "", []string{}
	}

	// Remove prefix
	content = content[1*len(prefix):]
	// Split by space to remove everything after the prefix
	contentArr := strings.Split(content, " ")
	// Get first element of slice (the command name)
	contentLower := strings.ToLower(contentArr[0])
	// Remove first element of slice (the command name)
	contentArr = append(contentArr[:0], contentArr[1:]...)
	return contentLower, contentArr
}

// getCommandWithName will return the found CommandInfo with a matching name or alias
func getCommandWithName(name string) *bot.CommandInfo {
	for _, cmd := range bot.Commands {
		if cmd.Name == name || util.SliceContains(cmd.Aliases, name) {
			return &cmd
		}
	}
	return nil
}

func FileExtMatches(s []string, file string) bool {
	found := false
	file = strings.ToLower(file)

	for _, e := range s {
		if filepath.Ext(file) == e {
			found = true
			break
		}
	}

	return found
}

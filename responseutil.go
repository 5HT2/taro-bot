package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"regexp"
	"strings"
)

// ResponseHandler will find a global response from the config and send it, if found
func ResponseHandler(e *gateway.MessageCreateEvent) {
	defer util.LogPanic()

	// TODO: Per-guild responses and configuration
	// TODO: compiling and caching support could be added here to improve speed
	for _, response := range bot.Responses {
		if findResponse(e, response) {
			sendResponse(e, response)
		}
	}
}

func sendResponse(e *gateway.MessageCreateEvent, response bot.ResponseInfo) {
	// Don't respond to bot messages.
	if e.Author.Bot {
		return
	}

	// If there is a channel whitelist, and it doesn't contain the original message's channel ID, return
	if len(response.LockChannels) > 0 && !util.SliceContains(response.LockChannels, int64(e.ChannelID)) {
		return
	}

	// If there is a user whitelist, and it doesn't contain the original author's ID, return
	if len(response.LockUsers) > 0 && !util.SliceContains(response.LockUsers, int64(e.Author.ID)) {
		return
	}

	embed := discord.Embed{
		Title:       response.Title,
		Description: response.Description,
		Color:       bot.DefaultColor,
	}
	msgContent := response.Description

	if response.Fn != nil {
		result := response.Fn(bot.ResponseReflection{E: e})
		if len(result) > 0 {
			slice := make([]interface{}, 0)
			for _, str := range result {
				slice = append(slice, str)
			}
			if response.Embed {
				embed.Description = fmt.Sprintf(embed.Description, slice...)
			} else {
				msgContent = fmt.Sprintf(embed.Description, slice...)
			}
		}
	}

	if response.Embed {
		_, err := SendCustomEmbed(e.ChannelID, embed)
		if err != nil {
			log.Printf("Error sending global response: %v\n", err)
		}
	} else {
		_, err := SendCustomMessage(e.ChannelID, msgContent)
		if err != nil {
			log.Printf("Error sending global response: %v\n", err)
		}
	}
}

func findResponse(e *gateway.MessageCreateEvent, response bot.ResponseInfo) bool {
	matched := 0
	message := []byte(e.Message.Content)
	for _, regex := range response.Regexes {
		// Allow using a variable in the regex to represent the current bot user
		// TODO: Documentation for these
		regex = strings.ReplaceAll(regex, "DISCORD_BOT_ID", bot.User.ID.String())

		found, err := regexp.Match(regex, message)
		if err != nil {
			log.Printf("Error matching \"%s\": %v\n", regex, err)
		}
		if found {
			matched += 1
		}

		if matched >= response.MatchMin {
			return true
		}
	}

	return false
}

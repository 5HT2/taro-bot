package cmd

import "C"
import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"regexp"
	"strings"
)

// ResponseHandler will find a global response from the config and send it, if found
func ResponseHandler(e *gateway.MessageCreateEvent) {
	defer util.LogPanic()

	// Don't respond to bot messages.
	if e.Author.Bot {
		return
	}

	// TODO: Per-guild responses and configuration
	// TODO: compiling and caching support could be added here to improve speed
	go func() {
		for _, response := range bot.Responses {
			runResponse(e, response)
		}
	}()
}

func runResponse(e *gateway.MessageCreateEvent, response bot.ResponseInfo) {
	if findResponse(e, response) {
		sendResponse(e, response)
	}
}

func sendResponse(e *gateway.MessageCreateEvent, response bot.ResponseInfo) {
	// If there is a channel whitelist, and it doesn't contain the original message's channel ID, return
	if e.ChannelID.IsValid() && len(response.LockChannels) > 0 && !util.SliceContains(response.LockChannels, int64(e.ChannelID)) {
		return
	}

	// If there is a user whitelist, and it doesn't contain the original author's ID, return
	if e.ChannelID.IsValid() && len(response.LockUsers) > 0 && !util.SliceContains(response.LockUsers, int64(e.Author.ID)) {
		return
	}

	if response.Fn != nil {
		response.Fn(bot.Response{E: e})
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

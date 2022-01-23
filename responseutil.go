package main

import (
	"fmt"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"regexp"
)

// ResponseHandler will find a global response from the config and send it, if found
func ResponseHandler(e *gateway.MessageCreateEvent) {
	// TODO: Per-guild responses and configuration
	// TODO: compiling and caching support could be added here to improve speed
	for _, response := range config.GlobalResponses {
		if findResponse(e, response) {
			sendResponse(e, response)
		}
	}
}

func sendResponse(e *gateway.MessageCreateEvent, response Response) {
	embed := discord.Embed{
		Title:       response.Title,
		Description: response.Description,
		Color:       defaultColor,
	}
	msgContent := response.Description

	if len(response.ReflectFunc) > 0 {
		result := CallStringFunc(ResponseReflection{e}, response.ReflectFunc)
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

func findResponse(e *gateway.MessageCreateEvent, response Response) bool {
	matched := 0
	message := []byte(e.Message.Content)
	for _, regex := range response.Regexes {
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

package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"regexp"
)

type Response struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Regexes     []string `json:"regexes"`
	MatchMin    int      `json:"match_min,omitempty"`
}

func ResponseHandler(e *gateway.MessageCreateEvent) {
	handleGlobalResponses(e)
}

func handleGlobalResponses(e *gateway.MessageCreateEvent) {
	// TODO: compiling and caching support could be added here to improve speed
	for _, response := range config.GlobalResponses {
		if findResponse(e, response) {
			embed := discord.Embed{Title: response.Title, Description: response.Description, Color: defaultColor}
			_, err := SendCustomEmbed(e.ChannelID, embed)
			if err != nil {
				log.Printf("Error sending global response: %v\n", err)
			}
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

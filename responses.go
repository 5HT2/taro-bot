package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
)

func RegisterResponses() {
	bot.Mutex.Lock()
	defer bot.Mutex.Unlock()

	bot.Responses = append(bot.Responses,
		bot.ResponseInfo{
			Fn:       PrefixResponse,
			Embed:    true,
			Regexes:  []string{"<@!?DISCORD_BOT_ID>", "prefix"},
			MatchMin: 2,
		},
	)
}

func PrefixResponse(r bot.ResponseReflection) string {
	prefix := defaultPrefix
	GuildContext(r.E.GuildID, func(g *GuildConfig) (*GuildConfig, string) {
		prefix = g.Prefix
		return g, "PrefixResponse"
	})

	return fmt.Sprintf("The current prefix is `%s`", prefix)
}

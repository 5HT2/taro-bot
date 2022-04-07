package main

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2C/http-bash-requests/httpBashRequests"
	"strings"
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	return &plugins.Plugin{
		Name:        "VintageStory",
		Description: "Manages VS Docker containers",
		Version:     "1.0.0",
		Commands:    []bot.CommandInfo{},
		Responses: []bot.ResponseInfo{{
			Fn:          VintageStoryRebootResponse,
			Embed:       true,
			Description: "%s",
			Regexes:     []string{"<@!?DISCORD_BOT_ID>", "vs", "restart"},
			MatchMin:    3,
		}},
	}
}

func VintageStoryRebootResponse(r bot.ResponseReflection) []string {
	servers := []string{"vintagestory0"}
	if strings.Contains(r.E.Content, "both") {
		servers = append(servers, "vintagestory1")
	} else if strings.Contains(r.E.Content, "test") {
		servers = []string{"vintagestory1"}
	}

	responses := make([]string, 0)
	for _, s := range servers {
		if res, err := httpBashRequests.Run("docker restart " + s); err != nil {
			responses = append(responses, "Response from `"+s+"`: `"+err.Error()+"`")
		} else {
			responses = append(responses, "Response from `"+s+"`: `"+string(res)+"`")
		}
	}

	return []string{"Okay, sent restart command(s). Responses:\n\n" + strings.Join(responses, "")}
}

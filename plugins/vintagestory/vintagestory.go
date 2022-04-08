package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2C/http-bash-requests/httpBashRequests"
	"github.com/diamondburned/arikawa/v3/discord"
	"log"
	"strings"
)

var (
	botID = discord.UserID(893216230410952785)
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	if bot.User.ID == botID {
		vintageStorySetup()
	}

	return &plugins.Plugin{
		Name:        "VintageStory",
		Description: "Manages VS Docker containers",
		Version:     "1.0.0",
		Commands:    []bot.CommandInfo{},
		Responses: []bot.ResponseInfo{{
			Fn:       VintageStoryRebootResponse,
			Embed:    true,
			Title:    "VintageStory",
			Regexes:  []string{"<@!?DISCORD_BOT_ID>", "vs", "restart"},
			MatchMin: 3,
		}},
	}
}

func VintageStoryRebootResponse(r bot.Response) string {
	if bot.User.ID != botID {
		return "Not setup for this server!"
	}

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

	return "Okay, sent restart command(s). Responses:\n\n" + strings.Join(responses, "")
}

func vintageStorySetup() {
	// TODO: This will have its own config value as a plugin
	vsChannel := 959129039401025606
	logVS := func(desc string, err error) {
		color := bot.DefaultColor
		embed := discord.Embed{
			Title:       "VintageStory",
			Description: desc,
			Color:       color,
		}

		if err != nil {
			embed.Description += err.Error()
			embed.Color = bot.ErrorColor
		}

		_, err = bot.Client.SendEmbeds(discord.ChannelID(vsChannel), embed)
		if err != nil {
			log.Printf("Error with logVS: %v\n", err)
		}
	}

	backupVS := func(name, path, backupName string) {
		logVS(fmt.Sprintf("Shutting down `%s`...", name), nil)
		if _, err := httpBashRequests.Run("docker stop " + name); err != nil {
			logVS("Error with Docker: ", err)
			return
		}

		if _, err := httpBashRequests.Run(fmt.Sprintf("sudo cp %sdefault.vcdbs %s%s", path, path, backupName)); err != nil {
			logVS("Error with copying file: ", err)
			return
		}

		if res, err := httpBashRequests.Run("docker start " + name); err != nil {
			logVS("Error with Docker: ", err)
			return
		} else {
			logVS(fmt.Sprintf("Started `%s`\n```\n%s\n```", name, res), nil)
		}
	}

	// Run a daily backup at 04:00
	if job, err := bot.Scheduler.Every(1).Day().At("04:00").Do(func() {
		backupVS("vintagestory0", "fs/vintagestory/Saves/", "daily.vcdbs")
		backupVS("vintagestory1", "fs/vs1/Saves/", "daily.vcdbs")
	}); err != nil {
		log.Printf("error setting up job: %v\n%v\n", job, err)
	} else {
		log.Printf("setup job: %v\n", job)
	}

	// Run a weekly backup at 04:15 on Sunday
	if job, err := bot.Scheduler.Cron("15 4 * * SUN").Do(func() {
		backupVS("vintagestory0", "fs/vintagestory/Saves/", "weekly.vcdbs")
		backupVS("vintagestory1", "fs/vs1/Saves/", "weekly.vcdbs")
	}); err != nil {
		log.Printf("error setting up job: %v\n%v\n", job, err)
	} else {
		log.Printf("setup job: %v\n", job)
	}
}

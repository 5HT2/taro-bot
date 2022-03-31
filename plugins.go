package main

import (
	"fmt"
	"github.com/5HT2C/http-bash-requests/httpBashRequests"
	"github.com/diamondburned/arikawa/v3/discord"
	"log"
	"net/http"
	"time"
)

// This is an external file to be used for more specific features.
// Ideally, these shouldn't be in the codebase at all.
// These are all planned to be migrated to plugins, once support
// for plugins has been added.

func SetupPlugins() {
	client := httpBashRequests.Client{Addr: "http://localhost:6016", HttpClient: &http.Client{Timeout: 5 * time.Minute}}
	httpBashRequests.Setup(&client)

	config.run(func(c *Config) {
		// TODO: This will have its own config value as a plugin
		if c.OperatorID == 242462997530804225 {
			vintageStorySetup()
		}
	})

	scheduler.StartAsync()
}

func vintageStorySetup() {
	// TODO: This will have its own config value as a plugin
	vsChannel := 959129039401025606
	logVS := func(desc string, err error) {
		color := defaultColor
		embed := discord.Embed{
			Title:       "VintageStory",
			Description: desc,
			Color:       color,
		}

		if err != nil {
			embed.Description += err.Error()
			embed.Color = errorColor
		}

		_, err = discordClient.SendEmbeds(discord.ChannelID(vsChannel), embed)
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
	if job, err := scheduler.Every(1).Day().At("04:00").Do(func() {
		backupVS("vintagestory", "fs/vintagestory/Saves/", "daily.vcdbs")
		backupVS("vintagestory1", "fs/vs1/Saves/", "daily.vcdbs")
	}); err != nil {
		log.Printf("error setting up job: %v\n%v\n", job, err)
	} else {
		log.Printf("setup job: %v\n", job)
	}

	// Run a weekly backup at 04:15 on Sunday
	if job, err := scheduler.Cron("15 4 * * SUN").Do(func() {
		backupVS("vintagestory", "fs/vintagestory/Saves/", "weekly.vcdbs")
		backupVS("vintagestory1", "fs/vs1/Saves/", "weekly.vcdbs")
	}); err != nil {
		log.Printf("error setting up job: %v\n%v\n", job, err)
	} else {
		log.Printf("setup job: %v\n", job)
	}
}

package main

import (
	"encoding/json"
	"github.com/diamondburned/arikawa/v3/discord"
	"log"
	"os"
	"sync"
	"time"
)

var (
	config        Config
	fileMode      = os.FileMode(0700)
	defaultPrefix = "."
)

type configOperation func(*Config)
type guildOperation func(*GuildConfig) (*GuildConfig, string)

// GuildContext will modify a GuildConfig non-concurrently.
// Avoid using inside a network or hang-able context whenever possible.
// TODO: Having one "context" per command would be nice to have.
func GuildContext(c discord.GuildID, g guildOperation) {
	id := int64(c)
	start := time.Now().UnixMilli()
	found := false

	config.run(func(c *Config) {
		// Try to find an existing config, and if so, replace it with the result of executed guildOperation
		// TODO: This isn't scalable with lots of Guilds, so a map would be preferable
		for n, guild := range c.GuildConfigs {
			if guild.ID == id {
				// Correct guild found, execute guildOperation
				res, fnName := g(&guild)
				c.GuildConfigs[n] = *res
				found = true

				exec := time.Now().UnixMilli()
				log.Printf("Execute: %vms (%s)\n", exec-start, fnName)
				break
			}
		}

		// If we didn't find an existing config, run guildOperation with the defaultConfig, and append it to the list
		if !found {
			defaultConfig := GuildConfig{ID: id, Prefix: defaultPrefix}
			c.PrefixCache[id] = defaultPrefix

			res, _ := g(&defaultConfig)
			c.GuildConfigs = append(c.GuildConfigs, *res)
		}
	})
}

// Config.run will modify a Config non-concurrently.
// Avoid using inside a network or hang-able context whenever possible.
func (c *Config) run(co configOperation) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	co(c)
}

type Config struct {
	Mutex           sync.Mutex       `json:"-"` // not saved in DB
	PrefixCache     map[int64]string `json:"-"` // not saved in DB // [guild id]prefix
	BotToken        string           `json:"bot_token"`
	OperatorChannel int64            `json:"operator_channel,omitempty"`
	OperatorID      int64            `json:"operator_id,omitempty"`
	GlobalResponses []Response       `json:"global_responses,omitempty"`
	GuildConfigs    []GuildConfig    `json:"guild_configs,omitempty"`
}

type GuildConfig struct {
	ID                   int64             `json:"id"`
	ArchiveRole          int64             `json:"archive_role,omitempty"`
	ArchiveCategory      int64             `json:"archive_category,omitempty"`
	Prefix               string            `json:"prefix,omitempty"`
	LogChannel           string            `json:"log_channel,omitempty"`
	Permissions          PermissionGroups  `json:"permissions,omitempty"`
	EnabledTopicChannels []int64           `json:"enabled_topic_channels,omitempty"`
	ActiveTopicVotes     []ActiveTopicVote `json:"active_topic_votes,omitempty"`
	TopicVoteThreshold   int64             `json:"topic_vote_threshold,omitempty"`
	TopicVoteEmoji       string            `json:"topic_vote_emoji,omitempty"`
	Starboard            StarboardConfig   `json:"starboard_config"`
}

// SetupConfigSaving will run SaveLocalInDatabase every 5 minutes with a ticker
func SetupConfigSaving() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				SaveConfig()
			}
		}
	}()
}

func LoadConfig() {
	bytes, err := os.ReadFile("config/config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v\n", err)
	}

	if err := json.Unmarshal(bytes, &config); err != nil {
		log.Fatalf("Error unmarshalling config: %v\n", err)
	}

	// Load prefix cache
	config.run(func(c *Config) {
		c.PrefixCache = make(map[int64]string, 0)

		for _, g := range c.GuildConfigs {
			c.PrefixCache[g.ID] = g.Prefix
		}
	})
}

func SaveConfig() {
	var bytes []byte
	var err error = nil

	config.run(func(c *Config) {
		bytes, err = json.MarshalIndent(c, "", "    ")
	})

	if err != nil {
		log.Printf("Failed to marshal config: %v\n", err)
		return
	}

	err = os.WriteFile("config/config.json", bytes, fileMode)
	if err != nil {
		log.Printf("Failed to write config: %v\n", err)
	} else {
		log.Printf("Successfully saved config\n")
	}
}

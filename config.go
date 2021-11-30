package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

var (
	config   Config
	fileMode = os.FileMode(0700)
)

type Config struct {
	BotToken         string            `json:"bot_token"`
	GlobalResponses  []Response        `json:"global_responses,omitempty"`
	GuildConfigs     []GuildConfig     `json:"guild_configs,omitempty"`
	StarboardConfigs []StarboardConfig `json:"starboard_configs,omitempty"`
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
}

func SaveConfig() {
	bytes, err := json.MarshalIndent(config, "", "    ")
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

func GetGuildConfig(guild int64) GuildConfig {
	defaultConfig := GuildConfig{ID: guild, Prefix: "."}

	if len(config.GuildConfigs) == 0 {
		return SetGuildConfig(defaultConfig)
	}

	for _, cfg := range config.GuildConfigs {
		if cfg.ID == guild {
			return cfg
		}
	}

	return SetGuildConfig(defaultConfig)
}

func SetGuildConfig(guildConfig GuildConfig) GuildConfig {
	for n, cfg := range config.GuildConfigs {
		if cfg.ID == guildConfig.ID {
			config.GuildConfigs[n] = guildConfig
			if *debug {
				SaveConfig()
			}
			return guildConfig
		}
	}

	// Append if not found in existing configs
	config.GuildConfigs = append(config.GuildConfigs, guildConfig)
	SaveConfig()
	return guildConfig
}

func GetStarboardConfig(guild int64) StarboardConfig {
	defaultConfig := StarboardConfig{ID: guild}

	if len(config.GuildConfigs) == 0 {
		return SetStarboardConfig(defaultConfig)
	}

	for _, cfg := range config.StarboardConfigs {
		if cfg.ID == guild {
			return cfg
		}
	}

	return SetStarboardConfig(defaultConfig)
}

func SetStarboardConfig(guildConfig StarboardConfig) StarboardConfig {
	for n, cfg := range config.StarboardConfigs {
		if cfg.ID == guildConfig.ID {
			config.StarboardConfigs[n] = guildConfig
			if *debug {
				SaveConfig()
			}
			return guildConfig
		}
	}

	// Append if not found in existing configs
	config.StarboardConfigs = append(config.StarboardConfigs, guildConfig)
	SaveConfig()
	return guildConfig
}

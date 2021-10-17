package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

var (
	config        Config
	fileMode      = os.FileMode(0700)
	defaultPrefix = "."
)

type Config struct {
	BotToken        string        `json:"bot_token"`
	GlobalResponses []Response    `json:"global_responses,omitempty"`
	GuildConfigs    []GuildConfig `json:"guild_configs,omitempty"`
}

type GuildConfig struct {
	ID          int64            `json:"id"`
	Prefix      string           `json:"prefix,omitempty"`
	LogChannel  string           `json:"log_channel,omitempty"`
	Permissions PermissionGroups `json:"permissions,omitempty"`
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
	defaultConfig := GuildConfig{ID: guild, Prefix: defaultPrefix}

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
			SaveConfig()
			return guildConfig
		}
	}

	// Append if not found in existing configs
	config.GuildConfigs = append(config.GuildConfigs, guildConfig)
	SaveConfig()
	return guildConfig
}

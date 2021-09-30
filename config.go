package main

import (
	"encoding/json"
	"github.com/diamondburned/arikawa/v3/discord"
	"log"
	"os"
	"strconv"
)

var (
	config Config
)

type Config struct {
	BotToken      string         `json:"bot_token"`
	AuditChannels []AuditChannel `json:"audit_channels,omitempty"`
}

type AuditChannel struct {
	GuildID   int64 `json:"guild_id"`
	ChannelID int64 `json:"channel_id"`
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

func setAuditChannel(channel discord.Channel, guildID int64, channelID int64) {
	// look for existing guild entry, and edit it
	for n, auditChannel := range config.AuditChannels {
		if auditChannel.GuildID == guildID {
			config.AuditChannels[n].ChannelID = channelID
			return
		}
	}

	// didn't find an existing entry, so add one
	auditChannel := AuditChannel{guildID, channelID}
	config.AuditChannels = append(config.AuditChannels, auditChannel)

	SendEmbed(channel.ID,
		"",
		"Set audit channel to <#"+strconv.FormatInt(channelID, 10)+">",
		successColor)
}

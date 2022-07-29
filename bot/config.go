package bot

import (
	"encoding/json"
	"fmt"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	C             Config
	fileMode      = os.FileMode(0700)
	DefaultPrefix = "."
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

	C.Run(func(c *Config) {
		// Try to find an existing config, and if so, replace it with the result of executed guildOperation
		// TODO: This isn't scalable with lots of Guilds, so a map would be preferable. See #6
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
			defaultConfig := GuildConfig{ID: id, Prefix: DefaultPrefix}
			c.PrefixCache[id] = DefaultPrefix

			res, _ := g(&defaultConfig)
			c.GuildConfigs = append(c.GuildConfigs, *res)
		}
	})
}

// Run will modify a Config non-concurrently.
// Avoid using inside a network or hang-able context whenever possible.
func (c *Config) Run(co configOperation) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	co(c)
}

type Config struct {
	Mutex           sync.Mutex       `json:"-"` // not saved in DB
	PrefixCache     map[int64]string `json:"-"` // not saved in DB // [guild id]prefix
	BotToken        string           `json:"bot_token"`
	ActivityName    string           `json:"activity_name,omitempty"` // See LoadActivityStatus
	ActivityUrl     string           `json:"activity_url,omitempty"`  // See LoadActivityStatus
	ActivityType    uint8            `json:"activity_type,omitempty"` // See LoadActivityStatus
	OperatorChannel int64            `json:"operator_channel,omitempty"`
	OperatorID      int64            `json:"operator_id,omitempty"`
	GuildConfigs    []GuildConfig    `json:"guild_configs,omitempty"`
}

type GuildConfig struct {
	ID                   int64             `json:"id"`
	Prefix               string            `json:"prefix,omitempty"`
	Permissions          PermissionGroups  `json:"permissions,omitempty"`
	ArchiveRole          int64             `json:"archive_role,omitempty"`           // TODO: Migrate
	ArchiveCategory      int64             `json:"archive_category,omitempty"`       // TODO: Migrate
	EnabledTopicChannels []int64           `json:"enabled_topic_channels,omitempty"` // TODO: Migrate
	ActiveTopicVotes     []ActiveTopicVote `json:"active_topic_votes,omitempty"`     // TODO: Migrate
	TopicVoteThreshold   int64             `json:"topic_vote_threshold,omitempty"`   // TODO: Migrate
	TopicVoteEmoji       string            `json:"topic_vote_emoji,omitempty"`       // TODO: Migrate
	Starboard            StarboardConfig   `json:"starboard_config"`                 // TODO: Migrate
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
		log.Fatalf("error loading config: %v\n", err)
	}

	if err := json.Unmarshal(bytes, &C); err != nil {
		log.Fatalf("error unmarshalling config: %v\n", err)
	}

	// Load prefix cache
	C.Run(func(c *Config) {
		c.PrefixCache = make(map[int64]string, 0)

		for _, g := range c.GuildConfigs {
			c.PrefixCache[g.ID] = g.Prefix
		}
	})
}

func SaveConfig() {
	var bytes []byte
	var err error = nil

	C.Run(func(c *Config) {
		bytes, err = json.MarshalIndent(c, "", "    ")
	})

	if err != nil {
		log.Printf("failed to marshal config: %v\n", err)
		return
	}

	err = os.WriteFile("config/config.json", bytes, fileMode)
	if err != nil {
		log.Printf("failed to write config: %v\n", err)
	} else {
		log.Printf("successfully saved taro config\n")
	}
}

// LoadActivityStatus will load the activity information from the config.
// Using USER_ID, USER_TAG and USER_USERNAME as replacements for the discord.Activity name are all supported.
// Setting URL is only useful for a Twitch or YouTube discord.StreamingActivity.
// The activity type uint8 is derived from its position in the list, eg, 0 == discord.GameActivity and 2 == discord.ListeningActivity.
func LoadActivityStatus() {
	name := ""
	url := ""
	var activityType uint8 = 0

	C.Run(func(c *Config) {
		name = c.ActivityName
		url = c.ActivityUrl
		activityType = c.ActivityType
	})
	name = strings.ReplaceAll(name, "USER_ID", fmt.Sprintf("%v", User.ID))
	name = strings.ReplaceAll(name, "USER_TAG", fmt.Sprintf("%v", User.Tag()))
	name = strings.ReplaceAll(name, "USER_USERNAME", fmt.Sprintf("%v", User.Username))

	if err := Client.Gateway().Send(Ctx, &gateway.UpdatePresenceCommand{
		Activities: []discord.Activity{{Name: name, URL: url, Type: discord.ActivityType(activityType)}},
	}); err != nil {
		log.Printf("error loading activity status: %v\n", err)
	}
}

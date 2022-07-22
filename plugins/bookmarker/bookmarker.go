package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"log"
	"reflect"
	"sync"
)

var (
	p     *plugins.Plugin
	mutex sync.Mutex

	embedColor discord.Color = 0x0099FF

	enabledFooter   = discord.EmbedFooter{Text: "Messages will be DMed to you when you react with a 🔖."}
	escapedBookmark = "%F0%9F%94%96"
)

type config struct {
	EnabledGuilds map[string]bool `json:"enabled_guilds,omitempty"` // [guild id]bool
}

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	// All the `FeatureNameInfo` fields are optional, and can be omitted.
	p = &plugins.Plugin{
		Name:        "Bookmarker",
		Description: "Bookmark messages to your DMs",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          BookmarkConfigCommand,
			FnName:      "BookmarkConfigCommand",
			Name:        "bookmarkconfig",
			Aliases:     []string{"bcfg"},
			Description: "Enable or disable bookmarking messages",
			GuildOnly:   true,
		}},
		ConfigType: reflect.TypeOf(config{}),
		Handlers: []bot.HandlerInfo{{
			Fn:     BookmarkReactionHandler,
			FnName: "BookmarkReactionHandler",
			FnType: reflect.TypeOf(func(*gateway.MessageReactionAddEvent) {}),
		}},
	}
	p.Config = p.LoadConfig()
	return p
}

func BookmarkConfigCommand(c bot.Command) error {
	mutex.Lock()
	defer mutex.Unlock()

	enabled := true // enabled by default
	id := c.E.GuildID.String()

	if p.Config != nil {
		enabledGuild, ok := p.Config.(config).EnabledGuilds[id]

		if ok {
			enabled = enabledGuild
		}
	}

	var err error = nil
	arg, _ := cmd.ParseStringArg(c.Args, 1, true)

	switch arg {
	case "toggle":
		enabled = !enabled
		if enabled {
			embed := cmd.MakeEmbed(p.Name, "Bookmarker enabled!", bot.SuccessColor)
			embed.Footer = &enabledFooter
			_, err = cmd.SendCustomEmbed(c.E.ChannelID, embed)
		} else {
			_, err = cmd.SendEmbed(c.E, p.Name, "Bookmarker disabled!", bot.ErrorColor)
		}
	default:
		if enabled {
			embed := cmd.MakeEmbed(p.Name, "Bookmarker is currently enabled!\nUse `bookmarkerconfig toggle` to disable it.", bot.SuccessColor)
			embed.Footer = &enabledFooter
			_, err = cmd.SendCustomEmbed(c.E.ChannelID, embed)
		} else {
			_, err = cmd.SendEmbed(c.E, p.Name, "Bookmarker is currently disabled!\nUse `bookmarkerconfig toggle` to enable it.", bot.ErrorColor)
		}
	}

	if p.Config == nil {
		guilds := make(map[string]bool)
		guilds[id] = enabled
		cfg := config{EnabledGuilds: guilds}
		p.Config = cfg
	} else {
		p.Config.(config).EnabledGuilds[id] = enabled
	}

	return err
}

func BookmarkReactionHandler(i interface{}) {
	mutex.Lock()
	defer mutex.Unlock()
	defer util.LogPanic()
	e := i.(*gateway.MessageReactionAddEvent)

	// Bot reacted
	if e.Member.User.Bot {
		return
	}

	// Not a bookmark emoji
	if e.Emoji.APIString().PathString() != escapedBookmark {
		return
	}

	sendBookmark := false

	if p.Config == nil {
		log.Println("here")
		// Enabled by default
		sendBookmark = true
	} else {
		enabled, ok := p.Config.(config).EnabledGuilds[e.GuildID.String()]
		log.Println("here2")

		// If not in the config (enabled by default) or explicitly enabled
		if !ok || enabled {
			log.Printf("here3 %v %v\n", ok, enabled)
			log.Printf("here4 %v\n", p.Config.(config).EnabledGuilds)

			sendBookmark = true
		}
	}

	if sendBookmark {
		msg, err := bot.Client.Message(e.ChannelID, e.MessageID)
		if err != nil {
			return
		}

		content := fmt.Sprintf("🔖 from <#%v>", e.ChannelID)
		field := discord.EmbedField{Name: "Source", Value: cmd.CreateMessageLink(int64(e.GuildID), msg, true)}
		footer := discord.EmbedFooter{Text: fmt.Sprintf("%v", msg.Author.ID)}

		description, image := cmd.GetEmbedAttachmentAndContent(*msg)

		embed := discord.Embed{
			Description: description,
			Author:      cmd.CreateEmbedAuthorUser(msg.Author),
			Timestamp:   msg.Timestamp,
			Fields:      []discord.EmbedField{field},
			Footer:      &footer,
			Image:       image,
			Color:       embedColor,
		}

		_, err = cmd.SendDirectMessageEmbedSafe(e.UserID, content, &embed)
		if err != nil {
			_, _ = cmd.SendCustomEmbed(e.ChannelID, cmd.MakeEmbed("Failed to send bookmark!\nServer -> Privacy Settings -> ✅ Allow direct messages from server members.", fmt.Sprintf("```\n%s\n```", err), bot.ErrorColor))
		}
	}
}

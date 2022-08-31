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
	"sort"
	"strconv"
	"time"
)

var (
	escapedStar = "%E2%AD%90"
	stars3Emoji = "⭐"
	stars5Emoji = "🌟"
	stars6Emoji = "💫"
	stars9Emoji = "✨"

	starboardColor discord.Color = 0xffac33
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	return &plugins.Plugin{
		Name:        "Starboard",
		Description: "Pin messages to a custom channel",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          StarboardConfigCommand,
			FnName:      "StarboardConfigCommand",
			Name:        "starboardconfig",
			Aliases:     []string{"starboardcfg", "scfg"},
			Description: "Configure Starboard",
			GuildOnly:   true,
		}, {
			Fn:          StarboardTopPostsCommand,
			FnName:      "StarboardTopPostsCommand",
			Name:        "starboardtopposts",
			Aliases:     []string{"sbtop"},
			Description: "Get the most starred posts in this guild!",
			GuildOnly:   true,
		}},
		Responses: []bot.ResponseInfo{},
		Handlers: []bot.HandlerInfo{{
			Fn:     StarboardReactionHandler,
			FnName: "StarboardReactionHandler",
			FnType: reflect.TypeOf(func(*gateway.MessageReactionAddEvent) {}),
		}},
	}
}

func StarboardTopPostsCommand(c bot.Command) error {
	nsfwArg, argErr := cmd.ParseStringArg(c.Args, 1, false)
	nsfw, argErr := cmd.ParseBoolArg(c.Args, 1)
	if argErr != nil && len(nsfwArg) > 0 {
		_, err := cmd.SendEmbed(c.E, c.Name,
			"Available arguments are:\n- `<show nsfw posts bool>`",
			bot.DefaultColor)
		return err
	}

	channel, err := bot.Client.Channel(c.E.ChannelID)
	if err != nil {
		return err

	}

	if nsfw && !channel.NSFW {
		_, err := cmd.SendEmbed(c.E, c.Name, "You can only use the `nsfw` arg in NSFW channels!", bot.ErrorColor)
		return err
	}

	posts := make([]bot.StarboardMessage, 0)
	bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		posts = append(posts, g.Starboard.Messages...)
		return g, "StarboardTopPostsCommand: get g.Starboard.Messages"
	})

	if len(posts) == 0 {
		_, err := cmd.SendEmbed(c.E, c.Name, "This server doesn't have any starboard posts. Try again when you have more!", bot.WarnColor)
		return err
	}

	// sort by number of stars
	sort.Slice(posts, func(i, j int) bool {
		return len(posts[i].Stars) > len(posts[j].Stars)
	})

	embeds := make([]discord.Embed, 0)
	limit := 0

	for _, p := range posts {
		limit++
		if limit > 5 {
			break
		}

		var embedAuthor discord.EmbedAuthor
		if member, err := bot.Client.Member(c.E.GuildID, discord.UserID(p.Author)); err == nil {
			embedAuthor = *cmd.CreateEmbedAuthor(*member)
		} else if user, err := bot.Client.User(discord.UserID(p.Author)); err == nil {
			embedAuthor = *cmd.CreateEmbedAuthorUser(*user)
		}

		field := discord.EmbedField{Name: "View Post", Value: cmd.CreateMessageLinkInt64(int64(c.E.GuildID), p.ID, p.CID, true, false)}
		footer := discord.EmbedFooter{Text: fmt.Sprintf("%v", p.Author)}
		embed := discord.Embed{
			Description: getEmojiChannelMention(len(p.Stars), p.CID),
			Author:      &embedAuthor,
			Fields:      []discord.EmbedField{field},
			Footer:      &footer,
			Color:       starboardColor,
		}

		embeds = append(embeds, embed)
	}

	_, err = bot.Client.SendEmbeds(c.E.ChannelID, embeds...)
	return err
}

func StarboardConfigCommand(c bot.Command) error {
	err := cmd.HasPermission("channels", c)
	if err == nil {
		if arg, _ := cmd.ParseStringArg(c.Args, 1, true); err != nil {
			return err
		} else {
			arg2, errParse := cmd.ParseChannelArg(c.Args, 2)
			var err error = nil

			bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
				switch arg {
				case "regular":
					if errParse != nil {
						g.Starboard.Channel = 0
						_, err = cmd.SendEmbed(c.E, "Starboard Channels", "⛔ Disabled regular starboard", bot.ErrorColor)
						return g, "StarboardConfigCommand: disable regular starboard"
					} else {
						g.Starboard.Channel = arg2
						_, err = cmd.SendEmbed(c.E, "Starboard Channels", "✅ Enabled regular starboard", bot.SuccessColor)
						return g, "StarboardConfigCommand: enable regular starboard"
					}
				case "nsfw":
					if errParse != nil {
						g.Starboard.NsfwChannel = 0
						_, err = cmd.SendEmbed(c.E, "Starboard Channels", "⛔ Disabled NSFW starboard", bot.ErrorColor)
						return g, "StarboardConfigCommand: disable nsfw starboard"
					} else {
						g.Starboard.NsfwChannel = arg2
						_, err = cmd.SendEmbed(c.E, "Starboard Channels", "✅ Enabled NSFW starboard", bot.SuccessColor)
						return g, "StarboardConfigCommand: enable nsfw starboard"
					}
				case "threshold":
					if arg3, errParse := cmd.ParseInt64Arg(c.Args, 2); errParse != nil {
						_, err = cmd.SendEmbed(c.E, "Starboard Threshold", fmt.Sprintf("Current star threshold is: %v", g.Starboard.Threshold), bot.DefaultColor)
					} else {
						if arg3 <= 0 {
							arg3 = 1
						}

						g.Starboard.Threshold = arg3
						_, err = cmd.SendEmbed(c.E, "Starboard Threshold", fmt.Sprintf("✅ Set threshold to: %v", arg3), bot.SuccessColor)
					}

					return g, "StarboardConfigCommand: set threshold"
				case "list":
					regularC := "✅ Regular Starboard (<#" + strconv.FormatInt(g.Starboard.Channel, 10) + ">)"
					nsfwC := "✅ NSFW Starboard (<#" + strconv.FormatInt(g.Starboard.NsfwChannel, 10) + ">)"
					if g.Starboard.Channel == 0 {
						regularC = "⛔ Regular Starboard"
					}
					if g.Starboard.NsfwChannel == 0 {
						nsfwC = "⛔ NSFW Starboard"
					}

					embed := discord.Embed{
						Title:       "Starboard Channels",
						Description: regularC + "\n" + nsfwC,
						Color:       bot.DefaultColor,
					}
					_, err = cmd.SendCustomEmbed(c.E.ChannelID, embed)
					return g, "StarboardConfigCommand: list starboard channels"
				default:
					_, err = cmd.SendEmbed(c.E,
						"Configure Starboard",
						"Available arguments are:\n- `list`\n- `threshold <threshold>`\n- `nsfw|regular [channel]`",
						bot.DefaultColor)
					return g, "StarboardConfigCommand: show help"
				}
			})
			return err
		}
	} else {
		return err
	}
}

func StarboardReactionHandler(i interface{}) {
	defer util.LogPanic()

	e := i.(*gateway.MessageReactionAddEvent)
	start := time.Now().UnixMilli()

	bot.GuildContext(e.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
		if g.Starboard.Threshold == 0 {
			g.Starboard.Threshold = 3
		}

		// Not starred by a guild member
		if e.Member == nil {
			log.Printf("Not a guild member\n")
			return g, "StarboardReactionHandler: check guild member"
		}

		// Not a star
		if e.Emoji.APIString().PathString() != escapedStar {
			return g, "StarboardReactionHandler: check reaction emoji"
		}

		msg, err := bot.Client.Message(e.ChannelID, e.MessageID)
		if err != nil {
			return g, "StarboardReactionHandler: get reaction message"
		}
		channel, err := bot.Client.Channel(e.ChannelID)
		if err != nil {
			return g, "StarboardReactionHandler: get reaction channel"
		}

		var sMsg *bot.StarboardMessage = nil
		newPost := true
		cID := int64(channel.ID)

		log.Printf("Checking channel for starboard message %s\n", cmd.CreateMessageLink(int64(e.GuildID), msg, false, false))

		// If user reacts to a post in a starboard channel
		if cID == g.Starboard.Channel || cID == g.Starboard.NsfwChannel {
			for _, m := range g.Starboard.Messages {
				if m.PostID == int64(msg.ID) {
					sMsg = &m
					newPost = false
					break
				}
			}
		} else { // else if a user reacts to a post in a regular channel
			for _, m := range g.Starboard.Messages {
				if m.ID == int64(msg.ID) {
					sMsg = &m
					newPost = false
					break
				}
			}

			// If starred before channel ID was added, and the reaction is from the origin channel, update the stored one
			if !newPost && sMsg.CID == 0 {
				sMsg.CID = int64(msg.ChannelID)
			}
		}

		if newPost {
			sMsg = &bot.StarboardMessage{
				Author: int64(msg.Author.ID),
				CID:    int64(msg.ChannelID),
				ID:     int64(msg.ID),
				PostID: 0,
				IsNsfw: channel.NSFW,
				Stars:  make([]int64, 0),
			}
		}

		// Channel to send starboard message to
		cID = g.Starboard.Channel
		if sMsg.IsNsfw == true {
			cID = g.Starboard.NsfwChannel
		}

		// Channel hasn't been set
		if cID == 0 {
			log.Printf("Channel ID is 0\n")
			return g, "StarboardReactionHandler: check cID"
		}

		// Get post channel and ensure it exists
		postChannel, err := bot.Client.Channel(discord.ChannelID(cID))
		if err != nil {
			log.Printf("Couldn't get post channel\n")
			return g, "StarboardReactionHandler: get post channel"
		}

		// When adding a new star, ensure star user is not the same as author
		// And also check if they've already been added
		sUserID := int64(e.Member.User.ID)
		if sMsg.Author != sUserID && !util.SliceContains(sMsg.Stars, sUserID) {
			sMsg.Stars = append(sMsg.Stars, sUserID)
		}
		log.Printf("sUserID: %v\nsMsg:%v\n", sUserID, sMsg)

		// Update our reactions in case any are missing from the API
		for _, reaction := range msg.Reactions {
			if reaction.Emoji.APIString().PathString() == escapedStar {
				userReactions, err := bot.Client.Reactions(msg.ChannelID, msg.ID, reaction.Emoji.APIString(), 0)
				if err != nil {
					log.Printf("Failed to get userReactions: %s\n", err)
					return g, "StarboardReactionHandler: update sMsg.Stars"
				}

				for _, userReaction := range userReactions {
					sUserID = int64(userReaction.ID)

					if sMsg.Author != sUserID && !util.SliceContains(sMsg.Stars, sUserID) {
						sMsg.Stars = append(sMsg.Stars, sUserID)
					}
				}
				break
			}
		}

		stars := len(sMsg.Stars)

		// Not enough stars in sMsg to make post
		if int64(stars) < g.Starboard.Threshold {
			log.Printf("Not enough stars: %v\n", sMsg.Stars)
			return g, "StarboardReactionHandler: check notEnoughStars"
		}

		content := getEmojiChannelMention(stars, sMsg.CID)

		// Attempt to get existing message, and make a new one if it isn't there
		pMsg, err := bot.Client.Message(postChannel.ID, discord.MessageID(sMsg.PostID))
		if err != nil {
			log.Printf("Couldn't get pMsg %v\n", err)

			//
			// Construct new starboard post if it couldn't retrieve an existing one

			member, err := bot.Client.Member(e.GuildID, discord.UserID(sMsg.Author))
			if err != nil {
				log.Printf("Couldn't get member %v\n", err)
				return g, "StarboardReactionHandler: get sMsg.Author"
			}

			description, image := cmd.GetEmbedAttachmentAndContent(*msg)
			field := discord.EmbedField{Name: "Source", Value: cmd.CreateMessageLink(int64(e.GuildID), msg, true, false)}
			footer := discord.EmbedFooter{Text: fmt.Sprintf("%v", sMsg.Author)}
			embed := discord.Embed{
				Description: description,
				Author:      cmd.CreateEmbedAuthor(*member),
				Fields:      []discord.EmbedField{field},
				Footer:      &footer,
				Timestamp:   msg.Timestamp,
				Color:       starboardColor,
				Image:       image,
			}

			log.Printf("Embed image: %v\n", embed.Image)

			msg, err = bot.Client.SendMessage(postChannel.ID, content, embed)
			if err != nil {
				log.Printf("Error sending starboard post: %v\n", err)
			} else {
				sMsg.PostID = int64(msg.ID)
			}
		} else {
			// Edit the post if it exists
			_, err = bot.Client.EditMessage(postChannel.ID, discord.MessageID(sMsg.PostID), content, pMsg.Embeds...)
			if err != nil {
				log.Printf("Error updating starboard post: %v\n", err)
			}
		}

		// Now that we have updated the stars and starboard post ID, save it in the config
		if newPost {
			g.Starboard.Messages = append(g.Starboard.Messages, *sMsg)
		} else {
			for i, m := range g.Starboard.Messages {
				if m.ID == sMsg.ID {
					g.Starboard.Messages[i] = *sMsg
				}
			}
		}

		return g, "StarboardReactionHandler: update post"
	})

	log.Printf("Execute: %vms (StarboardReactionHandler)\n", time.Now().UnixMilli()-start)
}

func getEmoji(stars int) (emoji string) {
	switch stars {
	case 0, 1, 2, 3, 4:
		emoji = stars3Emoji
	case 5:
		emoji = stars5Emoji
	case 6, 7, 8:
		emoji = stars6Emoji
	default:
		emoji = stars9Emoji
	}

	return emoji
}

func getEmojiChannelMention(stars int, channel int64) string {
	return fmt.Sprintf("%s **%v** <#%v>", getEmoji(stars), stars, channel)
}

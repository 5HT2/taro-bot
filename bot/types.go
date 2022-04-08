package bot

//
// For types that should be shared
//

import (
	"fmt"
	"github.com/diamondburned/arikawa/v3/gateway"
	"strings"
)

//
// CommandInfo is the info a command provides to register itself.
// Fn is the function that is executed to complete the Command.
// The Name and Aliases are used to call the command via Discord.
type CommandInfo struct {
	Fn          func(Command) error
	FnName      string
	Name        string
	Description string
	Aliases     []string
	GuildOnly   bool
}

// Command is passed to CommandInfo.Fn's arguments when a Command is executed.
type Command struct {
	E      *gateway.MessageCreateEvent
	FnName string
	Name   string
	Args   []string
}

func (i CommandInfo) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %v]", i.FnName, i.Name, i.Description, i.Aliases, i.GuildOnly)
}

func (i CommandInfo) MarkdownString() string {
	aliases := ""
	if len(i.Aliases) > 0 {
		aliases = "(" + strings.Join(i.Aliases, ", ") + ")"
	}
	description := i.Description
	if len(description) == 0 {
		description = "No Description"
	}

	return fmt.Sprintf("**%s** %s\n%s", i.Name, aliases, description)
}

//
// ResponseInfo is the info a response provides to register itself.
// Fn is the function that is executed to complete the Response.
// The Regexes are used to call the response via Discord.
type ResponseInfo struct {
	Fn           func(Response) string
	Embed        bool
	Title        string
	Regexes      []string
	MatchMin     int
	LockChannels []int64
	LockUsers    []int64
}

func (i ResponseInfo) String() string {
	return fmt.Sprintf("[%p, %v, %s]", i.Fn, i.MatchMin, i.Regexes)
}

// Response is passed to Response.Fn's arguments when a Response is executed.
type Response struct {
	E *gateway.MessageCreateEvent
}

//
// PermissionGroups is collection of "permissions". Each permission is a list of user IDs that have said permission.
// Switching this to a list of {Name, Users} would maybe be better code-wise.
type PermissionGroups struct {
	ManageChannels    []int64 `json:"manage_channels,omitempty"`
	ManagePermissions []int64 `json:"manage_permissions,omitempty"`
}

//
// ActiveTopicVote is used by topics.go
type ActiveTopicVote struct {
	Message int64  `json:"message"`
	Author  int64  `json:"author"`
	Topic   string `json:"topic"`
}

//
// StarboardConfig is used by commands.go and starboard.go
type StarboardConfig struct {
	Channel     int64              `json:"channel,omitempty"`      // channel post ID
	NsfwChannel int64              `json:"nsfw_channel,omitempty"` // nsfw post channel ID
	Messages    []StarboardMessage `json:"messages,omitempty"`
	Threshold   int64              `json:"threshold,omitempty"`
}

// StarboardMessage is used by starboard.go
type StarboardMessage struct {
	Author int64   `json:"author"`     // the original author ID
	CID    int64   `json:"channel_id"` // the original channel ID
	ID     int64   `json:"id"`         // the original message ID
	PostID int64   `json:"message"`    // the starboard post message ID
	IsNsfw bool    `json:"nsfw"`       // if the original message was made in an NSFW channel
	Stars  []int64 `json:"stars"`      // list of added user IDs
}

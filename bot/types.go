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
// CommandInfo is the info a command provides to register itself. FnName is the function called by reflection.
// The Name and Aliases are used to call the command via Discord.
type CommandInfo struct {
	FnName      string
	Name        string
	Description string
	Aliases     []string
	GuildOnly   bool
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
	Fn           func(ResponseReflection) string `json:"_,omitempty"` // the found function to call.
	Embed        bool                            `json:"embed"`
	Title        string                          `json:"title"`
	Regexes      []string                        `json:"regexes"`
	MatchMin     int                             `json:"match_min,omitempty"`
	LockChannels []int64                         `json:"lock_channels,omitempty"`
	LockUsers    []int64                         `json:"lock_users,omitempty"`
}

func (i ResponseInfo) String() string {
	return fmt.Sprintf("[%p, %v, %s]", i.Fn, i.MatchMin, i.Regexes)
}

type ResponseReflection struct {
	E *gateway.MessageCreateEvent
}

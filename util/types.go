package util

//
// For types that should be shared
//

import (
	"fmt"
	"strings"
)

//
// CommandInfo is the info a command provides to "register" itself. FnName is the function called by reflection.
// The Name and Aliases are used to call the command via Discord.
type CommandInfo struct {
	FnName      string
	Name        string
	Description string
	Aliases     []string
	GuildOnly   bool
}

func (ci CommandInfo) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %v]", ci.FnName, ci.Name, ci.Description, ci.Aliases, ci.GuildOnly)
}

func (ci CommandInfo) MarkdownString() string {
	aliases := ""
	if len(ci.Aliases) > 0 {
		aliases = "(" + strings.Join(ci.Aliases, ", ") + ")"
	}
	description := ci.Description
	if len(description) == 0 {
		description = "No Description"
	}

	return fmt.Sprintf("**%s** %s\n%s", ci.Name, aliases, description)
}

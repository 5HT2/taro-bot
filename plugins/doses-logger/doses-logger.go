package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/5HT2C/http-bash-requests/httpBashRequests"
	"net/http"
	"reflect"
	"strings"
)

var (
	p *plugins.Plugin
)

type config struct {
	FohToken string `json:"foh_token"`
}

func InitPlugin(i *plugins.PluginInit) *plugins.Plugin {
	p = &plugins.Plugin{
		Name:        "Doses Logger",
		Description: "An interface with the `doses-logger` CLI tool.",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          DoseCommand,
			FnName:      "DoseCommand",
			Name:        "dose",
			Description: "Manage medication and substance doses",
		}},
		ConfigType: reflect.TypeOf(config{}),
	}
	p.ConfigDir = i.ConfigDir
	p.Config = p.LoadConfig()
	return p
}

func DoseCommand(c bot.Command) error {
	if p.Config == nil || p.Config.(config).FohToken == "" {
		return bot.GenericError(c.FnName, "running command", "`foh_token` not set")
	}

	// Make URL of public file
	file := fmt.Sprintf("http://localhost:6010/media/doses-%v.json", c.E.Author.ID)

	// Get args to pass to command
	argsTmp, _ := cmd.ParseStringSliceArg(c.Args, 1, -1)
	args := make([]string, 0)
	frog := false

	// Strip non-allowed args being passed
	for _, arg := range argsTmp {
		// Strip leading dashes until we only have one
		for {
			// arg needs to be longer than 2, if it's shorter it can't have --[some flag]
			// arg also needs to have a dash in the first and second position, otherwise we don't need to strip a dash
			if len(arg) <= 2 || arg[0] != '-' || arg[1] != '-' {
				break
			}

			arg = arg[1:]
		}

		switch strings.ToLower(arg) {
		case "-url", "-token":
			continue
		case "-frog":
			frog = true
			file = "http://localhost:6010/media/doses.json"
			continue
		default:
			args = append(args, arg)
		}
	}

	// final arg parsing
	pArgs := strings.Join(args, " ")
	sep := ""
	if len(pArgs) > 0 {
		sep = " "
	}

	if frog && (strings.Contains(pArgs, "-add") || strings.Contains(pArgs, "-rm")) {
		return bot.GenericError(c.FnName, "parsing args", "`-frog` cannot be used with `-add` or `-rm`!")
	}

	parsedArgs := fmt.Sprintf(`%s%s-token=%s -url=%s`, pArgs, sep, p.Config.(config).FohToken, file)
	// end arg parsing

	// get dose db for user
	res, _ := http.Get(file)
	if res == nil {
		_, err := cmd.SendEmbed(c.E, c.Name, "`res` was nil, is fs-over-http running?", bot.ErrorColor)
		return err
	}

	// if not found, do we need to make a json file for the user?
	if res.StatusCode == 404 {
		file = fmt.Sprintf("http://localhost:6010/public/media/doses-%v.json", c.E.Author.ID)

		// TODO: Use http stdlib
		if res, err := httpBashRequests.Run(fmt.Sprintf("curl -X POST -H \"Auth: %s\" %s -F \"content=[]\"", p.Config.(config).FohToken, file)); err != nil {
			return err
		} else if _, err := cmd.SendEmbed(c.E, "", fmt.Sprintf("```\n%s\n```", util.TailLinesLimit(string(res), 2040)), bot.DefaultColor); err != nil {
			return err
		}

		cmd.CommandHandlerWithCommand(c.E, c.Name, c.Args)
		return nil
	} else if res.StatusCode != 200 { // another http error? (shouldn't happen ever)
		_, err := cmd.SendEmbed(c.E, c.Name, fmt.Sprintf("Status for %s was %v, do you need to make a new file?", file, res.StatusCode), bot.ErrorColor)
		return err
	}

	// now we execute the doses-logger
	if res, err := httpBashRequests.RunBinary(parsedArgs, "doses-logger/doses-logger", "", true); err != nil {
		return err
	} else {
		_, err := cmd.SendEmbed(c.E, "", fmt.Sprintf("```\n%s\n```", util.TailLinesLimit(string(res), 2040)), bot.DefaultColor)
		return err
	}
}

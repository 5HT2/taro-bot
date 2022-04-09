package plugins

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/feature"
)

// RegisterAll will register all bot features, and then load plugins
func RegisterAll(dir string) {
	bot.Mutex.Lock()
	defer bot.Mutex.Unlock()

	// This is done to clear the existing plugins that have already been registered, if this is called after the bot
	// has already been initialized. This allows reloading plugins at runtime.
	bot.Commands = make([]bot.CommandInfo, 0)
	bot.Responses = make([]bot.ResponseInfo, 0)

	// This registers the base features
	cmd.RegisterCommands()
	feature.RegisterResponses()

	// This registers the plugins we have downloaded
	// This does not build new plugins for us, which instead has to be done separately
	Load(dir)
}

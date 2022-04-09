package plugins

import (
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/feature"
	"log"
)

// ClearJobs will go through bot.Jobs and handle the de-registration of them
func ClearJobs() {
	for _, job := range bot.Jobs {
		_ = job.Scheduler.RemoveByTag(job.Tag)
	}

	bot.Jobs = make([]bot.JobInfo, 0)
}

// RegisterJobs will go through bot.Jobs and handle the re-registration of them
func RegisterJobs() {
	for _, job := range bot.Jobs {
		if rJob, err := job.Scheduler.Tag(job.Tag).Do(job.Fn); err != nil {
			log.Printf("failed to register job (%s): %v\n", job.Tag, err)
		} else {
			log.Printf("registered job: %v\n", rJob)
		}
	}
}

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

	// We want to do this before registering plugins, same as regular features
	ClearJobs()

	// This registers the plugins we have downloaded
	// This does not build new plugins for us, which instead has to be done separately
	Load(dir)

	// This registers the new jobs that plugins have scheduled
	RegisterJobs()
}

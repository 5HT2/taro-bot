package plugins

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/util"
	"io/ioutil"
	"log"
	"path/filepath"
	"plugin"
	"strings"
)

type PluginInit struct {
}

type Plugin struct {
	Name        string             // Name of the plugin to display to users
	Description string             // Description of what the plugin does
	Version     string             // Version in semver, e.g.., 1.1.0
	Commands    []bot.CommandInfo  // Commands to register, could be none
	Responses   []bot.ResponseInfo // Responses to register, could be none
	Jobs        []bot.JobInfo      // Jobs to register, could be none
}

func (p Plugin) String() string {
	return fmt.Sprintf("[%s, %s, %v, %s, %s, %s]", p.Name, p.Description, p.Version, p.Commands, p.Responses, p.Jobs)
}

// Register will register a plugin's commands, responses and jobs to the bot
func (p *Plugin) Register() {
	bot.Commands = append(bot.Commands, p.Commands...)
	bot.Responses = append(bot.Responses, p.Responses...)
	bot.Jobs = append(bot.Jobs, p.Jobs...) // these need to have RegisterJobs called in order to function
}

// Load will load all the plugins from dir specified in pluginList
func Load(dir, pluginList string) {
	d, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Printf("plugin loading failed: couldn't load dir: %s\n", err)
		return
	}

	plugins := parsePluginsList(pluginList)
	pluginInit := &PluginInit{}

	log.Printf("plugin list: [%s]\n", strings.Join(plugins, ", "))

	for _, entry := range d {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".so") && util.SliceContains(plugins, entry.Name()) {
			pluginPath := filepath.Join(dir, entry.Name())
			log.Printf("plugin found: %s\n", entry.Name())

			p, err := plugin.Open(pluginPath)
			if err != nil {
				log.Printf("plugin load failed: couldn't open plugin: %s (%s)\n", entry.Name(), err)
				continue
			}

			fn, err := p.Lookup("InitPlugin")
			if err != nil {
				log.Printf("plugin load failed: couldn't lookup symbols: %s (%s)\n", entry.Name(), err)
				continue
			}

			initFn := fn.(func(manager *PluginInit) *Plugin)
			if p := initFn(pluginInit); p != nil {
				p.Register()
				log.Printf("plugin registered: %s\n", p)
			} else {
				log.Printf("plugin load failed: %s (nil)\n", entry.Name())
			}
		}
	}
}

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
		// Run job if it doesn't have checking enabled, or if the condition is true
		if !job.CheckCondition || job.Condition {
			if rJob, err := job.Scheduler.Tag(job.Tag).Do(job.Fn); err != nil {
				log.Printf("failed to register job (%s): %v\n", job.Tag, err)
			} else {
				log.Printf("registered job: %v\n", rJob)
			}
		}
	}
}

// RegisterAll will register all bot features, and then load plugins
func RegisterAll(dir, pluginList string) {
	bot.Mutex.Lock()
	defer bot.Mutex.Unlock()

	// This is done to clear the existing plugins that have already been registered, if this is called after the bot
	// has already been initialized. This allows reloading plugins at runtime.
	bot.Commands = make([]bot.CommandInfo, 0)
	bot.Responses = make([]bot.ResponseInfo, 0)

	// We want to do this before registering plugins
	ClearJobs()

	// This registers the plugins we have downloaded
	// This does not build new plugins for us, which instead has to be done separately
	Load(dir, pluginList)

	// This registers the new jobs that plugins have scheduled
	RegisterJobs()
}

func parsePluginsList(pluginList string) []string {
	plugins := make([]string, 0)
	for _, s := range strings.Split(pluginList, " ") {
		p := strings.ToLower(s) + ".so"
		if !util.SliceContains(plugins, p) {
			plugins = append(plugins, p)
		}
	}
	return plugins
}

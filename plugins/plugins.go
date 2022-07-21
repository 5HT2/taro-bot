package plugins

import (
	"encoding/json"
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/gateway"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var (
	pathValidation = regexp.MustCompile(`[^a-z\d.]`)
	fileMode       = os.FileMode(0755)
	plugins        = make([]*Plugin, 0)
)

type PluginInit struct {
}

type Plugin struct {
	Name        string             // Name of the plugin to display to users
	Description string             // Description of what the plugin does
	Version     string             // Version in semver, e.g.., 1.1.0
	Config      interface{}        // Config is the Plugin's config, can be nil
	ConfigType  reflect.Type       // ConfigType is the type to validate parse the config with
	Commands    []bot.CommandInfo  // Commands to register, could be none
	Responses   []bot.ResponseInfo // Responses to register, could be none
	Jobs        []bot.JobInfo      // Jobs to register, could be none
	Handlers    []bot.HandlerInfo  // Handlers to register, could be none
}

func (p Plugin) String() string {
	return fmt.Sprintf("[%s, %s, %v, %s, %s, %s, %s, %s]", p.Name, p.Description, p.Version, p.ConfigType, p.Commands, p.Responses, p.Jobs, p.Handlers)
}

// Register will register a plugin's commands, responses and jobs to the bot
func (p *Plugin) Register() {
	plugins = append(plugins, p)

	bot.Commands = append(bot.Commands, p.Commands...)
	bot.Responses = append(bot.Responses, p.Responses...)
	bot.Jobs = append(bot.Jobs, p.Jobs...)             // these need to have RegisterJobs called in order to function
	bot.Handlers = append(bot.Handlers, p.Handlers...) // these need to have RegisterHandlers called in order to function
}

func (p *Plugin) LoadConfig() (i interface{}) {
	defer util.LogPanic() // This code is unsafe, we should log if it panics

	bytes, err := os.ReadFile(getConfigPath(p))
	if err != nil {
		log.Printf("plugin config reading failed (%s): %s\n", p.Name, err)
		return i
	}

	obj, err := util.NewInterface(p.ConfigType, bytes)
	if err != nil {
		log.Printf("plugin config unmarshalling failed (%s): %s\n", p.Name, err)
		return i
	}

	log.Printf("plugin config loaded for %s\n", p.Name)
	return obj
}

func (p *Plugin) SaveConfig() {
	if p.Config == nil || p.ConfigType == nil {
		log.Printf("skipping saving %s\n", p.Name)
		return
	}

	// This is faster than checking if it exists
	_ = os.Mkdir("config/"+getConfigDir(p), fileMode)

	if bytes, err := json.MarshalIndent(p.Config, "", "    "); err != nil {
		log.Printf("plugin config marshalling failed (%s): %s\n", p.Name, err)
	} else {
		if err = os.WriteFile(getConfigPath(p), bytes, fileMode); err != nil {
			log.Printf("plugin config writing failed (%s): %s\n", p.Name, err)
		} else {
			log.Printf("saved config for %s\n", p.Name)
		}
	}
}

// SaveConfig will save all plugin configs
func SaveConfig() {
	for _, p := range plugins {
		p.SaveConfig()
	}
}

// SetupConfigSaving will run each plugin's SaveConfig every 5 minutes with a ticker
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
		RegisterJob(job)
	}
}

// RegisterJob registers a job for use with gocron. Ensure you add the job to bot.Jobs for de-registration with ClearJobs.
func RegisterJob(job bot.JobInfo) {
	// Run job if it doesn't have checking enabled, or if the condition is true
	if !job.CheckCondition || job.Condition {
		if rJob, err := job.Scheduler.Tag(job.Tag).Do(job.Fn); err != nil {
			log.Printf("failed to register job (%s): %v\n", job.Tag, err)
		} else {
			log.Printf("registered job: %v\n", rJob)
		}
	}
}

// ClearHandlers will go through bot.Handlers and handle the de-registration of them
func ClearHandlers() {
	for _, handler := range bot.Handlers {
		if handler.FnRm != nil {
			handler.FnRm()
		}
	}

	bot.Handlers = make([]bot.HandlerInfo, 0)
}

// RegisterHandlers will go through bot.Handlers and handle the re-registration of them
func RegisterHandlers() {
	for n, i := range bot.Handlers {
		// This is necessary because the loop mutates bot.Handlers as an invisible side effect.
		// Removing this will cause ghosts to enter your computer and call bot.Client.AddHandler even when fn == nil
		handler := bot.HandlerInfo{Fn: i.Fn, FnName: i.FnName, FnType: i.FnType}

		var fn any
		// Implement necessary handler types here when failing to register.
		// This has to be done manually, by hand, because Go is unable to pass a real type as a parameter.
		// Believe me, I tried doing so with reflection and got nothing to show for it after 5 hours.
		// If this behavior changes as Go finally figures out their situation with generics, that would be
		// nice to implement here, as a consideration for the future.
		switch handler.FnType {
		case reflect.TypeOf(func(e *gateway.MessageReactionAddEvent) {}):
			fn = func(e *gateway.MessageReactionAddEvent) {
				handler.Fn(e)
			}
		case reflect.TypeOf(func(e *gateway.MessageReactionRemoveEvent) {}):
			fn = func(e *gateway.MessageReactionRemoveEvent) {
				handler.Fn(e)
			}
		case reflect.TypeOf(func(e *gateway.GuildMemberAddEvent) {}):
			fn = func(e *gateway.GuildMemberAddEvent) {
				handler.Fn(e)
			}
		case reflect.TypeOf(func(e *gateway.GuildMemberRemoveEvent) {}):
			fn = func(e *gateway.GuildMemberRemoveEvent) {
				handler.Fn(e)
			}
		default:
			log.Printf("failed to register handler (%s): type %v not recognized\n", handler.FnName, handler.FnType)
			continue
		}

		if fn != nil {
			rm := bot.Client.AddHandler(fn)
			bot.Handlers[n].FnRm = rm
			log.Printf("registered handler: %v\n", bot.Handlers[n])
		}
	}
}

// RegisterAll will register all bot features, and then load plugins
func RegisterAll(dir, pluginList string) {
	bot.Mutex.Lock()
	defer bot.Mutex.Unlock()

	// This is done to clear the existing plugins that have already been registered, if this is called after the bot
	// has already been initialized. This allows reloading plugins at runtime.
	plugins = make([]*Plugin, 0)
	bot.Commands = make([]bot.CommandInfo, 0)
	bot.Responses = make([]bot.ResponseInfo, 0)

	// We want to do this before registering plugins
	ClearJobs()
	ClearHandlers()

	// This registers the plugins we have downloaded
	// This does not build new plugins for us, which instead has to be done separately
	Load(dir, pluginList)

	// This registers the new jobs that plugins have scheduled, and the handlers that they return
	RegisterJobs()
	RegisterHandlers()

	// This enables config saving for all loaded plugins
	SetupConfigSaving()
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

func getConfigPath(p *Plugin) string {
	return fmt.Sprintf("config/%s/%s.json", getConfigDir(p), p.Version)
}

func getConfigDir(p *Plugin) string {
	return pathValidation.ReplaceAllString(strings.ToLower(p.Name), "-")
}

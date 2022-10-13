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
	"strings"
	"time"
)

var (
	fileMode = os.FileMode(0755)
	plugins  = make([]*Plugin, 0)
)

type PluginInit struct {
	ConfigDir string
}

type Plugin struct {
	Name        string             // Name of the plugin to display to users
	Description string             // Description of what the plugin does
	Version     string             // Version in semver, e.g.., 1.1.0
	Config      interface{}        // Config is the Plugin's config, can be nil
	ConfigDir   string             // ConfigDir is the name of the config directory
	ConfigType  reflect.Type       // ConfigType is the type to validate parse the config with
	Commands    []bot.CommandInfo  // Commands to register, could be none
	Responses   []bot.ResponseInfo // Responses to register, could be none
	Handlers    []bot.HandlerInfo  // Handlers to register, could be none
	Jobs        []bot.JobInfo      // Jobs to register, could be none
	StartupFn   func()             // ShutdownFn is a function to be called when the bot starts up
	ShutdownFn  func()             // ShutdownFn is a function to be called when the bot shuts down
}

func (p Plugin) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s, %s]", p.Name, p.Description, p.Version, p.ConfigDir, p.ConfigType, p.Commands, p.Responses, p.Handlers, p.Jobs)
}

// Register will register a plugin's commands, responses and jobs to the bot
func (p *Plugin) Register() {
	plugins = append(plugins, p)

	bot.Commands = append(bot.Commands, p.Commands...)
	bot.Responses = append(bot.Responses, p.Responses...)
	bot.Handlers = append(bot.Handlers, p.Handlers...) // these need to have RegisterHandlers called in order to function
	bot.Jobs = append(bot.Jobs, p.Jobs...)             // these need to have RegisterJobs called in order to function
}

func (p *Plugin) LoadConfig() (i interface{}) {
	defer util.LogPanic() // This code is unsafe, we should log if it panics

	if p.ConfigDir == "" {
		log.Fatalln("plugin config load failed: p.ConfigDir is unset!")
	}

	bytes, err := os.ReadFile(getConfigPath(p))
	if err != nil {
		log.Printf("plugin config reading failed (%s): %s\n", p.Name, err)
		return i
	}

	obj, err := util.NewInterface(p.ConfigType, bytes) // unsafe
	if err != nil {
		log.Printf("plugin config unmarshalling failed (%s): %s\n", p.Name, err)
		return i
	}

	log.Printf("plugin config loaded for %s\n", p.Name)
	return obj
}

func (p *Plugin) SaveConfig() {
	if p.Config == nil || p.ConfigType == nil || p.ConfigDir == "" {
		log.Printf("skipping saving %s\n", p.Name)
		return
	}

	// This is faster than checking if it exists
	_ = os.Mkdir("config/"+p.ConfigDir, fileMode)

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

// Startup will run the startup function for all plugins
func Startup() {
	for _, p := range plugins {
		if p.StartupFn != nil {
			p.StartupFn()
		}
	}
}

// Shutdown will run the shutdown function for all plugins
func Shutdown() {
	for _, p := range plugins {
		if p.ShutdownFn != nil {
			p.ShutdownFn()
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

// Load will load all the plugins
func Load(dir string) {
	d, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Printf("plugin loading failed: couldn't load dir: %s\n", err)
		return
	}

	plugins := parsePluginsList()

	log.Printf("plugin list: [%s]\n", strings.Join(plugins, ", "))

	for _, entry := range d {
		func() {
			defer util.LogPanic() // plugins can panic when returning their PluginInit

			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".so") && util.SliceContains(plugins, entry.Name()) {
				pluginPath := filepath.Join(dir, entry.Name())
				log.Printf("plugin found: %s\n", entry.Name())

				p, err := plugin.Open(pluginPath)
				if err != nil {
					log.Printf("plugin load failed: couldn't open plugin: %s (%s)\n", entry.Name(), err)
					return
				}

				fn, err := p.Lookup("InitPlugin")
				if err != nil {
					log.Printf("plugin load failed: couldn't lookup symbols: %s (%s)\n", entry.Name(), err)
					return
				}

				if fn == nil {
					log.Printf("plugin load failed: fn nil\n")
					return
				}

				// Pass the ConfigDir to the PluginInit, so plugins can access it while loading their initial config.
				// This requires an extra step on the user's part when writing a plugin, but the plugin loading will fail
				// and let the user know if they forgot to do so. This isn't ideal, but it allows the renaming of plugin
				// names, without breaking the config or relying on parsing to be consistent.
				pluginInit := &PluginInit{ConfigDir: strings.TrimSuffix(entry.Name(), ".so")}
				// Create the init function to execute, to attempt plugin registration.
				initFn := fn.(func(manager *PluginInit) *Plugin)

				if p := initFn(pluginInit); p != nil {
					p.Register()
					log.Printf("plugin registered: %s\n", p)
				} else {
					log.Printf("plugin load failed: %s (nil)\n", entry.Name())
				}
			}
		}()
	}
}

// ClearJobs will clear all registered jobs
func ClearJobs() {
	bot.Scheduler.Clear()
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
	if rJob, err := job.Fn(); err != nil {
		log.Printf("failed to register job (%s): %v\n", job.Name, err)
	} else {
		log.Printf("registered job (%s): %v\n", job.Name, rJob)
	}
}

// RegisterJobConcurrent registers a job with RegisterJob concurrently, and optionally adds the job to bot.Jobs to be tracked.
func RegisterJobConcurrent(job bot.JobInfo, addGlobally bool) {
	bot.Mutex.Lock()
	defer bot.Mutex.Unlock()

	if addGlobally {
		bot.Jobs = append(bot.Jobs, job)
	}

	RegisterJob(job)
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
func RegisterAll(dir string) {
	bot.Mutex.Lock()
	defer bot.Mutex.Unlock()

	// This is done to clear the existing plugins that have already been registered, if this is called after the bot
	// has already been initialized. This allows reloading plugins at runtime.
	plugins = make([]*Plugin, 0)
	bot.Commands = make([]bot.CommandInfo, 0)
	bot.Responses = make([]bot.ResponseInfo, 0)

	// We want to do this before registering plugins
	ClearHandlers()
	ClearJobs()

	// This registers the plugins we have downloaded
	// This does not build new plugins for us, which instead has to be done separately
	Load(dir)

	// This registers the new jobs that plugins have scheduled, and the handlers that they return
	RegisterHandlers()
	RegisterJobs()

	// This enables config saving for all loaded plugins
	SetupConfigSaving()

	// This runs the startup sequence for all loaded plugins that have it
	Startup()
}

func parsePluginsList() []string {
	plugins := make([]string, 0)

	for _, p := range bot.P.LoadedPlugins {
		if p == "default" {
			continue
		}

		p += ".so"
		if !util.SliceContains(plugins, p) {
			plugins = append(plugins, p)
		}
	}

	if len(bot.P.LoadedPlugins) > 0 || util.SliceContains(bot.P.LoadedPlugins, "default") {
		for _, p := range bot.DefaultPlugins {
			p += ".so"
			if !util.SliceContains(plugins, p) {
				plugins = append(plugins, p)
			}
		}
	}
	return plugins
}

func getConfigPath(p *Plugin) string {
	return fmt.Sprintf("config/%s/%s.json", p.ConfigDir, p.Version)
}

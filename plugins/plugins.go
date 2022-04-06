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
	Version     int64              // Version in semver, i.e., 1.1.0 becomes 110
	Commands    []util.CommandInfo // Commands to register, could be none
}

func (p Plugin) String() string {
	return fmt.Sprintf("[%s, %s, %v, %s]", p.Name, p.Description, p.Version, p.Commands)
}

func (p *Plugin) Register() {
	// TODO: Maybe check if FnName collides? Shouldn't be a huge deal honestly
	log.Printf("registering plugin: %s\n", p)
	bot.Commands = append(bot.Commands, p.Commands...)
}

func Load() {
	d, err := ioutil.ReadDir("bin")
	if err != nil {
		log.Printf("couldn't load bin dir for plugins: %s\n", err)
		return
	}

	pluginInit := &PluginInit{}

	for _, entry := range d {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".so") {
			pluginPath := filepath.Join("bin", entry.Name())
			log.Printf("found plugin: %s\n", entry.Name())

			p, err := plugin.Open(pluginPath)
			if err != nil {
				log.Printf("couldn't open plugin: %s (%s)\n", entry.Name(), err)
				continue
			}

			fn, err := p.Lookup("InitPlugin")
			if err != nil {
				log.Printf("couldn't lookup plugin symbols: %s (%s)\n", entry.Name(), err)
				continue
			}

			initFn := fn.(func(manager *PluginInit) *Plugin)
			if p := initFn(pluginInit); p != nil {
				p.Register()
				log.Printf("loaded plugin: %s\n", entry.Name())
			} else {
				log.Printf("couldn't load plugin: %s (nil)\n", entry.Name())
			}
		}
	}
}

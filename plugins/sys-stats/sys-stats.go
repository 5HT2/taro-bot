package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	cu "github.com/5HT2/taro-bot/util/cpu"
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/uptime"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	runningFetches = make(map[string]chan bool)
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	return &plugins.Plugin{
		Name:        "System Stats",
		Description: "Provides system statistics",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          SysStatsCommand,
			FnName:      "SysStatsCommand",
			Name:        "systemstats",
			Aliases:     []string{"stats", "stat", "sysstat"},
			Description: "Provides system statistics",
		}},
	}
}

func spacedString(s string, offset int) string {
	return fmt.Sprintf("%s%s", s, strings.Repeat(" ", offset-len(s)))
}

func SysStatsCommand(c bot.Command) error {
	// If we have a running fetch command in this guild, cancel it
	if quit, ok := runningFetches[c.E.GuildID.String()]; ok {
		quit <- true
	}

	// Start a new quit channel
	runningFetches[c.E.GuildID.String()] = make(chan bool)

	// Determine displayed shell based on
	shell := "$"
	if err := cmd.HasPermission(c, cmd.PermOperator); err == nil {
		shell = "#"
	}
	if err := cmd.HasPermission(c, cmd.PermModerate); err == nil {
		shell = "#"
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return bot.GenericError(c.FnName, "getting hostname", err.Error())
	}
	hostname = "taro@" + hostname

	// Get kernel release number
	kernelRelease, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		// Try for darwin
		if out, err := exec.Command("uname", "-r").CombinedOutput(); err != nil {
			return bot.GenericSyntaxError(c.FnName, "getting kernel release", err.Error())
		} else {
			kernelRelease = append(out, []byte("-macOS")...)
		}
	}

	// Get current uptime
	uptimeDuration, err := uptime.Get()
	if err != nil {
		return bot.GenericError(c.FnName, "getting uptime", err.Error())
	}

	var days int64 = 0
	hours := uptimeDuration.Hours()
	if hours >= 24.0 {
		days = int64(hours / 24)
	}
	hoursInt := int64(hours) - (days * 24)
	hoursS := "s"
	if hoursInt == 1 {
		hoursS = ""
	}

	// Get starting CPU usage
	cpuBefore, err := cpu.Get()
	if err != nil {
		return bot.GenericError(c.FnName, "getting cpu info", err.Error())
	}

	// Fetch data to display inside fetch
	s := []string{
		hostname,
		strings.ReplaceAll(fmt.Sprintf("%s", kernelRelease), "\n", ""),
		fmt.Sprintf("%v days, %v hour%s", days, hoursInt, hoursS),
		"[calculating...]", // We have to wait at least one second, so instead we update it later
		"[calculating...]", // Not necessary to wait for memory, technically, but it looks nicer this way
	}

	// Calculate longest number of trailing spaces required for width, based on data line length
	o := 0
	for _, i := range s {
		if len(i) > o {
			o = len(i)
		}
	}
	o += 1

	// Generate the fetch text art with given data
	generateFetch := func() string {
		info := fmt.Sprintf("%s:~%s %s\n", hostname, shell, c.Name)
		info += fmt.Sprintf(" ┌──────────────%s─┐\n", strings.Repeat("─", o))
		info += fmt.Sprintf(" │ Hostname  >  %s │\n", spacedString(s[0], o))
		info += fmt.Sprintf(" │ Kernel    >  %s │\n", spacedString(s[1], o))
		info += fmt.Sprintf(" │ Uptime    >  %s │\n", spacedString(s[2], o))
		info += fmt.Sprintf(" │ CPU Load  >  %s │\n", spacedString(s[3], o))
		info += fmt.Sprintf(" │ Memory    >  %s │\n", spacedString(s[4], o))
		info += fmt.Sprintf(" └──────────────%s─┘\n", strings.Repeat("─", o))
		return fmt.Sprintf("```yml\n%s```", info)
	}

	// Start a goroutine for 60 seconds which updates CPU and memory in the fetch image.
	// If another stats command is started in the same guild, this fetch is cancelled in favor of the new one.
	go func() {
		msg, _ := cmd.SendMessage(c.E, generateFetch())
		i := 0

		for {
			select { // only allow one running fetch per guild, by cancelling when receiving a quit signal from another message
			case <-runningFetches[c.E.GuildID.String()]:
				return
			default:
				i++
				time.Sleep(time.Duration(1500) * time.Millisecond)

				cpuAfter, err := cpu.Get()
				if err != nil {
					log.Printf("%s\n", bot.GenericError(c.FnName, "getting cpu info", err.Error()))
					break
				}

				mem, err := memory.Get()
				if err != nil {
					log.Printf("%s\n", bot.GenericError(c.FnName, "getting memory info", err.Error()))
					break
				}

				s[3] = fmt.Sprintf(
					"%.2f%% (%v cores)",
					float64(cpuAfter.User-cpuBefore.User)/float64(cpuAfter.Total-cpuBefore.Total)*100,
					cu.GetCoresStr(cpuAfter),
				)

				s[4] = fmt.Sprintf(
					"%.2f GB/%.2f GB",
					float64(mem.Used)/1024*0.000001, float64(mem.Total)/1024*0.000001,
				)

				msg, _ = bot.Client.EditMessage(c.E.ChannelID, msg.ID, generateFetch())

				// 40 iterations * 1500ms = 1 minute
				if i == 40 {
					return
				}
			}
		}
	}()

	return nil
}

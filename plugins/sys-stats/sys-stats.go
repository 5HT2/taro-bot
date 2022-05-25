package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2C/http-bash-requests/httpBashRequests"
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/uptime"
	"log"
	"os"
	"strings"
	"time"
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
			Aliases:     []string{"stats"},
			Description: "Provides system statistics",
		}},
	}
}

func spacedString(s string, offset int) string {
	return fmt.Sprintf("%s%s", s, strings.Repeat(" ", offset-len(s)))
}

func SysStatsCommand(c bot.Command) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	hostname = "taro@" + hostname

	kernel, err := httpBashRequests.Run("uname --kernel-release | tr -d '\\n'")
	if err != nil {
		kernel = []byte("unknown")
		log.Printf("couldn't get kernel: %s\n", err)
	}

	mem, err := memory.Get()
	if err != nil {
		return err
	}

	uptimeDuration, err := uptime.Get()
	if err != nil {
		return err
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

	cpuBefore, err := cpu.Get()
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(1) * time.Second)
	cpuAfter, err := cpu.Get()
	if err != nil {
		return err
	}

	s := []string{
		hostname,
		string(kernel),
		fmt.Sprintf("%v days, %v hour%s", days, hoursInt, hoursS),
		fmt.Sprintf("%.2f%%", float64(cpuAfter.User-cpuBefore.User)/float64(cpuAfter.Total-cpuBefore.Total)*100),
		fmt.Sprintf("%.1f GB/%.1f GB", float64(mem.Used)/1024*0.000001, float64(mem.Total)/1024*0.000001),
	}

	o := 0
	for _, i := range s {
		if len(i) > o {
			o = len(i)
		}
	}
	o += 1

	info := fmt.Sprintf(" ┌──────────────%s─┐\n", strings.Repeat("─", o))
	info += fmt.Sprintf(" │ Hostname  >  %s │\n", spacedString(s[0], o))
	info += fmt.Sprintf(" │ Kernel    >  %s │\n", spacedString(s[1], o))
	info += fmt.Sprintf(" │ Uptime    >  %s │\n", spacedString(s[2], o))
	info += fmt.Sprintf(" │ CPU Load  >  %s │\n", spacedString(s[3], o))
	info += fmt.Sprintf(" │ Memory    >  %s │\n", spacedString(s[4], o))
	info += fmt.Sprintf(" └──────────────%s─┘\n", strings.Repeat("─", o))

	_, err = cmd.SendMessage(c.E, fmt.Sprintf("```yml\n%s```", info))
	return err
}

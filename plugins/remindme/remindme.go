package main

import (
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/go-co-op/gocron"
	"log"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	p     *plugins.Plugin
	mutex sync.Mutex
)

type config struct {
	Reminders map[string]Reminder `json:"reminders"` // [msg id]Reminder
}

type Reminder struct {
	ID        int64             `json:"id"`        // ID of the original message used to create the reminder, used as a unique identifier
	Time      time.Time         `json:"time"`      // Time in epoch seconds, to send the reminder
	Channel   int64             `json:"channel"`   // Channel to send message inside
	Guild     int64             `json:"guild"`     // Guild the message originates from
	User      discord.User      `json:"user"`      // User object of the reminder author
	Timestamp discord.Timestamp `json:"timestamp"` // Timestamp of the original message
	DM        bool              `json:"dm"`        // DM is if the reminder is in the user's direct messages with the bot
	Contents  string            `json:"contents"`  // Contents of the message to send
}

func (r Reminder) String() string {
	return fmt.Sprintf("[%v, %v, %v, %v, %v, %v, %v, \"%s\"]", r.ID, r.Time.Unix(), r.Channel, r.Guild, r.User.ID, r.Timestamp, r.DM, r.Contents)
}

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	p = &plugins.Plugin{
		Name:        "Remind Me",
		Description: "Set a reminder for yourself at a later date!",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          RemindMeCommand,
			FnName:      "RemindMeCommand",
			Name:        "remindme",
			Aliases:     []string{"remind", "r"},
			Description: "This command is an example",
		}},
		ConfigType: reflect.TypeOf(config{}),
	}
	p.Config = p.LoadConfig()
	p.Jobs = generateJobs() // even if the plugin is reloaded, InitPlugin should re-add the reminders with this
	return p
}

func RemindMeCommand(c bot.Command) error {
	duration, err := cmd.ParseDurationArg(c.Args, 1)
	if err != nil {
		return err
	}

	if duration < 0 {
		return bot.GenericError("RemindMeCommand", "getting date", fmt.Sprintf("you cannot set reminders in the past (%s ago)", duration))
	}

	args, _ := cmd.ParseStringSliceArg(c.Args, 2, -1)
	content := strings.Join(args, " ")
	if len(args) == 0 {
		content = "No reminder message set!"
	}

	t := time.Now().Add(duration)
	reminder := Reminder{
		ID:        int64(c.E.ID),
		Time:      t,
		Channel:   int64(c.E.ChannelID),
		Guild:     int64(c.E.GuildID),
		User:      c.E.Author,
		Timestamp: c.E.Timestamp,
		DM:        !c.E.GuildID.IsValid(),
		Contents:  content,
	}

	createAndRegisterReminder(reminder)

	_, err1 := cmd.SendEmbed(
		c.E,
		p.Name,
		fmt.Sprintf("Successfully created reminder for <t:%v:R>, you will be reminded on <t:%v:F>!", t.Unix(), t.Unix()),
		bot.SuccessColor,
	)
	return err1
}

// createAndRegisterReminder will create a job from a Reminder and register it right away
func createAndRegisterReminder(r Reminder) {
	job := createJob(r)
	id := strconv.FormatInt(r.ID, 10)

	if p.Config == nil || p.Config.(config).Reminders == nil {
		reminders := make(map[string]Reminder, 0)
		reminders[id] = r
		cfg := config{Reminders: reminders}
		p.Config = cfg
	} else {
		p.Config.(config).Reminders[id] = r
	}

	plugins.RegisterJobConcurrent(job, true)
}

// generateJobs will generate the initial jobs needed from a plugin load
func generateJobs() []bot.JobInfo {
	if p.Config == nil || p.Config.(config).Reminders == nil {
		return []bot.JobInfo{}
	}

	jobs := make([]bot.JobInfo, 0)

	for _, reminder := range p.Config.(config).Reminders {
		jobs = append(jobs, createJob(reminder))
	}

	return jobs
}

func createJob(r Reminder) bot.JobInfo {
	fn := func() {
		mutex.Lock()
		defer mutex.Unlock()

		field := discord.EmbedField{Name: "Source", Value: cmd.CreateMessageLinkInt64(r.Guild, r.ID, r.Channel, true, r.DM)}
		footer := discord.EmbedFooter{Text: r.User.ID.String()}
		embed := &discord.Embed{
			Description: r.Contents,
			Author:      cmd.CreateEmbedAuthorUser(r.User),
			Fields:      []discord.EmbedField{field},
			Footer:      &footer,
			Timestamp:   r.Timestamp,
			Color:       bot.BlueColor,
		}

		var err error

		if r.DM {
			_, err = cmd.SendDirectMessageEmbedSafe(r.User.ID, fmt.Sprintf("📝 from <#%v>", r.Channel), embed)
		} else {
			_, err = cmd.SendMessageEmbedSafe(discord.ChannelID(r.Channel), fmt.Sprintf("📝 from <#%v> <@%v>", r.Channel, r.User.ID), embed)
		}

		if err != nil {
			log.Printf("failed to deliver reminder: %v\n%s\n", err, r)
		}

		// Remove after attempting to send reminder
		if p.Config != nil {
			delete(p.Config.(config).Reminders, strconv.FormatInt(r.ID, 10))
		}
	}

	job := bot.JobInfo{
		Fn: func() (*gocron.Job, error) {
			return bot.Scheduler.Every(1).LimitRunsTo(1).StartAt(r.Time).Do(fn)
		},
		Name: fmt.Sprintf("remindme-plugin-schedule-%v", r.ID),
	}

	return job
}

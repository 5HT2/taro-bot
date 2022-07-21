package bot

import (
	"context"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/go-co-op/gocron"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	Commands  = make([]CommandInfo, 0)
	Responses = make([]ResponseInfo, 0)
	Jobs      = make([]JobInfo, 0)
	Handlers  = make([]HandlerInfo, 0)
	Mutex     = sync.Mutex{}

	HttpClient     = http.Client{Timeout: 5 * time.Second}
	Client         state.State
	Ctx            = context.Background()
	User           *discord.User
	PermissionsHex = 278404582480 // this is currently only used in base.go, but it is in shared.go because it is bot-level and should be set by the person maintaining the bot code
	Scheduler      = gocron.NewScheduler(getTimeZone())

	SuccessColor discord.Color = 0x3cde5a
	ErrorColor   discord.Color = 0xde413c
	WarnColor    discord.Color = 0xde953c
	DefaultColor discord.Color = 0x493cde
	WhiteColor   discord.Color = 0xfefefe
)

func getTimeZone() *time.Location {
	tzEnv := os.Getenv("TZ")
	if len(tzEnv) == 0 {
		tzEnv = "Local"
	}

	l, err := time.LoadLocation(tzEnv)
	if err != nil {
		log.Printf("error loading timezone, defaulting to UTC: %v\n", err)
		return time.UTC
	}

	log.Printf("using location \"%s\" for timezone\n", l)
	return l
}

package bot

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/session"
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
	Client         session.Session
	User           *discord.User
	PermissionsHex = 278136147008
	Scheduler      = gocron.NewScheduler(getTimeZone())

	SuccessColor   discord.Color = 0x3cde5a
	ErrorColor     discord.Color = 0xde413c
	WarnColor      discord.Color = 0xde953c
	DefaultColor   discord.Color = 0x493cde
	StarboardColor discord.Color = 0xffac33
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

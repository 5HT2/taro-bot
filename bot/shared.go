package bot

import (
	"net/http"
	"sync"
	"time"
)

var (
	Commands  = make([]CommandInfo, 0)
	Responses = make([]ResponseInfo, 0)
	Mutex     = sync.Mutex{}

	HttpClient = http.Client{Timeout: 5 * time.Second}
)

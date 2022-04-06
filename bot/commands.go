package bot

import (
	"github.com/5HT2/taro-bot/util"
	"sync"
)

var (
	Commands      = make([]util.CommandInfo, 0)
	CommandsMutex = sync.Mutex{}
)

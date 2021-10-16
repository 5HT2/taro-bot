package main

import "github.com/diamondburned/arikawa/v3/gateway"

type Response struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ReflectFunc string   `json:"reflect_func,omitempty"`
	Regexes     []string `json:"regexes"`
	MatchMin    int      `json:"match_min,omitempty"`
}

type ResponseReflection struct {
	e *gateway.MessageCreateEvent
}

func (rr ResponseReflection) PrefixResponse() string {
	return GetGuildConfig(int64(rr.e.GuildID)).Prefix
}

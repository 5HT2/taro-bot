package main

import "strings"

var (
	commands = map[string]string{
		"ping":  "PingCommand",
		"fuck":  "FuckCommand",
		"kirby": "KirbyCommand",
	}
)

func (c Command) PingCommand() {
	msg, err := SendEmbed(c,
		"Ping!",
		"Unfinished", // TODO do
		defaultColor)
	if err != nil {
		_, _ = SendEmbed(c, "Pong!", msg.Timestamp.Format(timeFormat), defaultColor)
	}
}

func (c Command) FuckCommand() {
	_, _ = SendEmbed(c, "that's right", "fucker\n\nthis is all automatically reflected with generics", successColor)
}

func (c Command) KirbyCommand() {
	content := strings.Join(strings.Split(c.e.Content, " ")[1:], " ")
	_, _ = SendMessage(c, "<:kirbyfeet:893291555744542730>")
	_, _ = SendMessage(c, content)
}

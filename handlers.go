package main

import (
	"github.com/diamondburned/arikawa/v3/gateway"
	"reflect"
	"strings"
)

// CommandHandler will parse commands and run the appropriate command func
// TODO: add custom prefix support
func CommandHandler(e *gateway.MessageCreateEvent) {
	if strings.HasPrefix(e.Message.Content, ".") {

	}
	reflect.ValueOf(&e).MethodByName("GFG").Call([]reflect.Value{})
}

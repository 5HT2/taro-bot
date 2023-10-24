//go:build darwin && cgo
// +build darwin,cgo

package cpu

import (
	"github.com/mackerelio/go-osstat/cpu"
)

func GetCores(s *cpu.Stats) int {
	return -1
}

func GetCoresStr(s *cpu.Stats) string {
	return "?"
}

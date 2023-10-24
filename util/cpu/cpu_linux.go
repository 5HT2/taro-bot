//go:build linux
// +build linux

package cpu

import (
	"fmt"
	"github.com/mackerelio/go-osstat/cpu"
)

func GetCores(s *cpu.Stats) int {
	return s.CPUCount
}

func GetCoresStr(s *cpu.Stats) string {
	return fmt.Sprintf("%v", s.CPUCount)
}

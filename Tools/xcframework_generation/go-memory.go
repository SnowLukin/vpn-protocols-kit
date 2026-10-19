package main

import "C"
import (
	"math"
	"runtime/debug"
)

// GOGC below this makes the collector run almost constantly, and a negative
// value turns it off.
const minGCPercent = 10

// LibXraySetGoMemorySettings overrides the collector settings libXray forces
// on iOS at every xray start (GOGC 10 and a 30 MiB soft memory limit). The
// settings are process-wide, so calling this after the start wins. gcPercent
// below 10 is raised to 10, memoryLimitMiB <= 0 removes the limit.
//
//export LibXraySetGoMemorySettings
func LibXraySetGoMemorySettings(gcPercent, memoryLimitMiB C.int) {
	setGoMemorySettings(int(gcPercent), int(memoryLimitMiB))
}

func setGoMemorySettings(gcPercent, memoryLimitMiB int) {
	debug.SetGCPercent(max(gcPercent, minGCPercent))
	limit := int64(math.MaxInt64)
	if memoryLimitMiB > 0 {
		limit = int64(memoryLimitMiB) << 20
	}
	debug.SetMemoryLimit(limit)
}

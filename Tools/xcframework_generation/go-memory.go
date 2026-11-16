package main

import "C"
import (
	"math"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/xtls/libxray/memory"
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

// LibXraySetFreeOSMemoryInterval sets how often libXray returns memory to the
// OS on iOS (every second by default); seconds <= 0 stops that loop. Called
// before an xray start, it also sets the interval the loop starts with.
//
//export LibXraySetFreeOSMemoryInterval
func LibXraySetFreeOSMemoryInterval(seconds C.int) {
	memory.SetForceFreeInterval(time.Duration(seconds) * time.Second)
}

// LibXrayFreeOSMemory runs a full collection and returns as much memory to the
// OS as possible. It blocks for the whole collection.
//
//export LibXrayFreeOSMemory
func LibXrayFreeOSMemory() {
	debug.FreeOSMemory()
}

// LibXraySetGoMaxProcs limits how many threads run Go code at once; n <= 0
// returns to the runtime default.
//
//export LibXraySetGoMaxProcs
func LibXraySetGoMaxProcs(n C.int) {
	setGoMaxProcs(int(n))
}

func setGoMaxProcs(n int) {
	if n <= 0 {
		runtime.SetDefaultGOMAXPROCS()
		return
	}
	runtime.GOMAXPROCS(n)
}

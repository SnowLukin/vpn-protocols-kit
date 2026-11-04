package main

import (
	"math"
	"runtime"
	"runtime/debug"
	"testing"
)

func restoreGoMemoryDefaults(t *testing.T) {
	t.Cleanup(func() {
		debug.SetGCPercent(100)
		debug.SetMemoryLimit(math.MaxInt64)
	})
}

func currentGoMemorySettings() (gcPercent int, memoryLimit int64) {
	gcPercent = debug.SetGCPercent(-1)
	debug.SetGCPercent(gcPercent)
	return gcPercent, debug.SetMemoryLimit(-1)
}

func TestSetGoMemorySettingsAppliesValues(t *testing.T) {
	restoreGoMemoryDefaults(t)

	setGoMemorySettings(100, 64)

	if gc, limit := currentGoMemorySettings(); gc != 100 || limit != 64<<20 {
		t.Fatalf("got GOGC %d, limit %d", gc, limit)
	}
}

func TestSetGoMemorySettingsZeroLimitRemovesIt(t *testing.T) {
	restoreGoMemoryDefaults(t)
	debug.SetMemoryLimit(30 << 20)

	setGoMemorySettings(100, 0)

	if _, limit := currentGoMemorySettings(); limit != math.MaxInt64 {
		t.Fatalf("limit stayed %d", limit)
	}
}

func TestSetGoMemorySettingsKeepsCollectorOn(t *testing.T) {
	restoreGoMemoryDefaults(t)

	setGoMemorySettings(-1, 0)

	if gc, _ := currentGoMemorySettings(); gc != minGCPercent {
		t.Fatalf("got GOGC %d", gc)
	}
}

func TestSetGoMaxProcsLimitsAndRestoresDefault(t *testing.T) {
	defaultProcs := runtime.GOMAXPROCS(0)
	t.Cleanup(func() { runtime.GOMAXPROCS(defaultProcs) })

	setGoMaxProcs(1)
	if got := runtime.GOMAXPROCS(0); got != 1 {
		t.Fatalf("got GOMAXPROCS %d", got)
	}

	setGoMaxProcs(0)
	if got := runtime.GOMAXPROCS(0); got != defaultProcs {
		t.Fatalf("got GOMAXPROCS %d, want default %d", got, defaultProcs)
	}
}

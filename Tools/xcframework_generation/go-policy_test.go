package main

import (
	"encoding/json"
	"math"
	"runtime/debug"
	"testing"
)

func resetGCPolicy(t *testing.T) {
	t.Cleanup(func() {
		debug.SetGCPercent(100)
		debug.SetMemoryLimit(math.MaxInt64)
		setFreeOSMemoryInterval(0)
		stopManagedXray()
	})
}

func TestApplyGCPolicyNeverTurnsCollectorOff(t *testing.T) {
	resetGCPolicy(t)
	for _, percent := range []int{-1, 0, 5} {
		applyGCPolicy(gcPolicy{gcPercent: percent})
		if got := goMemStats()["gogc"]; got != minGCPercent {
			t.Fatalf("gcPercent %d applied as GOGC %v", percent, got)
		}
	}
}

func TestApplyGCPolicyMemoryLimit(t *testing.T) {
	resetGCPolicy(t)
	applyGCPolicy(gcPolicy{gcPercent: 100, memoryLimitMiB: 40})
	if got := debug.SetMemoryLimit(-1); got != 40<<20 {
		t.Fatalf("limit %d, want 40 MiB", got)
	}
	applyGCPolicy(gcPolicy{gcPercent: 100, memoryLimitMiB: 0})
	if got := debug.SetMemoryLimit(-1); got != math.MaxInt64 {
		t.Fatalf("limit %d, want none", got)
	}
}

func TestApplyGCPolicyStopsPeriodicFree(t *testing.T) {
	resetGCPolicy(t)
	applyGCPolicy(gcPolicy{gcPercent: 100, freeIntervalMs: 1000})
	if freeIntervalMs.Load() != 1000 {
		t.Fatal("periodic free not enabled")
	}
	applyGCPolicy(gcPolicy{gcPercent: 100, freeIntervalMs: -5})
	if freeIntervalMs.Load() != 0 {
		t.Fatal("periodic free still enabled")
	}
}

func TestManagedStartClosesPreviousInstance(t *testing.T) {
	resetGCPolicy(t)
	config, _ := json.Marshal(map[string]any{
		"outbounds": []map[string]any{{"protocol": "freedom"}},
	})
	policy := gcPolicy{gcPercent: 100}
	if err := runManagedXray(t.TempDir(), string(config), policy); err != nil {
		t.Fatal(err)
	}
	first := managedInstance
	if err := runManagedXray(t.TempDir(), string(config), policy); err != nil {
		t.Fatal(err)
	}
	if first.IsRunning() || !managedXrayRunning() {
		t.Fatal("restart must close the previous instance and keep the new one")
	}
	if _, stopped := stopManagedXray(); !stopped || managedXrayRunning() {
		t.Fatal("stop did not close the managed instance")
	}
	if _, stopped := stopManagedXray(); stopped {
		t.Fatal("second stop reported a managed instance")
	}
}

func TestManagedStartRejectsBrokenConfig(t *testing.T) {
	resetGCPolicy(t)
	if err := runManagedXray(t.TempDir(), "{", gcPolicy{gcPercent: 100}); err == nil {
		t.Fatal("broken config started")
	}
	if managedXrayRunning() {
		t.Fatal("failed start left an instance")
	}
}

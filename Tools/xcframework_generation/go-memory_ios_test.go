//go:build ios

package main

import (
	"math"
	"testing"

	lxray "github.com/xtls/libxray/xray"
)

// The override only works while libXray applies its policy synchronously
// inside the start call. A libXray update that moves it elsewhere fails here.
//
// With the ios tag a macOS test binary often dies at startup with "non-Go code
// disabled sigaltstack", even with no tests selected. Run with async
// preemption off:
//
//	GODEBUG=asyncpreemptoff=1 go test -tags ios -run SetGoMemorySettings .
func TestSetGoMemorySettingsOverridesLibXrayStart(t *testing.T) {
	restoreGoMemoryDefaults(t)
	if err := lxray.RunXrayFromJSON(t.TempDir(), `{"outbounds":[{"protocol":"freedom"}]}`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lxray.StopXray() })
	if gc, limit := currentGoMemorySettings(); gc != 10 || limit != 30<<20 {
		t.Fatalf("libXray start left GOGC %d, limit %d", gc, limit)
	}

	setGoMemorySettings(100, 0)

	if gc, limit := currentGoMemorySettings(); gc != 100 || limit != math.MaxInt64 {
		t.Fatalf("got GOGC %d, limit %d", gc, limit)
	}
}

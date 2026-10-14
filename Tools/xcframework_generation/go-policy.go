package main

import "C"
import (
	"encoding/json"
	"math"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xtls/libxray/nodep"
	lxray "github.com/xtls/libxray/xray"
	xraycore "github.com/xtls/xray-core/core"
)

// GOGC below this makes the collector run almost constantly, and a negative
// value turns it off: a bad remote value must not do either.
const minGCPercent = 10

var (
	managedMu       sync.Mutex
	managedInstance *xraycore.Instance
	freeIntervalMs  atomic.Int64
	freeLoopOnce    sync.Once
)

type gcPolicy struct {
	gcPercent      int
	memoryLimitMiB int
	freeIntervalMs int
	maxProcs       int
}

// LibXrayRunXrayFromJSONManaged starts xray without libXray's forced iOS
// policy (GOGC 10, 30 MiB limit and a FreeOSMemory loop added on every start).
// The policy is applied before the instance starts. memoryLimitMiB <= 0 means
// no limit, freeIntervalMs <= 0 no periodic FreeOSMemory, maxProcs <= 0 keeps
// GOMAXPROCS. A running managed instance is closed first.
//
//export LibXrayRunXrayFromJSONManaged
func LibXrayRunXrayFromJSONManaged(datDir, configJSON *C.char, gcPercent, memoryLimitMiB, freeIntervalMs, maxProcs C.int) *C.char {
	policy := gcPolicy{int(gcPercent), int(memoryLimitMiB), int(freeIntervalMs), int(maxProcs)}
	var response nodep.CallResponse[string]
	err := runManagedXray(C.GoString(datDir), C.GoString(configJSON), policy)
	return C.CString(response.EncodeToBase64("", err))
}

func runManagedXray(datDir, configJSON string, policy gcPolicy) error {
	managedMu.Lock()
	defer managedMu.Unlock()
	if managedInstance != nil {
		_ = managedInstance.Close()
		managedInstance = nil
	}
	applyGCPolicy(policy)
	lxray.InitEnv(datDir)
	config, err := xraycore.LoadConfig("json", strings.NewReader(configJSON))
	if err != nil {
		return err
	}
	instance, err := xraycore.New(config)
	if err != nil {
		return err
	}
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		return err
	}
	managedInstance = instance
	debug.FreeOSMemory()
	return nil
}

func applyGCPolicy(policy gcPolicy) {
	debug.SetGCPercent(max(policy.gcPercent, minGCPercent))
	limit := int64(math.MaxInt64)
	if policy.memoryLimitMiB > 0 {
		limit = int64(policy.memoryLimitMiB) << 20
	}
	debug.SetMemoryLimit(limit)
	if policy.maxProcs > 0 {
		runtime.GOMAXPROCS(policy.maxProcs)
	}
	setFreeOSMemoryInterval(policy.freeIntervalMs)
}

func setFreeOSMemoryInterval(intervalMs int) {
	freeIntervalMs.Store(int64(max(intervalMs, 0)))
	if intervalMs <= 0 {
		return
	}
	freeLoopOnce.Do(func() {
		go func() {
			for {
				ms := freeIntervalMs.Load()
				if ms <= 0 {
					time.Sleep(time.Second)
					continue
				}
				time.Sleep(time.Duration(ms) * time.Millisecond)
				if freeIntervalMs.Load() > 0 {
					debug.FreeOSMemory()
				}
			}
		}()
	})
}

func stopManagedXray() (string, bool) {
	managedMu.Lock()
	instance := managedInstance
	managedInstance = nil
	managedMu.Unlock()
	if instance == nil {
		return "", false
	}
	var response nodep.CallResponse[string]
	return response.EncodeToBase64("", instance.Close()), true
}

func managedXrayRunning() bool {
	managedMu.Lock()
	defer managedMu.Unlock()
	return managedInstance != nil && managedInstance.IsRunning()
}

// LibXrayFreeOSMemory forces a GC and returns free pages to the OS once.
//
//export LibXrayFreeOSMemory
func LibXrayFreeOSMemory() {
	debug.FreeOSMemory()
}

var goStatNames = []string{
	"/memory/classes/total:bytes",
	"/memory/classes/heap/released:bytes",
	"/memory/classes/heap/objects:bytes",
	"/memory/classes/heap/stacks:bytes",
	"/memory/classes/heap/free:bytes",
	"/sched/goroutines:goroutines",
	"/sched/threads/total:threads",
	"/gc/cycles/total:gc-cycles",
	"/cpu/classes/gc/total:cpu-seconds",
	"/cpu/classes/total:cpu-seconds",
	"/gc/limiter/last-enabled:gc-cycle",
	"/gc/gogc:percent",
}

// LibXrayGoMemStats returns Go memory and GC counters as JSON; the caller
// frees the string.
//
//export LibXrayGoMemStats
func LibXrayGoMemStats() *C.char {
	data, err := json.Marshal(goMemStats())
	if err != nil {
		return nil
	}
	return C.CString(string(data))
}

func goMemStats() map[string]float64 {
	samples := make([]metrics.Sample, len(goStatNames))
	for i, name := range goStatNames {
		samples[i].Name = name
	}
	metrics.Read(samples)
	value := func(i int) float64 {
		switch samples[i].Value.Kind() {
		case metrics.KindUint64:
			return float64(samples[i].Value.Uint64())
		case metrics.KindFloat64:
			return samples[i].Value.Float64()
		}
		return -1
	}
	const mib = 1 << 20
	return map[string]float64{
		"go_mapped_mb":       (value(0) - value(1)) / mib,
		"go_heap_objects_mb": value(2) / mib,
		"go_stacks_mb":       value(3) / mib,
		"go_heap_free_mb":    value(4) / mib,
		"goroutines":         value(5),
		"go_threads":         value(6),
		"gc_cycles":          value(7),
		"gc_cpu_s":           value(8),
		"go_cpu_s":           value(9),
		"gc_limiter_cycle":   value(10),
		"gomaxprocs":         float64(runtime.GOMAXPROCS(0)),
		"go_mem_limit_mb":    float64(debug.SetMemoryLimit(-1)) / mib,
		"gogc":               value(11),
	}
}

package loghistory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	applog "github.com/xtls/xray-core/app/log"
	commonlog "github.com/xtls/xray-core/common/log"
)

type blockedSink struct {
	mu      sync.Mutex
	entered chan struct{}
	release chan struct{}
	records [][]byte
	closed  bool
}

func (s *blockedSink) writeRecord(data []byte) error {
	select {
	case s.entered <- struct{}{}:
	default:
	}
	<-s.release
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, data)
	return nil
}
func (s *blockedSink) Close() error { s.mu.Lock(); defer s.mu.Unlock(); s.closed = true; return nil }
func (s *blockedSink) State() State { return State{Version: 1, Status: "active"} }

func TestHandlerBoundsQueueReportsDropsAndDrainsBeforeClose(t *testing.T) {
	cfg := testConfig(t)
	cfg.Policy.MaxPendingBytes = 512
	sink := &blockedSink{entered: make(chan struct{}, 1), release: make(chan struct{})}
	h := newHandler(cfg, sink)
	h.Handle(&commonlog.GeneralMessage{Severity: commonlog.Severity_Info, Content: "first"})
	<-sink.entered
	for i := 0; i < 30; i++ {
		h.Handle(&commonlog.GeneralMessage{Severity: commonlog.Severity_Warning, Content: strings.Repeat("x", 160)})
	}
	closed := make(chan struct{})
	go func() { _ = h.Close(); close(closed) }()
	select {
	case <-closed:
		t.Fatal("Close returned before accepted record was written")
	case <-time.After(10 * time.Millisecond):
	}
	close(sink.release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not drain")
	}
	if !sink.closed {
		t.Fatal("sink was not closed")
	}
	if len(sink.records) > 4 {
		t.Fatalf("queue was not bounded: %d", len(sink.records))
	}
	var first, last map[string]any
	_ = json.Unmarshal(sink.records[0], &first)
	_ = json.Unmarshal(sink.records[len(sink.records)-1], &last)
	if first["message"] != "first" || last["kind"] != "dropped" {
		t.Fatalf("lost event or drop marker: %s", sink.records)
	}
	count := len(sink.records)
	h.Handle(&commonlog.GeneralMessage{Content: "after close"})
	_ = h.Close()
	if len(sink.records) != count {
		t.Fatal("closed handler accepted event")
	}
}

func TestConfiguredFactoryPersistsAcrossXraySessions(t *testing.T) {
	cfg := testConfig(t)
	manager := &Manager{}
	if err := manager.Configure(&cfg); err != nil {
		t.Fatal(err)
	}
	h, err := manager.create(applog.LogType_File, applog.HandlerCreatorOptions{Path: filepath.Join(cfg.Directory, "xray-core.log")})
	if err != nil {
		t.Fatal(err)
	}
	h.Handle(&commonlog.GeneralMessage{Severity: commonlog.Severity_Error, Content: "throwable\n  at frame"})
	if err := h.(*Handler).Close(); err != nil {
		t.Fatal(err)
	}
	cfg.ConnectionID = "connection-b"
	if err := manager.Configure(&cfg); err != nil {
		t.Fatal(err)
	}
	h, err = manager.create(applog.LogType_File, applog.HandlerCreatorOptions{Path: filepath.Join(cfg.Directory, "xray-core.log")})
	if err != nil {
		t.Fatal(err)
	}
	h.Handle(&commonlog.GeneralMessage{Severity: commonlog.Severity_Info, Content: "reconnected"})
	_ = h.(*Handler).Close()
	records, _ := readRecords(t, cfg.Directory)
	if len(records) != 2 || records[0]["message"] != "throwable\n  at frame" || records[1]["connectionId"] != "connection-b" {
		t.Fatalf("lost history: %#v", records)
	}
	if state := manager.State(); state == nil || state.Status != "closed" {
		t.Fatalf("unexpected live state: %#v", state)
	}
}

func TestFactoryFailureIsVisibleWithoutAbortingVPN(t *testing.T) {
	cfg := testConfig(t)
	cfg.Directory = filepath.Join(cfg.Directory, "missing", "\x00")
	m := &Manager{}
	if err := m.Configure(&cfg); err != nil {
		t.Fatal(err)
	}
	h, err := m.create(applog.LogType_File, applog.HandlerCreatorOptions{Path: filepath.Join(cfg.Directory, "xray-core.log")})
	if err != nil || h == nil {
		t.Fatal("log IO failure aborted VPN startup")
	}
	h.Handle(&commonlog.GeneralMessage{Content: "safe to discard"})
	state := m.State()
	if state == nil || state.Status != "failed" || strings.Contains(state.ErrorCode, cfg.Directory) {
		t.Fatalf("missing safe failure: %#v", state)
	}
}

func TestDisableClosesActiveHistoryWithoutRestartingXray(t *testing.T) {
	cfg := testConfig(t)
	m := &Manager{}
	if err := m.Configure(&cfg); err != nil {
		t.Fatal(err)
	}
	h, err := m.create(applog.LogType_File, applog.HandlerCreatorOptions{Path: filepath.Join(cfg.Directory, "xray-core.log")})
	if err != nil {
		t.Fatal(err)
	}
	h.Handle(&commonlog.GeneralMessage{Severity: commonlog.Severity_Info, Content: "before revoke"})
	if err := m.Configure(nil); err != nil {
		t.Fatal(err)
	}
	h.Handle(&commonlog.GeneralMessage{Severity: commonlog.Severity_Info, Content: "after revoke"})
	if err := h.(*Handler).Close(); err != nil {
		t.Fatal(err)
	}
	records, _ := readRecords(t, cfg.Directory)
	if len(records) != 1 || records[0]["message"] != "before revoke" {
		t.Fatalf("history continued after revoke: %#v", records)
	}
	if state := m.State(); state == nil || state.Status != "closed" {
		t.Fatalf("missing closed live state: %#v", state)
	}
	cfg.ConnectionID = "next-connection"
	if err := m.Configure(&cfg); err != nil {
		t.Fatal(err)
	}
}

func TestPreparedMarkerCannotReopenPlainFileAfterRevoke(t *testing.T) {
	cfg := testConfig(t)
	marker := filepath.Join(cfg.Directory, "xray-core.log")
	m := &Manager{}
	if err := m.Configure(&cfg); err != nil {
		t.Fatal(err)
	}
	if err := m.Configure(nil); err != nil {
		t.Fatal(err)
	}
	h, err := m.create(applog.LogType_File, applog.HandlerCreatorOptions{Path: marker})
	if err != nil || h == nil {
		t.Fatal("revoked collection must not abort prepared VPN start")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("prepared marker reopened an unbounded plain file after revoke")
	}
}

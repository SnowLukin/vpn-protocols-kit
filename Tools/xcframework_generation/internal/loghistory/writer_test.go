package loghistory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testConfig(t *testing.T) Config {
	t.Helper()
	return Config{Directory: t.TempDir(), ConnectionID: "connection-a", Policy: Policy{MaxSegmentBytes: 512, MaxSourceBytes: 2048, MaxSegments: 4}}
}

func readRecords(t *testing.T, directory string) ([]map[string]any, int64) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(directory, "xray-core.*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	var total int64
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) > 512 {
			t.Fatalf("segment exceeds budget: %d", len(data))
		}
		total += int64(len(data))
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line == "" {
				continue
			}
			var record map[string]any
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				t.Fatalf("broken record: %v", err)
			}
			records = append(records, record)
		}
	}
	if len(paths) > 4 {
		t.Fatalf("too many parts: %d", len(paths))
	}
	return records, total
}

func TestRotationKeepsWholeRecentEventsWithinSourceBudget(t *testing.T) {
	cfg := testConfig(t)
	w, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for i := 0; i < 40; i++ {
		err := w.Write(Event{Timestamp: time.Unix(int64(i), 0), Level: "warning", Message: fmt.Sprintf("row-%02d %s", i, strings.Repeat("ж", 70))})
		if err != nil {
			t.Fatal(err)
		}
	}
	records, total := readRecords(t, cfg.Directory)
	if total > cfg.Policy.MaxSourceBytes {
		t.Fatalf("source exceeds budget: %d", total)
	}
	if len(records) == 0 || !strings.HasPrefix(records[len(records)-1]["message"].(string), "row-39") {
		t.Fatal("latest event was lost")
	}
	for _, record := range records {
		if strings.HasPrefix(record["message"].(string), "row-00") {
			t.Fatal("oldest event was not evicted")
		}
	}
}

func TestReconnectKeepsHistoryAndMultilineIdentity(t *testing.T) {
	cfg := testConfig(t)
	w, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	message := "first\n  YAML: \"quoted\"\n\\path\\value\tкир"
	if err = w.Write(Event{Timestamp: time.Unix(100, 123456000), Level: "error", Message: message}); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	cfg.ConnectionID = "connection-b"
	w, err = New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err = w.Write(Event{Timestamp: time.Unix(101, 0), Level: "info", Message: "next session"}); err != nil {
		t.Fatal(err)
	}
	records, _ := readRecords(t, cfg.Directory)
	if len(records) != 2 {
		t.Fatalf("want both sessions, got %d", len(records))
	}
	if records[0]["message"] != message || records[0]["connectionId"] != "connection-a" || records[1]["connectionId"] != "connection-b" {
		t.Fatalf("history changed: %#v", records)
	}
	if records[0]["source"] != "xray" || records[0]["timestamp"] != "1970-01-01T00:01:40.123456Z" {
		t.Fatal("invalid source/time envelope")
	}
}

func TestCrashTailNeverAbsorbsNextEvent(t *testing.T) {
	cfg := testConfig(t)
	first := filepath.Join(cfg.Directory, "xray-core.00000000000000000001.jsonl")
	if err := os.WriteFile(first, []byte(`{"v":1,"message":"unfinished`), 0600); err != nil {
		t.Fatal(err)
	}
	w, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err = w.Write(Event{Timestamp: time.Unix(100, 0), Level: "info", Message: "after restart"}); err != nil {
		t.Fatal(err)
	}
	old, _ := os.ReadFile(first)
	if string(old) != `{"v":1,"message":"unfinished` {
		t.Fatal("crash evidence was overwritten")
	}
	second, err := os.ReadFile(filepath.Join(cfg.Directory, "xray-core.00000000000000000002.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err = json.Unmarshal(second, &record); err != nil || record["message"] != "after restart" {
		t.Fatal("new event was glued to crash tail")
	}
}

func TestRotationFailureStopsGrowthAndReportsFailure(t *testing.T) {
	cfg := testConfig(t)
	w, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	event := Event{Timestamp: time.Unix(100, 0), Level: "info", Message: strings.Repeat("x", 240)}
	if err = w.Write(event); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(cfg.Directory, "xray-core.00000000000000000001.jsonl")
	before, _ := os.Stat(first)
	if err = os.Mkdir(filepath.Join(cfg.Directory, "xray-core.00000000000000000002.jsonl"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = w.Write(event); err == nil {
		t.Fatal("rotation failure was hidden")
	}
	after, _ := os.Stat(first)
	if after.Size() != before.Size() {
		t.Fatal("old file grew after rotation failed")
	}
	state := w.State()
	if state.Status != "failed" || state.ErrorCode == "" || strings.Contains(state.ErrorCode, cfg.Directory) {
		t.Fatalf("unsafe or missing failure: %#v", state)
	}
}

func TestOversizedEventIsExplicitAndBounded(t *testing.T) {
	cfg := testConfig(t)
	w, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err = w.Write(Event{Timestamp: time.Unix(100, 0), Level: "error", Message: strings.Repeat("\x00", 10000)}); err != nil {
		t.Fatal(err)
	}
	records, _ := readRecords(t, cfg.Directory)
	if len(records) != 1 || !strings.Contains(records[0]["message"].(string), "omitted") {
		t.Fatal("oversized event silently changed or lost")
	}
}

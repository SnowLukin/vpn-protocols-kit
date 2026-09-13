package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SnowLukin/vpn_protocols-kit/internal/loghistory"
	commonlog "github.com/xtls/xray-core/common/log"
	"github.com/xtls/xray-core/core"
)

func TestXrayUsesHistoryFactoryAcrossActualCoreRestarts(t *testing.T) {
	directory := t.TempDir()
	cfg := loghistory.Config{Directory: directory, Policy: loghistory.DefaultPolicy()}
	for _, session := range []string{"first-connection", "second-connection"} {
		cfg.ConnectionID = session
		if err := xrayLogHistory.Configure(&cfg); err != nil {
			t.Fatal(err)
		}
		config, _ := json.Marshal(map[string]any{
			"log":       map[string]any{"error": filepath.Join(directory, "xray-core.log"), "loglevel": "info"},
			"outbounds": []map[string]any{{"protocol": "freedom"}},
		})
		instance, err := core.StartInstance("json", config)
		if err != nil {
			t.Fatal(err)
		}
		commonlog.Record(&commonlog.GeneralMessage{Severity: commonlog.Severity_Error, Content: "history-proof " + session + "\n  at frame"})
		if err := instance.Close(); err != nil {
			t.Fatal(err)
		}
	}
	defer xrayLogHistory.Configure(nil)
	files, err := filepath.Glob(filepath.Join(directory, "xray-core.*.jsonl"))
	if err != nil || len(files) == 0 {
		t.Fatal("actual Xray file handler did not use history")
	}
	found := map[string]bool{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			var record map[string]any
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				t.Fatal(err)
			}
			message, _ := record["message"].(string)
			if strings.HasPrefix(message, "history-proof ") {
				id, _ := record["connectionId"].(string)
				if message != "history-proof "+id+"\n  at frame" {
					t.Fatalf("lost multiline identity: %s", line)
				}
				found[id] = true
			}
		}
	}
	if !found["first-connection"] || !found["second-connection"] {
		t.Fatalf("lost reconnect history: %#v", found)
	}
	if _, err := os.Stat(filepath.Join(directory, "xray-core.log")); !os.IsNotExist(err) {
		t.Fatal("legacy unbounded file writer was still opened")
	}
}

package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"github.com/xtls/libxray/xray"
	"github.com/xtls/xray-core/common/platform"
)

func decodeCallResponse(t *testing.T, raw string) (success bool, message string) {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("response is not base64: %v", err)
	}
	var response struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	return response.Success, response.Error
}

func TestRunXrayFromJSONStartsAndStopsCore(t *testing.T) {
	datDir := t.TempDir()
	config := `{"inbounds":[{"listen":"127.0.0.1","port":0,"protocol":"socks"}],"outbounds":[{"protocol":"freedom"}]}`

	success, message := decodeCallResponse(t, runXrayFromJSON(datDir, config))
	if !success || message != "" {
		t.Fatalf("start failed: %q", message)
	}
	defer xray.StopXray()

	if !xray.GetXrayState() {
		t.Fatal("xray is not running after a successful start")
	}
	if os.Getenv(platform.AssetLocation) != datDir {
		t.Fatalf("asset dir is %q, want %q", os.Getenv(platform.AssetLocation), datDir)
	}
	if success, message := decodeCallResponse(t, callResponse(xray.StopXray())); !success {
		t.Fatalf("stop failed: %q", message)
	}
	if xray.GetXrayState() {
		t.Fatal("xray still runs after stop")
	}
}

func TestRunXrayFromJSONReportsConfigError(t *testing.T) {
	success, message := decodeCallResponse(t, runXrayFromJSON(t.TempDir(), `{"outbounds": [`))
	if success || message == "" {
		t.Fatalf("broken config: success=%v error=%q", success, message)
	}
	if xray.GetXrayState() {
		t.Fatal("xray runs after a failed start")
	}
}

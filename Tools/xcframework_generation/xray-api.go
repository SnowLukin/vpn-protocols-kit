package main

import "C"
import (
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/SnowLukin/vpn_protocols-kit/internal/loghistory"
	"github.com/xtls/libxray/xray"
	"github.com/xtls/xray-core/common/platform"
)

var xrayLogHistory loghistory.Manager

func init() {
	if err := xrayLogHistory.Register(); err != nil {
		panic(err)
	}
}

//export LibXrayConfigureLogHistory
func LibXrayConfigureLogHistory(configJSON *C.char) C.int {
	raw := C.GoString(configJSON)
	if raw == "" {
		if xrayLogHistory.Configure(nil) != nil {
			return -1
		}
		return 0
	}
	if len(raw) > 8192 {
		return -1
	}
	var config loghistory.Config
	if json.Unmarshal([]byte(raw), &config) != nil || xrayLogHistory.Configure(&config) != nil {
		return -1
	}
	return 0
}

//export LibXrayGetLogHistoryState
func LibXrayGetLogHistoryState() *C.char {
	state := xrayLogHistory.State()
	if state == nil {
		return nil
	}
	data, err := json.Marshal(state)
	if err != nil {
		return nil
	}
	return C.CString(string(data))
}

// xray-core reads geo files and certificates from these process variables;
// libXray no longer takes a datDir and expects them in the config env.
func setAssetDir(datDir string) {
	if datDir == "" {
		return
	}
	os.Setenv(platform.AssetLocation, datDir)
	os.Setenv(platform.CertLocation, datDir)
}

// The response keeps the shape Swift parses: base64 of JSON with "error".
func callResponse(err error) string {
	response := struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}{Success: err == nil}
	if err != nil {
		response.Error = err.Error()
	}
	data, _ := json.Marshal(&response)
	return base64.StdEncoding.EncodeToString(data)
}

func runXrayFromJSON(datDir, configJSON string) string {
	setAssetDir(datDir)
	return callResponse(xray.RunXrayFromJSON(configJSON))
}

// Run Xray instance with JSON configuration.
// datDir means the dir which geosite.dat and geoip.dat are in.
//
//export LibXrayRunXrayFromJSON
func LibXrayRunXrayFromJSON(datDir, configJSON *C.char) *C.char {
	return C.CString(runXrayFromJSON(C.GoString(datDir), C.GoString(configJSON)))
}

// Stop Xray instance.
//
//export LibXrayStopXray
func LibXrayStopXray() *C.char {
	return C.CString(callResponse(xray.StopXray()))
}

// Xray's version
//
//export LibXrayXrayVersion
func LibXrayXrayVersion() *C.char {
	return C.CString(xray.XrayVersion())
}

// Get Xray State
//
//export LibXrayGetXrayState
func LibXrayGetXrayState() C.int {
	if xray.GetXrayState() {
		return 1
	}
	return 0
}

package main

import "C"
import (
	"encoding/base64"
	"encoding/json"

	libxray "github.com/xtls/libxray"
)

// Run Xray instance.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// maxMemory means the soft memory limit of golang, see SetMemoryLimit to find more information.
//
//export LibXrayRunXray
func LibXrayRunXray(datDir, configPath *C.char, maxMemory int64) *C.char {
	request := libxray.RunXrayRequest{
		DatDir:     C.GoString(datDir),
		ConfigPath: C.GoString(configPath),
	}
	requestBytes, err := json.Marshal(&request)
	if err != nil {
		return C.CString(err.Error())
	}
	base64Text := base64.StdEncoding.EncodeToString(requestBytes)
	result := libxray.RunXray(base64Text)
	return C.CString(result)
}

// Run Xray instance with JSON configuration
//
//export LibXrayRunXrayFromJSON
func LibXrayRunXrayFromJSON(datDir, configJSON *C.char) *C.char {
	request := libxray.RunXrayFromJSONRequest{
		DatDir:     C.GoString(datDir),
		ConfigJSON: C.GoString(configJSON),
	}
	requestBytes, err := json.Marshal(&request)
	if err != nil {
		return C.CString(err.Error())
	}
	base64Text := base64.StdEncoding.EncodeToString(requestBytes)
	result := libxray.RunXrayFromJSON(base64Text)
	return C.CString(result)
}

// Stop Xray instance.
//
//export LibXrayStopXray
func LibXrayStopXray() *C.char {
	result := libxray.StopXray()
	return C.CString(result)
}

// Xray's version
//
//export LibXrayXrayVersion
func LibXrayXrayVersion() *C.char {
	result := libxray.XrayVersion()
	return C.CString(result)
}

// Get Xray State
//
//export LibXrayGetXrayState
func LibXrayGetXrayState() C.int {
	if libxray.GetXrayState() {
		return 1
	}
	return 0
}

// Test Xray Config
//
//export LibXrayTestXray
func LibXrayTestXray(datDir, configPath *C.char) *C.char {
	request := libxray.TestXrayRequest{
		DatDir:     C.GoString(datDir),
		ConfigPath: C.GoString(configPath),
	}
	requestBytes, err := json.Marshal(&request)
	if err != nil {
		return C.CString(err.Error())
	}
	base64Text := base64.StdEncoding.EncodeToString(requestBytes)
	result := libxray.TestXray(base64Text)
	return C.CString(result)
}

// Ping Xray config
//
//export LibXrayPing
func LibXrayPing(datDir, configPath *C.char, timeout C.int, url, proxy *C.char) *C.char {
	// Create a simple JSON for ping request
	request := map[string]interface{}{
		"datDir":     C.GoString(datDir),
		"configPath": C.GoString(configPath),
		"timeout":    int(timeout),
		"url":        C.GoString(url),
		"proxy":      C.GoString(proxy),
	}
	requestBytes, err := json.Marshal(&request)
	if err != nil {
		return C.CString("")
	}
	base64Text := base64.StdEncoding.EncodeToString(requestBytes)
	result := libxray.Ping(base64Text)
	return C.CString(result)
}

// Query inbound and outbound stats
//
//export LibXrayQueryStats
func LibXrayQueryStats(server *C.char) *C.char {
	base64Text := base64.StdEncoding.EncodeToString([]byte(C.GoString(server)))
	result := libxray.QueryStats(base64Text)
	return C.CString(result)
}

// Count geo data
//
//export LibXrayCountGeoData
func LibXrayCountGeoData(datDir, name, geoType *C.char) *C.char {
	request := libxray.CountGeoDataRequest{
		DatDir:  C.GoString(datDir),
		Name:    C.GoString(name),
		GeoType: C.GoString(geoType),
	}
	requestBytes, err := json.Marshal(&request)
	if err != nil {
		return C.CString(err.Error())
	}
	base64Text := base64.StdEncoding.EncodeToString(requestBytes)
	result := libxray.CountGeoData(base64Text)
	return C.CString(result)
}

// Read geo files
//
//export LibXrayReadGeoFiles
func LibXrayReadGeoFiles(data *C.char) *C.char {
	base64Text := base64.StdEncoding.EncodeToString([]byte(C.GoString(data)))
	result := libxray.ReadGeoFiles(base64Text)
	return C.CString(result)
}

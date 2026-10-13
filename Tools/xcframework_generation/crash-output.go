package main

import "C"
import (
	"errors"
	"os"
	"runtime/debug"
)

// LibXraySetCrashOutput makes the Go runtime copy its fatal report to the file
// at path, truncating it. Xray and AmneziaWG share one Go runtime, so a single
// call covers both. Returns NULL on success, otherwise an error message the
// caller must free.
//
//export LibXraySetCrashOutput
func LibXraySetCrashOutput(path *C.char) *C.char {
	if err := setCrashOutput(C.GoString(path)); err != nil {
		return C.CString(err.Error())
	}
	return nil
}

func setCrashOutput(path string) error {
	if path == "" {
		return errors.New("crash output path is empty")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	// SetCrashOutput keeps its own duplicate of the descriptor.
	defer file.Close()
	return debug.SetCrashOutput(file, debug.CrashOptions{})
}

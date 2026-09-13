package loghistory

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const stem = "xray-core"

type Policy struct {
	MaxSegmentBytes int64 `json:"maxSegmentBytes"`
	MaxSourceBytes  int64 `json:"maxSourceBytes"`
	MaxSegments     int   `json:"maxSegments"`
	MaxPendingBytes int64 `json:"maxPendingBytes,omitempty"`
}

func DefaultPolicy() Policy {
	return Policy{MaxSegmentBytes: 2 * 1024 * 1024, MaxSourceBytes: 8 * 1024 * 1024, MaxSegments: 4, MaxPendingBytes: 512 * 1024}
}

func (p Policy) Validate() error {
	if p.MaxSegmentBytes < 512 || p.MaxSourceBytes < p.MaxSegmentBytes || p.MaxSegments < 1 || int64(p.MaxSegments) > p.MaxSourceBytes/p.MaxSegmentBytes || p.MaxPendingBytes < 0 {
		return errors.New("invalid_log_history_policy")
	}
	return nil
}

type Config struct {
	Directory    string `json:"directory"`
	ConnectionID string `json:"connectionId"`
	Policy       Policy `json:"policy"`
}

type Event struct {
	Timestamp time.Time
	Level     string
	Message   string
	Kind      string
}

type envelope struct {
	Version      int       `json:"v"`
	Timestamp    time.Time `json:"timestamp"`
	Source       string    `json:"source"`
	Level        string    `json:"level"`
	ConnectionID *string   `json:"connectionId"`
	Kind         string    `json:"kind,omitempty"`
	Message      string    `json:"message"`
}

type State struct {
	Version      int     `json:"v"`
	Status       string  `json:"status"`
	ConnectionID *string `json:"connectionId,omitempty"`
	ErrorCode    string  `json:"errorCode,omitempty"`
}

type part struct {
	path     string
	size     int64
	sequence uint64
	legacy   bool
}

type Writer struct {
	mu       sync.Mutex
	config   Config
	state    State
	parts    []part
	file     *os.File
	sequence uint64
	closed   bool
}

type fileError struct {
	code  string
	cause error
}

func (e *fileError) Error() string { return e.code }
func (e *fileError) Unwrap() error { return e.cause }

func SafeErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var failure *fileError
	if errors.As(err, &failure) {
		return failure.code
	}
	return "io_unknown"
}

func New(config Config) (*Writer, error) {
	if config.Policy == (Policy{}) {
		config.Policy = DefaultPolicy()
	}
	if config.Policy.MaxPendingBytes == 0 {
		config.Policy.MaxPendingBytes = DefaultPolicy().MaxPendingBytes
	}
	if err := config.Policy.Validate(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(config.Directory) || len(config.ConnectionID) > 128 {
		return nil, errors.New("invalid_log_history_config")
	}
	w := &Writer{config: config, state: State{Version: 1, Status: "active", ConnectionID: connectionID(config.ConnectionID)}}
	if err := os.MkdirAll(config.Directory, 0700); err != nil {
		return nil, w.fail("io_open", err)
	}
	if err := w.load(); err != nil {
		return nil, w.fail("io_read", err)
	}
	if len(w.parts) > 0 {
		last := w.parts[len(w.parts)-1]
		if !last.legacy && last.size < config.Policy.MaxSegmentBytes {
			complete, err := completeFile(last.path, last.size)
			if err != nil {
				return nil, w.fail("io_read", err)
			}
			if complete {
				w.file, err = os.OpenFile(last.path, os.O_WRONLY|os.O_APPEND, 0600)
				if err != nil {
					return nil, w.fail("io_open", err)
				}
			}
		}
	}
	if w.file == nil {
		if err := w.openNext(); err != nil {
			return nil, w.fail("io_open", err)
		}
	}
	if err := w.trim(0); err != nil {
		return nil, w.fail("io_rotate", err)
	}
	if err := w.saveState(); err != nil {
		return nil, w.fail("io_state", err)
	}
	return w, nil
}

func connectionID(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func completeFile(path string, size int64) (bool, error) {
	if size == 0 {
		return true, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	var last [1]byte
	_, err = f.ReadAt(last[:], size-1)
	return last[0] == '\n', err
}

func (w *Writer) load() error {
	entries, err := os.ReadDir(w.config.Directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		name := entry.Name()
		legacy := name == stem+".log" || name == stem+".log.old"
		var sequence uint64
		if !legacy {
			if !strings.HasPrefix(name, stem+".") || !strings.HasSuffix(name, ".jsonl") {
				continue
			}
			number := strings.TrimSuffix(strings.TrimPrefix(name, stem+"."), ".jsonl")
			if len(number) != 20 {
				continue
			}
			sequence, err = strconv.ParseUint(number, 10, 64)
			if err != nil || sequence == 0 {
				continue
			}
			if sequence > w.sequence {
				w.sequence = sequence
			}
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		w.parts = append(w.parts, part{path: filepath.Join(w.config.Directory, name), size: info.Size(), sequence: sequence, legacy: legacy})
	}
	sort.Slice(w.parts, func(i, j int) bool {
		if w.parts[i].legacy != w.parts[j].legacy {
			return w.parts[i].legacy
		}
		if w.parts[i].sequence != w.parts[j].sequence {
			return w.parts[i].sequence < w.parts[j].sequence
		}
		return strings.HasSuffix(w.parts[i].path, ".old") && !strings.HasSuffix(w.parts[j].path, ".old")
	})
	return nil
}

func (w *Writer) openNext() error {
	if w.sequence == ^uint64(0) {
		return errors.New("log_history_sequence_exhausted")
	}
	w.sequence++
	path := filepath.Join(w.config.Directory, fmt.Sprintf("%s.%020d.jsonl", stem, w.sequence))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	w.file = file
	w.parts = append(w.parts, part{path: path, sequence: w.sequence})
	return nil
}

func encode(config Config, event Event) ([]byte, error) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if int64(len(event.Message)) > min(config.Policy.MaxSegmentBytes, pendingLimit(config.Policy)) {
		event.Message = "<message omitted: exceeds log event budget>"
	}
	record := envelope{Version: 1, Timestamp: event.Timestamp.UTC().Truncate(time.Microsecond), Source: "xray", Level: event.Level, ConnectionID: connectionID(config.ConnectionID), Kind: event.Kind, Message: event.Message}
	data, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	if int64(len(data)+1) > config.Policy.MaxSegmentBytes {
		record.Message = "<message omitted: exceeds log segment budget>"
		data, err = json.Marshal(record)
		if err != nil {
			return nil, err
		}
		if int64(len(data)+1) > config.Policy.MaxSegmentBytes {
			return nil, errors.New("log_event_metadata_exceeds_budget")
		}
	}
	return append(data, '\n'), nil
}

func (w *Writer) Write(event Event) error {
	data, err := encode(w.config, event)
	if err != nil {
		return err
	}
	return w.writeRecord(data)
}

func (w *Writer) writeRecord(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.state.Status == "failed" {
		return &fileError{code: w.state.ErrorCode}
	}
	if w.closed {
		return errors.New("log_history_closed")
	}

	current := &w.parts[len(w.parts)-1]
	if current.size+int64(len(data)) > w.config.Policy.MaxSegmentBytes {
		if err := w.file.Close(); err != nil {
			return w.fail("io_close", err)
		}
		w.file = nil
		if err := w.openNext(); err != nil {
			return w.fail("io_rotate", err)
		}
	}
	if err := w.trim(int64(len(data))); err != nil {
		return w.fail("io_rotate", err)
	}
	written, err := w.file.Write(data)
	w.parts[len(w.parts)-1].size += int64(written)
	if err == nil && written != len(data) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return w.fail("io_write", err)
	}
	return nil
}

func (w *Writer) trim(incoming int64) error {
	var total int64
	for _, part := range w.parts {
		total += part.size
	}
	for len(w.parts) > w.config.Policy.MaxSegments || total+incoming > w.config.Policy.MaxSourceBytes {
		if len(w.parts) <= 1 {
			return errors.New("log_history_budget_exhausted")
		}
		oldest := w.parts[0]
		if err := os.Remove(oldest.path); err != nil {
			return err
		}
		total -= oldest.size
		w.parts = w.parts[1:]
	}
	return nil
}

func (w *Writer) State() State {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	if w.file != nil {
		if err := w.file.Sync(); err != nil {
			return w.fail("io_flush", err)
		}
		if err := w.file.Close(); err != nil {
			return w.fail("io_close", err)
		}
		w.file = nil
	}
	if w.state.Status != "failed" {
		w.state.Status = "closed"
		if err := w.saveState(); err != nil {
			return w.fail("io_state", err)
		}
	}
	return nil
}

func (w *Writer) fail(code string, cause error) error {
	w.state.Status = "failed"
	w.state.ErrorCode = code
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	_ = w.saveState()
	return &fileError{code: code, cause: cause}
}

func (w *Writer) saveState() error {
	data, err := json.Marshal(w.state)
	if err != nil {
		return err
	}
	if len(data) > 4096 {
		return errors.New("log_history_state_exceeds_budget")
	}
	path := filepath.Join(w.config.Directory, stem+".state.json")
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

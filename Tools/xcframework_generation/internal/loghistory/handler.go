package loghistory

import (
	"fmt"
	"strings"
	"sync"
	"time"

	commonlog "github.com/xtls/xray-core/common/log"
)

type recordSink interface {
	writeRecord([]byte) error
	State() State
	Close() error
}

type Handler struct {
	mu         sync.Mutex
	config     Config
	sink       recordSink
	queue      [][]byte
	pending    int64
	dropped    uint64
	closing    bool
	failed     bool
	wake       chan struct{}
	done       chan struct{}
	closeError error
}

func pendingLimit(policy Policy) int64 {
	if policy.MaxPendingBytes == 0 {
		return DefaultPolicy().MaxPendingBytes
	}
	return policy.MaxPendingBytes
}

func newHandler(config Config, sink recordSink) *Handler {
	h := &Handler{config: config, sink: sink, wake: make(chan struct{}, 1), done: make(chan struct{})}
	go h.run()
	return h
}

func (h *Handler) Handle(message commonlog.Message) {
	event := Event{Timestamp: time.Now(), Level: "info"}
	switch message := message.(type) {
	case *commonlog.GeneralMessage:
		event.Level = strings.ToLower(message.Severity.String())
		if text, ok := message.Content.(string); ok {
			event.Message = text
		} else {
			event.Message = fmt.Sprint(message.Content)
		}
	default:
		event.Message = message.String()
	}
	data, err := encode(h.config, event)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closing || h.failed {
		return
	}
	if h.dropped > 0 {
		marker, _ := encode(h.config, Event{Timestamp: time.Now(), Level: "warning", Kind: "dropped", Message: fmt.Sprintf("%d log events dropped: pending byte limit", h.dropped)})
		if int64(len(marker)) <= pendingLimit(h.config.Policy)-h.pending {
			h.queue = append(h.queue, marker)
			h.pending += int64(len(marker))
			h.dropped = 0
		}
	}
	if err != nil || int64(len(data)) > pendingLimit(h.config.Policy)-h.pending {
		if h.dropped < ^uint64(0) {
			h.dropped++
		}
	} else {
		h.queue = append(h.queue, data)
		h.pending += int64(len(data))
	}
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

func (h *Handler) run() {
	defer close(h.done)
	for {
		<-h.wake
		for {
			h.mu.Lock()
			var data []byte
			queued := len(h.queue) > 0
			if queued {
				data = h.queue[0]
				h.queue[0] = nil
				h.queue = h.queue[1:]
			} else if h.dropped > 0 {
				data, _ = encode(h.config, Event{Timestamp: time.Now(), Level: "warning", Kind: "dropped", Message: fmt.Sprintf("%d log events dropped: pending byte limit", h.dropped)})
				h.dropped = 0
				h.pending += int64(len(data))
			} else if h.closing {
				h.mu.Unlock()
				h.closeError = h.sink.Close()
				return
			} else {
				h.mu.Unlock()
				break
			}
			h.mu.Unlock()
			err := h.sink.writeRecord(data)
			h.mu.Lock()
			h.pending -= int64(len(data))
			if err != nil {
				h.failed = true
				h.queue = nil
				h.pending = 0
				h.dropped = 0
				h.closeError = err
				h.mu.Unlock()
				_ = h.sink.Close()
				return
			}
			h.mu.Unlock()
		}
	}
}

func (h *Handler) Close() error {
	h.mu.Lock()
	h.closing = true
	select {
	case h.wake <- struct{}{}:
	default:
	}
	h.mu.Unlock()
	<-h.done
	return h.closeError
}

func (h *Handler) State() State { return h.sink.State() }

func (h *Handler) finished() bool {
	select {
	case <-h.done:
		return true
	default:
		return false
	}
}

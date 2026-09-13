package loghistory

import (
	"errors"
	"path/filepath"
	"sync"

	applog "github.com/xtls/xray-core/app/log"
	commonlog "github.com/xtls/xray-core/common/log"
)

type Manager struct {
	mu     sync.Mutex
	config *Config
	// A prepared VPN config must not reopen a plain file after collection is revoked.
	markerPath string
	handler    *Handler
	failure    *State
}

func (m *Manager) Register() error {
	return applog.RegisterHandlerCreator(applog.LogType_File, m.create)
}

func (m *Manager) Configure(config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if config == nil {
		m.config = nil
		if m.handler != nil {
			_ = m.handler.Close()
		}
		return nil
	}
	if m.handler != nil && !m.handler.finished() {
		return errors.New("log_history_busy")
	}
	copy := *config
	if copy.Policy == (Policy{}) {
		copy.Policy = DefaultPolicy()
	}
	if err := copy.Policy.Validate(); err != nil {
		return err
	}
	if !filepath.IsAbs(copy.Directory) || len(copy.ConnectionID) > 128 {
		return errors.New("invalid_log_history_config")
	}
	m.config = &copy
	m.markerPath = filepath.Join(copy.Directory, "xray-core.log")

	m.handler = nil
	m.failure = nil
	return nil
}

func (m *Manager) State() *State {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.handler != nil {
		state := m.handler.State()
		return &state
	}
	if m.failure != nil {
		state := *m.failure
		return &state
	}
	return nil
}

func (m *Manager) create(_ applog.LogType, options applog.HandlerCreatorOptions) (commonlog.Handler, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if filepath.Clean(options.Path) != m.markerPath {
		creator, err := commonlog.CreateFileLogWriter(options.Path)
		if err != nil {
			return nil, err
		}
		return commonlog.NewLogger(creator), nil
	}
	if m.config == nil {
		return discardedHandler{}, nil
	}

	if m.handler != nil && !m.handler.finished() {
		return discardedHandler{}, nil
	}
	writer, err := New(*m.config)
	if err != nil {
		m.failure = &State{Version: 1, Status: "failed", ConnectionID: connectionID(m.config.ConnectionID), ErrorCode: SafeErrorCode(err)}
		return discardedHandler{}, nil
	}
	m.handler = newHandler(*m.config, writer)
	return m.handler, nil
}

type discardedHandler struct{}

func (discardedHandler) Handle(commonlog.Message) {}

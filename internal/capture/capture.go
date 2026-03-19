package capture

import (
	"strings"
	"sync"
	"time"
)

type State string

const (
	StateInactive State = "inactive"
	StateActive   State = "active"
	StatePaused   State = "paused"
)

type Manager struct {
	mu      sync.RWMutex
	state   State
	sources map[string]SourceState
}

func NewManager() *Manager {
	now := time.Now().UTC()
	return &Manager{
		state: StateInactive,
		sources: map[string]SourceState{
			"microphone": {Status: "unknown", UpdatedAt: now},
			"screen":     {Status: "unknown", UpdatedAt: now},
			"companion":  {Status: "unknown", UpdatedAt: now},
		},
	}
}

func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = StateActive
}

func (m *Manager) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = StatePaused
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = StateInactive
}

func (m *Manager) State() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

type SourceState struct {
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *Manager) SetSourceStatus(source, status, reason string) {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		return
	}
	status = normalizeSourceStatus(status)
	if status == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sources == nil {
		m.sources = map[string]SourceState{}
	}
	m.sources[source] = SourceState{
		Status:    status,
		Reason:    strings.TrimSpace(reason),
		UpdatedAt: time.Now().UTC(),
	}
}

func (m *Manager) SourceStatuses() map[string]SourceState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]SourceState, len(m.sources))
	for k, v := range m.sources {
		out[k] = v
	}
	return out
}

func (m *Manager) ReducedCapabilities() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.sources))
	for k, v := range m.sources {
		if v.Status == "available" || v.Status == "unknown" {
			continue
		}
		out = append(out, k)
	}
	return out
}

func normalizeSourceStatus(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "available":
		return "available"
	case "degraded":
		return "degraded"
	case "unavailable":
		return "unavailable"
	default:
		return ""
	}
}

package audio

import (
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	ModePushToTalk = "push_to_talk"
	ModeAlwaysOn   = "always_on"
)

type Device struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type State struct {
	Mode            string    `json:"mode"`
	InputDeviceID   string    `json:"input_device_id,omitempty"`
	InputDeviceName string    `json:"input_device_name,omitempty"`
	Muted           bool      `json:"muted"`
	Paused          bool      `json:"paused"`
	LastLevel       float64   `json:"last_level"`
	LevelValid      bool      `json:"level_valid"`
	LevelMin        float64   `json:"level_min"`
	LevelMax        float64   `json:"level_max"`
	LastLevelAt     time.Time `json:"last_level_at,omitempty"`
}

type Controller struct {
	mu      sync.RWMutex
	state   State
	devices []Device
}

func NewController() *Controller {
	devs := defaultDevices()
	st := State{
		Mode:       ModeAlwaysOn,
		LevelMin:   0.02,
		LevelMax:   0.95,
		LevelValid: false,
	}
	if len(devs) > 0 {
		st.InputDeviceID = devs[0].ID
		st.InputDeviceName = devs[0].Label
	}
	return &Controller{
		state:   st,
		devices: devs,
	}
}

func (c *Controller) State() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *Controller) Devices() []Device {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Device, len(c.devices))
	copy(out, c.devices)
	return out
}

func (c *Controller) SetDevices(devices []Device) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Device, 0, len(devices))
	for _, d := range devices {
		id := strings.TrimSpace(d.ID)
		label := strings.TrimSpace(d.Label)
		if id == "" {
			continue
		}
		if label == "" {
			label = id
		}
		out = append(out, Device{ID: id, Label: label})
	}
	if len(out) == 0 {
		out = defaultDevices()
	}
	c.devices = out
	if c.state.InputDeviceID == "" {
		c.state.InputDeviceID = out[0].ID
		c.state.InputDeviceName = out[0].Label
		return
	}
	for _, d := range out {
		if d.ID == c.state.InputDeviceID {
			c.state.InputDeviceName = d.Label
			return
		}
	}
	c.state.InputDeviceID = out[0].ID
	c.state.InputDeviceName = out[0].Label
}

type Config struct {
	Mode          string  `json:"mode"`
	InputDeviceID string  `json:"input_device_id"`
	Muted         *bool   `json:"muted,omitempty"`
	Paused        *bool   `json:"paused,omitempty"`
	LevelMin      float64 `json:"level_min,omitempty"`
	LevelMax      float64 `json:"level_max,omitempty"`
}

func (c *Controller) Configure(cfg Config) State {
	c.mu.Lock()
	defer c.mu.Unlock()
	mode := normalizeMode(cfg.Mode)
	if mode != "" {
		c.state.Mode = mode
	}
	if cfg.Muted != nil {
		c.state.Muted = *cfg.Muted
	}
	if cfg.Paused != nil {
		c.state.Paused = *cfg.Paused
	}
	if cfg.LevelMin > 0 && cfg.LevelMin < 1 {
		c.state.LevelMin = cfg.LevelMin
	}
	if cfg.LevelMax > 0 && cfg.LevelMax <= 1 {
		c.state.LevelMax = cfg.LevelMax
	}
	if c.state.LevelMax < c.state.LevelMin {
		c.state.LevelMax = c.state.LevelMin
	}
	if id := strings.TrimSpace(cfg.InputDeviceID); id != "" {
		c.state.InputDeviceID = id
		c.state.InputDeviceName = id
		for _, d := range c.devices {
			if d.ID == id {
				c.state.InputDeviceName = d.Label
				break
			}
		}
	}
	return c.state
}

func (c *Controller) UpdateLevel(level float64) State {
	c.mu.Lock()
	defer c.mu.Unlock()
	if level < 0 {
		level = 0
	}
	if level > 1 {
		level = 1
	}
	c.state.LastLevel = level
	c.state.LastLevelAt = time.Now().UTC()
	c.state.LevelValid = level >= c.state.LevelMin && level <= c.state.LevelMax
	return c.state
}

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ModePushToTalk:
		return ModePushToTalk
	case ModeAlwaysOn:
		return ModeAlwaysOn
	default:
		return ""
	}
}

func defaultDevices() []Device {
	switch runtime.GOOS {
	case "darwin":
		return []Device{
			{ID: "default", Label: "System Default (macOS)"},
		}
	case "windows":
		return []Device{
			{ID: "default", Label: "System Default (Windows)"},
		}
	default:
		return []Device{
			{ID: "default", Label: "System Default (Linux)"},
		}
	}
}

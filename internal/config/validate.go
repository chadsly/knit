package config

import (
	"fmt"
	"strings"
	"time"
)

var allowedCapabilities = map[string]struct{}{
	"capture":     {},
	"submit":      {},
	"config":      {},
	"config_read": {},
	"purge":       {},
	"read":        {},
	"logs":        {},
	"*":           {},
}

var allowedVideoModes = map[string]struct{}{
	"event_triggered": {},
	"on_demand":       {},
	"continuous":      {},
}

var allowedTranscriptionModes = map[string]struct{}{
	"faster_whisper": {},
	"local":          {},
	"lmstudio":       {},
	"remote":         {},
}

// Validate checks if a config is internally consistent and safe to apply.
func Validate(cfg Config) error {
	if strings.TrimSpace(cfg.HTTPListenAddr) == "" {
		return fmt.Errorf("http listen address cannot be empty")
	}
	if strings.TrimSpace(cfg.SQLitePath) == "" {
		return fmt.Errorf("sqlite path cannot be empty")
	}
	if strings.TrimSpace(cfg.UIRuntime) == "" {
		return fmt.Errorf("ui runtime cannot be empty")
	}
	if cfg.PointerSampleHz < 1 || cfg.PointerSampleHz > 120 {
		return fmt.Errorf("pointer sample hz must be between 1 and 120")
	}

	if cfg.ArtifactMaxFiles <= 0 {
		return fmt.Errorf("artifact max files must be greater than zero")
	}

	if err := validateDuration("audio retention", cfg.AudioRetention); err != nil {
		return err
	}
	if err := validateDuration("screenshot retention", cfg.ScreenshotRetention); err != nil {
		return err
	}
	if err := validateDuration("video retention", cfg.VideoRetention); err != nil {
		return err
	}
	if err := validateDuration("transcript retention", cfg.TranscriptRetention); err != nil {
		return err
	}
	if err := validateDuration("structured retention", cfg.StructuredRetention); err != nil {
		return err
	}
	if err := validateDuration("purge interval", cfg.PurgeInterval); err != nil {
		return err
	}
	if cfg.PurgeScheduleEnabled && cfg.PurgeInterval <= 0 {
		return fmt.Errorf("purge interval must be positive when purge schedule is enabled")
	}

	mode := strings.TrimSpace(cfg.VideoMode)
	if mode == "" {
		return fmt.Errorf("video mode cannot be empty")
	}
	if _, ok := allowedVideoModes[mode]; !ok {
		return fmt.Errorf("unsupported video mode: %s", mode)
	}

	transcriptionMode := strings.TrimSpace(cfg.TranscriptionMode)
	if transcriptionMode == "" {
		return fmt.Errorf("transcription mode cannot be empty")
	}
	if _, ok := allowedTranscriptionModes[transcriptionMode]; !ok {
		return fmt.Errorf("unsupported transcription mode: %s", transcriptionMode)
	}

	for _, cap := range cfg.ControlCapabilities {
		cap = strings.TrimSpace(cap)
		if cap == "" {
			return fmt.Errorf("control capability cannot be empty")
		}
		if _, ok := allowedCapabilities[cap]; !ok {
			return fmt.Errorf("unsupported control capability: %s", cap)
		}
	}

	for _, item := range cfg.OutboundAllowlist {
		if strings.TrimSpace(item) == "" {
			return fmt.Errorf("allowlist entries cannot be empty")
		}
	}
	for _, item := range cfg.BlockedTargets {
		if strings.TrimSpace(item) == "" {
			return fmt.Errorf("blocklist entries cannot be empty")
		}
	}
	for _, provider := range cfg.AllowedSubmitProviders {
		if strings.TrimSpace(provider) == "" {
			return fmt.Errorf("allowed submit providers cannot contain empty entries")
		}
	}
	if strings.TrimSpace(cfg.VersionPin) != "" && strings.TrimSpace(cfg.BuildID) != "" && strings.TrimSpace(cfg.VersionPin) != strings.TrimSpace(cfg.BuildID) {
		return fmt.Errorf("version pin must match build id when both are set")
	}

	return nil
}

func validateDuration(name string, d time.Duration) error {
	if d < 0 {
		return fmt.Errorf("%s cannot be negative", name)
	}
	return nil
}

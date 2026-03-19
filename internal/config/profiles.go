package config

import "time"

type PublicConfig struct {
	LocalProfile           string   `json:"local_profile,omitempty"`
	EnvironmentName        string   `json:"environment_name,omitempty"`
	BuildID                string   `json:"build_id,omitempty"`
	VersionPin             string   `json:"version_pin,omitempty"`
	ManagedDeploymentID    string   `json:"managed_deployment_id,omitempty"`
	TranscriptionMode      string   `json:"transcription_mode,omitempty"`
	AllowRemoteSTT         bool     `json:"allow_remote_stt"`
	AllowRemoteSubmit      bool     `json:"allow_remote_submit"`
	ControlCapabilities    []string `json:"control_capabilities"`
	ConfigLocked           bool     `json:"config_locked"`
	CaptureSettingsLocked  bool     `json:"capture_settings_locked"`
	WindowScopedCapture    bool     `json:"window_scoped_capture"`
	BrowserAppFirst        bool     `json:"browser_app_first"`
	UIRuntime              string   `json:"ui_runtime"`
	AutoStartEnabled       bool     `json:"auto_start_enabled"`
	PointerSampleHz        int      `json:"pointer_sample_hz"`
	VideoEnabledDefault    bool     `json:"video_enabled_default"`
	VideoMode              string   `json:"video_mode"`
	AudioRetention         string   `json:"audio_retention"`
	ScreenshotRetention    string   `json:"screenshot_retention"`
	VideoRetention         string   `json:"video_retention"`
	TranscriptRetention    string   `json:"transcript_retention"`
	StructuredRetention    string   `json:"structured_retention"`
	PurgeSchedule          bool     `json:"purge_schedule"`
	PurgeInterval          string   `json:"purge_interval"`
	ArtifactMaxFiles       int      `json:"artifact_max_files"`
	OutboundAllowlist      []string `json:"outbound_allowlist"`
	BlockedTargets         []string `json:"blocked_targets"`
	AllowedSubmitProviders []string `json:"allowed_submit_providers,omitempty"`
	SIEMLogPath            string   `json:"siem_log_path,omitempty"`
}

func ExportPublic(cfg Config) PublicConfig {
	return PublicConfig{
		LocalProfile:           cfg.LocalProfile,
		EnvironmentName:        cfg.EnvironmentName,
		BuildID:                cfg.BuildID,
		VersionPin:             cfg.VersionPin,
		ManagedDeploymentID:    cfg.ManagedDeploymentID,
		TranscriptionMode:      cfg.TranscriptionMode,
		AllowRemoteSTT:         cfg.AllowRemoteSTT,
		AllowRemoteSubmit:      cfg.AllowRemoteSubmission,
		ControlCapabilities:    append([]string(nil), cfg.ControlCapabilities...),
		ConfigLocked:           cfg.ConfigLocked,
		CaptureSettingsLocked:  cfg.CaptureSettingsLocked,
		WindowScopedCapture:    cfg.WindowScopedCapture,
		BrowserAppFirst:        cfg.BrowserAppFirst,
		UIRuntime:              cfg.UIRuntime,
		AutoStartEnabled:       cfg.AutoStartEnabled,
		PointerSampleHz:        cfg.PointerSampleHz,
		VideoEnabledDefault:    cfg.VideoEnabledByDefault,
		VideoMode:              cfg.VideoMode,
		AudioRetention:         cfg.AudioRetention.String(),
		ScreenshotRetention:    cfg.ScreenshotRetention.String(),
		VideoRetention:         cfg.VideoRetention.String(),
		TranscriptRetention:    cfg.TranscriptRetention.String(),
		StructuredRetention:    cfg.StructuredRetention.String(),
		PurgeSchedule:          cfg.PurgeScheduleEnabled,
		PurgeInterval:          cfg.PurgeInterval.String(),
		ArtifactMaxFiles:       cfg.ArtifactMaxFiles,
		OutboundAllowlist:      append([]string(nil), cfg.OutboundAllowlist...),
		BlockedTargets:         append([]string(nil), cfg.BlockedTargets...),
		AllowedSubmitProviders: append([]string(nil), cfg.AllowedSubmitProviders...),
		SIEMLogPath:            cfg.SIEMLogPath,
	}
}

func ApplyPublic(base Config, in PublicConfig) Config {
	if in.LocalProfile != "" {
		base.LocalProfile = in.LocalProfile
	}
	if in.EnvironmentName != "" {
		base.EnvironmentName = in.EnvironmentName
	}
	if in.BuildID != "" {
		base.BuildID = in.BuildID
	}
	if in.VersionPin != "" || base.VersionPin != "" {
		base.VersionPin = in.VersionPin
	}
	if in.ManagedDeploymentID != "" || base.ManagedDeploymentID != "" {
		base.ManagedDeploymentID = in.ManagedDeploymentID
	}
	if in.TranscriptionMode != "" {
		base.TranscriptionMode = in.TranscriptionMode
	}
	base.AllowRemoteSTT = in.AllowRemoteSTT
	base.AllowRemoteSubmission = in.AllowRemoteSubmit
	if in.ControlCapabilities != nil {
		base.ControlCapabilities = append([]string(nil), in.ControlCapabilities...)
	}
	base.ConfigLocked = in.ConfigLocked
	base.CaptureSettingsLocked = in.CaptureSettingsLocked
	base.WindowScopedCapture = in.WindowScopedCapture
	base.BrowserAppFirst = in.BrowserAppFirst
	if in.UIRuntime != "" {
		base.UIRuntime = in.UIRuntime
	}
	base.AutoStartEnabled = in.AutoStartEnabled
	if in.PointerSampleHz > 0 {
		base.PointerSampleHz = in.PointerSampleHz
	}
	base.VideoEnabledByDefault = in.VideoEnabledDefault
	if in.VideoMode != "" {
		base.VideoMode = in.VideoMode
	}
	if d, err := time.ParseDuration(in.AudioRetention); err == nil {
		base.AudioRetention = d
	}
	if d, err := time.ParseDuration(in.ScreenshotRetention); err == nil {
		base.ScreenshotRetention = d
	}
	if d, err := time.ParseDuration(in.VideoRetention); err == nil {
		base.VideoRetention = d
	}
	if d, err := time.ParseDuration(in.TranscriptRetention); err == nil {
		base.TranscriptRetention = d
	}
	if d, err := time.ParseDuration(in.StructuredRetention); err == nil {
		base.StructuredRetention = d
	}
	if d, err := time.ParseDuration(in.PurgeInterval); err == nil {
		base.PurgeInterval = d
	}
	base.PurgeScheduleEnabled = in.PurgeSchedule
	if in.ArtifactMaxFiles > 0 {
		base.ArtifactMaxFiles = in.ArtifactMaxFiles
	}
	if in.OutboundAllowlist != nil {
		base.OutboundAllowlist = append([]string(nil), in.OutboundAllowlist...)
	}
	if in.BlockedTargets != nil {
		base.BlockedTargets = append([]string(nil), in.BlockedTargets...)
	}
	if in.AllowedSubmitProviders != nil {
		base.AllowedSubmitProviders = append([]string(nil), in.AllowedSubmitProviders...)
	}
	if in.SIEMLogPath != "" || base.SIEMLogPath != "" {
		base.SIEMLogPath = in.SIEMLogPath
	}
	return base
}

func Profile(name string) (PublicConfig, bool) {
	profiles := Profiles()
	cfg, ok := profiles[name]
	return cfg, ok
}

func Profiles() map[string]PublicConfig {
	return map[string]PublicConfig{
		"personal_local_dev": {
			LocalProfile:          "personal_local_dev",
			EnvironmentName:       "local-dev",
			TranscriptionMode:     "remote",
			AllowRemoteSTT:        true,
			AllowRemoteSubmit:     true,
			ControlCapabilities:   []string{"capture", "submit", "config", "config_read", "purge", "read", "logs"},
			ConfigLocked:          false,
			CaptureSettingsLocked: false,
			WindowScopedCapture:   true,
			BrowserAppFirst:       true,
			UIRuntime:             "native_tray_plus_local_web_ui",
			AutoStartEnabled:      false,
			PointerSampleHz:       30,
			VideoEnabledDefault:   false,
			VideoMode:             "event_triggered",
			AudioRetention:        "0s",
			ScreenshotRetention:   (14 * 24 * time.Hour).String(),
			VideoRetention:        (7 * 24 * time.Hour).String(),
			TranscriptRetention:   (30 * 24 * time.Hour).String(),
			StructuredRetention:   (30 * 24 * time.Hour).String(),
			PurgeSchedule:         true,
			PurgeInterval:         (30 * time.Minute).String(),
			ArtifactMaxFiles:      2000,
		},
		"enterprise_managed_workstation": {
			LocalProfile:           "enterprise_managed_workstation",
			EnvironmentName:        "enterprise-managed",
			TranscriptionMode:      "remote",
			AllowRemoteSTT:         true,
			AllowRemoteSubmit:      true,
			ControlCapabilities:    []string{"capture", "submit", "config", "config_read", "purge", "read", "logs"},
			ConfigLocked:           true,
			CaptureSettingsLocked:  true,
			ManagedDeploymentID:    "enterprise-managed-default",
			WindowScopedCapture:    true,
			BrowserAppFirst:        true,
			UIRuntime:              "native_tray_plus_local_web_ui",
			AutoStartEnabled:       false,
			PointerSampleHz:        30,
			VideoEnabledDefault:    false,
			VideoMode:              "event_triggered",
			AudioRetention:         "0s",
			ScreenshotRetention:    (7 * 24 * time.Hour).String(),
			VideoRetention:         (3 * 24 * time.Hour).String(),
			TranscriptRetention:    (14 * 24 * time.Hour).String(),
			StructuredRetention:    (14 * 24 * time.Hour).String(),
			PurgeSchedule:          true,
			PurgeInterval:          (15 * time.Minute).String(),
			ArtifactMaxFiles:       1000,
			AllowedSubmitProviders: []string{"codex_cli", "claude_cli", "opencode_cli", "codex", "codex_api", "claude_api"},
			SIEMLogPath:            ".knit/audit.siem.jsonl",
		},
		"high_security_restricted_mode": {
			LocalProfile:           "high_security_restricted_mode",
			EnvironmentName:        "high-security",
			TranscriptionMode:      "local",
			AllowRemoteSTT:         false,
			AllowRemoteSubmit:      false,
			ControlCapabilities:    []string{"capture", "submit", "read", "logs"},
			ConfigLocked:           true,
			CaptureSettingsLocked:  true,
			ManagedDeploymentID:    "high-security-default",
			WindowScopedCapture:    true,
			BrowserAppFirst:        true,
			UIRuntime:              "native_tray_plus_local_web_ui",
			AutoStartEnabled:       false,
			PointerSampleHz:        30,
			VideoEnabledDefault:    false,
			VideoMode:              "event_triggered",
			AudioRetention:         "0s",
			ScreenshotRetention:    (24 * time.Hour).String(),
			VideoRetention:         (12 * time.Hour).String(),
			TranscriptRetention:    (72 * time.Hour).String(),
			StructuredRetention:    (72 * time.Hour).String(),
			PurgeSchedule:          true,
			PurgeInterval:          (10 * time.Minute).String(),
			ArtifactMaxFiles:       300,
			AllowedSubmitProviders: []string{"codex_cli", "claude_cli", "opencode_cli"},
			SIEMLogPath:            ".knit/audit.siem.jsonl",
		},
	}
}

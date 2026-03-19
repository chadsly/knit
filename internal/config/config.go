package config

import "time"

// Config contains defaults aligned to the current v1 decisions.
type Config struct {
	HTTPListenAddr        string
	DataDir               string
	SQLitePath            string
	UserConfigPath        string
	LocalProfile          string
	EnvironmentName       string
	BuildID               string
	VersionPin            string
	ManagedDeploymentID   string
	ControlToken          string
	ControlCapabilities   []string
	ConfigLocked          bool
	CaptureSettingsLocked bool

	WindowScopedCapture bool
	BrowserAppFirst     bool
	UIRuntime           string
	AutoStartEnabled    bool
	PointerSampleHz     int

	TranscriptionMode     string
	TranscriptionProvider string

	VideoEnabledByDefault bool
	VideoMode             string
	AllowRemoteSTT        bool
	AllowRemoteSubmission bool

	AudioRetention       time.Duration
	ScreenshotRetention  time.Duration
	VideoRetention       time.Duration
	TranscriptRetention  time.Duration
	StructuredRetention  time.Duration
	PurgeScheduleEnabled bool
	PurgeInterval        time.Duration
	ArtifactMaxFiles     int

	OutboundAllowlist      []string
	BlockedTargets         []string
	AllowedSubmitProviders []string
	SIEMLogPath            string
}

func Default() Config {
	return Config{
		HTTPListenAddr:      "127.0.0.1:7777",
		DataDir:             "./.knit",
		SQLitePath:          "knit.db",
		LocalProfile:        "local-default",
		EnvironmentName:     "local-dev",
		BuildID:             "",
		VersionPin:          "",
		ManagedDeploymentID: "",
		ControlCapabilities: []string{"capture", "submit", "config", "purge", "read"},

		CaptureSettingsLocked: false,
		WindowScopedCapture:   true,
		BrowserAppFirst:       true,
		UIRuntime:             "native_tray_plus_local_web_ui",
		AutoStartEnabled:      false,
		PointerSampleHz:       30,

		TranscriptionMode:     "faster_whisper",
		TranscriptionProvider: "managed_faster_whisper_stt",

		VideoEnabledByDefault: false,
		VideoMode:             "event_triggered",
		AllowRemoteSTT:        true,
		AllowRemoteSubmission: true,

		AudioRetention:       0,
		ScreenshotRetention:  14 * 24 * time.Hour,
		VideoRetention:       7 * 24 * time.Hour,
		TranscriptRetention:  30 * 24 * time.Hour,
		StructuredRetention:  30 * 24 * time.Hour,
		PurgeScheduleEnabled: true,
		PurgeInterval:        30 * time.Minute,
		ArtifactMaxFiles:     2000,

		OutboundAllowlist:      []string{},
		BlockedTargets:         []string{},
		AllowedSubmitProviders: []string{},
		SIEMLogPath:            "",
	}
}

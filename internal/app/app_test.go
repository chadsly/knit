package app

import "testing"

func TestDefaultConfigUsesManagedFasterWhisperByDefault(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.TranscriptionMode != "faster_whisper" {
		t.Fatalf("expected default transcription mode faster_whisper, got %q", cfg.TranscriptionMode)
	}
	if cfg.TranscriptionProvider != "managed_faster_whisper_stt" {
		t.Fatalf("expected default transcription provider managed_faster_whisper_stt, got %q", cfg.TranscriptionProvider)
	}
}

func TestDefaultConfigReadsEnvironmentControls(t *testing.T) {
	t.Setenv("KNIT_PROFILE", "enterprise_managed_workstation")
	t.Setenv("KNIT_ENVIRONMENT", "staging")
	t.Setenv("KNIT_BUILD_ID", "build-123")
	t.Setenv("KNIT_VERSION_PIN", "build-123")
	t.Setenv("KNIT_MANAGED_DEPLOYMENT_ID", "fleet-a")
	t.Setenv("KNIT_TRANSCRIPTION_MODE", "local")
	t.Setenv("KNIT_ALLOW_REMOTE_STT", "false")
	t.Setenv("KNIT_ALLOW_REMOTE_SUBMISSION", "0")
	t.Setenv("KNIT_AUTO_START", "1")
	t.Setenv("KNIT_CONTROL_CAPABILITIES", "read,submit")
	t.Setenv("KNIT_CAPTURE_SETTINGS_LOCKED", "1")
	t.Setenv("KNIT_POINTER_SAMPLE_HZ", "45")
	t.Setenv("KNIT_ALLOWED_SUBMIT_PROVIDERS", "codex_cli,claude_cli")
	t.Setenv("KNIT_SIEM_JSONL_PATH", "siem/audit.jsonl")

	cfg := DefaultConfig()
	if cfg.LocalProfile != "enterprise_managed_workstation" {
		t.Fatalf("unexpected local profile: %s", cfg.LocalProfile)
	}
	if cfg.EnvironmentName != "staging" {
		t.Fatalf("unexpected environment: %s", cfg.EnvironmentName)
	}
	if cfg.BuildID != "build-123" {
		t.Fatalf("unexpected build id: %s", cfg.BuildID)
	}
	if cfg.VersionPin != "build-123" {
		t.Fatalf("unexpected version pin: %s", cfg.VersionPin)
	}
	if cfg.ManagedDeploymentID != "fleet-a" {
		t.Fatalf("unexpected managed deployment id: %s", cfg.ManagedDeploymentID)
	}
	if cfg.TranscriptionMode != "local" {
		t.Fatalf("unexpected transcription mode: %s", cfg.TranscriptionMode)
	}
	if cfg.AllowRemoteSTT {
		t.Fatalf("expected remote stt disabled")
	}
	if cfg.AllowRemoteSubmission {
		t.Fatalf("expected remote submission disabled")
	}
	if !cfg.AutoStartEnabled {
		t.Fatalf("expected auto-start enabled from env")
	}
	if len(cfg.ControlCapabilities) != 2 || cfg.ControlCapabilities[0] != "read" || cfg.ControlCapabilities[1] != "submit" {
		t.Fatalf("unexpected capabilities: %#v", cfg.ControlCapabilities)
	}
	if !cfg.CaptureSettingsLocked {
		t.Fatalf("expected capture settings locked from env")
	}
	if cfg.PointerSampleHz != 45 {
		t.Fatalf("unexpected pointer sample hz: %d", cfg.PointerSampleHz)
	}
	if len(cfg.AllowedSubmitProviders) != 2 {
		t.Fatalf("unexpected allowed providers: %#v", cfg.AllowedSubmitProviders)
	}
	if cfg.SIEMLogPath != "siem/audit.jsonl" {
		t.Fatalf("unexpected siem log path: %s", cfg.SIEMLogPath)
	}
}

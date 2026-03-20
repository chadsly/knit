package config

import "testing"

func TestProfileApplyAndExport(t *testing.T) {
	base := Default()
	p, ok := Profile("high_security_restricted_mode")
	if !ok {
		t.Fatalf("expected profile")
	}
	updated := ApplyPublic(base, p)
	if !updated.ConfigLocked {
		t.Fatalf("expected config lock enabled for high-security profile")
	}
	if !updated.CaptureSettingsLocked {
		t.Fatalf("expected capture settings lock enabled for high-security profile")
	}
	if updated.AllowRemoteSTT {
		t.Fatalf("expected high-security profile to disable remote stt")
	}
	if updated.AllowRemoteSubmission {
		t.Fatalf("expected high-security profile to disable remote submissions")
	}
	if updated.ArtifactMaxFiles != p.ArtifactMaxFiles {
		t.Fatalf("expected artifact max files %d, got %d", p.ArtifactMaxFiles, updated.ArtifactMaxFiles)
	}
	out := ExportPublic(updated)
	if out.VideoMode == "" {
		t.Fatalf("expected video mode")
	}
	if !out.WindowScopedCapture {
		t.Fatalf("expected window scoped capture true")
	}
	if out.TranscriptionMode != "local" {
		t.Fatalf("expected local transcription mode export, got %q", out.TranscriptionMode)
	}
	if out.AutoStartEnabled {
		t.Fatalf("expected profile export to keep auto-start disabled by default")
	}
	if out.ManagedDeploymentID == "" {
		t.Fatalf("expected managed deployment id to export")
	}
	if len(out.AllowedSubmitProviders) == 0 {
		t.Fatalf("expected submit provider allowlist to export")
	}
}

func TestValidateDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.TranscriptionMode != "faster_whisper" {
		t.Fatalf("expected default transcription mode faster_whisper, got %q", cfg.TranscriptionMode)
	}
	if cfg.TranscriptionProvider != "managed_faster_whisper_stt" {
		t.Fatalf("expected default transcription provider managed_faster_whisper_stt, got %q", cfg.TranscriptionProvider)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestValidateRejectsInvalidCapability(t *testing.T) {
	cfg := Default()
	cfg.ControlCapabilities = []string{"capture", "invalid-capability"}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected invalid capability to fail validation")
	}
}

func TestValidateAllowsEnterpriseCapabilitiesAndVersionPin(t *testing.T) {
	cfg := Default()
	cfg.ControlCapabilities = []string{"read", "logs", "config_read"}
	cfg.BuildID = "build-123"
	cfg.VersionPin = "build-123"
	cfg.AllowedSubmitProviders = []string{"codex_cli", "claude_cli", "claude_api"}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected enterprise config to be valid: %v", err)
	}
}

func TestEnterpriseProfileIncludesClaudeAPIWhenRemoteSubmitAllowed(t *testing.T) {
	p, ok := Profile("enterprise_managed_workstation")
	if !ok {
		t.Fatalf("expected enterprise profile")
	}
	found := false
	for _, provider := range p.AllowedSubmitProviders {
		if provider == "claude_api" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected enterprise profile to allow claude_api")
	}
}

func TestValidateRejectsMismatchedVersionPin(t *testing.T) {
	cfg := Default()
	cfg.BuildID = "build-a"
	cfg.VersionPin = "build-b"
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected mismatched version pin to fail validation")
	}
}

func TestDefaultUsesEmbeddedVersionDefaults(t *testing.T) {
	prevBuildID := EmbeddedBuildID
	prevVersionPin := EmbeddedVersionPin
	EmbeddedBuildID = "0.1.2"
	EmbeddedVersionPin = "0.1.2"
	t.Cleanup(func() {
		EmbeddedBuildID = prevBuildID
		EmbeddedVersionPin = prevVersionPin
	})

	cfg := Default()
	if cfg.BuildID != "0.1.2" {
		t.Fatalf("expected embedded build id default, got %q", cfg.BuildID)
	}
	if cfg.VersionPin != "0.1.2" {
		t.Fatalf("expected embedded version pin default, got %q", cfg.VersionPin)
	}
}

func TestValidateRejectsInvalidTranscriptionMode(t *testing.T) {
	cfg := Default()
	cfg.TranscriptionMode = "unsupported"
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected invalid transcription mode to fail validation")
	}
}

func TestValidateAcceptsLMStudioTranscriptionMode(t *testing.T) {
	cfg := Default()
	cfg.TranscriptionMode = "lmstudio"
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected lmstudio transcription mode to be valid: %v", err)
	}
}

func TestValidateAcceptsManagedFasterWhisperTranscriptionMode(t *testing.T) {
	cfg := Default()
	cfg.TranscriptionMode = "faster_whisper"
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected faster_whisper transcription mode to be valid: %v", err)
	}
}

func TestValidateRejectsInvalidVideoMode(t *testing.T) {
	cfg := Default()
	cfg.VideoMode = "bad-mode"
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected invalid video mode to fail validation")
	}
}

func TestValidateRejectsInvalidPointerSampleRate(t *testing.T) {
	cfg := Default()
	cfg.PointerSampleHz = 0
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected pointer sample rate lower bound validation error")
	}
	cfg.PointerSampleHz = 121
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected pointer sample rate upper bound validation error")
	}
}

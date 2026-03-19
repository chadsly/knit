package operatorstate

import (
	"os"
	"strings"
	"testing"

	"knit/internal/audio"
	"knit/internal/config"
)

func TestApplyRestoresRuntimeAndAudioState(t *testing.T) {
	t.Setenv("KNIT_CODEX_WORKDIR", "")
	t.Setenv("KNIT_DEFAULT_PROVIDER", "")
	t.Setenv("KNIT_CODEX_SANDBOX", "")
	t.Setenv("KNIT_CODEX_APPROVAL_POLICY", "")
	t.Setenv("KNIT_CHECK_UPDATES_ON_STARTUP", "")
	t.Setenv("KNIT_CLAUDE_API_MODEL", "")
	t.Setenv("ANTHROPIC_BASE_URL", "")
	t.Setenv("KNIT_CLAUDE_API_TIMEOUT_SECONDS", "")
	t.Setenv("KNIT_TRANSCRIPTION_MODE", "")
	t.Setenv("KNIT_FASTER_WHISPER_MODEL", "")
	t.Setenv("KNIT_FASTER_WHISPER_DEVICE", "")
	t.Setenv("KNIT_FASTER_WHISPER_COMPUTE_TYPE", "")

	cfg := config.Default()
	controller := audio.NewController()
	state := &State{
		Version: 1,
		System: System{
			AutoStartEnabled:      true,
			CheckUpdatesOnStartup: false,
		},
		RuntimeCodex: RuntimeCodex{
			DefaultProvider:         "codex_cli",
			CodexWorkdir:            "/tmp/repo",
			ClaudeAPIModel:          "claude-test-model",
			AnthropicBaseURL:        "https://api.anthropic.test",
			ClaudeAPITimeoutSeconds: 75,
		},
		RuntimeTranscription: RuntimeTranscription{
			Mode:        "faster_whisper",
			Model:       "small",
			Device:      "cpu",
			ComputeType: "int8",
		},
		Audio: Audio{
			Mode:          audio.ModePushToTalk,
			InputDeviceID: "mic-2",
			Muted:         true,
			Paused:        false,
			LevelMin:      0.05,
			LevelMax:      0.8,
		},
	}

	cfg = Apply(cfg, controller, state)

	if cfg.TranscriptionMode != "faster_whisper" {
		t.Fatalf("expected faster_whisper mode, got %q", cfg.TranscriptionMode)
	}
	if got := os.Getenv("KNIT_CODEX_WORKDIR"); got != "/tmp/repo" {
		t.Fatalf("expected restored codex workdir, got %q", got)
	}
	if got := os.Getenv("KNIT_DEFAULT_PROVIDER"); got != "codex_cli" {
		t.Fatalf("expected restored default provider, got %q", got)
	}
	if got := os.Getenv("KNIT_CODEX_SANDBOX"); got != DefaultLocalCodingSandbox {
		t.Fatalf("expected default sandbox %q, got %q", DefaultLocalCodingSandbox, got)
	}
	if got := os.Getenv("KNIT_CODEX_APPROVAL_POLICY"); got != DefaultLocalCodingApproval {
		t.Fatalf("expected default approval %q, got %q", DefaultLocalCodingApproval, got)
	}
	if got := os.Getenv("KNIT_CLAUDE_API_MODEL"); got != "claude-test-model" {
		t.Fatalf("expected restored Claude API model, got %q", got)
	}
	if got := os.Getenv("ANTHROPIC_BASE_URL"); got != "https://api.anthropic.test" {
		t.Fatalf("expected restored Anthropic base url, got %q", got)
	}
	if got := os.Getenv("KNIT_CLAUDE_API_TIMEOUT_SECONDS"); got != "75" {
		t.Fatalf("expected restored Claude API timeout, got %q", got)
	}
	if got := os.Getenv("KNIT_AUTO_START"); got != "1" {
		t.Fatalf("expected restored auto-start env, got %q", got)
	}
	if got := os.Getenv("KNIT_CHECK_UPDATES_ON_STARTUP"); got != "0" {
		t.Fatalf("expected restored update-check env, got %q", got)
	}
	if !cfg.AutoStartEnabled {
		t.Fatalf("expected restored auto-start config")
	}
	if got := os.Getenv("KNIT_FASTER_WHISPER_MODEL"); got != "small" {
		t.Fatalf("expected restored faster whisper model, got %q", got)
	}
	audioState := controller.State()
	if audioState.Mode != audio.ModePushToTalk || audioState.InputDeviceID != "mic-2" || !audioState.Muted {
		t.Fatalf("unexpected restored audio state: %#v", audioState)
	}
}

func TestCaptureDefaultsUpdateCheckOnStartupAndReadsEnvOverride(t *testing.T) {
	t.Setenv("KNIT_CHECK_UPDATES_ON_STARTUP", "")
	state := Capture(config.Default(), audio.NewController().State())
	if !state.System.CheckUpdatesOnStartup {
		t.Fatalf("expected startup update check to default on")
	}

	t.Setenv("KNIT_CHECK_UPDATES_ON_STARTUP", "0")
	state = Capture(config.Default(), audio.NewController().State())
	if state.System.CheckUpdatesOnStartup {
		t.Fatalf("expected env override to disable startup update check")
	}
}

func TestNormalizeRuntimeCodexDefaultsFillsSandboxAndApproval(t *testing.T) {
	state := NormalizeRuntimeCodexDefaults(RuntimeCodex{})
	if state.CodexSandbox != DefaultLocalCodingSandbox {
		t.Fatalf("expected sandbox default %q, got %q", DefaultLocalCodingSandbox, state.CodexSandbox)
	}
	if state.CodexApproval != DefaultLocalCodingApproval {
		t.Fatalf("expected approval default %q, got %q", DefaultLocalCodingApproval, state.CodexApproval)
	}
	if state.DefaultProvider != "codex_cli" {
		t.Fatalf("expected default provider codex_cli, got %q", state.DefaultProvider)
	}
	if !strings.Contains(state.CLIAdapterCmd, "knit-codex-cli-adapter.sh") {
		t.Fatalf("expected bundled codex cli adapter command, got %q", state.CLIAdapterCmd)
	}
	if !strings.Contains(state.ClaudeCLIAdapterCmd, "knit-claude-cli-adapter.sh") {
		t.Fatalf("expected bundled claude cli adapter command, got %q", state.ClaudeCLIAdapterCmd)
	}
	if !strings.Contains(state.OpenCodeCLIAdapterCmd, "knit-opencode-cli-adapter.sh") {
		t.Fatalf("expected bundled opencode cli adapter command, got %q", state.OpenCodeCLIAdapterCmd)
	}
	if state.DeliveryIntentProfile != DefaultDeliveryIntentProfile {
		t.Fatalf("expected default delivery intent %q, got %q", DefaultDeliveryIntentProfile, state.DeliveryIntentProfile)
	}
	if !strings.Contains(state.ImplementChangesPrompt, "canonical Knit feedback payload JSON") {
		t.Fatalf("expected default implement-changes prompt to be populated")
	}
	if !strings.Contains(state.DraftPlanPrompt, "Produce a concrete implementation plan") {
		t.Fatalf("expected default draft-plan prompt to be populated")
	}
	if !strings.Contains(state.CreateJiraTicketsPrompt, "Jira-ready implementation tickets") {
		t.Fatalf("expected default jira prompt to be populated")
	}
}

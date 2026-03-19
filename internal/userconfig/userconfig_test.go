package userconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"knit/internal/audio"
	"knit/internal/config"
	"knit/internal/operatorstate"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	cfg.UserConfigPath = filepath.Join(cfg.DataDir, "knit.toml")
	state := operatorstate.Capture(cfg, audio.NewController().State())
	state.System.AutoStartEnabled = true
	state.RuntimeCodex.DefaultProvider = "claude_api"
	state.RuntimeCodex.CodexWorkdir = "/tmp/knit-workdir"
	state.RuntimeCodex.DeliveryIntentProfile = "draft_plan"
	state.RuntimeCodex.DraftPlanPrompt = "Plan only.\nDo not edit files."
	state.RuntimeTranscription.Mode = "remote"
	state.Audio.Mode = audio.ModePushToTalk
	state.Audio.Muted = true

	path, err := Save(cfg, state)
	if err != nil {
		t.Fatalf("save user config: %v", err)
	}
	if path != cfg.UserConfigPath {
		t.Fatalf("expected config path %q, got %q", cfg.UserConfigPath, path)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read user config: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "[runtime_codex]") {
		t.Fatalf("expected runtime_codex section in saved config")
	}
	if !strings.Contains(text, `default_provider = "claude_api"`) {
		t.Fatalf("expected saved default provider in config, got:\n%s", text)
	}
	if !strings.Contains(text, `default_template = "draft_plan"`) {
		t.Fatalf("expected saved prompt template in config, got:\n%s", text)
	}

	loaded, err := Load(cfg, audio.NewController().State())
	if err != nil {
		t.Fatalf("load user config: %v", err)
	}
	if !loaded.Exists {
		t.Fatalf("expected loaded config to exist")
	}
	if loaded.State.RuntimeCodex.DefaultProvider != "claude_api" {
		t.Fatalf("expected loaded default provider claude_api, got %q", loaded.State.RuntimeCodex.DefaultProvider)
	}
	if loaded.State.RuntimeCodex.DeliveryIntentProfile != "draft_plan" {
		t.Fatalf("expected loaded delivery intent draft_plan, got %q", loaded.State.RuntimeCodex.DeliveryIntentProfile)
	}
	if got := loaded.State.RuntimeCodex.DraftPlanPrompt; got != "Plan only.\nDo not edit files." {
		t.Fatalf("expected loaded draft prompt, got %q", got)
	}
	if loaded.State.Audio.Mode != audio.ModePushToTalk || !loaded.State.Audio.Muted {
		t.Fatalf("expected loaded audio state to round-trip, got %#v", loaded.State.Audio)
	}
}

func TestRenderIncludesCommentedDefaults(t *testing.T) {
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	state := operatorstate.Capture(cfg, audio.NewController().State())
	_, text := Export(cfg, state)

	required := []string{
		"# Secrets do not belong here.",
		"# allowed: faster_whisper, local, lmstudio, remote",
		"# choose from: codex_cli, claude_cli, opencode_cli, codex_api, claude_api",
		"# allowed: series, parallel",
		"# allowed: always_on, push_to_talk",
		"# allow_remote_submit = true",
		`default_provider = "codex_cli"`,
		`default_template = "implement_changes"`,
		"# implement_changes_text = '''",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected rendered config to include %q\n%s", fragment, text)
		}
	}
}

func TestResolvePathPrefersRootConfigAndFallsBackToDataDir(t *testing.T) {
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	cfg := config.Default()
	cfg.DataDir = filepath.Join(root, ".knit")

	if got := ResolvePath(cfg); got != "knit.toml" {
		t.Fatalf("expected root config path by default, got %q", got)
	}

	dataPath := filepath.Join(cfg.DataDir, "knit.toml")
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.WriteFile(dataPath, []byte("[config]\n"), 0o600); err != nil {
		t.Fatalf("write data-dir config: %v", err)
	}
	if got := ResolvePath(cfg); got != dataPath {
		t.Fatalf("expected existing data-dir config path, got %q", got)
	}

	rootPath := filepath.Join(root, "knit.toml")
	if err := os.WriteFile(rootPath, []byte("[config]\n"), 0o600); err != nil {
		t.Fatalf("write root config: %v", err)
	}
	if got := ResolvePath(cfg); got != "knit.toml" {
		t.Fatalf("expected existing root config path to win, got %q", got)
	}
}

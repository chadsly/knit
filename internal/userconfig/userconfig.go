package userconfig

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"knit/internal/audio"
	"knit/internal/config"
	"knit/internal/operatorstate"
)

const defaultFilename = "knit.toml"

const (
	controlCapabilitiesComment  = "allowed: capture, submit, config, config_read, purge, read, logs, *"
	transcriptionModesComment   = "allowed: faster_whisper, local, lmstudio, remote"
	uiRuntimeComment            = "allowed today: native_tray_plus_local_web_ui"
	videoModesComment           = "allowed: event_triggered, on_demand, continuous"
	submitProvidersComment      = "choose from: codex_cli, claude_cli, opencode_cli, codex_api, claude_api"
	submitExecutionModesComment = "allowed: series, parallel"
	codexSandboxComment         = "allowed: read-only, workspace-write, danger-full-access"
	codexApprovalComment        = "allowed: untrusted, on-request, never"
	audioModesComment           = "allowed: always_on, push_to_talk"
	promptTemplatesComment      = "allowed: implement_changes, draft_plan, create_jira_tickets"
)

type File struct {
	Config               config.PublicConfig         `toml:"config"`
	RuntimeCodex         runtimeCodexSection         `toml:"runtime_codex"`
	RuntimeTranscription runtimeTranscriptionSection `toml:"runtime_transcription"`
	Audio                audioSection                `toml:"audio"`
	Prompts              promptSection               `toml:"prompts"`
}

type runtimeCodexSection struct {
	DefaultProvider         string `toml:"default_provider,omitempty"`
	CLIAdapterCmd           string `toml:"cli_adapter_cmd,omitempty"`
	CLITimeoutSeconds       int    `toml:"cli_timeout_seconds,omitempty"`
	ClaudeCLIAdapterCmd     string `toml:"claude_cli_adapter_cmd,omitempty"`
	ClaudeCLITimeoutSeconds int    `toml:"claude_cli_timeout_seconds,omitempty"`
	OpenCodeCLIAdapterCmd   string `toml:"opencode_cli_adapter_cmd,omitempty"`
	OpenCodeCLITimeoutSecs  int    `toml:"opencode_cli_timeout_seconds,omitempty"`
	SubmitExecMode          string `toml:"submit_execution_mode,omitempty"`
	CodexWorkdir            string `toml:"codex_workdir,omitempty"`
	CodexOutputDir          string `toml:"codex_output_dir,omitempty"`
	CodexSandbox            string `toml:"codex_sandbox,omitempty"`
	CodexApproval           string `toml:"codex_approval_policy,omitempty"`
	CodexProfile            string `toml:"codex_profile,omitempty"`
	CodexModel              string `toml:"codex_model,omitempty"`
	CodexReasoning          string `toml:"codex_reasoning_effort,omitempty"`
	OpenAIBaseURL           string `toml:"openai_base_url,omitempty"`
	CodexAPITimeoutSeconds  int    `toml:"codex_api_timeout_seconds,omitempty"`
	OpenAIOrgID             string `toml:"openai_org_id,omitempty"`
	OpenAIProjectID         string `toml:"openai_project_id,omitempty"`
	ClaudeAPIModel          string `toml:"claude_api_model,omitempty"`
	AnthropicBaseURL        string `toml:"anthropic_base_url,omitempty"`
	ClaudeAPITimeoutSeconds int    `toml:"claude_api_timeout_seconds,omitempty"`
	PostSubmitRebuild       string `toml:"post_submit_rebuild_cmd,omitempty"`
	PostSubmitVerify        string `toml:"post_submit_verify_cmd,omitempty"`
	PostSubmitTimeout       int    `toml:"post_submit_timeout_seconds,omitempty"`
	CodexSkipRepoCheck      bool   `toml:"codex_skip_git_repo_check"`
}

type runtimeTranscriptionSection struct {
	Mode          string `toml:"mode,omitempty"`
	BaseURL       string `toml:"base_url,omitempty"`
	Model         string `toml:"model,omitempty"`
	Device        string `toml:"device,omitempty"`
	ComputeType   string `toml:"compute_type,omitempty"`
	Language      string `toml:"language,omitempty"`
	LocalCommand  string `toml:"local_command,omitempty"`
	TimeoutSecond int    `toml:"timeout_seconds,omitempty"`
}

type audioSection struct {
	Mode          string  `toml:"mode,omitempty"`
	InputDeviceID string  `toml:"input_device_id,omitempty"`
	Muted         bool    `toml:"muted"`
	Paused        bool    `toml:"paused"`
	LevelMin      float64 `toml:"level_min,omitempty"`
	LevelMax      float64 `toml:"level_max,omitempty"`
}

type promptSection struct {
	DefaultTemplate       string `toml:"default_template,omitempty"`
	ImplementChangesText  string `toml:"implement_changes_text,omitempty"`
	DraftPlanText         string `toml:"draft_plan_text,omitempty"`
	CreateJiraTicketsText string `toml:"create_jira_tickets_text,omitempty"`
}

type Loaded struct {
	Path   string
	Config config.Config
	State  operatorstate.State
	Exists bool
}

func ResolvePath(cfg config.Config) string {
	if strings.TrimSpace(cfg.UserConfigPath) != "" {
		return strings.TrimSpace(cfg.UserConfigPath)
	}
	rootPath := defaultFilename
	if _, err := os.Stat(rootPath); err == nil {
		return rootPath
	}
	dataDir := strings.TrimSpace(cfg.DataDir)
	if dataDir == "" {
		dataDir = ".knit"
	}
	dataPath := filepath.Join(dataDir, defaultFilename)
	if _, err := os.Stat(dataPath); err == nil {
		return dataPath
	}
	return rootPath
}

func Load(baseCfg config.Config, audioState audio.State) (Loaded, error) {
	path := ResolvePath(baseCfg)
	loaded := Loaded{
		Path:   path,
		Config: baseCfg,
		State:  defaultState(baseCfg, audioState),
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return loaded, nil
		}
		return loaded, fmt.Errorf("read user config: %w", err)
	}
	var file File
	if _, err := toml.Decode(string(raw), &file); err != nil {
		return loaded, fmt.Errorf("decode user config toml: %w", err)
	}
	loaded.Exists = true
	loaded.Config = config.ApplyPublic(baseCfg, file.Config)
	applyRuntimeCodexSection(&loaded.State.RuntimeCodex, file.RuntimeCodex)
	applyRuntimeTranscriptionSection(&loaded.State.RuntimeTranscription, file.RuntimeTranscription)
	applyAudioSection(&loaded.State.Audio, file.Audio)
	applyPromptSection(&loaded.State.RuntimeCodex, file.Prompts)
	loaded.State.RuntimeCodex = operatorstate.NormalizeRuntimeCodexDefaults(loaded.State.RuntimeCodex)
	return loaded, nil
}

func Save(cfg config.Config, state operatorstate.State) (string, error) {
	path := ResolvePath(cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return path, fmt.Errorf("create user config dir: %w", err)
	}
	state.RuntimeCodex = operatorstate.NormalizeRuntimeCodexDefaults(state.RuntimeCodex)
	text := renderFile(cfg, state)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		return path, fmt.Errorf("write user config: %w", err)
	}
	return path, nil
}

func Export(cfg config.Config, state operatorstate.State) (string, string) {
	return ResolvePath(cfg), renderFile(cfg, state)
}

func defaultState(cfg config.Config, audioState audio.State) operatorstate.State {
	rt := operatorstate.Capture(cfg, audioState)
	rt.RuntimeCodex = operatorstate.NormalizeRuntimeCodexDefaults(rt.RuntimeCodex)
	return rt
}

func applyRuntimeCodexSection(dst *operatorstate.RuntimeCodex, src runtimeCodexSection) {
	if dst == nil {
		return
	}
	dst.DefaultProvider = strings.TrimSpace(src.DefaultProvider)
	dst.CLIAdapterCmd = strings.TrimSpace(src.CLIAdapterCmd)
	dst.CLITimeoutSeconds = src.CLITimeoutSeconds
	dst.ClaudeCLIAdapterCmd = strings.TrimSpace(src.ClaudeCLIAdapterCmd)
	dst.ClaudeCLITimeoutSeconds = src.ClaudeCLITimeoutSeconds
	dst.OpenCodeCLIAdapterCmd = strings.TrimSpace(src.OpenCodeCLIAdapterCmd)
	dst.OpenCodeCLITimeoutSecs = src.OpenCodeCLITimeoutSecs
	dst.SubmitExecMode = strings.TrimSpace(src.SubmitExecMode)
	dst.CodexWorkdir = strings.TrimSpace(src.CodexWorkdir)
	dst.CodexOutputDir = strings.TrimSpace(src.CodexOutputDir)
	dst.CodexSandbox = strings.TrimSpace(src.CodexSandbox)
	dst.CodexApproval = strings.TrimSpace(src.CodexApproval)
	dst.CodexProfile = strings.TrimSpace(src.CodexProfile)
	dst.CodexModel = strings.TrimSpace(src.CodexModel)
	dst.CodexReasoning = strings.TrimSpace(src.CodexReasoning)
	dst.OpenAIBaseURL = strings.TrimSpace(src.OpenAIBaseURL)
	dst.CodexAPITimeoutSeconds = src.CodexAPITimeoutSeconds
	dst.OpenAIOrgID = strings.TrimSpace(src.OpenAIOrgID)
	dst.OpenAIProjectID = strings.TrimSpace(src.OpenAIProjectID)
	dst.ClaudeAPIModel = strings.TrimSpace(src.ClaudeAPIModel)
	dst.AnthropicBaseURL = strings.TrimSpace(src.AnthropicBaseURL)
	dst.ClaudeAPITimeoutSeconds = src.ClaudeAPITimeoutSeconds
	dst.PostSubmitRebuild = strings.TrimSpace(src.PostSubmitRebuild)
	dst.PostSubmitVerify = strings.TrimSpace(src.PostSubmitVerify)
	dst.PostSubmitTimeout = src.PostSubmitTimeout
	dst.CodexSkipRepoCheck = src.CodexSkipRepoCheck
}

func applyRuntimeTranscriptionSection(dst *operatorstate.RuntimeTranscription, src runtimeTranscriptionSection) {
	if dst == nil {
		return
	}
	dst.Mode = strings.TrimSpace(src.Mode)
	dst.BaseURL = strings.TrimSpace(src.BaseURL)
	dst.Model = strings.TrimSpace(src.Model)
	dst.Device = strings.TrimSpace(src.Device)
	dst.ComputeType = strings.TrimSpace(src.ComputeType)
	dst.Language = strings.TrimSpace(src.Language)
	dst.LocalCommand = strings.TrimSpace(src.LocalCommand)
	dst.TimeoutSecond = src.TimeoutSecond
}

func applyAudioSection(dst *operatorstate.Audio, src audioSection) {
	if dst == nil {
		return
	}
	if strings.TrimSpace(src.Mode) != "" {
		dst.Mode = strings.TrimSpace(src.Mode)
	}
	if strings.TrimSpace(src.InputDeviceID) != "" {
		dst.InputDeviceID = strings.TrimSpace(src.InputDeviceID)
	}
	dst.Muted = src.Muted
	dst.Paused = src.Paused
	if src.LevelMin > 0 {
		dst.LevelMin = src.LevelMin
	}
	if src.LevelMax > 0 {
		dst.LevelMax = src.LevelMax
	}
}

func applyPromptSection(dst *operatorstate.RuntimeCodex, src promptSection) {
	if dst == nil {
		return
	}
	dst.DeliveryIntentProfile = strings.TrimSpace(src.DefaultTemplate)
	dst.ImplementChangesPrompt = strings.TrimSpace(src.ImplementChangesText)
	dst.DraftPlanPrompt = strings.TrimSpace(src.DraftPlanText)
	dst.CreateJiraTicketsPrompt = strings.TrimSpace(src.CreateJiraTicketsText)
}

func renderFile(cfg config.Config, state operatorstate.State) string {
	cfgPublic := config.ExportPublic(cfg)
	rc := operatorstate.NormalizeRuntimeCodexDefaults(state.RuntimeCodex)
	rt := state.RuntimeTranscription
	audioState := state.Audio
	var b bytes.Buffer
	b.WriteString("# Knit user configuration\n")
	b.WriteString("# This file is app-managed. Edit it if you want, but Knit may rewrite comments/order on save.\n")
	b.WriteString("# Secrets do not belong here. Use .env or your OS keychain for API keys and tokens.\n\n")

	writeSectionHeader(&b, "config")
	writeString(&b, "local_profile", cfgPublic.LocalProfile, "local-default", "")
	writeString(&b, "environment_name", cfgPublic.EnvironmentName, "local-dev", "")
	writeString(&b, "build_id", cfgPublic.BuildID, "", "optional build identifier shown in runtime state")
	writeString(&b, "version_pin", cfgPublic.VersionPin, "", "optional managed deployment pin")
	writeString(&b, "managed_deployment_id", cfgPublic.ManagedDeploymentID, "", "optional deployment identifier")
	writeString(&b, "transcription_mode", cfgPublic.TranscriptionMode, "faster_whisper", transcriptionModesComment)
	writeBool(&b, "allow_remote_stt", cfgPublic.AllowRemoteSTT, true, "")
	writeBool(&b, "allow_remote_submit", cfgPublic.AllowRemoteSubmit, true, "")
	writeStringArray(&b, "control_capabilities", cfgPublic.ControlCapabilities, []string{"capture", "submit", "config", "purge", "read"}, controlCapabilitiesComment)
	writeBool(&b, "config_locked", cfgPublic.ConfigLocked, false, "")
	writeBool(&b, "capture_settings_locked", cfgPublic.CaptureSettingsLocked, false, "")
	writeBool(&b, "window_scoped_capture", cfgPublic.WindowScopedCapture, true, "")
	writeBool(&b, "browser_app_first", cfgPublic.BrowserAppFirst, true, "")
	writeString(&b, "ui_runtime", cfgPublic.UIRuntime, "native_tray_plus_local_web_ui", uiRuntimeComment)
	writeBool(&b, "auto_start_enabled", cfgPublic.AutoStartEnabled, false, "")
	writeInt(&b, "pointer_sample_hz", cfgPublic.PointerSampleHz, 30, "")
	writeBool(&b, "video_enabled_default", cfgPublic.VideoEnabledDefault, false, "")
	writeString(&b, "video_mode", cfgPublic.VideoMode, "event_triggered", videoModesComment)
	writeString(&b, "audio_retention", cfgPublic.AudioRetention, "0s", "")
	writeString(&b, "screenshot_retention", cfgPublic.ScreenshotRetention, "336h0m0s", "")
	writeString(&b, "video_retention", cfgPublic.VideoRetention, "168h0m0s", "")
	writeString(&b, "transcript_retention", cfgPublic.TranscriptRetention, "720h0m0s", "")
	writeString(&b, "structured_retention", cfgPublic.StructuredRetention, "720h0m0s", "")
	writeBool(&b, "purge_schedule", cfgPublic.PurgeSchedule, true, "")
	writeString(&b, "purge_interval", cfgPublic.PurgeInterval, "30m0s", "")
	writeInt(&b, "artifact_max_files", cfgPublic.ArtifactMaxFiles, 2000, "")
	writeStringArray(&b, "outbound_allowlist", cfgPublic.OutboundAllowlist, nil, "optional explicit outbound allowlist")
	writeStringArray(&b, "blocked_targets", cfgPublic.BlockedTargets, nil, "optional explicit outbound blocklist")
	writeStringArray(&b, "allowed_submit_providers", cfgPublic.AllowedSubmitProviders, nil, submitProvidersComment)
	writeString(&b, "siem_log_path", cfgPublic.SIEMLogPath, "", "optional SIEM JSONL mirror path")

	writeSectionHeader(&b, "runtime_codex")
	writeString(&b, "default_provider", rc.DefaultProvider, "codex_cli", submitProvidersComment)
	writeString(&b, "cli_adapter_cmd", rc.CLIAdapterCmd, "", "")
	writeInt(&b, "cli_timeout_seconds", rc.CLITimeoutSeconds, 600, "")
	writeString(&b, "claude_cli_adapter_cmd", rc.ClaudeCLIAdapterCmd, "", "")
	writeInt(&b, "claude_cli_timeout_seconds", rc.ClaudeCLITimeoutSeconds, 600, "")
	writeString(&b, "opencode_cli_adapter_cmd", rc.OpenCodeCLIAdapterCmd, "", "")
	writeInt(&b, "opencode_cli_timeout_seconds", rc.OpenCodeCLITimeoutSecs, 600, "")
	writeString(&b, "submit_execution_mode", rc.SubmitExecMode, "series", submitExecutionModesComment)
	writeString(&b, "codex_workdir", rc.CodexWorkdir, "", "")
	writeString(&b, "codex_output_dir", rc.CodexOutputDir, "", "")
	writeString(&b, "codex_sandbox", rc.CodexSandbox, operatorstate.DefaultLocalCodingSandbox, codexSandboxComment)
	writeString(&b, "codex_approval_policy", rc.CodexApproval, operatorstate.DefaultLocalCodingApproval, codexApprovalComment)
	writeString(&b, "codex_profile", rc.CodexProfile, "", "")
	writeString(&b, "codex_model", rc.CodexModel, "", "")
	writeString(&b, "codex_reasoning_effort", rc.CodexReasoning, "", "")
	writeString(&b, "openai_base_url", rc.OpenAIBaseURL, "", "")
	writeInt(&b, "codex_api_timeout_seconds", rc.CodexAPITimeoutSeconds, 60, "")
	writeString(&b, "openai_org_id", rc.OpenAIOrgID, "", "")
	writeString(&b, "openai_project_id", rc.OpenAIProjectID, "", "")
	writeString(&b, "anthropic_base_url", rc.AnthropicBaseURL, "", "")
	writeInt(&b, "claude_api_timeout_seconds", rc.ClaudeAPITimeoutSeconds, 60, "")
	writeString(&b, "claude_api_model", rc.ClaudeAPIModel, "", "")
	writeString(&b, "post_submit_rebuild_cmd", rc.PostSubmitRebuild, "", "")
	writeString(&b, "post_submit_verify_cmd", rc.PostSubmitVerify, "", "")
	writeInt(&b, "post_submit_timeout_seconds", rc.PostSubmitTimeout, 600, "")
	writeBool(&b, "codex_skip_git_repo_check", rc.CodexSkipRepoCheck, true, "")

	writeSectionHeader(&b, "runtime_transcription")
	writeString(&b, "mode", rt.Mode, cfg.TranscriptionMode, transcriptionModesComment)
	writeString(&b, "base_url", rt.BaseURL, "", "")
	writeString(&b, "model", rt.Model, "", "")
	writeString(&b, "device", rt.Device, "", "")
	writeString(&b, "compute_type", rt.ComputeType, "", "")
	writeString(&b, "language", rt.Language, "", "")
	writeString(&b, "local_command", rt.LocalCommand, "", "")
	writeInt(&b, "timeout_seconds", rt.TimeoutSecond, 0, "")

	writeSectionHeader(&b, "audio")
	writeString(&b, "mode", audioState.Mode, audio.ModeAlwaysOn, audioModesComment)
	writeString(&b, "input_device_id", audioState.InputDeviceID, "", "")
	writeBool(&b, "muted", audioState.Muted, false, "")
	writeBool(&b, "paused", audioState.Paused, false, "")
	writeFloat(&b, "level_min", audioState.LevelMin, 0.02, "")
	writeFloat(&b, "level_max", audioState.LevelMax, 0.95, "")

	writeSectionHeader(&b, "prompts")
	writeString(&b, "default_template", rc.DeliveryIntentProfile, operatorstate.DefaultDeliveryIntentProfile, promptTemplatesComment)
	writeMultilineString(&b, "implement_changes_text", rc.ImplementChangesPrompt, operatorstate.DefaultPromptImplementChanges(), "")
	writeMultilineString(&b, "draft_plan_text", rc.DraftPlanPrompt, operatorstate.DefaultPromptDraftPlan(), "")
	writeMultilineString(&b, "create_jira_tickets_text", rc.CreateJiraTicketsPrompt, operatorstate.DefaultPromptCreateJiraTickets(), "")

	return b.String()
}

func writeSectionHeader(b *bytes.Buffer, name string) {
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString("[")
	b.WriteString(name)
	b.WriteString("]\n")
}

func writeComment(b *bytes.Buffer, comment string) {
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return
	}
	b.WriteString("# ")
	b.WriteString(comment)
	b.WriteString("\n")
}

func writeString(b *bytes.Buffer, key, value, defaultValue, comment string) {
	writeComment(b, comment)
	if strings.TrimSpace(value) == "" {
		if strings.TrimSpace(defaultValue) == "" {
			fmt.Fprintf(b, "# %s = \"\"\n", key)
		} else {
			fmt.Fprintf(b, "# %s = %q\n", key, defaultValue)
		}
		return
	}
	fmt.Fprintf(b, "%s = %q\n", key, value)
}

func writeMultilineString(b *bytes.Buffer, key, value, defaultValue, comment string) {
	writeComment(b, comment)
	if strings.TrimSpace(value) == "" || value == defaultValue {
		fmt.Fprintf(b, "# %s = '''\n", key)
		for _, line := range strings.Split(defaultValue, "\n") {
			b.WriteString("# ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("# '''\n")
		return
	}
	fmt.Fprintf(b, "%s = '''\n%s\n'''\n", key, value)
}

func writeBool(b *bytes.Buffer, key string, value, defaultValue bool, comment string) {
	writeComment(b, comment)
	if value == defaultValue {
		fmt.Fprintf(b, "# %s = %t\n", key, defaultValue)
		return
	}
	fmt.Fprintf(b, "%s = %t\n", key, value)
}

func writeInt(b *bytes.Buffer, key string, value, defaultValue int, comment string) {
	writeComment(b, comment)
	if value == 0 || value == defaultValue {
		fmt.Fprintf(b, "# %s = %d\n", key, defaultValue)
		return
	}
	fmt.Fprintf(b, "%s = %d\n", key, value)
}

func writeFloat(b *bytes.Buffer, key string, value, defaultValue float64, comment string) {
	writeComment(b, comment)
	if value == 0 || value == defaultValue {
		fmt.Fprintf(b, "# %s = %s\n", key, strconv.FormatFloat(defaultValue, 'f', -1, 64))
		return
	}
	fmt.Fprintf(b, "%s = %s\n", key, strconv.FormatFloat(value, 'f', -1, 64))
}

func writeStringArray(b *bytes.Buffer, key string, values, defaultValues []string, comment string) {
	writeComment(b, comment)
	render := func(items []string) string {
		if len(items) == 0 {
			return "[]"
		}
		quoted := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			quoted = append(quoted, strconv.Quote(item))
		}
		if len(quoted) == 0 {
			return "[]"
		}
		return "[" + strings.Join(quoted, ", ") + "]"
	}
	if len(values) == 0 {
		fmt.Fprintf(b, "# %s = %s\n", key, render(defaultValues))
		return
	}
	fmt.Fprintf(b, "%s = %s\n", key, render(values))
}

package operatorstate

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"knit/internal/agents"
	"knit/internal/audio"
	"knit/internal/config"
	"knit/internal/transcription"
)

type State struct {
	Version              int                  `json:"version"`
	System               System               `json:"system"`
	RuntimeCodex         RuntimeCodex         `json:"runtime_codex"`
	RuntimeTranscription RuntimeTranscription `json:"runtime_transcription"`
	Audio                Audio                `json:"audio"`
	Extensions           ExtensionState       `json:"extensions,omitempty"`
}

type System struct {
	AutoStartEnabled      bool `json:"auto_start_enabled"`
	CheckUpdatesOnStartup bool `json:"check_updates_on_startup"`
}

type RuntimeCodex struct {
	DefaultProvider         string `json:"default_provider,omitempty"`
	CLIAdapterCmd           string `json:"cli_adapter_cmd,omitempty"`
	CLITimeoutSeconds       int    `json:"cli_timeout_seconds,omitempty"`
	ClaudeCLIAdapterCmd     string `json:"claude_cli_adapter_cmd,omitempty"`
	ClaudeCLITimeoutSeconds int    `json:"claude_cli_timeout_seconds,omitempty"`
	OpenCodeCLIAdapterCmd   string `json:"opencode_cli_adapter_cmd,omitempty"`
	OpenCodeCLITimeoutSecs  int    `json:"opencode_cli_timeout_seconds,omitempty"`
	SubmitExecMode          string `json:"submit_execution_mode,omitempty"`
	CodexWorkdir            string `json:"codex_workdir,omitempty"`
	CodexOutputDir          string `json:"codex_output_dir,omitempty"`
	CodexSandbox            string `json:"codex_sandbox,omitempty"`
	CodexApproval           string `json:"codex_approval_policy,omitempty"`
	CodexProfile            string `json:"codex_profile,omitempty"`
	CodexModel              string `json:"codex_model,omitempty"`
	CodexReasoning          string `json:"codex_reasoning_effort,omitempty"`
	OpenAIBaseURL           string `json:"openai_base_url,omitempty"`
	CodexAPITimeoutSeconds  int    `json:"codex_api_timeout_seconds,omitempty"`
	OpenAIOrgID             string `json:"openai_org_id,omitempty"`
	OpenAIProjectID         string `json:"openai_project_id,omitempty"`
	ClaudeAPIModel          string `json:"claude_api_model,omitempty"`
	AnthropicBaseURL        string `json:"anthropic_base_url,omitempty"`
	ClaudeAPITimeoutSeconds int    `json:"claude_api_timeout_seconds,omitempty"`
	PostSubmitRebuild       string `json:"post_submit_rebuild_cmd,omitempty"`
	PostSubmitVerify        string `json:"post_submit_verify_cmd,omitempty"`
	PostSubmitTimeout       int    `json:"post_submit_timeout_seconds,omitempty"`
	CodexSkipRepoCheck      bool   `json:"codex_skip_git_repo_check"`
	DeliveryIntentProfile   string `json:"delivery_intent_profile,omitempty"`
	ImplementChangesPrompt  string `json:"implement_changes_prompt,omitempty"`
	DraftPlanPrompt         string `json:"draft_plan_prompt,omitempty"`
	CreateJiraTicketsPrompt string `json:"create_jira_tickets_prompt,omitempty"`
}

type RuntimeTranscription struct {
	Mode          string `json:"mode,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
	Model         string `json:"model,omitempty"`
	Device        string `json:"device,omitempty"`
	ComputeType   string `json:"compute_type,omitempty"`
	Language      string `json:"language,omitempty"`
	LocalCommand  string `json:"local_command,omitempty"`
	TimeoutSecond int    `json:"timeout_seconds,omitempty"`
}

type Audio struct {
	Mode          string  `json:"mode,omitempty"`
	InputDeviceID string  `json:"input_device_id,omitempty"`
	Muted         bool    `json:"muted"`
	Paused        bool    `json:"paused"`
	LevelMin      float64 `json:"level_min,omitempty"`
	LevelMax      float64 `json:"level_max,omitempty"`
}

type ExtensionState struct {
	Pairings []ExtensionPairing `json:"pairings,omitempty"`
}

type ExtensionPairing struct {
	ID           string     `json:"id"`
	Name         string     `json:"name,omitempty"`
	Browser      string     `json:"browser,omitempty"`
	Platform     string     `json:"platform,omitempty"`
	Capabilities []string   `json:"capabilities,omitempty"`
	TokenHash    string     `json:"token_hash,omitempty"`
	CreatedAt    time.Time  `json:"created_at,omitempty"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}

const (
	DefaultLocalCodingSandbox    = "workspace-write"
	DefaultLocalCodingApproval   = "never"
	DefaultDeliveryIntentProfile = agents.IntentImplementChanges
	DefaultCheckUpdatesOnStartup = true
)

func NormalizeRuntimeCodexDefaults(state RuntimeCodex) RuntimeCodex {
	if strings.TrimSpace(state.CodexSandbox) == "" {
		state.CodexSandbox = DefaultLocalCodingSandbox
	}
	if strings.TrimSpace(state.CodexApproval) == "" {
		state.CodexApproval = DefaultLocalCodingApproval
	}
	if strings.TrimSpace(state.DefaultProvider) == "" {
		state.DefaultProvider = "codex_cli"
	}
	if strings.TrimSpace(state.CLIAdapterCmd) == "" {
		state.CLIAdapterCmd = defaultBundledCLIAdapterCommand("knit-codex-cli-adapter.sh")
	}
	if strings.TrimSpace(state.ClaudeCLIAdapterCmd) == "" {
		state.ClaudeCLIAdapterCmd = defaultBundledCLIAdapterCommand("knit-claude-cli-adapter.sh")
	}
	if strings.TrimSpace(state.OpenCodeCLIAdapterCmd) == "" {
		state.OpenCodeCLIAdapterCmd = defaultBundledCLIAdapterCommand("knit-opencode-cli-adapter.sh")
	}
	switch strings.TrimSpace(state.DeliveryIntentProfile) {
	case agents.IntentCreateJira:
	default:
		state.DeliveryIntentProfile = DefaultDeliveryIntentProfile
	}
	if strings.TrimSpace(state.ImplementChangesPrompt) == "" {
		state.ImplementChangesPrompt = DefaultPromptImplementChanges()
	}
	if strings.TrimSpace(state.CreateJiraTicketsPrompt) == "" {
		state.CreateJiraTicketsPrompt = DefaultPromptCreateJiraTickets()
	}
	return state
}

func DefaultPromptImplementChanges() string {
	return agents.DefaultInstructionTemplate(agents.IntentImplementChanges)
}

func DefaultPromptCreateJiraTickets() string {
	return agents.DefaultInstructionTemplate(agents.IntentCreateJira)
}

func defaultBundledCLIAdapterCommand(scriptName string) string {
	scriptName = strings.TrimSpace(scriptName)
	if scriptName == "" {
		return ""
	}
	for _, candidate := range bundledCLIAdapterCandidates(scriptName) {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		return shellQuote(candidate)
	}
	return ""
}

func bundledCLIAdapterCandidates(scriptName string) []string {
	roots := []string{}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		roots = append(roots, cwd)
	}
	if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
		roots = append(roots, filepath.Dir(exe))
	}
	seen := map[string]struct{}{}
	candidates := make([]string, 0, 8)
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		for {
			candidate := filepath.Clean(filepath.Join(root, "scripts", scriptName))
			if _, ok := seen[candidate]; !ok {
				seen[candidate] = struct{}{}
				candidates = append(candidates, candidate)
			}
			parent := filepath.Dir(root)
			if parent == root {
				break
			}
			root = parent
		}
	}
	return candidates
}

func shellQuote(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return "'" + strings.ReplaceAll(raw, "'", `'"'"'`) + "'"
}

func Capture(cfg config.Config, audioState audio.State) State {
	return State{
		Version: 1,
		System: System{
			AutoStartEnabled:      readEnvBool("KNIT_AUTO_START"),
			CheckUpdatesOnStartup: readEnvBoolDefault("KNIT_CHECK_UPDATES_ON_STARTUP", DefaultCheckUpdatesOnStartup),
		},
		RuntimeCodex: NormalizeRuntimeCodexDefaults(RuntimeCodex{
			DefaultProvider:         strings.TrimSpace(os.Getenv("KNIT_DEFAULT_PROVIDER")),
			CLIAdapterCmd:           strings.TrimSpace(os.Getenv("KNIT_CLI_ADAPTER_CMD")),
			CLITimeoutSeconds:       readEnvInt("KNIT_CLI_TIMEOUT_SECONDS"),
			ClaudeCLIAdapterCmd:     strings.TrimSpace(os.Getenv("KNIT_CLAUDE_CLI_ADAPTER_CMD")),
			ClaudeCLITimeoutSeconds: readEnvInt("KNIT_CLAUDE_CLI_TIMEOUT_SECONDS"),
			OpenCodeCLIAdapterCmd:   strings.TrimSpace(os.Getenv("KNIT_OPENCODE_CLI_ADAPTER_CMD")),
			OpenCodeCLITimeoutSecs:  readEnvInt("KNIT_OPENCODE_CLI_TIMEOUT_SECONDS"),
			SubmitExecMode:          strings.TrimSpace(os.Getenv("KNIT_SUBMIT_EXECUTION_MODE")),
			CodexWorkdir:            strings.TrimSpace(os.Getenv("KNIT_CODEX_WORKDIR")),
			CodexOutputDir:          strings.TrimSpace(os.Getenv("KNIT_CODEX_OUTPUT_DIR")),
			CodexSandbox:            strings.TrimSpace(os.Getenv("KNIT_CODEX_SANDBOX")),
			CodexApproval:           strings.TrimSpace(os.Getenv("KNIT_CODEX_APPROVAL_POLICY")),
			CodexProfile:            strings.TrimSpace(os.Getenv("KNIT_CODEX_PROFILE")),
			CodexModel:              firstNonEmpty(strings.TrimSpace(os.Getenv("KNIT_CODEX_MODEL")), strings.TrimSpace(os.Getenv("CODEX_MODEL"))),
			CodexReasoning:          strings.TrimSpace(os.Getenv("KNIT_CODEX_REASONING_EFFORT")),
			OpenAIBaseURL:           strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
			CodexAPITimeoutSeconds:  readEnvInt("CODEX_TIMEOUT_SECONDS"),
			OpenAIOrgID:             firstNonEmpty(strings.TrimSpace(os.Getenv("OPENAI_ORG_ID")), strings.TrimSpace(os.Getenv("OPENAI_ORGANIZATION"))),
			OpenAIProjectID:         strings.TrimSpace(os.Getenv("OPENAI_PROJECT_ID")),
			ClaudeAPIModel:          strings.TrimSpace(os.Getenv("KNIT_CLAUDE_API_MODEL")),
			AnthropicBaseURL:        strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL")),
			ClaudeAPITimeoutSeconds: readEnvInt("KNIT_CLAUDE_API_TIMEOUT_SECONDS"),
			PostSubmitRebuild:       strings.TrimSpace(os.Getenv("KNIT_POST_SUBMIT_REBUILD_CMD")),
			PostSubmitVerify:        strings.TrimSpace(os.Getenv("KNIT_POST_SUBMIT_VERIFY_CMD")),
			PostSubmitTimeout:       readEnvInt("KNIT_POST_SUBMIT_TIMEOUT_SECONDS"),
			CodexSkipRepoCheck:      readEnvBool("KNIT_CODEX_SKIP_GIT_REPO_CHECK"),
		}),
		RuntimeTranscription: RuntimeTranscription{
			Mode:          strings.TrimSpace(cfg.TranscriptionMode),
			BaseURL:       captureTranscriptionBaseURL(cfg),
			Model:         captureTranscriptionModel(cfg),
			Device:        strings.TrimSpace(os.Getenv("KNIT_FASTER_WHISPER_DEVICE")),
			ComputeType:   strings.TrimSpace(os.Getenv("KNIT_FASTER_WHISPER_COMPUTE_TYPE")),
			Language:      strings.TrimSpace(os.Getenv("KNIT_FASTER_WHISPER_LANGUAGE")),
			LocalCommand:  strings.TrimSpace(os.Getenv("KNIT_LOCAL_STT_CMD")),
			TimeoutSecond: captureTranscriptionTimeout(cfg),
		},
		Audio: Audio{
			Mode:          audioState.Mode,
			InputDeviceID: audioState.InputDeviceID,
			Muted:         audioState.Muted,
			Paused:        audioState.Paused,
			LevelMin:      audioState.LevelMin,
			LevelMax:      audioState.LevelMax,
		},
	}
}

func Apply(cfg config.Config, controller *audio.Controller, state *State) config.Config {
	if state == nil {
		return cfg
	}
	state.RuntimeCodex = NormalizeRuntimeCodexDefaults(state.RuntimeCodex)
	cfg.AutoStartEnabled = state.System.AutoStartEnabled
	applyEnv("KNIT_AUTO_START", boolToEnv(state.System.AutoStartEnabled))
	applyEnv("KNIT_CHECK_UPDATES_ON_STARTUP", boolToEnv(state.System.CheckUpdatesOnStartup))
	applyRuntimeCodex(state.RuntimeCodex)
	cfg.TranscriptionMode = strings.TrimSpace(firstNonEmpty(state.RuntimeTranscription.Mode, cfg.TranscriptionMode))
	applyRuntimeTranscription(cfg.TranscriptionMode, state.RuntimeTranscription)
	if controller != nil {
		muted := state.Audio.Muted
		paused := state.Audio.Paused
		controller.Configure(audio.Config{
			Mode:          state.Audio.Mode,
			InputDeviceID: state.Audio.InputDeviceID,
			Muted:         &muted,
			Paused:        &paused,
			LevelMin:      state.Audio.LevelMin,
			LevelMax:      state.Audio.LevelMax,
		})
	}
	cfg.TranscriptionProvider = transcription.NewProviderFromEnv(cfg.TranscriptionMode).Name()
	return cfg
}

func applyRuntimeCodex(state RuntimeCodex) {
	applyEnv("KNIT_DEFAULT_PROVIDER", state.DefaultProvider)
	applyEnv("KNIT_CLI_ADAPTER_CMD", state.CLIAdapterCmd)
	applyEnvInt("KNIT_CLI_TIMEOUT_SECONDS", state.CLITimeoutSeconds)
	applyEnv("KNIT_CLAUDE_CLI_ADAPTER_CMD", state.ClaudeCLIAdapterCmd)
	applyEnvInt("KNIT_CLAUDE_CLI_TIMEOUT_SECONDS", state.ClaudeCLITimeoutSeconds)
	applyEnv("KNIT_OPENCODE_CLI_ADAPTER_CMD", state.OpenCodeCLIAdapterCmd)
	applyEnvInt("KNIT_OPENCODE_CLI_TIMEOUT_SECONDS", state.OpenCodeCLITimeoutSecs)
	applyEnv("KNIT_SUBMIT_EXECUTION_MODE", state.SubmitExecMode)
	applyEnv("KNIT_CODEX_WORKDIR", state.CodexWorkdir)
	applyEnv("KNIT_CODEX_OUTPUT_DIR", state.CodexOutputDir)
	applyEnv("KNIT_CODEX_SANDBOX", state.CodexSandbox)
	applyEnv("KNIT_CODEX_APPROVAL_POLICY", state.CodexApproval)
	applyEnv("KNIT_CODEX_PROFILE", state.CodexProfile)
	applyEnv("KNIT_CODEX_MODEL", state.CodexModel)
	applyEnv("CODEX_MODEL", state.CodexModel)
	applyEnv("KNIT_CODEX_REASONING_EFFORT", state.CodexReasoning)
	applyEnv("OPENAI_BASE_URL", state.OpenAIBaseURL)
	applyEnvInt("CODEX_TIMEOUT_SECONDS", state.CodexAPITimeoutSeconds)
	applyEnv("OPENAI_ORG_ID", state.OpenAIOrgID)
	applyEnv("OPENAI_ORGANIZATION", state.OpenAIOrgID)
	applyEnv("OPENAI_PROJECT_ID", state.OpenAIProjectID)
	applyEnv("KNIT_CLAUDE_API_MODEL", state.ClaudeAPIModel)
	applyEnv("ANTHROPIC_BASE_URL", state.AnthropicBaseURL)
	applyEnvInt("KNIT_CLAUDE_API_TIMEOUT_SECONDS", state.ClaudeAPITimeoutSeconds)
	applyEnv("KNIT_POST_SUBMIT_REBUILD_CMD", state.PostSubmitRebuild)
	applyEnv("KNIT_POST_SUBMIT_VERIFY_CMD", state.PostSubmitVerify)
	applyEnvInt("KNIT_POST_SUBMIT_TIMEOUT_SECONDS", state.PostSubmitTimeout)
	if state.CodexSkipRepoCheck {
		applyEnv("KNIT_CODEX_SKIP_GIT_REPO_CHECK", "1")
	} else {
		applyEnv("KNIT_CODEX_SKIP_GIT_REPO_CHECK", "0")
	}
}

func applyRuntimeTranscription(mode string, state RuntimeTranscription) {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = "faster_whisper"
	}
	applyEnv("KNIT_TRANSCRIPTION_MODE", mode)
	switch mode {
	case "remote":
		applyEnv("OPENAI_BASE_URL", state.BaseURL)
		applyEnv("OPENAI_STT_MODEL", state.Model)
	case "lmstudio":
		applyEnv("KNIT_LMSTUDIO_BASE_URL", state.BaseURL)
		applyEnv("KNIT_LMSTUDIO_STT_MODEL", state.Model)
		applyEnvInt("KNIT_LMSTUDIO_STT_TIMEOUT_SECONDS", state.TimeoutSecond)
	case "local":
		applyEnv("KNIT_LOCAL_STT_CMD", state.LocalCommand)
		applyEnvInt("KNIT_LOCAL_STT_TIMEOUT_SECONDS", state.TimeoutSecond)
	case "faster_whisper":
		applyEnv("KNIT_FASTER_WHISPER_MODEL", transcription.NormalizeFasterWhisperModel(state.Model))
		applyEnv("KNIT_FASTER_WHISPER_DEVICE", state.Device)
		applyEnv("KNIT_FASTER_WHISPER_COMPUTE_TYPE", state.ComputeType)
		applyEnv("KNIT_FASTER_WHISPER_LANGUAGE", state.Language)
		applyEnvInt("KNIT_FASTER_WHISPER_TIMEOUT_SECONDS", state.TimeoutSecond)
	}
}

func captureTranscriptionBaseURL(cfg config.Config) string {
	switch strings.ToLower(strings.TrimSpace(cfg.TranscriptionMode)) {
	case "remote":
		return strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	case "lmstudio":
		return firstNonEmpty(strings.TrimSpace(os.Getenv("KNIT_LMSTUDIO_BASE_URL")), strings.TrimSpace(os.Getenv("LMSTUDIO_BASE_URL")))
	default:
		return ""
	}
}

func captureTranscriptionModel(cfg config.Config) string {
	switch strings.ToLower(strings.TrimSpace(cfg.TranscriptionMode)) {
	case "remote":
		return firstNonEmpty(strings.TrimSpace(os.Getenv("OPENAI_STT_MODEL")), "gpt-4o-mini-transcribe")
	case "lmstudio":
		return firstNonEmpty(strings.TrimSpace(os.Getenv("KNIT_LMSTUDIO_STT_MODEL")), strings.TrimSpace(os.Getenv("LMSTUDIO_STT_MODEL")), "whisper-large-v3-turbo")
	case "faster_whisper":
		return transcription.NormalizeFasterWhisperModel(os.Getenv("KNIT_FASTER_WHISPER_MODEL"))
	default:
		return ""
	}
}

func captureTranscriptionTimeout(cfg config.Config) int {
	switch strings.ToLower(strings.TrimSpace(cfg.TranscriptionMode)) {
	case "lmstudio":
		return readEnvInt("KNIT_LMSTUDIO_STT_TIMEOUT_SECONDS")
	case "local":
		return readEnvInt("KNIT_LOCAL_STT_TIMEOUT_SECONDS")
	case "faster_whisper":
		return readEnvInt("KNIT_FASTER_WHISPER_TIMEOUT_SECONDS")
	default:
		return 0
	}
}

func applyEnv(key, value string) {
	if strings.TrimSpace(value) == "" {
		_ = os.Unsetenv(key)
		return
	}
	_ = os.Setenv(key, strings.TrimSpace(value))
}

func boolToEnv(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func applyEnvInt(key string, value int) {
	if value <= 0 {
		_ = os.Unsetenv(key)
		return
	}
	_ = os.Setenv(key, strconv.Itoa(value))
}

func readEnvInt(key string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	return v
}

func readEnvBool(key string) bool {
	return readEnvBoolDefault(key, false)
}

func readEnvBoolDefault(key string, fallback bool) bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

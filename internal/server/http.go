package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"knit/internal/agents"
	"knit/internal/audio"
	"knit/internal/audit"
	"knit/internal/companion"
	"knit/internal/config"
	"knit/internal/operatorstate"
	"knit/internal/platform"
	"knit/internal/privileged"
	"knit/internal/redaction"
	"knit/internal/security"
	"knit/internal/session"
	"knit/internal/storage"
	"knit/internal/transcription"
	"knit/internal/userconfig"
)

const (
	maxTranscriptionLocalCommandLen = 2048
	maxTranscriptionTimeoutSeconds  = 600
)

var transcriptionLanguagePattern = regexp.MustCompile(`^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8}){0,2}$`)

type Server struct {
	cfgMu        sync.RWMutex
	cfg          config.Config
	runtimeMu    sync.RWMutex
	runtime      operatorstate.State
	nonceMu      sync.Mutex
	nonces       map[string]time.Time
	pairMu       sync.Mutex
	pendingPairs map[string]pendingExtensionPairing
	sttMu        sync.RWMutex
	submitMu     sync.Mutex
	submitQ      int
	submitRun    int
	submitSeq    int64
	// submit workers and execution state
	submitSeriesCh      chan submitJob
	submitPending       []submitJob
	submitRunning       map[string]submitJob
	submitCancel        map[string]context.CancelFunc
	submitCanceled      map[string]string
	submitAttempts      []submitAttempt
	submitRecoveryNotes []string
	parallelPending     int
	parallelHasSuccess  bool
	parallelPostRunning bool
	submitQueuePath     string
	sessions            *session.Service
	privilegedCapture   *privileged.CaptureBroker
	audit               *audit.Logger
	agents              *agents.Registry
	store               storage.Store
	artifacts           *storage.ArtifactStore
	autoStart           *platform.AutoStartManager
	latency             *latencyBook
	stt                 transcription.Provider
	postSubmitRunner    func() *postSubmitResult
	updateHTTPClient    *http.Client
	updateReleaseAPIURL string
	httpSrv             *http.Server
}

type startSessionRequest struct {
	TargetWindow string `json:"target_window"`
	TargetURL    string `json:"target_url"`
	ReviewMode   string `json:"review_mode,omitempty"`
}

const defaultSessionTargetWindow = "Browser Review"

type feedbackRequest struct {
	RawTranscript string `json:"raw_transcript"`
	Normalized    string `json:"normalized"`
	PointerX      int    `json:"pointer_x"`
	PointerY      int    `json:"pointer_y"`
	Window        string `json:"window"`
	ScreenshotRef string `json:"screenshot_ref"`
	VideoClipRef  string `json:"video_clip_ref"`
	ReviewMode    string `json:"review_mode,omitempty"`
	ExperimentID  string `json:"experiment_id,omitempty"`
	Variant       string `json:"variant,omitempty"`
	LaserMode     bool   `json:"laser_mode,omitempty"`
}

type reviewNoteRequest struct {
	Author string `json:"author"`
	Note   string `json:"note"`
}

type reviewModeRequest struct {
	Mode string `json:"mode"`
}

type audioConfigRequest struct {
	Mode          string  `json:"mode"`
	InputDeviceID string  `json:"input_device_id"`
	Muted         *bool   `json:"muted,omitempty"`
	Paused        *bool   `json:"paused,omitempty"`
	LevelMin      float64 `json:"level_min,omitempty"`
	LevelMax      float64 `json:"level_max,omitempty"`
}

type audioDevicesRequest struct {
	Devices []audio.Device `json:"devices"`
}

type audioLevelRequest struct {
	Level float64 `json:"level"`
}

type captureSourceRequest struct {
	Source string `json:"source"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type approveRequest struct {
	Summary string `json:"summary"`
}

type cancelSubmitAttemptRequest struct {
	AttemptID string `json:"attempt_id"`
}

type rerunSubmitAttemptRequest struct {
	AttemptID string `json:"attempt_id"`
}

func canonicalPackageHasInput(pkg *session.CanonicalPackage) bool {
	if pkg == nil {
		return false
	}
	return len(pkg.ChangeRequests) > 0 || len(pkg.Artifacts) > 0
}

type submitRequest struct {
	Provider              string   `json:"provider"`
	IntentProfile         string   `json:"intent_profile,omitempty"`
	InstructionText       string   `json:"instruction_text,omitempty"`
	CustomInstructions    string   `json:"custom_instructions,omitempty"`
	AllowLargeInlineMedia bool     `json:"allow_large_inline_media,omitempty"`
	RedactReplayValues    bool     `json:"redact_replay_values,omitempty"`
	OmitVideoClips        bool     `json:"omit_video_clips,omitempty"`
	OmitVideoEventIDs     []string `json:"omit_video_event_ids,omitempty"`
}

type runtimeTranscriptionRequest struct {
	Mode          string  `json:"mode"`
	BaseURL       *string `json:"base_url,omitempty"`
	Model         *string `json:"model,omitempty"`
	Device        *string `json:"device,omitempty"`
	ComputeType   *string `json:"compute_type,omitempty"`
	Language      *string `json:"language,omitempty"`
	LocalCommand  *string `json:"local_command,omitempty"`
	TimeoutSecond *int    `json:"timeout_seconds,omitempty"`
}

type payloadPreviewRequest struct {
	Provider              string   `json:"provider"`
	IntentProfile         string   `json:"intent_profile,omitempty"`
	InstructionText       string   `json:"instruction_text,omitempty"`
	CustomInstructions    string   `json:"custom_instructions,omitempty"`
	AllowLargeInlineMedia bool     `json:"allow_large_inline_media,omitempty"`
	RedactReplayValues    bool     `json:"redact_replay_values,omitempty"`
	OmitVideoClips        bool     `json:"omit_video_clips,omitempty"`
	OmitVideoEventIDs     []string `json:"omit_video_event_ids,omitempty"`
}

type feedbackUpdateTextRequest struct {
	EventID string `json:"event_id"`
	Text    string `json:"text"`
}

type feedbackDeleteRequest struct {
	EventID string `json:"event_id"`
}

type replaySettingsRequest struct {
	CaptureInputValues *bool `json:"capture_input_values,omitempty"`
}

type runtimeCodexRequest struct {
	DefaultProvider         string `json:"default_provider,omitempty"`
	CLIAdapterCmd           string `json:"cli_adapter_cmd"`
	CLITimeoutSeconds       *int   `json:"cli_timeout_seconds,omitempty"`
	ClaudeCLIAdapterCmd     string `json:"claude_cli_adapter_cmd,omitempty"`
	ClaudeCLITimeoutSeconds *int   `json:"claude_cli_timeout_seconds,omitempty"`
	OpenCodeCLIAdapterCmd   string `json:"opencode_cli_adapter_cmd,omitempty"`
	OpenCodeCLITimeoutSecs  *int   `json:"opencode_cli_timeout_seconds,omitempty"`
	SubmitExecMode          string `json:"submit_execution_mode,omitempty"`
	CodexWorkdir            string `json:"codex_workdir"`
	CodexOutputDir          string `json:"codex_output_dir"`
	CodexSandbox            string `json:"codex_sandbox"`
	CodexApproval           string `json:"codex_approval_policy"`
	CodexProfile            string `json:"codex_profile"`
	CodexModel              string `json:"codex_model"`
	CodexReasoning          string `json:"codex_reasoning_effort"`
	OpenAIBaseURL           string `json:"openai_base_url,omitempty"`
	CodexAPITimeoutSeconds  *int   `json:"codex_api_timeout_seconds,omitempty"`
	OpenAIOrgID             string `json:"openai_org_id,omitempty"`
	OpenAIProjectID         string `json:"openai_project_id,omitempty"`
	ClaudeAPIModel          string `json:"claude_api_model,omitempty"`
	AnthropicBaseURL        string `json:"anthropic_base_url,omitempty"`
	ClaudeAPITimeoutSeconds *int   `json:"claude_api_timeout_seconds,omitempty"`
	PostSubmitRebuild       string `json:"post_submit_rebuild_cmd"`
	PostSubmitVerify        string `json:"post_submit_verify_cmd"`
	PostSubmitTimeout       *int   `json:"post_submit_timeout_seconds,omitempty"`
	DeliveryIntentProfile   string `json:"delivery_intent_profile,omitempty"`
	ImplementChangesPrompt  string `json:"implement_changes_prompt,omitempty"`
	DraftPlanPrompt         string `json:"draft_plan_prompt,omitempty"`
	CreateJiraTicketsPrompt string `json:"create_jira_tickets_prompt,omitempty"`
	CodexSkipRepoCheck      *bool  `json:"codex_skip_git_repo_check,omitempty"`
}

type configImportRequest struct {
	Profile string               `json:"profile"`
	Config  *config.PublicConfig `json:"config,omitempty"`
}

func New(
	cfg config.Config,
	sessions *session.Service,
	privilegedCapture *privileged.CaptureBroker,
	auditLogger *audit.Logger,
	agentRegistry *agents.Registry,
	store storage.Store,
	artifactStore *storage.ArtifactStore,
	autoStart *platform.AutoStartManager,
	transcriptionProvider transcription.Provider,
) *Server {
	s := &Server{
		cfg:                 cfg,
		nonces:              map[string]time.Time{},
		pendingPairs:        map[string]pendingExtensionPairing{},
		sessions:            sessions,
		privilegedCapture:   privilegedCapture,
		audit:               auditLogger,
		agents:              agentRegistry,
		store:               store,
		artifacts:           artifactStore,
		autoStart:           autoStart,
		latency:             newLatencyBook(512),
		stt:                 transcriptionProvider,
		postSubmitRunner:    runPostSubmitAutomation,
		submitRunning:       map[string]submitJob{},
		submitCancel:        map[string]context.CancelFunc{},
		submitCanceled:      map[string]string{},
		submitQueuePath:     filepath.Join(cfg.DataDir, "submit_queue.json"),
		updateHTTPClient:    &http.Client{Timeout: 5 * time.Second},
		updateReleaseAPIURL: defaultReleaseCheckAPIURL,
	}
	if privilegedCapture != nil {
		s.runtime = operatorstate.Capture(cfg, privilegedCapture.AudioState())
	} else {
		s.runtime = operatorstate.State{Version: 1}
	}
	if store != nil {
		if persisted, err := store.LoadOperatorState(); err == nil && persisted != nil {
			persisted.Version = 1
			if privilegedCapture != nil {
				persisted.Audio = s.runtime.Audio
			}
			s.runtime = *persisted
		}
	}
	s.runtime.RuntimeCodex = operatorstate.NormalizeRuntimeCodexDefaults(s.runtime.RuntimeCodex)
	s.applyRuntimeStateToProcess()
	s.agents = s.buildAgentRegistry()
	s.stt = s.buildTranscriptionProvider(cfg)
	s.postSubmitRunner = func() *postSubmitResult {
		return runPostSubmitAutomationFor(s.currentRuntimeCodex())
	}
	s.initSubmitWorkers()

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/favicon.ico", s.handleFavicon)
	mux.HandleFunc("/docs", s.handleDocsBrowser)
	mux.HandleFunc("/docs/assets/", s.handleDocsAsset)
	mux.HandleFunc("/floating-composer", s.handleFloatingComposer)
	mux.HandleFunc("/companion.js", s.handleCompanionScript)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/update/check", s.handleUpdateCheck)
	mux.HandleFunc("/api/session/start", s.handleSessionStart)
	mux.HandleFunc("/api/session/pause", s.handleSessionPause)
	mux.HandleFunc("/api/session/resume", s.handleSessionResume)
	mux.HandleFunc("/api/session/stop", s.handleSessionStop)
	mux.HandleFunc("/api/session/feedback", s.handleFeedback)
	mux.HandleFunc("/api/session/feedback/note", s.handleFeedbackNote)
	mux.HandleFunc("/api/session/feedback/clip", s.handleFeedbackClip)
	mux.HandleFunc("/api/session/feedback/discard-last", s.handleFeedbackDiscardLast)
	mux.HandleFunc("/api/session/feedback/update-text", s.handleFeedbackUpdateText)
	mux.HandleFunc("/api/session/feedback/delete", s.handleFeedbackDelete)
	mux.HandleFunc("/api/session/replay/settings", s.handleReplaySettings)
	mux.HandleFunc("/api/session/review-note", s.handleReviewNote)
	mux.HandleFunc("/api/session/review-mode", s.handleReviewMode)
	mux.HandleFunc("/api/session/approve", s.handleApprove)
	mux.HandleFunc("/api/session/payload/preview", s.handlePayloadPreview)
	mux.HandleFunc("/api/session/replay/export", s.handleReplayExport)
	mux.HandleFunc("/api/session/submit", s.handleSubmit)
	mux.HandleFunc("/api/session/attempt/log", s.handleAttemptLog)
	mux.HandleFunc("/api/session/attempt/cancel", s.handleCancelSubmitAttempt)
	mux.HandleFunc("/api/session/attempt/rerun", s.handleRerunSubmitAttempt)
	mux.HandleFunc("/api/session/open-last-log", s.handleOpenLastLog)
	mux.HandleFunc("/api/session/history", s.handleHistory)
	mux.HandleFunc("/api/extension/pair/start", s.handleExtensionPairStart)
	mux.HandleFunc("/api/extension/pair/complete", s.handleExtensionPairComplete)
	mux.HandleFunc("/api/extension/pairings", s.handleExtensionPairings)
	mux.HandleFunc("/api/extension/pair/revoke", s.handleExtensionPairRevoke)
	mux.HandleFunc("/api/extension/session", s.handleExtensionSession)
	mux.HandleFunc("/api/audit/export", s.handleAuditExport)
	mux.HandleFunc("/api/audio/devices", s.handleAudioDevices)
	mux.HandleFunc("/api/audio/config", s.handleAudioConfig)
	mux.HandleFunc("/api/audio/level", s.handleAudioLevel)
	mux.HandleFunc("/api/capture/kill", s.handleKillCapture)
	mux.HandleFunc("/api/capture/source", s.handleCaptureSource)
	mux.HandleFunc("/api/companion/pointer", s.handleCompanionPointer)
	mux.HandleFunc("/api/purge/session", s.handlePurgeSession)
	mux.HandleFunc("/api/purge/all", s.handlePurgeAll)
	mux.HandleFunc("/api/config/export", s.handleConfigExport)
	mux.HandleFunc("/api/config/import", s.handleConfigImport)
	mux.HandleFunc("/api/runtime/codex", s.handleRuntimeCodex)
	mux.HandleFunc("/api/runtime/codex/options", s.handleRuntimeCodexOptions)
	mux.HandleFunc("/api/runtime/transcription", s.handleRuntimeTranscription)
	mux.HandleFunc("/api/runtime/transcription/health", s.handleRuntimeTranscriptionHealth)
	mux.HandleFunc("/api/docs/catalog", s.handleDocsCatalog)
	mux.HandleFunc("/api/docs/view", s.handleDocsView)
	mux.HandleFunc("/api/fs/list", s.handleFSList)
	mux.HandleFunc("/api/fs/pickdir", s.handleFSPickDir)

	s.httpSrv = &http.Server{
		Addr:              cfg.HTTPListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpSrv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpSrv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) currentConfig() config.Config {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg
}

func (s *Server) updateConfig(cfg config.Config) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	s.cfg = cfg
}

func (s *Server) currentSTT() transcription.Provider {
	s.sttMu.RLock()
	defer s.sttMu.RUnlock()
	return s.stt
}

func (s *Server) setSTTProvider(p transcription.Provider) {
	s.sttMu.Lock()
	defer s.sttMu.Unlock()
	s.stt = p
}

func (s *Server) currentRuntimeState() operatorstate.State {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return s.runtime
}

func (s *Server) currentRuntimeCodex() operatorstate.RuntimeCodex {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return s.runtime.RuntimeCodex
}

func (s *Server) currentRuntimeTranscription() operatorstate.RuntimeTranscription {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return s.runtime.RuntimeTranscription
}

func (s *Server) updateRuntimeState(mut func(*operatorstate.State)) operatorstate.State {
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()
	mut(&s.runtime)
	return s.runtime
}

func (s *Server) applyRuntimeStateToProcess() {
	cfg := s.currentConfig()
	state := s.currentRuntimeState()
	s.updateConfig(operatorstate.Apply(cfg, nil, &state))
}

func (s *Server) persistOperatorState(cfg config.Config) error {
	state := s.currentRuntimeState()
	state.Version = 1
	if strings.TrimSpace(state.RuntimeTranscription.Mode) == "" {
		state.RuntimeTranscription.Mode = strings.TrimSpace(cfg.TranscriptionMode)
	}
	if s.store != nil {
		if err := s.store.SaveOperatorState(&state); err != nil {
			return err
		}
	}
	_, err := userconfig.Save(cfg, state)
	return err
}

func (s *Server) requireAuth(w http.ResponseWriter, r *http.Request) bool {
	cfg := s.currentConfig()
	if strings.TrimSpace(cfg.ControlToken) == "" && len(s.extensionPairingsPublic()) == 0 {
		return true
	}
	provided, authCtx, ok := s.authenticateRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	if !s.authorizedForCapability(authCtx.Capabilities, r.URL.Path, r.Method) {
		http.Error(w, "forbidden: capability not allowed", http.StatusForbidden)
		return false
	}
	s.withAuthContext(r, authCtx)
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		if !s.validateReplayHeaders(provided, r) {
			http.Error(w, "unauthorized: missing or replayed nonce", http.StatusUnauthorized)
			return false
		}
	}
	return true
}

func (s *Server) authorizedForCapability(caps []string, path, method string) bool {
	required := requiredCapability(path, method)
	if required == "" {
		return true
	}
	if len(caps) == 0 {
		return true
	}
	allowed := false
	for _, c := range caps {
		c = strings.TrimSpace(c)
		if capabilityAllows(c, required) {
			allowed = true
			break
		}
	}
	return allowed
}

func capabilityAllows(granted, required string) bool {
	granted = strings.TrimSpace(strings.ToLower(granted))
	required = strings.TrimSpace(strings.ToLower(required))
	if granted == "*" || granted == required {
		return true
	}
	switch required {
	case "logs":
		return granted == "read"
	case "config_read":
		return granted == "config" || granted == "read"
	default:
		return false
	}
}

func requiredCapability(path, method string) string {
	switch path {
	case "/api/session/attempt/log", "/api/session/open-last-log", "/api/audit/export":
		return "logs"
	case "/api/config/export", "/api/extension/pairings":
		return "config_read"
	case "/api/extension/pair/start", "/api/extension/pair/revoke":
		return "config"
	}
	if method == http.MethodGet {
		return "read"
	}
	switch path {
	case "/api/session/approve", "/api/session/submit", "/api/session/payload/preview", "/api/session/open-last-log", "/api/session/attempt/cancel":
		return "submit"
	case "/api/config/import", "/api/runtime/codex", "/api/runtime/transcription", "/api/fs/pickdir":
		return "config"
	case "/api/purge/session", "/api/purge/all":
		return "purge"
	default:
		return "capture"
	}
}

func stringSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out[trimmed] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *Server) validateReplayHeaders(token string, r *http.Request) bool {
	nonce := strings.TrimSpace(r.Header.Get("X-Knit-Nonce"))
	tsRaw := strings.TrimSpace(r.Header.Get("X-Knit-Timestamp"))
	if nonce == "" || tsRaw == "" {
		return false
	}
	ts, err := strconv.ParseInt(tsRaw, 10, 64)
	if err != nil {
		return false
	}
	now := time.Now().UTC()
	tm := time.UnixMilli(ts)
	if tm.Before(now.Add(-2*time.Minute)) || tm.After(now.Add(2*time.Minute)) {
		return false
	}
	key := token + ":" + nonce

	s.nonceMu.Lock()
	defer s.nonceMu.Unlock()
	// prune stale nonces
	for k, seenAt := range s.nonces {
		if seenAt.Before(now.Add(-3 * time.Minute)) {
			delete(s.nonces, k)
		}
	}
	if _, exists := s.nonces[key]; exists {
		return false
	}
	s.nonces[key] = now
	return true
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	cfg := s.currentConfig()
	page := strings.ReplaceAll(indexHTML, "__KNIT_TOKEN__", cfg.ControlToken)
	_, _ = w.Write([]byte(page))
}

func (s *Server) handleFloatingComposer(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	cfg := s.currentConfig()
	page := strings.ReplaceAll(floatingComposerHTML, "__KNIT_TOKEN__", cfg.ControlToken)
	_, _ = w.Write([]byte(page))
}

func (s *Server) handleCompanionScript(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	cfg := s.currentConfig()
	script := strings.ReplaceAll(companionJS, "__KNIT_TOKEN__", cfg.ControlToken)
	_, _ = w.Write([]byte(script))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	cfg := s.currentConfig()
	writeJSON(w, map[string]any{
		"ok":                true,
		"current_version":   runtimeVersion(cfg),
		"build_id":          cfg.BuildID,
		"version_pin":       cfg.VersionPin,
		"platform_profile":  platform.CurrentProfile(),
		"runtime_platform":  platform.CurrentRuntimeGuide(),
		"ui_runtime":        cfg.UIRuntime,
		"auto_start":        s.autoStart != nil && s.autoStart.Status().Enabled,
		"integrity_enabled": strings.TrimSpace(os.Getenv("KNIT_BINARY_SHA256")) != "" || strings.TrimSpace(os.Getenv("KNIT_BINARY_CHECKSUMS_FILE")) != "" || strings.TrimSpace(os.Getenv("KNIT_RELEASE_MANIFEST_FILE")) != "",
	})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	cfg := s.currentConfig()
	transcriptionProviderName := ""
	transcriptionMode := cfg.TranscriptionMode
	transcriptionEndpoint := ""
	sttProvider := s.currentSTT()
	if sttProvider != nil {
		transcriptionProviderName = sttProvider.Name()
		transcriptionMode = sttProvider.Mode()
		transcriptionEndpoint = sttProvider.Endpoint()
	}
	var pointerLatest any
	if curr := s.sessions.Current(); curr != nil {
		ptr, _ := s.privilegedCapture.PointerSnapshot(curr.ID)
		pointerLatest = ptr
	}
	audioState := map[string]any{}
	audioDevices := []audio.Device{}
	if s.privilegedCapture != nil {
		audioState = map[string]any{
			"state":   s.privilegedCapture.AudioState(),
			"devices": s.privilegedCapture.AudioDevices(),
		}
		audioDevices = s.privilegedCapture.AudioDevices()
	}
	autoStartStatus := platform.AutoStartStatus{}
	if s.autoStart != nil {
		autoStartStatus = s.autoStart.Status()
	}
	writeJSON(w, map[string]any{
		"capture_state":            s.privilegedCapture.State(),
		"capture_sources":          s.privilegedCapture.SourceStatuses(),
		"native_capture_modules":   platform.NativeCaptureModules(),
		"platform_profile":         platform.CurrentProfile(),
		"runtime_platform":         platform.CurrentRuntimeGuide(),
		"reduced_capabilities":     s.privilegedCapture.ReducedCapabilities(),
		"session":                  s.sessions.Current(),
		"pointer_latest":           pointerLatest,
		"audio":                    audioState,
		"audio_devices":            audioDevices,
		"auto_start":               autoStartStatus,
		"submit_queue":             s.submitQueueState(),
		"submit_attempts":          s.submitAttemptsSnapshot(),
		"submit_recovery_notices":  s.submitRecoveryNotesSnapshot(),
		"latency_metrics":          s.latency.snapshot(),
		"adapters":                 s.agents.Names(),
		"config_locked":            cfg.ConfigLocked,
		"capture_settings_locked":  cfg.CaptureSettingsLocked,
		"window_scoped":            cfg.WindowScopedCapture,
		"video_default":            cfg.VideoEnabledByDefault,
		"video_mode":               cfg.VideoMode,
		"transcription_mode":       transcriptionMode,
		"transcription_provider":   transcriptionProviderName,
		"transcription_endpoint":   transcriptionEndpoint,
		"allow_remote_stt":         cfg.AllowRemoteSTT,
		"allow_remote_submission":  cfg.AllowRemoteSubmission,
		"local_profile":            cfg.LocalProfile,
		"environment_name":         cfg.EnvironmentName,
		"current_version":          runtimeVersion(cfg),
		"build_id":                 cfg.BuildID,
		"version_pin":              cfg.VersionPin,
		"managed_deployment_id":    cfg.ManagedDeploymentID,
		"user_config_path":         userconfig.ResolvePath(cfg),
		"ui_runtime":               cfg.UIRuntime,
		"pointer_sample_hz":        cfg.PointerSampleHz,
		"sqlite_path":              cfg.SQLitePath,
		"control_capabilities":     cfg.ControlCapabilities,
		"allowed_submit_providers": append([]string(nil), cfg.AllowedSubmitProviders...),
		"siem_log_enabled":         strings.TrimSpace(cfg.SIEMLogPath) != "",
		"outbound_allowlist_size":  len(cfg.OutboundAllowlist),
		"outbound_blocklist_size":  len(cfg.BlockedTargets),
		"runtime_codex":            s.runtimeAgentState(),
		"runtime_transcription":    s.runtimeTranscriptionState(cfg, sttProvider),
		"update_check_on_startup":  s.runtime.System.CheckUpdatesOnStartup,
		"extension_pairings":       s.extensionPairingsPublic(),
	})
}

func (s *Server) handleSessionStart(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	cfg := s.currentConfig()
	var req startSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.TargetWindow = strings.TrimSpace(req.TargetWindow)
	if req.TargetWindow == "" {
		req.TargetWindow = defaultSessionTargetWindow
	}
	if req.TargetURL != "" && !redaction.URLAllowed(req.TargetURL, cfg.OutboundAllowlist, cfg.BlockedTargets) {
		http.Error(w, "target_url blocked by policy", http.StatusForbidden)
		return
	}
	s.privilegedCapture.Start()
	s.privilegedCapture.SetSourceStatus("screen", "degraded", "visual capture not enabled")
	s.privilegedCapture.SetSourceStatus("microphone", "degraded", "audio capture not started")
	s.privilegedCapture.SetSourceStatus("companion", "degraded", "browser companion not attached")
	sess := s.sessions.StartWithMeta(req.TargetWindow, req.TargetURL, cfg.LocalProfile, cfg.EnvironmentName, cfg.BuildID)
	if strings.TrimSpace(req.ReviewMode) != "" {
		if updated, err := s.sessions.SetReviewMode(req.ReviewMode); err == nil && updated != nil {
			sess = updated
		}
	}
	if err := s.store.UpsertSession(sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.auditWriteRequest(r, audit.Event{Type: "session_started", SessionID: sess.ID, Details: map[string]any{"target_window": req.TargetWindow, "target_url": req.TargetURL}})
	writeJSON(w, sess)
}

func (s *Server) handleSessionPause(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	s.privilegedCapture.Pause()
	s.privilegedCapture.SetSourceStatus("screen", "degraded", "capture paused")
	s.privilegedCapture.SetSourceStatus("microphone", "degraded", "capture paused")
	if err := s.sessions.Pause(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if curr := s.sessions.Current(); curr != nil {
		if err := s.store.UpsertSession(curr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.auditWriteRequest(r, audit.Event{Type: "session_paused", SessionID: curr.ID})
	}
	writeJSON(w, s.sessions.Current())
}

func (s *Server) handleSessionResume(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	s.privilegedCapture.Start()
	s.privilegedCapture.SetSourceStatus("screen", "degraded", "awaiting visual stream")
	s.privilegedCapture.SetSourceStatus("microphone", "degraded", "awaiting audio input")
	if err := s.sessions.Resume(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if curr := s.sessions.Current(); curr != nil {
		if err := s.store.UpsertSession(curr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.auditWriteRequest(r, audit.Event{Type: "session_resumed", SessionID: curr.ID})
	}
	writeJSON(w, s.sessions.Current())
}

func (s *Server) handleSessionStop(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	s.privilegedCapture.Stop()
	if err := s.sessions.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if curr := s.sessions.Current(); curr != nil {
		if err := s.store.UpsertSession(curr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.auditWriteRequest(r, audit.Event{Type: "session_stopped", SessionID: curr.ID})
	}
	writeJSON(w, s.sessions.Current())
}

func (s *Server) handleFeedback(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req feedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	evt := session.FeedbackEvt{
		RawTranscript:  redaction.Text(req.RawTranscript),
		NormalizedText: redaction.Text(req.Normalized),
		Pointer: session.PointerCtx{
			X:      req.PointerX,
			Y:      req.PointerY,
			Window: req.Window,
		},
		ScreenshotRef: req.ScreenshotRef,
		VideoClipRef:  req.VideoClipRef,
		Confidence:    0.5,
		ReviewMode:    strings.TrimSpace(req.ReviewMode),
		ExperimentID:  strings.TrimSpace(req.ExperimentID),
		Variant:       strings.TrimSpace(req.Variant),
		LaserMode:     req.LaserMode,
	}
	curr, err := s.sessions.AddFeedback(evt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.UpsertSession(curr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.auditWriteRequest(r, audit.Event{Type: "feedback_captured", SessionID: curr.ID, Details: map[string]any{"feedback_count": len(curr.Feedback), "source": s.requestSource(r)}})
	writeJSON(w, curr)
}

func (s *Server) handleFeedbackDiscardLast(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	curr, discarded, err := s.sessions.DiscardLastFeedback()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if curr != nil {
		if err := s.store.UpsertSession(curr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	sessionID := ""
	if curr != nil {
		sessionID = curr.ID
	}
	discardedEventID := ""
	if discarded != nil {
		discardedEventID = discarded.ID
	}
	_ = s.audit.Write(audit.Event{
		Type:      "feedback_discarded_last",
		SessionID: sessionID,
		Details: map[string]any{
			"discarded_event_id": discardedEventID,
		},
	})
	writeJSON(w, map[string]any{
		"session":            curr,
		"discarded_event_id": discardedEventID,
	})
}

func (s *Server) handleFeedbackUpdateText(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req feedbackUpdateTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	text := redaction.Text(req.Text)
	curr, updated, err := s.sessions.UpdateFeedbackText(req.EventID, text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if curr != nil {
		if err := s.store.UpsertSession(curr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	sessionID := ""
	if curr != nil {
		sessionID = curr.ID
	}
	_ = s.audit.Write(audit.Event{
		Type:      "feedback_text_updated",
		SessionID: sessionID,
		Details: map[string]any{
			"event_id": updated.ID,
		},
	})
	writeJSON(w, map[string]any{
		"session":  curr,
		"event_id": updated.ID,
		"text":     updated.NormalizedText,
	})
}

func (s *Server) handleFeedbackDelete(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req feedbackDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	curr, deleted, err := s.sessions.DeleteFeedback(req.EventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if curr != nil {
		if err := s.store.UpsertSession(curr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	sessionID := ""
	if curr != nil {
		sessionID = curr.ID
	}
	_ = s.audit.Write(audit.Event{
		Type:      "feedback_deleted",
		SessionID: sessionID,
		Details: map[string]any{
			"event_id": deleted.ID,
		},
	})
	writeJSON(w, map[string]any{
		"session":          curr,
		"deleted_event_id": deleted.ID,
	})
}

func (s *Server) handleReplaySettings(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req replaySettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.CaptureInputValues == nil {
		http.Error(w, "capture_input_values is required", http.StatusBadRequest)
		return
	}
	curr, err := s.sessions.SetCaptureInputValues(*req.CaptureInputValues)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.UpsertSession(curr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.audit.Write(audit.Event{
		Type:      "replay_settings_updated",
		SessionID: curr.ID,
		Details: map[string]any{
			"capture_input_values": curr.CaptureInputValues,
		},
	})
	writeJSON(w, map[string]any{"session": curr})
}

func (s *Server) handleReviewMode(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req reviewModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	curr, err := s.sessions.SetReviewMode(req.Mode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if curr != nil {
		if err := s.store.UpsertSession(curr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	_ = s.audit.Write(audit.Event{
		Type:      "review_mode_updated",
		SessionID: curr.ID,
		Details: map[string]any{
			"review_mode": curr.ReviewMode,
		},
	})
	writeJSON(w, map[string]any{"session": curr})
}

func (s *Server) handleReviewNote(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req reviewNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	curr, note, err := s.sessions.AddReviewNote(req.Author, req.Note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if curr != nil {
		if err := s.store.UpsertSession(curr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	sessionID := ""
	noteID := ""
	if curr != nil {
		sessionID = curr.ID
	}
	if note != nil {
		noteID = note.ID
	}
	_ = s.audit.Write(audit.Event{
		Type:      "review_note_added",
		SessionID: sessionID,
		Details: map[string]any{
			"review_note_id": noteID,
			"author":         req.Author,
		},
	})
	writeJSON(w, map[string]any{"session": curr, "review_note": note})
}

func (s *Server) handleFeedbackNote(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	curr := s.sessions.Current()
	if curr == nil {
		http.Error(w, session.ErrNoSession.Error(), http.StatusBadRequest)
		return
	}
	cfg := s.currentConfig()

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, "invalid multipart payload", http.StatusBadRequest)
		return
	}

	raw := strings.TrimSpace(r.FormValue("raw_transcript"))
	normalized := strings.TrimSpace(r.FormValue("normalized"))
	if normalized == "" {
		normalized = raw
	}
	audioRef := ""

	sttProvider := s.currentSTT()
	if raw == "" {
		audio, audioName, err := readMultipartFile(r, "audio")
		if err == nil && len(audio) > 0 && sttProvider != nil {
			defer security.ZeroBytes(audio)
			if s.privilegedCapture != nil {
				audioState := s.privilegedCapture.AudioState()
				if audioState.Muted {
					http.Error(w, "audio capture is muted", http.StatusConflict)
					return
				}
				if audioState.Paused {
					http.Error(w, "audio capture is paused", http.StatusConflict)
					return
				}
			}
			if cfg.AudioRetention > 0 {
				ext := strings.TrimPrefix(filepath.Ext(audioName), ".")
				if ext == "" {
					ext = "webm"
				}
				ref, saveErr := s.artifacts.Save("audio", curr.ID, audio, ext)
				if saveErr == nil {
					audioRef = ref
				}
			}
			if sttProvider.Mode() == "remote" && !cfg.AllowRemoteSTT {
				s.privilegedCapture.SetSourceStatus("microphone", "degraded", "remote transcription blocked by policy")
				http.Error(w, "remote transcription is disabled by policy", http.StatusForbidden)
				return
			}
			if sttProvider.Mode() == "remote" && !endpointAllowedByPolicy(sttProvider.Endpoint(), cfg) {
				s.privilegedCapture.SetSourceStatus("microphone", "degraded", "transcription endpoint blocked by policy")
				http.Error(w, "transcription endpoint blocked by policy", http.StatusForbidden)
				return
			}
			tmpFile, err := writeTempAudio(audioName, audio)
			if err != nil {
				s.privilegedCapture.SetSourceStatus("microphone", "degraded", "temp audio write failed")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer tmpFile.cleanup()
			transcript, trErr := sttProvider.Transcribe(r.Context(), tmpFile.path)
			if trErr != nil {
				s.privilegedCapture.SetSourceStatus("microphone", "degraded", "transcription failed")
				http.Error(w, trErr.Error(), http.StatusBadGateway)
				return
			}
			raw = strings.TrimSpace(transcript)
			normalized = raw
			s.privilegedCapture.SetSourceStatus("microphone", "available", "audio captured and transcribed")
		}
	}

	if raw == "" {
		s.privilegedCapture.SetSourceStatus("microphone", "degraded", "no transcript or audio provided")
		http.Error(w, "raw_transcript or audio file is required", http.StatusBadRequest)
		return
	}

	if handled, payload, err := s.handleVoiceCommand(curr, raw); handled {
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{
			"command_handled": true,
			"command_result":  payload,
		})
		s.latency.observe("feedback_note_ms", time.Since(startedAt))
		return
	}

	pointer, path := s.privilegedCapture.PointerSnapshot(curr.ID)
	replaySteps := s.privilegedCapture.PointerReplaySnapshot(curr.ID)
	sensitiveContext := redaction.SensitiveContext(curr.TargetWindow, curr.TargetURL, pointer.TargetLabel, pointer.TargetID, pointer.TargetTestID)
	outOfScopeVisual := curr.TargetURL != "" && pointer.URL != "" && !sameHost(curr.TargetURL, pointer.URL)
	if outOfScopeVisual {
		s.privilegedCapture.SetSourceStatus("screen", "degraded", "visual capture outside approved target scope")
	}

	screenshotRef := ""
	if screenshotBytes, screenshotName, err := readMultipartFile(r, "screenshot"); err == nil && len(screenshotBytes) > 0 {
		defer security.ZeroBytes(screenshotBytes)
		if !sensitiveContext && !outOfScopeVisual {
			ext := strings.TrimPrefix(filepath.Ext(screenshotName), ".")
			if ext == "" {
				ext = "png"
			}
			compressed, compressedExt := compressScreenshot(screenshotBytes)
			if compressedExt != "" {
				ext = compressedExt
			}
			ref, saveErr := s.artifacts.Save("screenshot", curr.ID, compressed, ext)
			if saveErr != nil {
				http.Error(w, saveErr.Error(), http.StatusInternalServerError)
				return
			}
			screenshotRef = ref
			s.privilegedCapture.SetSourceStatus("screen", "available", "screenshot captured")
		}
	}

	clipRef := ""
	var clipMeta *session.VideoMetadata
	if clipBytes, clipName, err := readMultipartFile(r, "clip"); err == nil && len(clipBytes) > 0 {
		defer security.ZeroBytes(clipBytes)
		if !sensitiveContext && !outOfScopeVisual {
			ext := strings.TrimPrefix(filepath.Ext(clipName), ".")
			if ext == "" {
				ext = "webm"
			}
			ref, saveErr := s.artifacts.Save("clip", curr.ID, clipBytes, ext)
			if saveErr != nil {
				http.Error(w, saveErr.Error(), http.StatusInternalServerError)
				return
			}
			clipRef = ref
			clipMeta = parseClipVideoMetadata(r)
			s.privilegedCapture.SetSourceStatus("screen", "available", "video clip captured")
		}
	}

	visualTarget := strings.TrimSpace(pointer.TargetTag)
	if pointer.TargetID != "" {
		visualTarget += "#" + pointer.TargetID
	}
	if pointer.TargetTestID != "" {
		visualTarget += "[data-testid=" + pointer.TargetTestID + "]"
	}
	if pointer.TargetSelector != "" {
		if visualTarget != "" {
			visualTarget += " | "
		}
		visualTarget += pointer.TargetSelector
	}
	reviewMode := strings.TrimSpace(r.FormValue("review_mode"))
	experimentID := strings.TrimSpace(r.FormValue("experiment_id"))
	variant := strings.TrimSpace(r.FormValue("variant"))
	laserMode := false
	if v := strings.ToLower(strings.TrimSpace(r.FormValue("laser_mode"))); v == "1" || v == "true" || v == "yes" {
		laserMode = true
	}
	var laserPath []session.PointerSample
	if rawLaserPath := strings.TrimSpace(r.FormValue("laser_path_json")); rawLaserPath != "" {
		_ = json.Unmarshal([]byte(rawLaserPath), &laserPath)
	}
	evt := session.FeedbackEvt{
		RawTranscript:   redaction.Text(raw),
		NormalizedText:  redaction.Text(normalized),
		Pointer:         pointer,
		PointerPath:     path,
		AudioRef:        audioRef,
		VisualTargetRef: visualTarget,
		ScreenshotRef:   screenshotRef,
		VideoClipRef:    clipRef,
		Confidence:      0.75,
		ReviewMode:      reviewMode,
		ExperimentID:    experimentID,
		Variant:         variant,
		LaserMode:       laserMode,
		LaserPath:       laserPath,
		Video:           clipMeta,
		Replay:          buildReplayBundleForNote(pointer, path, replaySteps, curr.CaptureInputValues),
	}
	updated, err := s.sessions.AddFeedback(evt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.UpsertSession(updated); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	newEvt := updated.Feedback[len(updated.Feedback)-1]
	_ = s.audit.Write(audit.Event{Type: "feedback_note_captured", SessionID: updated.ID, Details: map[string]any{
		"event_id":           newEvt.ID,
		"has_screenshot":     screenshotRef != "",
		"has_clip":           clipRef != "",
		"sensitive_suppress": sensitiveContext,
		"transcription_mode": s.sttMode(),
	}})
	s.latency.observe("feedback_note_ms", time.Since(startedAt))
	writeJSON(w, map[string]any{"session": updated, "event_id": newEvt.ID, "screenshot_ref": screenshotRef, "clip_ref": clipRef, "sensitive_context_suppressed": sensitiveContext})
}

func (s *Server) handleFeedbackClip(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method == http.MethodGet {
		if !s.requireAuth(w, r) {
			return
		}
		s.handleFeedbackClipFetch(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	curr := s.sessions.Current()
	if curr == nil {
		http.Error(w, session.ErrNoSession.Error(), http.StatusBadRequest)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, "invalid multipart payload", http.StatusBadRequest)
		return
	}
	eventID := strings.TrimSpace(r.FormValue("event_id"))
	if eventID == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}
	clipBytes, clipName, err := readMultipartFile(r, "clip")
	if err != nil || len(clipBytes) == 0 {
		http.Error(w, "clip file is required", http.StatusBadRequest)
		return
	}
	defer security.ZeroBytes(clipBytes)
	pointer, _ := s.privilegedCapture.PointerSnapshot(curr.ID)
	if curr.TargetURL != "" && pointer.URL != "" && !sameHost(curr.TargetURL, pointer.URL) {
		s.privilegedCapture.SetSourceStatus("screen", "degraded", "clip outside approved target scope")
		http.Error(w, "clip outside session target scope", http.StatusForbidden)
		return
	}
	ext := strings.TrimPrefix(filepath.Ext(clipName), ".")
	if ext == "" {
		ext = "webm"
	}
	clipRef, saveErr := s.artifacts.Save("clip", curr.ID, clipBytes, ext)
	if saveErr != nil {
		http.Error(w, saveErr.Error(), http.StatusInternalServerError)
		return
	}
	videoMeta := parseClipVideoMetadata(r)
	updated, err := s.sessions.AttachClip(eventID, clipRef, videoMeta)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.UpsertSession(updated); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.privilegedCapture.SetSourceStatus("screen", "available", "clip attached")
	clipDetails := map[string]any{"event_id": eventID, "clip_ref": clipRef}
	if videoMeta != nil {
		clipDetails["codec"] = videoMeta.Codec
		clipDetails["scope"] = videoMeta.Scope
		clipDetails["has_audio"] = videoMeta.HasAudio
		clipDetails["pointer_overlay"] = videoMeta.PointerOverlay
		clipDetails["duration_ms"] = videoMeta.DurationMS
	}
	_ = s.audit.Write(audit.Event{Type: "feedback_clip_attached", SessionID: updated.ID, Details: clipDetails})
	s.latency.observe("feedback_clip_attach_ms", time.Since(startedAt))
	writeJSON(w, map[string]any{"ok": true, "event_id": eventID, "clip_ref": clipRef})
}

func (s *Server) handleFeedbackClipFetch(w http.ResponseWriter, r *http.Request) {
	curr := s.sessions.Current()
	if curr == nil {
		http.Error(w, session.ErrNoSession.Error(), http.StatusBadRequest)
		return
	}
	eventID := strings.TrimSpace(r.URL.Query().Get("event_id"))
	if eventID == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}
	var (
		clipRef string
		codec   string
	)
	for _, evt := range curr.Feedback {
		if strings.TrimSpace(evt.ID) != eventID {
			continue
		}
		clipRef = strings.TrimSpace(evt.VideoClipRef)
		if evt.Video != nil {
			codec = strings.TrimSpace(evt.Video.Codec)
		}
		break
	}
	if clipRef == "" {
		http.Error(w, "clip not found for event", http.StatusNotFound)
		return
	}
	payload, err := s.artifacts.Load(clipRef)
	if err != nil {
		http.Error(w, fmt.Sprintf("load clip artifact: %v", err), http.StatusInternalServerError)
		return
	}
	mimeType := inferArtifactMIMEType(clipRef, payload)
	w.Header().Set("Content-Type", mimeType)
	if codec != "" {
		w.Header().Set("X-Knit-Video-Codec", codec)
	}
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(payload)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req approveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	pkg, err := s.sessions.Approve(redaction.Text(req.Summary))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	curr := s.sessions.Current()
	if curr != nil {
		if pkg, err = s.attachReplayArtifacts(pkg, curr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := s.sessions.ReplaceApprovedPackage(pkg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := s.store.UpsertSession(curr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := s.store.SaveCanonicalPackage(pkg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.auditWriteRequest(r, audit.Event{Type: "session_approved", SessionID: pkg.SessionID, Details: map[string]any{"change_requests": len(pkg.ChangeRequests), "source": s.requestSource(r)}})
	writeJSON(w, pkg)
}

func (s *Server) handlePayloadPreview(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req payloadPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Provider = strings.TrimSpace(req.Provider)
	if req.Provider == "" {
		req.Provider = s.resolveProvider("")
	} else {
		req.Provider = canonicalProviderAlias(req.Provider, s.agents.Names())
	}
	if !submitProviderAllowed(req.Provider, s.currentConfig()) {
		http.Error(w, "provider blocked by policy", http.StatusForbidden)
		return
	}
	pkg, err := s.sessions.ApprovedPackage()
	if err != nil {
		http.Error(w, "session must be explicitly approved before payload preview", http.StatusPreconditionFailed)
		return
	}
	transmissionPkg, transmissionWarnings, _, err := s.buildTransmissionPackage(pkg, transmissionOptions{
		AllowLargeInline:   req.AllowLargeInlineMedia,
		RedactReplayValues: req.RedactReplayValues,
		OmitVideoClips:     req.OmitVideoClips,
		OmitVideoEventIDs:  stringSet(req.OmitVideoEventIDs),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !canonicalPackageHasInput(transmissionPkg) {
		http.Error(w, "capture at least one note before previewing or submitting", http.StatusConflict)
		return
	}
	rc := s.currentRuntimeCodex()
	intent := agents.NormalizeDeliveryIntent(agents.DeliveryIntent{
		Profile:            req.IntentProfile,
		InstructionText:    req.InstructionText,
		CustomInstructions: req.CustomInstructions,
	})
	payload, err := agents.PreviewProviderPayloadWithConfig(req.Provider, redactPackageForTransmission(*transmissionPkg), rc.CodexModel, rc.ClaudeAPIModel, intent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	curr := s.sessions.Current()
	if curr == nil {
		http.Error(w, session.ErrNoSession.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, payloadPreviewResponse{
		Provider: req.Provider,
		Payload:  payload,
		Preview:  mergePayloadPreviewWarnings(s.buildRenderedPayloadPreview(transmissionPkg, curr, req.Provider, intent), transmissionWarnings),
	})
}

func (s *Server) handleReplayExport(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	eventID := strings.TrimSpace(r.URL.Query().Get("event_id"))
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if eventID == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}
	if format != "json" && format != "playwright" {
		http.Error(w, "format must be json or playwright", http.StatusBadRequest)
		return
	}
	pkg, err := s.sessions.ApprovedPackage()
	if err != nil {
		http.Error(w, "session must be explicitly approved before replay export", http.StatusPreconditionFailed)
		return
	}
	var replay *session.ReplayBundle
	for _, change := range pkg.ChangeRequests {
		if change.EventID == eventID {
			replay = change.Replay
			break
		}
	}
	if replay == nil {
		http.Error(w, "replay bundle not found", http.StatusNotFound)
		return
	}
	if format == "json" {
		payload, err := json.MarshalIndent(replay, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "replay-"+eventID+".json"))
		_, _ = w.Write(payload)
		return
	}
	script := strings.TrimSpace(replay.PlaywrightScript)
	if script == "" {
		http.Error(w, "playwright script unavailable", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "replay-"+eventID+".spec.ts"))
	_, _ = w.Write([]byte(script))
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Provider = strings.TrimSpace(req.Provider)
	if req.Provider == "" {
		req.Provider = s.resolveProvider("")
	} else {
		req.Provider = canonicalProviderAlias(req.Provider, s.agents.Names())
	}
	cfg := s.currentConfig()
	if !submitProviderAllowed(req.Provider, cfg) {
		http.Error(w, "provider blocked by policy", http.StatusForbidden)
		return
	}
	if s.agents.IsRemote(req.Provider) && !cfg.AllowRemoteSubmission {
		http.Error(w, "remote submission is disabled by policy", http.StatusForbidden)
		return
	}
	if s.agents.IsRemote(req.Provider) {
		if endpoint := s.agents.Endpoint(req.Provider); endpoint != "" && !endpointAllowedByPolicy(endpoint, cfg) {
			http.Error(w, "submission endpoint blocked by policy", http.StatusForbidden)
			return
		}
	}

	approvedPkg, err := s.sessions.ApprovedPackage()
	if err != nil {
		http.Error(w, "session must be explicitly approved before submission", http.StatusPreconditionFailed)
		return
	}
	transmissionPkg, transmissionWarnings, requiresDecision, err := s.buildTransmissionPackage(approvedPkg, transmissionOptions{
		AllowLargeInline:   req.AllowLargeInlineMedia,
		RedactReplayValues: req.RedactReplayValues,
		OmitVideoClips:     req.OmitVideoClips,
		OmitVideoEventIDs:  stringSet(req.OmitVideoEventIDs),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !canonicalPackageHasInput(transmissionPkg) {
		http.Error(w, "capture at least one note before submission", http.StatusConflict)
		return
	}
	if requiresDecision {
		http.Error(w, strings.Join(transmissionWarnings, " "), http.StatusConflict)
		return
	}
	if _, err := s.sessions.ReserveApprovedPackage(); err != nil {
		http.Error(w, "session must be explicitly approved before submission", http.StatusPreconditionFailed)
		return
	}
	redactedPkg := redactPackageForTransmission(*transmissionPkg)
	rc := s.currentRuntimeCodex()
	intent := agents.NormalizeDeliveryIntent(agents.DeliveryIntent{
		Profile:            req.IntentProfile,
		InstructionText:    req.InstructionText,
		CustomInstructions: req.CustomInstructions,
	})
	providerPayload, err := agents.PreviewProviderPayloadWithConfig(req.Provider, redactedPkg, rc.CodexModel, rc.ClaudeAPIModel, intent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if curr := s.sessions.Current(); curr != nil {
		if err := s.store.UpsertSession(curr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	authCtx := s.requestAuthContext(r)
	attempt := s.enqueueSubmitJob(req.Provider, redactedPkg, providerPayload, intent, s.requestSource(r), authCtx.Actor)
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]any{
		"attempt_id":         attempt.AttemptID,
		"provider":           attempt.Provider,
		"intent_profile":     attempt.IntentProfile,
		"intent_label":       attempt.IntentLabel,
		"session_id":         attempt.SessionID,
		"status":             attempt.Status,
		"execution_mode":     attempt.Mode,
		"queue_position":     attempt.QueuePos,
		"source":             attempt.Source,
		"submit_queue_state": s.submitQueueState(),
	})
}

func buildReplayBundleForNote(pointer session.PointerCtx, path []session.PointerSample, steps []session.ReplayStep, captureInputValues bool) *session.ReplayBundle {
	if len(steps) == 0 && len(path) == 0 && pointer.DOM == nil && len(pointer.Console) == 0 && len(pointer.Network) == 0 {
		return nil
	}
	mode := "redacted"
	if captureInputValues {
		mode = "opt_in"
	}
	replaySteps := cloneReplaySteps(steps)
	if !captureInputValues {
		for i := range replaySteps {
			replaySteps[i].Value = ""
			replaySteps[i].ValueCaptured = false
			if strings.EqualFold(strings.TrimSpace(replaySteps[i].Type), "input") || strings.EqualFold(strings.TrimSpace(replaySteps[i].Type), "change") {
				replaySteps[i].ValueRedacted = true
			}
		}
	}
	return &session.ReplayBundle{
		URL:              strings.TrimSpace(pointer.URL),
		Route:            strings.TrimSpace(pointer.Route),
		TargetTag:        strings.TrimSpace(pointer.TargetTag),
		TargetID:         strings.TrimSpace(pointer.TargetID),
		TargetTestID:     strings.TrimSpace(pointer.TargetTestID),
		TargetLabel:      strings.TrimSpace(pointer.TargetLabel),
		TargetSelector:   strings.TrimSpace(pointer.TargetSelector),
		ValueCaptureMode: mode,
		PointerPath:      append([]session.PointerSample(nil), path...),
		Steps:            replaySteps,
		DOM:              cloneDOMInspection(pointer.DOM),
		Console:          append([]session.ConsoleEntry(nil), pointer.Console...),
		Network:          append([]session.NetworkEntry(nil), pointer.Network...),
	}
}

func (s *Server) attachReplayArtifacts(pkg *session.CanonicalPackage, curr *session.Session) (*session.CanonicalPackage, error) {
	if pkg == nil || curr == nil {
		return pkg, nil
	}
	byEventID := make(map[string]session.FeedbackEvt, len(curr.Feedback))
	for _, evt := range curr.Feedback {
		byEventID[evt.ID] = evt
	}
	for i := range pkg.ChangeRequests {
		change := &pkg.ChangeRequests[i]
		evt, ok := byEventID[change.EventID]
		if !ok || evt.Replay == nil {
			continue
		}
		replay := cloneReplayBundle(evt.Replay)
		replay.PlaywrightScript = session.GeneratePlaywrightScript(change.Summary, replay)
		replay.Exports = nil

		jsonPayload, err := json.MarshalIndent(replay, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal replay bundle: %w", err)
		}
		jsonRef, err := s.artifacts.Save("replay", pkg.SessionID, jsonPayload, "json")
		if err != nil {
			return nil, fmt.Errorf("save replay bundle: %w", err)
		}
		pkg.Artifacts = append(pkg.Artifacts, session.ArtifactRef{Kind: "replay_json", Ref: jsonRef, EventID: change.EventID})
		replay.Exports = append(replay.Exports, session.ReplayExport{
			Kind:     "json",
			Ref:      jsonRef,
			Filename: "replay-" + change.EventID + ".json",
		})

		if replay.PlaywrightScript != "" {
			scriptRef, err := s.artifacts.Save("playwright", pkg.SessionID, []byte(replay.PlaywrightScript), "ts")
			if err != nil {
				return nil, fmt.Errorf("save playwright replay script: %w", err)
			}
			pkg.Artifacts = append(pkg.Artifacts, session.ArtifactRef{Kind: "playwright_script", Ref: scriptRef, EventID: change.EventID})
			replay.Exports = append(replay.Exports, session.ReplayExport{
				Kind:     "playwright",
				Ref:      scriptRef,
				Filename: "replay-" + change.EventID + ".spec.ts",
			})
		}
		change.Replay = replay
	}
	return pkg, nil
}

func statusOrEmpty(step *postSubmitStepResult) string {
	if step == nil {
		return ""
	}
	return step.Status
}

func providerSet(adapters []string) map[string]struct{} {
	out := make(map[string]struct{}, len(adapters))
	for _, name := range adapters {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		out[trimmed] = struct{}{}
	}
	return out
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstProvider(adapters []string) string {
	for _, name := range adapters {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func providerAlias(provider string) string {
	switch strings.TrimSpace(provider) {
	case "cli":
		return "codex_cli"
	default:
		return strings.TrimSpace(provider)
	}
}

func durationFromSeconds(seconds int, fallback time.Duration) time.Duration {
	if seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func (s *Server) buildAgentRegistry() *agents.Registry {
	rc := s.currentRuntimeCodex()
	return agents.NewRegistry(
		agents.NewCodexAPIAdapter(
			strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
			rc.CodexModel,
			rc.OpenAIBaseURL,
			durationFromSeconds(rc.CodexAPITimeoutSeconds, 60*time.Second),
			firstNonEmptyString(rc.OpenAIOrgID, os.Getenv("OPENAI_ORGANIZATION")),
			rc.OpenAIProjectID,
		),
		agents.NewClaudeAPIAdapter(
			strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
			rc.ClaudeAPIModel,
			rc.AnthropicBaseURL,
			durationFromSeconds(rc.ClaudeAPITimeoutSeconds, 60*time.Second),
		),
		agents.NewCLIAdapter("codex_cli", rc.CLIAdapterCmd, durationFromSeconds(rc.CLITimeoutSeconds, agentsDefaultCLITimeout())),
		agents.NewCLIAdapter("claude_cli", rc.ClaudeCLIAdapterCmd, durationFromSeconds(rc.ClaudeCLITimeoutSeconds, agentsDefaultCLITimeout())),
		agents.NewCLIAdapter("opencode_cli", rc.OpenCodeCLIAdapterCmd, durationFromSeconds(rc.OpenCodeCLITimeoutSecs, agentsDefaultCLITimeout())),
	)
}

func agentsDefaultCLITimeout() time.Duration {
	return 10 * time.Minute
}

func (s *Server) buildTranscriptionProvider(cfg config.Config) transcription.Provider {
	rt := s.currentRuntimeTranscription()
	mode := strings.ToLower(strings.TrimSpace(rt.Mode))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(cfg.TranscriptionMode))
	}
	switch mode {
	case "local":
		return transcription.NewLocalCLIProvider(rt.LocalCommand, durationFromSeconds(rt.TimeoutSecond, 90*time.Second))
	case "faster_whisper":
		runtimeDir := filepath.Join(cfg.DataDir, "runtime", "faster-whisper")
		return transcription.NewManagedFasterWhisperProvider(
			runtimeDir,
			os.Getenv("KNIT_FASTER_WHISPER_PYTHON"),
			rt.Model,
			rt.Device,
			rt.ComputeType,
			rt.Language,
			durationFromSeconds(rt.TimeoutSecond, 120*time.Second),
		)
	case "lmstudio":
		return transcription.NewLMStudioSpeechToTextProvider(
			firstNonEmptyString(os.Getenv("KNIT_LMSTUDIO_API_KEY"), os.Getenv("LMSTUDIO_API_KEY")),
			rt.BaseURL,
			rt.Model,
			durationFromSeconds(rt.TimeoutSecond, 90*time.Second),
		)
	case "remote":
		fallthrough
	default:
		return transcription.NewOpenAISpeechToTextProvider(
			strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
			rt.BaseURL,
			rt.Model,
			90*time.Second,
		)
	}
}

func canonicalProviderAlias(provider string, adapters []string) string {
	p := strings.TrimSpace(provider)
	if p == "" {
		return ""
	}
	available := providerSet(adapters)
	if _, ok := available[p]; ok {
		return p
	}
	if alias := providerAlias(p); alias != "" {
		if _, ok := available[alias]; ok {
			return alias
		}
	}
	return p
}

func validateTranscriptionRuntimeRequest(mode string, req runtimeTranscriptionRequest) error {
	if req.TimeoutSecond != nil {
		if *req.TimeoutSecond < 0 {
			return fmt.Errorf("timeout_seconds must be 0 or between 1 and %d", maxTranscriptionTimeoutSeconds)
		}
		if *req.TimeoutSecond > maxTranscriptionTimeoutSeconds {
			return fmt.Errorf("timeout_seconds must be 0 or between 1 and %d", maxTranscriptionTimeoutSeconds)
		}
	}
	if req.Language != nil {
		language := strings.TrimSpace(*req.Language)
		if language != "" && !transcriptionLanguagePattern.MatchString(language) {
			return fmt.Errorf("language must be a short language tag such as en or en-US")
		}
	}
	if req.LocalCommand != nil {
		command := strings.TrimSpace(*req.LocalCommand)
		if len(command) > maxTranscriptionLocalCommandLen {
			return fmt.Errorf("local_command exceeds max length of %d", maxTranscriptionLocalCommandLen)
		}
		for _, r := range command {
			if r == '\n' || r == '\r' || r == 0 {
				return fmt.Errorf("local_command must be single-line text")
			}
			if unicode.IsControl(r) && r != '\t' {
				return fmt.Errorf("local_command contains unsupported control characters")
			}
		}
	}
	switch mode {
	case "local":
		if req.LocalCommand != nil {
			command := strings.TrimSpace(*req.LocalCommand)
			if command == "" {
				return nil
			}
		}
	case "faster_whisper":
		if req.Language != nil {
			trimmed := strings.TrimSpace(*req.Language)
			if trimmed != "" && len(trimmed) > 24 {
				return fmt.Errorf("language is too long")
			}
		}
	}
	return nil
}

func resolveProvider(requested string, adapters []string) string {
	available := providerSet(adapters)
	if len(available) == 0 {
		return ""
	}
	p := canonicalProviderAlias(requested, adapters)
	if p != "" {
		if _, ok := available[p]; ok {
			return p
		}
	}
	p = canonicalProviderAlias(os.Getenv("KNIT_DEFAULT_PROVIDER"), adapters)
	if p != "" {
		if _, ok := available[p]; ok {
			return p
		}
	}
	if _, ok := available["codex_cli"]; ok {
		return "codex_cli"
	}
	if _, ok := available["cli"]; ok {
		return "cli"
	}
	if first := firstProvider(adapters); first != "" {
		return first
	}
	return "codex_cli"
}

func (s *Server) resolveProvider(requested string) string {
	adapters := allowedSubmitProviders(s.agents.Names(), s.currentConfig())
	available := providerSet(adapters)
	p := canonicalProviderAlias(requested, adapters)
	if p != "" {
		if _, ok := available[p]; ok {
			return p
		}
	}
	defaultProvider := strings.TrimSpace(s.currentRuntimeCodex().DefaultProvider)
	p = canonicalProviderAlias(defaultProvider, adapters)
	if p != "" {
		if _, ok := available[p]; ok {
			return p
		}
	}
	return resolveProvider("", adapters)
}

func (s *Server) runtimeAgentState() map[string]any {
	rc := s.currentRuntimeCodex()
	adapters := allowedSubmitProviders(s.agents.Names(), s.currentConfig())
	return map[string]any{
		"default_provider":             s.resolveProvider(""),
		"available_providers":          adapters,
		"cli_adapter_cmd":              rc.CLIAdapterCmd,
		"cli_timeout_seconds":          intString(rc.CLITimeoutSeconds),
		"claude_cli_adapter_cmd":       rc.ClaudeCLIAdapterCmd,
		"claude_cli_timeout_seconds":   intString(rc.ClaudeCLITimeoutSeconds),
		"opencode_cli_adapter_cmd":     rc.OpenCodeCLIAdapterCmd,
		"opencode_cli_timeout_seconds": intString(rc.OpenCodeCLITimeoutSecs),
		"submit_execution_mode":        s.submitExecutionMode(),
		"codex_workdir":                rc.CodexWorkdir,
		"codex_output_dir":             rc.CodexOutputDir,
		"codex_sandbox":                rc.CodexSandbox,
		"codex_approval_policy":        rc.CodexApproval,
		"codex_profile":                rc.CodexProfile,
		"codex_model":                  rc.CodexModel,
		"codex_reasoning_effort":       rc.CodexReasoning,
		"openai_base_url":              rc.OpenAIBaseURL,
		"codex_api_timeout_seconds":    intString(rc.CodexAPITimeoutSeconds),
		"openai_org_id":                rc.OpenAIOrgID,
		"openai_project_id":            rc.OpenAIProjectID,
		"openai_api_key_configured":    strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != "",
		"claude_api_model":             rc.ClaudeAPIModel,
		"anthropic_base_url":           rc.AnthropicBaseURL,
		"claude_api_timeout_seconds":   intString(rc.ClaudeAPITimeoutSeconds),
		"anthropic_api_key_configured": strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) != "",
		"post_submit_rebuild_cmd":      rc.PostSubmitRebuild,
		"post_submit_verify_cmd":       rc.PostSubmitVerify,
		"post_submit_timeout_seconds":  intString(rc.PostSubmitTimeout),
		"delivery_intent_profile":      rc.DeliveryIntentProfile,
		"implement_changes_prompt":     rc.ImplementChangesPrompt,
		"create_jira_tickets_prompt":   rc.CreateJiraTicketsPrompt,
		"codex_skip_git_repo_check":    rc.CodexSkipRepoCheck,
	}
}

func allowedSubmitProviders(providers []string, cfg config.Config) []string {
	if len(cfg.AllowedSubmitProviders) == 0 {
		return append([]string(nil), providers...)
	}
	allowed := providerSet(cfg.AllowedSubmitProviders)
	out := make([]string, 0, len(providers))
	for _, provider := range providers {
		trimmed := strings.TrimSpace(provider)
		if trimmed == "" {
			continue
		}
		if _, ok := allowed[trimmed]; ok {
			out = append(out, trimmed)
		}
	}
	return out
}

func submitProviderAllowed(provider string, cfg config.Config) bool {
	provider = strings.TrimSpace(provider)
	if provider == "" || len(cfg.AllowedSubmitProviders) == 0 {
		return true
	}
	for _, allowed := range cfg.AllowedSubmitProviders {
		if strings.EqualFold(strings.TrimSpace(allowed), provider) {
			return true
		}
	}
	return false
}

func (s *Server) runtimeTranscriptionState(cfg config.Config, p transcription.Provider) map[string]any {
	rt := s.currentRuntimeTranscription()
	mode := strings.TrimSpace(firstNonEmptyString(rt.Mode, cfg.TranscriptionMode))
	state := map[string]any{
		"mode":            mode,
		"provider":        "",
		"endpoint":        "",
		"model":           "",
		"device":          "",
		"compute_type":    "",
		"language":        "",
		"local_command":   "",
		"timeout_seconds": "",
	}
	if p != nil {
		state["provider"] = p.Name()
		state["endpoint"] = p.Endpoint()
	}
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "remote":
		state["model"] = firstNonEmptyString(rt.Model, "gpt-4o-mini-transcribe")
		if strings.TrimSpace(state["endpoint"].(string)) == "" {
			state["endpoint"] = firstNonEmptyString(rt.BaseURL, "https://api.openai.com")
		}
	case "lmstudio":
		state["model"] = firstNonEmptyString(rt.Model, "whisper-large-v3-turbo")
		if strings.TrimSpace(state["endpoint"].(string)) == "" {
			state["endpoint"] = firstNonEmptyString(rt.BaseURL, "http://127.0.0.1:1234")
		}
		state["timeout_seconds"] = intString(rt.TimeoutSecond)
	case "faster_whisper":
		model := transcription.NormalizeFasterWhisperModel(rt.Model)
		device := firstNonEmptyString(rt.Device, "cpu")
		computeType := firstNonEmptyString(rt.ComputeType, "int8")
		state["model"] = model
		state["device"] = device
		state["compute_type"] = computeType
		state["language"] = strings.TrimSpace(rt.Language)
		state["timeout_seconds"] = intString(rt.TimeoutSecond)
	case "local":
		state["local_command"] = strings.TrimSpace(rt.LocalCommand)
		state["timeout_seconds"] = intString(rt.TimeoutSecond)
	default:
		state["timeout_seconds"] = ""
	}
	return state
}

func intString(v int) string {
	if v <= 0 {
		return ""
	}
	return strconv.Itoa(v)
}

func ptrString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func ptrInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func (s *Server) sttMode() string {
	sttProvider := s.currentSTT()
	if sttProvider == nil {
		return ""
	}
	return sttProvider.Mode()
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	list, err := s.store.ListSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func (s *Server) handleAudioDevices(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	if s.currentConfig().CaptureSettingsLocked {
		http.Error(w, "capture settings are locked by policy", http.StatusLocked)
		return
	}
	if s.privilegedCapture == nil {
		writeJSON(w, map[string]any{"state": nil, "devices": []audio.Device{}})
		return
	}
	if r.Method == http.MethodPost {
		var req audioDevicesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		s.privilegedCapture.SetAudioDevices(req.Devices)
	}
	writeJSON(w, map[string]any{"state": s.privilegedCapture.AudioState(), "devices": s.privilegedCapture.AudioDevices()})
}

func (s *Server) handleAudioConfig(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	if s.currentConfig().CaptureSettingsLocked {
		http.Error(w, "capture settings are locked by policy", http.StatusLocked)
		return
	}
	if s.privilegedCapture == nil {
		http.Error(w, "audio controller unavailable", http.StatusServiceUnavailable)
		return
	}
	var req audioConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	state := s.privilegedCapture.ConfigureAudio(audio.Config{
		Mode:          req.Mode,
		InputDeviceID: req.InputDeviceID,
		Muted:         req.Muted,
		Paused:        req.Paused,
		LevelMin:      req.LevelMin,
		LevelMax:      req.LevelMax,
	})
	if err := s.persistOperatorState(s.currentConfig()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"state": state, "devices": s.privilegedCapture.AudioDevices()})
}

func (s *Server) handleAudioLevel(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	if s.privilegedCapture == nil {
		http.Error(w, "audio controller unavailable", http.StatusServiceUnavailable)
		return
	}
	var req audioLevelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	state := s.privilegedCapture.UpdateAudioLevel(req.Level)
	writeJSON(w, map[string]any{"state": state})
}

func (s *Server) handleCaptureSource(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	if s.currentConfig().CaptureSettingsLocked {
		http.Error(w, "capture settings are locked by policy", http.StatusLocked)
		return
	}
	var req captureSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	source := strings.TrimSpace(req.Source)
	status := strings.TrimSpace(req.Status)
	if source == "" || status == "" {
		http.Error(w, "source and status are required", http.StatusBadRequest)
		return
	}
	switch status {
	case "available", "degraded", "unavailable":
	default:
		http.Error(w, "unsupported source status", http.StatusBadRequest)
		return
	}
	s.privilegedCapture.SetSourceStatus(source, status, req.Reason)
	writeJSON(w, map[string]any{
		"capture_sources":      s.privilegedCapture.SourceStatuses(),
		"reduced_capabilities": s.privilegedCapture.ReducedCapabilities(),
	})
}

func (s *Server) handleAttemptLog(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	attemptID := strings.TrimSpace(r.URL.Query().Get("attempt_id"))
	if attemptID == "" {
		http.Error(w, "attempt_id is required", http.StatusBadRequest)
		return
	}
	attempt, ok := s.submitAttemptByID(attemptID)
	if !ok {
		http.Error(w, "attempt not found", http.StatusNotFound)
		return
	}
	ref := strings.TrimSpace(attempt.ExecutionRef)
	if ref == "" {
		ref = strings.TrimSpace(attempt.Ref)
	}
	logPath, err := resolveLocalLogPath(ref)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	offset := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		v, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || v < 0 {
			http.Error(w, "offset must be a non-negative integer", http.StatusBadRequest)
			return
		}
		offset = v
	}
	limit := 12000
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		v, parseErr := strconv.Atoi(raw)
		if parseErr != nil || v <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		if v > 128000 {
			v = 128000
		}
		limit = v
	}

	tail := parseAttemptLogBool(r.URL.Query().Get("tail"))
	chunk, nextOffset, eof, truncatedHead, err := readLogChunk(logPath, offset, limit, tail)
	if err != nil {
		http.Error(w, "read log chunk failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{
		"attempt_id":     attempt.AttemptID,
		"status":         attempt.Status,
		"path":           logPath,
		"offset":         offset,
		"next_offset":    nextOffset,
		"eof":            eof,
		"truncated_head": truncatedHead,
		"chunk":          chunk,
	})
}

func (s *Server) handleCancelSubmitAttempt(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req cancelSubmitAttemptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	authCtx := s.requestAuthContext(r)
	attempt, found, err := s.cancelSubmitAttempt(req.AttemptID, s.requestSource(r), authCtx.Actor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !found {
		http.Error(w, "attempt not found", http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{
		"attempt":            attempt,
		"submit_queue_state": s.submitQueueState(),
		"submit_attempts":    s.submitAttemptsSnapshot(),
	})
}

func (s *Server) handleRerunSubmitAttempt(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req rerunSubmitAttemptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	attemptID := strings.TrimSpace(req.AttemptID)
	if attemptID == "" {
		http.Error(w, "attempt_id is required", http.StatusBadRequest)
		return
	}
	original, found := s.submitAttemptByID(attemptID)
	if !found {
		http.Error(w, "attempt not found", http.StatusNotFound)
		return
	}
	if strings.TrimSpace(original.SessionID) == "" {
		http.Error(w, "attempt cannot be rerun without a saved session package", http.StatusBadRequest)
		return
	}
	provider := strings.TrimSpace(original.Provider)
	if provider == "" {
		provider = s.resolveProvider("")
	} else {
		provider = canonicalProviderAlias(provider, s.agents.Names())
	}
	cfg := s.currentConfig()
	if !submitProviderAllowed(provider, cfg) {
		http.Error(w, "provider blocked by policy", http.StatusForbidden)
		return
	}
	if s.agents.IsRemote(provider) && !cfg.AllowRemoteSubmission {
		http.Error(w, "remote submission is disabled by policy", http.StatusForbidden)
		return
	}
	if s.agents.IsRemote(provider) {
		if endpoint := s.agents.Endpoint(provider); endpoint != "" && !endpointAllowedByPolicy(endpoint, cfg) {
			http.Error(w, "submission endpoint blocked by policy", http.StatusForbidden)
			return
		}
	}
	pkg, err := s.store.LoadLatestCanonicalPackage(original.SessionID)
	if err != nil {
		http.Error(w, "load canonical package failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if pkg == nil {
		http.Error(w, "approved package not found for this run", http.StatusNotFound)
		return
	}
	redactedPkg := redactPackageForTransmission(*pkg)
	rc := s.currentRuntimeCodex()
	intent := agents.NormalizeDeliveryIntent(agents.DeliveryIntent{
		Profile:            original.IntentProfile,
		InstructionText:    original.InstructionText,
		CustomInstructions: original.CustomInstructions,
	})
	providerPayload, err := agents.PreviewProviderPayloadWithConfig(provider, redactedPkg, rc.CodexModel, rc.ClaudeAPIModel, intent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	authCtx := s.requestAuthContext(r)
	attempt := s.enqueueSubmitJob(provider, redactedPkg, providerPayload, intent, s.requestSource(r), authCtx.Actor)
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]any{
		"attempt":            attempt,
		"attempt_id":         attempt.AttemptID,
		"provider":           attempt.Provider,
		"intent_profile":     attempt.IntentProfile,
		"intent_label":       attempt.IntentLabel,
		"session_id":         attempt.SessionID,
		"status":             attempt.Status,
		"execution_mode":     attempt.Mode,
		"queue_position":     attempt.QueuePos,
		"source":             attempt.Source,
		"submit_queue_state": s.submitQueueState(),
		"submit_attempts":    s.submitAttemptsSnapshot(),
	})
}

func (s *Server) handleOpenLastLog(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	curr := s.sessions.Current()
	if curr == nil {
		http.Error(w, "no active session", http.StatusBadRequest)
		return
	}
	logPath := ""
	if latest, ok := s.latestSubmitAttemptLogPath(); ok {
		logPath = latest
	} else {
		resolved, err := resolveLocalLogPath(curr.VersionReference)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		logPath = resolved
	}
	if err := openLocalPath(logPath); err != nil {
		http.Error(w, "open log failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	_ = s.audit.Write(audit.Event{Type: "submission_log_opened", SessionID: curr.ID, Details: map[string]any{"path": logPath}})
	writeJSON(w, map[string]any{"ok": true, "path": logPath})
}

func (s *Server) latestSubmitAttemptLogPath() (string, bool) {
	attempts := s.submitAttemptsSnapshot()
	for _, attempt := range attempts {
		ref := strings.TrimSpace(attempt.ExecutionRef)
		if ref == "" {
			ref = strings.TrimSpace(attempt.Ref)
		}
		if ref == "" {
			continue
		}
		path, err := resolveLocalLogPath(ref)
		if err == nil {
			return path, true
		}
	}
	return "", false
}

func (s *Server) handleKillCapture(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	s.privilegedCapture.Stop()
	s.privilegedCapture.SetSourceStatus("screen", "unavailable", "capture killed by operator")
	s.privilegedCapture.SetSourceStatus("microphone", "unavailable", "capture killed by operator")
	if curr := s.sessions.Current(); curr != nil {
		_ = s.sessions.Pause()
		paused := s.sessions.Current()
		if err := s.store.UpsertSession(paused); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = s.audit.Write(audit.Event{Type: "capture_killed", SessionID: curr.ID})
	}
	writeJSON(w, map[string]any{"capture_state": s.privilegedCapture.State()})
}

func (s *Server) handleCompanionPointer(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	curr := s.sessions.Current()
	if curr == nil {
		http.Error(w, "no active session", http.StatusBadRequest)
		return
	}
	var evt companion.PointerEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if evt.SessionID == "" {
		evt.SessionID = curr.ID
	}
	if evt.SessionID != curr.ID {
		http.Error(w, "session mismatch", http.StatusBadRequest)
		return
	}
	if evt.Window == "" {
		evt.Window = curr.TargetWindow
	}
	if curr.TargetURL != "" && evt.URL != "" && !sameHost(curr.TargetURL, evt.URL) {
		http.Error(w, "pointer event outside session target scope", http.StatusForbidden)
		return
	}
	evt = sanitizeCompanionPointerEvent(evt)
	s.privilegedCapture.AddPointer(evt)
	sessionURL := strings.TrimSpace(evt.URL)
	if sessionURL != "" && !redaction.URLAllowed(sessionURL, s.currentConfig().OutboundAllowlist, s.currentConfig().BlockedTargets) {
		sessionURL = ""
	}
	if updated, changed, err := s.sessions.UpdateTargetContext(strings.TrimSpace(evt.Window), sessionURL); err == nil && changed && updated != nil {
		if err := s.store.UpsertSession(updated); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.auditWriteRequest(r, audit.Event{
			Type:      "session_target_updated",
			SessionID: updated.ID,
			Details: map[string]any{
				"target_window": updated.TargetWindow,
				"target_url":    updated.TargetURL,
			},
		})
	}
	s.privilegedCapture.SetSourceStatus("companion", "available", "browser companion attached")
	s.latency.observe("pointer_ingest_ms", time.Since(startedAt))
	writeJSON(w, map[string]any{"ok": true})
}

func sanitizeCompanionPointerEvent(evt companion.PointerEvent) companion.PointerEvent {
	evt.EventType = truncateCompanionField(strings.ToLower(evt.EventType), 32)
	evt.Window = redaction.Text(truncateCompanionField(evt.Window, 160))
	evt.URL = sanitizeCompanionURL(evt.URL)
	evt.Route = truncateCompanionField(evt.Route, 256)
	evt.TargetTag = truncateCompanionField(strings.ToLower(evt.TargetTag), 32)
	evt.TargetID = redaction.Text(truncateCompanionField(evt.TargetID, 120))
	evt.TargetTestID = redaction.Text(truncateCompanionField(evt.TargetTestID, 120))
	evt.TargetRole = truncateCompanionField(strings.ToLower(evt.TargetRole), 64)
	evt.TargetLabel = redaction.Text(truncateCompanionField(evt.TargetLabel, 200))
	evt.TargetSelector = truncateCompanionField(evt.TargetSelector, 240)
	evt.DOM = sanitizeDOMInspection(evt.DOM)
	evt.Console = sanitizeConsoleEntries(evt.Console)
	evt.Network = sanitizeNetworkEntries(evt.Network)
	evt.Key = truncateCompanionField(evt.Key, 64)
	evt.Code = truncateCompanionField(evt.Code, 64)
	evt.Modifiers = sanitizeReplayModifiers(evt.Modifiers)
	evt.InputType = truncateCompanionField(strings.ToLower(evt.InputType), 32)
	evt.MouseButton = max(0, min(evt.MouseButton, 2))
	evt.ClickCount = max(0, min(evt.ClickCount, 3))
	evt.Value, evt.ValueRedacted = sanitizeCompanionInputValue(evt)
	if evt.Value == "" {
		evt.ValueCaptured = false
	}
	if evt.ValueCaptured && evt.Value != "" {
		evt.ValueRedacted = false
	}
	return evt
}

func truncateCompanionField(value string, max int) string {
	trimmed := strings.TrimSpace(value)
	if max <= 0 || len(trimmed) <= max {
		return trimmed
	}
	return trimmed[:max]
}

func sanitizeCompanionURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return truncateCompanionField(trimmed, 512)
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return truncateCompanionField(parsed.String(), 512)
}

func sanitizeDOMInspection(in *session.DOMInspection) *session.DOMInspection {
	if in == nil {
		return nil
	}
	out := &session.DOMInspection{
		Tag:         truncateCompanionField(strings.ToLower(in.Tag), 32),
		ID:          redaction.Text(truncateCompanionField(in.ID, 120)),
		TestID:      redaction.Text(truncateCompanionField(in.TestID, 120)),
		Role:        truncateCompanionField(strings.ToLower(in.Role), 64),
		Label:       redaction.Text(truncateCompanionField(in.Label, 200)),
		Selector:    truncateCompanionField(in.Selector, 240),
		TextPreview: redaction.Text(truncateCompanionField(in.TextPreview, 240)),
		OuterHTML:   redaction.Text(truncateCompanionField(in.OuterHTML, 400)),
	}
	if len(in.Attributes) > 0 {
		out.Attributes = map[string]string{}
		count := 0
		for k, v := range in.Attributes {
			if count >= 8 {
				break
			}
			key := truncateCompanionField(strings.ToLower(k), 64)
			if key == "" {
				continue
			}
			out.Attributes[key] = redaction.Text(truncateCompanionField(v, 160))
			count++
		}
	}
	return out
}

func sanitizeConsoleEntries(entries []session.ConsoleEntry) []session.ConsoleEntry {
	if len(entries) == 0 {
		return nil
	}
	limit := len(entries)
	if limit > 8 {
		limit = 8
	}
	out := make([]session.ConsoleEntry, 0, limit)
	for i := len(entries) - limit; i < len(entries); i++ {
		entry := entries[i]
		out = append(out, session.ConsoleEntry{
			Level:     truncateCompanionField(strings.ToLower(entry.Level), 16),
			Message:   redaction.Text(truncateCompanionField(entry.Message, 400)),
			Timestamp: entry.Timestamp,
		})
	}
	return out
}

func sanitizeNetworkEntries(entries []session.NetworkEntry) []session.NetworkEntry {
	if len(entries) == 0 {
		return nil
	}
	limit := len(entries)
	if limit > 8 {
		limit = 8
	}
	out := make([]session.NetworkEntry, 0, limit)
	for i := len(entries) - limit; i < len(entries); i++ {
		entry := entries[i]
		status := entry.Status
		if status < 0 {
			status = 0
		}
		out = append(out, session.NetworkEntry{
			Kind:       truncateCompanionField(strings.ToLower(entry.Kind), 16),
			Method:     truncateCompanionField(strings.ToUpper(entry.Method), 16),
			URL:        sanitizeCompanionURL(entry.URL),
			Status:     status,
			OK:         entry.OK,
			DurationMS: max(0, entry.DurationMS),
			Timestamp:  entry.Timestamp,
		})
	}
	return out
}

func sanitizeReplayModifiers(modifiers []string) []string {
	if len(modifiers) == 0 {
		return nil
	}
	out := make([]string, 0, len(modifiers))
	seen := map[string]struct{}{}
	for _, modifier := range modifiers {
		value := truncateCompanionField(strings.ToLower(modifier), 16)
		switch value {
		case "alt", "control", "meta", "shift":
		default:
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func sanitizeCompanionInputValue(evt companion.PointerEvent) (string, bool) {
	if !evt.ValueCaptured {
		return "", evt.ValueRedacted
	}
	classification := classifyReplayInputSensitivity(evt)
	value := truncateCompanionField(strings.ReplaceAll(evt.Value, "\n", " "), 400)
	switch classification {
	case "password":
		return "", true
	case "token":
		return maskReplayTokenValue(value), false
	case "card":
		return maskReplayCardValue(value), false
	}
	return value, false
}

func classifyReplayInputSensitivity(evt companion.PointerEvent) string {
	inputType := strings.ToLower(strings.TrimSpace(evt.InputType))
	haystack := strings.ToLower(strings.Join([]string{
		evt.TargetID,
		evt.TargetTestID,
		evt.TargetLabel,
	}, " "))
	switch {
	case inputType == "password" || inputType == "hidden" || inputType == "file":
		return "password"
	case strings.Contains(haystack, "password"),
		strings.Contains(haystack, "passcode"),
		strings.Contains(haystack, "secret"),
		strings.Contains(haystack, "otp"),
		strings.Contains(haystack, "one-time"),
		strings.Contains(haystack, "mfa"),
		strings.Contains(haystack, "2fa"),
		strings.Contains(haystack, "pin"),
		strings.Contains(haystack, "cvv"),
		strings.Contains(haystack, "cvc"),
		strings.Contains(haystack, "ssn"):
		return "password"
	case strings.Contains(haystack, "token"),
		strings.Contains(haystack, "credential"),
		strings.Contains(haystack, "auth"),
		strings.Contains(haystack, "api key"),
		strings.Contains(haystack, "apikey"),
		strings.Contains(haystack, "bearer"),
		strings.Contains(haystack, "access key"),
		strings.Contains(haystack, "secret key"),
		strings.Contains(haystack, "client secret"):
		return "token"
	case strings.Contains(haystack, "card"),
		strings.Contains(haystack, "credit"),
		strings.Contains(haystack, "debit"),
		strings.Contains(haystack, "amex"),
		strings.Contains(haystack, "visa"),
		strings.Contains(haystack, "mastercard"),
		strings.Contains(haystack, "discover"),
		strings.Contains(haystack, "pan"):
		return "card"
	default:
		return ""
	}
}

func maskReplayTokenValue(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	if len(text) <= 8 {
		return "[masked token]"
	}
	return text[:4] + "..." + text[len(text)-4:]
}

func maskReplayCardValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	var digits strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	clean := digits.String()
	if clean == "" {
		return ""
	}
	last4 := clean
	if len(last4) > 4 {
		last4 = last4[len(last4)-4:]
	}
	return "**** **** **** " + last4
}

func cloneReplayBundle(in *session.ReplayBundle) *session.ReplayBundle {
	if in == nil {
		return nil
	}
	out := *in
	out.PointerPath = append([]session.PointerSample(nil), in.PointerPath...)
	out.Steps = cloneReplaySteps(in.Steps)
	out.DOM = cloneDOMInspection(in.DOM)
	out.Console = append([]session.ConsoleEntry(nil), in.Console...)
	out.Network = append([]session.NetworkEntry(nil), in.Network...)
	out.Exports = append([]session.ReplayExport(nil), in.Exports...)
	return &out
}

func cloneReplaySteps(in []session.ReplayStep) []session.ReplayStep {
	if len(in) == 0 {
		return nil
	}
	out := make([]session.ReplayStep, len(in))
	for i, step := range in {
		out[i] = step
		out[i].Modifiers = append([]string(nil), step.Modifiers...)
		out[i].DOM = cloneDOMInspection(step.DOM)
	}
	return out
}

func cloneDOMInspection(in *session.DOMInspection) *session.DOMInspection {
	if in == nil {
		return nil
	}
	out := *in
	if len(in.Attributes) > 0 {
		out.Attributes = make(map[string]string, len(in.Attributes))
		for k, v := range in.Attributes {
			out.Attributes[k] = v
		}
	}
	return &out
}

func (s *Server) handlePurgeSession(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	curr := s.sessions.Current()
	if curr == nil {
		http.Error(w, "no active session", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteSessionByID(curr.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	deletedArtifacts, _ := s.artifacts.RemoveBySession(curr.ID)
	s.privilegedCapture.Stop()
	s.privilegedCapture.SetSourceStatus("screen", "unavailable", "no active session")
	s.privilegedCapture.SetSourceStatus("microphone", "unavailable", "no active session")
	s.privilegedCapture.SetSourceStatus("companion", "unavailable", "no active session")
	s.sessions.DropCurrent()
	_ = s.audit.Write(audit.Event{Type: "session_purged", SessionID: curr.ID, Details: map[string]any{"artifacts_deleted": deletedArtifacts}})
	writeJSON(w, map[string]any{"ok": true, "session_id": curr.ID, "artifacts_deleted": deletedArtifacts})
}

func (s *Server) handlePurgeAll(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	if err := s.store.PurgeAll(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	deletedArtifacts, _ := s.artifacts.PurgeAll()
	s.privilegedCapture.Stop()
	s.privilegedCapture.SetSourceStatus("screen", "unavailable", "no active session")
	s.privilegedCapture.SetSourceStatus("microphone", "unavailable", "no active session")
	s.privilegedCapture.SetSourceStatus("companion", "unavailable", "no active session")
	s.sessions.ResetAll()
	_ = s.audit.Write(audit.Event{Type: "all_data_purged", Details: map[string]any{"artifacts_deleted": deletedArtifacts}})
	writeJSON(w, map[string]any{"ok": true, "artifacts_deleted": deletedArtifacts})
}

func (s *Server) handleConfigExport(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	cfg := s.currentConfig()
	configPath, configText := userconfig.Export(cfg, s.currentRuntimeState())
	writeJSON(w, map[string]any{
		"config":      config.ExportPublic(cfg),
		"profiles":    config.Profiles(),
		"config_path": configPath,
		"config_toml": configText,
	})
}

func (s *Server) handleAuditExport(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	if s.audit == nil {
		http.Error(w, "audit unavailable", http.StatusServiceUnavailable)
		return
	}
	limit := 500
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 5000 {
			limit = parsed
		}
	}
	events, err := s.audit.Export(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"events": events,
		"count":  len(events),
	})
}

func (s *Server) handleConfigImport(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	baseCfg := s.currentConfig()
	if baseCfg.ConfigLocked {
		http.Error(w, "config is locked by policy", http.StatusLocked)
		return
	}
	var req configImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	cfg := baseCfg
	prevMode := strings.TrimSpace(baseCfg.TranscriptionMode)
	if req.Profile != "" {
		profile, ok := config.Profile(req.Profile)
		if !ok {
			http.Error(w, "unknown config profile", http.StatusBadRequest)
			return
		}
		cfg = config.ApplyPublic(cfg, profile)
	}
	if req.Config != nil {
		cfg = config.ApplyPublic(cfg, *req.Config)
	}
	if err := config.Validate(cfg); err != nil {
		http.Error(w, "invalid config: "+err.Error(), http.StatusBadRequest)
		return
	}
	if s.autoStart != nil {
		if _, err := s.autoStart.Ensure(cfg.AutoStartEnabled); err != nil {
			http.Error(w, "invalid auto-start configuration: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	// keep control-sensitive fields immutable via import.
	if cfg.ControlToken == "" {
		cfg.ControlToken = s.currentConfig().ControlToken
	}
	s.updateRuntimeState(func(state *operatorstate.State) {
		state.System.AutoStartEnabled = cfg.AutoStartEnabled
	})
	if strings.TrimSpace(cfg.TranscriptionMode) != prevMode {
		s.updateRuntimeState(func(state *operatorstate.State) {
			state.RuntimeTranscription.Mode = strings.TrimSpace(cfg.TranscriptionMode)
		})
		provider := s.buildTranscriptionProvider(cfg)
		s.setSTTProvider(provider)
		cfg.TranscriptionProvider = provider.Name()
	}
	s.updateConfig(cfg)
	if err := s.persistOperatorState(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.audit.Write(audit.Event{Type: "config_imported", Details: map[string]any{"profile": req.Profile}})
	writeJSON(w, map[string]any{"ok": true, "config": config.ExportPublic(cfg)})
}

func (s *Server) handleRuntimeTranscription(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	cfg := s.currentConfig()
	if cfg.CaptureSettingsLocked {
		http.Error(w, "capture settings are locked by policy", http.StatusLocked)
		return
	}
	if cfg.ConfigLocked {
		http.Error(w, "config is locked by policy", http.StatusLocked)
		return
	}
	var req runtimeTranscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(cfg.TranscriptionMode))
	}
	if err := validateTranscriptionRuntimeRequest(mode, req); err != nil {
		http.Error(w, "invalid transcription runtime: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg.TranscriptionMode = mode
	if err := config.Validate(cfg); err != nil {
		http.Error(w, "invalid config: "+err.Error(), http.StatusBadRequest)
		return
	}

	nextRuntime := s.updateRuntimeState(func(state *operatorstate.State) {
		rt := &state.RuntimeTranscription
		rt.Mode = mode
		switch mode {
		case "remote":
			rt.BaseURL = strings.TrimSpace(ptrString(req.BaseURL))
			rt.Model = firstNonEmptyString(strings.TrimSpace(ptrString(req.Model)), "gpt-4o-mini-transcribe")
			rt.Device = ""
			rt.ComputeType = ""
			rt.Language = ""
			rt.LocalCommand = ""
			rt.TimeoutSecond = 0
		case "lmstudio":
			rt.BaseURL = strings.TrimSpace(ptrString(req.BaseURL))
			rt.Model = firstNonEmptyString(strings.TrimSpace(ptrString(req.Model)), "whisper-large-v3-turbo")
			rt.Device = ""
			rt.ComputeType = ""
			rt.Language = ""
			rt.LocalCommand = ""
			rt.TimeoutSecond = ptrInt(req.TimeoutSecond)
		case "local":
			rt.BaseURL = ""
			rt.Model = ""
			rt.Device = ""
			rt.ComputeType = ""
			rt.Language = ""
			rt.LocalCommand = strings.TrimSpace(ptrString(req.LocalCommand))
			rt.TimeoutSecond = ptrInt(req.TimeoutSecond)
		case "faster_whisper":
			rt.BaseURL = ""
			rt.Model = transcription.NormalizeFasterWhisperModel(ptrString(req.Model))
			rt.Device = firstNonEmptyString(strings.TrimSpace(ptrString(req.Device)), "cpu")
			rt.ComputeType = firstNonEmptyString(strings.TrimSpace(ptrString(req.ComputeType)), "int8")
			rt.Language = strings.TrimSpace(ptrString(req.Language))
			rt.LocalCommand = ""
			rt.TimeoutSecond = ptrInt(req.TimeoutSecond)
		}
	})

	provider := s.buildTranscriptionProvider(cfg)
	s.setSTTProvider(provider)
	cfg.TranscriptionProvider = provider.Name()
	s.updateConfig(cfg)
	if err := s.persistOperatorState(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = s.audit.Write(audit.Event{
		Type: "runtime_transcription_updated",
		Details: map[string]any{
			"mode":     mode,
			"provider": provider.Name(),
			"endpoint": provider.Endpoint(),
		},
	})
	writeJSON(w, map[string]any{
		"ok":                    true,
		"runtime_transcription": s.runtimeTranscriptionState(config.Config{TranscriptionMode: nextRuntime.RuntimeTranscription.Mode}, provider),
	})
}

func (s *Server) handleRuntimeTranscriptionHealth(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	cfg := s.currentConfig()
	provider := s.currentSTT()
	if provider == nil {
		http.Error(w, "transcription provider not configured", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	err := transcription.HealthCheckProvider(ctx, provider)
	status := "ok"
	message := "transcription provider reachable"
	if err != nil {
		status = "error"
		message = err.Error()
	}
	writeJSON(w, map[string]any{
		"status":                status,
		"message":               message,
		"mode":                  cfg.TranscriptionMode,
		"provider":              provider.Name(),
		"endpoint":              provider.Endpoint(),
		"runtime_transcription": s.runtimeTranscriptionState(cfg, provider),
	})
}

func (s *Server) handleRuntimeCodex(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	cfg := s.currentConfig()
	if cfg.ConfigLocked {
		http.Error(w, "config is locked by policy", http.StatusLocked)
		return
	}
	var req runtimeCodexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.updateRuntimeState(func(state *operatorstate.State) {
		rc := &state.RuntimeCodex
		if strings.TrimSpace(req.DefaultProvider) != "" {
			rc.DefaultProvider = canonicalProviderAlias(strings.TrimSpace(req.DefaultProvider), s.agents.Names())
		}
		rc.CLIAdapterCmd = strings.TrimSpace(req.CLIAdapterCmd)
		if req.CLITimeoutSeconds != nil {
			rc.CLITimeoutSeconds = max(0, *req.CLITimeoutSeconds)
		}
		rc.ClaudeCLIAdapterCmd = strings.TrimSpace(req.ClaudeCLIAdapterCmd)
		if req.ClaudeCLITimeoutSeconds != nil {
			rc.ClaudeCLITimeoutSeconds = max(0, *req.ClaudeCLITimeoutSeconds)
		}
		rc.OpenCodeCLIAdapterCmd = strings.TrimSpace(req.OpenCodeCLIAdapterCmd)
		if req.OpenCodeCLITimeoutSecs != nil {
			rc.OpenCodeCLITimeoutSecs = max(0, *req.OpenCodeCLITimeoutSecs)
		}
		if strings.TrimSpace(req.SubmitExecMode) != "" {
			rc.SubmitExecMode = normalizeSubmitExecutionMode(req.SubmitExecMode)
		}
		rc.CodexWorkdir = strings.TrimSpace(req.CodexWorkdir)
		rc.CodexOutputDir = strings.TrimSpace(req.CodexOutputDir)
		rc.CodexSandbox = strings.TrimSpace(req.CodexSandbox)
		rc.CodexApproval = strings.TrimSpace(req.CodexApproval)
		rc.CodexProfile = strings.TrimSpace(req.CodexProfile)
		rc.CodexModel = strings.TrimSpace(req.CodexModel)
		rc.CodexReasoning = strings.TrimSpace(req.CodexReasoning)
		rc.OpenAIBaseURL = strings.TrimSpace(req.OpenAIBaseURL)
		if req.CodexAPITimeoutSeconds != nil {
			rc.CodexAPITimeoutSeconds = max(0, *req.CodexAPITimeoutSeconds)
		}
		rc.OpenAIOrgID = strings.TrimSpace(req.OpenAIOrgID)
		rc.OpenAIProjectID = strings.TrimSpace(req.OpenAIProjectID)
		rc.ClaudeAPIModel = strings.TrimSpace(req.ClaudeAPIModel)
		rc.AnthropicBaseURL = strings.TrimSpace(req.AnthropicBaseURL)
		if req.ClaudeAPITimeoutSeconds != nil {
			rc.ClaudeAPITimeoutSeconds = max(0, *req.ClaudeAPITimeoutSeconds)
		}
		rc.PostSubmitRebuild = strings.TrimSpace(req.PostSubmitRebuild)
		rc.PostSubmitVerify = strings.TrimSpace(req.PostSubmitVerify)
		if req.PostSubmitTimeout != nil {
			rc.PostSubmitTimeout = max(0, *req.PostSubmitTimeout)
		}
		if strings.TrimSpace(req.DeliveryIntentProfile) != "" {
			rc.DeliveryIntentProfile = strings.TrimSpace(req.DeliveryIntentProfile)
		}
		rc.ImplementChangesPrompt = strings.TrimSpace(req.ImplementChangesPrompt)
		rc.DraftPlanPrompt = strings.TrimSpace(req.DraftPlanPrompt)
		rc.CreateJiraTicketsPrompt = strings.TrimSpace(req.CreateJiraTicketsPrompt)
		if req.CodexSkipRepoCheck != nil {
			rc.CodexSkipRepoCheck = *req.CodexSkipRepoCheck
		}
		*rc = operatorstate.NormalizeRuntimeCodexDefaults(*rc)
	})
	s.applyRuntimeStateToProcess()
	s.agents = s.buildAgentRegistry()
	_ = s.audit.Write(audit.Event{
		Type: "runtime_codex_updated",
		Details: map[string]any{
			"default_provider":      s.resolveProvider(""),
			"has_cli_adapter_cmd":   strings.TrimSpace(req.CLIAdapterCmd) != "",
			"has_claude_cli_cmd":    strings.TrimSpace(req.ClaudeCLIAdapterCmd) != "",
			"has_opencode_cli_cmd":  strings.TrimSpace(req.OpenCodeCLIAdapterCmd) != "",
			"has_anthropic_base":    strings.TrimSpace(req.AnthropicBaseURL) != "",
			"workdir":               strings.TrimSpace(req.CodexWorkdir),
			"sandbox":               strings.TrimSpace(req.CodexSandbox),
			"approval_policy":       strings.TrimSpace(req.CodexApproval),
			"submit_execution_mode": s.submitExecutionMode(),
			"has_post_rebuild":      strings.TrimSpace(req.PostSubmitRebuild) != "",
			"has_post_verify":       strings.TrimSpace(req.PostSubmitVerify) != "",
		},
	})
	if err := s.persistOperatorState(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"ok":            true,
		"runtime_codex": s.runtimeAgentState(),
	})
}

func (s *Server) handleRuntimeCodexOptions(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	opts, err := codexOptionsFetcher(r.Context())
	if err != nil {
		http.Error(w, "failed to load codex runtime options: "+err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, opts)
}

func (s *Server) handleFSList(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	reqPath := strings.TrimSpace(r.URL.Query().Get("path"))
	basePath := reqPath
	if basePath == "" {
		if wd, err := os.Getwd(); err == nil {
			basePath = wd
		}
	}
	if basePath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			basePath = home
		}
	}
	if basePath == "" {
		http.Error(w, "unable to resolve base path", http.StatusInternalServerError)
		return
	}
	clean := filepath.Clean(basePath)
	abs, err := filepath.Abs(clean)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		http.Error(w, "path is not a directory", http.StatusBadRequest)
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		http.Error(w, "unable to read directory", http.StatusInternalServerError)
		return
	}
	type dirEntry struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	dirs := make([]dirEntry, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		dirs = append(dirs, dirEntry{
			Name: name,
			Path: filepath.Join(abs, name),
		})
	}
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	parent := filepath.Dir(abs)
	if parent == abs {
		parent = ""
	}
	writeJSON(w, map[string]any{
		"current_path": abs,
		"parent_path":  parent,
		"dirs":         dirs,
	})
}

func (s *Server) handleFSPickDir(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	path, err := pickDirectoryNative()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"path": path})
}

func readLogChunk(path string, offset int64, limit int, tail bool) (string, int64, bool, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", offset, true, false, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", offset, true, false, err
	}
	size := info.Size()
	if offset < 0 {
		offset = 0
	}
	if tail {
		if int64(limit) < size {
			offset = size - int64(limit)
		} else {
			offset = 0
		}
	} else if offset > size {
		offset = size
	}
	if limit <= 0 {
		limit = 12000
	}

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return "", offset, true, false, err
	}
	buf, err := io.ReadAll(io.LimitReader(f, int64(limit)))
	if err != nil {
		return "", offset, true, false, err
	}
	next := offset + int64(len(buf))
	return string(buf), next, next >= size, offset > 0, nil
}

func parseAttemptLogBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func allowCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Knit-Token, X-Knit-Nonce, X-Knit-Timestamp")
}

func readMultipartFile(r *http.Request, field string) ([]byte, string, error) {
	f, hdr, err := r.FormFile(field)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 64<<20))
	if err != nil {
		return nil, "", err
	}
	return b, hdr.Filename, nil
}

type tempAudio struct {
	path string
}

func writeTempAudio(name string, payload []byte) (*tempAudio, error) {
	ext := strings.TrimPrefix(filepath.Ext(name), ".")
	if ext == "" {
		ext = "webm"
	}
	f, err := os.CreateTemp("", "knit-audio-*."+ext)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.Write(payload); err != nil {
		return nil, err
	}
	return &tempAudio{path: f.Name()}, nil
}

func (t *tempAudio) cleanup() {
	if t == nil || t.path == "" {
		return
	}
	_ = os.Remove(t.path)
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, fmt.Sprintf("encode json response: %v", err), http.StatusInternalServerError)
	}
}

func parseClipVideoMetadata(r *http.Request) *session.VideoMetadata {
	if r == nil {
		return nil
	}
	meta := &session.VideoMetadata{
		Scope:          strings.TrimSpace(r.FormValue("video_scope")),
		Window:         strings.TrimSpace(r.FormValue("video_window")),
		Codec:          strings.TrimSpace(r.FormValue("video_codec")),
		HasAudio:       parseFormBool(r.FormValue("video_has_audio")),
		PointerOverlay: parseFormBool(r.FormValue("video_pointer_overlay")),
		RegionX:        parseFormInt(r.FormValue("video_region_x")),
		RegionY:        parseFormInt(r.FormValue("video_region_y")),
		RegionW:        parseFormInt(r.FormValue("video_region_w")),
		RegionH:        parseFormInt(r.FormValue("video_region_h")),
		DurationMS:     parseFormInt64(r.FormValue("clip_duration_ms")),
	}
	if v := parseFormTime(r.FormValue("clip_started_at")); v != nil {
		meta.StartedAt = v
	}
	if v := parseFormTime(r.FormValue("clip_ended_at")); v != nil {
		meta.EndedAt = v
	}
	if meta.Scope == "" && meta.Window == "" && meta.Codec == "" &&
		!meta.HasAudio && !meta.PointerOverlay &&
		meta.RegionX == 0 && meta.RegionY == 0 && meta.RegionW == 0 && meta.RegionH == 0 &&
		meta.DurationMS == 0 && meta.StartedAt == nil && meta.EndedAt == nil {
		return nil
	}
	return meta
}

func parseFormBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseFormInt(v string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n
}

func parseFormInt64(v string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return n
}

func parseFormTime(v string) *time.Time {
	s := strings.TrimSpace(v)
	if s == "" {
		return nil
	}
	if ts, err := time.Parse(time.RFC3339Nano, s); err == nil {
		out := ts.UTC()
		return &out
	}
	if ts, err := time.Parse(time.RFC3339, s); err == nil {
		out := ts.UTC()
		return &out
	}
	return nil
}

func sameHost(a, b string) bool {
	ua, errA := url.Parse(a)
	ub, errB := url.Parse(b)
	if errA != nil || errB != nil {
		return false
	}
	hostA := canonicalScopeHost(ua)
	hostB := canonicalScopeHost(ub)
	if hostA == "" || hostB == "" {
		return false
	}
	return strings.EqualFold(hostA, hostB)
}

func canonicalScopeHost(u *url.URL) string {
	if u == nil {
		return ""
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return ""
	}
	if isLoopbackHost(host) {
		return "loopback"
	}
	port := strings.TrimSpace(u.Port())
	if port == "" {
		port = defaultPortForScheme(u.Scheme)
	}
	if port == "" {
		return strings.ToLower(host)
	}
	return strings.ToLower(net.JoinHostPort(host, port))
}

func isLoopbackHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	default:
		return false
	}
}

func defaultPortForScheme(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

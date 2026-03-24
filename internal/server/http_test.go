package server

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"knit/internal/agents"
	"knit/internal/audio"
	"knit/internal/audit"
	"knit/internal/capture"
	"knit/internal/companion"
	"knit/internal/config"
	"knit/internal/operatorstate"
	"knit/internal/platform"
	"knit/internal/privileged"
	"knit/internal/security"
	"knit/internal/session"
	"knit/internal/storage"
	"knit/internal/transcription"
)

func TestStateIncludesConfigLockFlag(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.ConfigLocked = true
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	locked, _ := payload["config_locked"].(bool)
	if !locked {
		t.Fatalf("expected config_locked=true in state payload")
	}
}

func TestStateIncludesEnterpriseControls(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.CaptureSettingsLocked = true
	cfg.ManagedDeploymentID = "fleet-a"
	cfg.VersionPin = "0.9.0"
	cfg.BuildID = "0.9.0"
	cfg.AllowedSubmitProviders = []string{"codex_cli"}
	cfg.SIEMLogPath = "siem/audit.jsonl"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if locked, _ := payload["capture_settings_locked"].(bool); !locked {
		t.Fatalf("expected capture_settings_locked=true")
	}
	if got := payload["managed_deployment_id"]; got != "fleet-a" {
		t.Fatalf("expected managed_deployment_id=fleet-a, got %v", got)
	}
	if got := payload["version_pin"]; got != "0.9.0" {
		t.Fatalf("expected version_pin=0.9.0, got %v", got)
	}
	if got := payload["current_version"]; got != "0.9.0" {
		t.Fatalf("expected current_version=0.9.0, got %v", got)
	}
	if enabled, _ := payload["update_check_on_startup"].(bool); !enabled {
		t.Fatalf("expected update_check_on_startup=true")
	}
	allowed, _ := payload["allowed_submit_providers"].([]any)
	if len(allowed) != 1 || allowed[0] != "codex_cli" {
		t.Fatalf("expected allowed provider list, got %#v", payload["allowed_submit_providers"])
	}
	if enabled, _ := payload["siem_log_enabled"].(bool); !enabled {
		t.Fatalf("expected siem_log_enabled=true")
	}
	runtimePlatform, _ := payload["runtime_platform"].(map[string]any)
	if runtimePlatform["host_target"] == "" || runtimePlatform["runtime_summary"] == "" {
		t.Fatalf("expected runtime_platform metadata, got %#v", runtimePlatform)
	}
}

func TestStateIncludesCommonSubmitFailureOutcomesForMainUI(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	srv.submitMu.Lock()
	srv.submitAttempts = []submitAttempt{
		{
			AttemptID:      "attempt-no-input",
			Status:         "submitted",
			OutcomeCode:    submitOutcomeNoInput,
			OutcomeTitle:   "No input",
			OutcomeMessage: "Knit submitted this run without any captured change requests or artifacts, so the coding agent had nothing to change.",
		},
		{
			AttemptID:      "attempt-trusted-directory",
			Status:         "failed",
			OutcomeCode:    submitOutcomeTrustedDir,
			OutcomeTitle:   "Trusted directory required",
			OutcomeMessage: "Go back to Capture, Review, and Send, open Settings, then check Workspace first. If the wrong repository is selected, choose the correct workspace for this project and rerun. If the workspace is already correct, open Settings > Agent and switch Sandbox to danger-full-access before rerunning. Workspace used: /tmp/ruddur.",
		},
		{
			AttemptID:      "attempt-wrong-workspace",
			Status:         "submitted",
			OutcomeCode:    submitOutcomeWrongWorkspace,
			OutcomeTitle:   "Wrong workspace",
			OutcomeMessage: "Go back to Capture, Review, and Send, open Settings > Workspace, and choose the repository that matches this request before rerunning. Workspace used: /tmp/ruddur.",
		},
		{
			AttemptID:      "attempt-read-only",
			Status:         "submitted",
			OutcomeCode:    submitOutcomeReadOnly,
			OutcomeTitle:   "Read-only",
			OutcomeMessage: "Go back to Capture, Review, and Send, open Settings > Agent, and switch Sandbox to danger-full-access before rerunning.",
		},
	}
	srv.submitMu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode state payload: %v", err)
	}
	attempts, _ := payload["submit_attempts"].([]any)
	if len(attempts) != 4 {
		t.Fatalf("expected 4 submit attempts, got %#v", payload["submit_attempts"])
	}

	want := map[string]struct {
		code     string
		title    string
		contains string
	}{
		"attempt-no-input":          {code: submitOutcomeNoInput, title: "No input", contains: "had nothing to change"},
		"attempt-trusted-directory": {code: submitOutcomeTrustedDir, title: "Trusted directory required", contains: "open Settings, then check Workspace first"},
		"attempt-wrong-workspace":   {code: submitOutcomeWrongWorkspace, title: "Wrong workspace", contains: "open Settings > Workspace"},
		"attempt-read-only":         {code: submitOutcomeReadOnly, title: "Read-only", contains: "open Settings > Agent"},
	}
	for _, raw := range attempts {
		attempt, _ := raw.(map[string]any)
		id, _ := attempt["attempt_id"].(string)
		exp, ok := want[id]
		if !ok {
			t.Fatalf("unexpected attempt in state payload: %#v", attempt)
		}
		if got := attempt["outcome_code"]; got != exp.code {
			t.Fatalf("expected outcome_code %q for %s, got %#v", exp.code, id, got)
		}
		if got := attempt["outcome_title"]; got != exp.title {
			t.Fatalf("expected outcome_title %q for %s, got %#v", exp.title, id, got)
		}
		message, _ := attempt["outcome_message"].(string)
		if !strings.Contains(message, exp.contains) {
			t.Fatalf("expected outcome_message for %s to contain %q, got %q", id, exp.contains, message)
		}
	}
}

func TestUpdateCheckEndpointReturnsLatestRelease(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.BuildID = "0.1.0"
	cfg.VersionPin = "0.1.0"
	srv := newTestServer(t, cfg)

	releaseAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","html_url":"https://github.com/chadsly/knit/releases/tag/v0.2.0"}`))
	}))
	defer releaseAPI.Close()

	srv.updateHTTPClient = releaseAPI.Client()
	srv.updateReleaseAPIURL = releaseAPI.URL

	req := httptest.NewRequest(http.MethodGet, "/api/update/check", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload updateCheckResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != "update_available" {
		t.Fatalf("expected update_available status, got %#v", payload)
	}
	if !payload.UpdateAvailable {
		t.Fatalf("expected update_available=true, got %#v", payload)
	}
	if payload.CurrentVersion != "0.1.0" {
		t.Fatalf("expected current version 0.1.0, got %#v", payload)
	}
	if payload.LatestVersion != "0.2.0" {
		t.Fatalf("expected latest version 0.2.0, got %#v", payload)
	}
	if payload.ReleaseURL != "https://github.com/chadsly/knit/releases/tag/v0.2.0" {
		t.Fatalf("expected release URL in payload, got %#v", payload)
	}
}

func TestHealthIncludesRuntimePlatformMetadata(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.BuildID = "build-123"
	cfg.VersionPin = "build-123"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected ok=true")
	}
	runtimePlatform, _ := payload["runtime_platform"].(map[string]any)
	if runtimePlatform["host_target"] == "" || runtimePlatform["runtime_summary"] == "" {
		t.Fatalf("expected runtime platform metadata, got %#v", runtimePlatform)
	}
}

func TestCompanionScriptUsesConfiguredPointerSamplingRate(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.PointerSampleHz = 60
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/companion.js", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "const hz = Number(state && state.pointer_sample_hz || 30);") {
		t.Fatalf("expected companion script to read pointer_sample_hz from state")
	}
	if !strings.Contains(body, "moveThrottleMS = Math.max(8, Math.round(1000 / hz));") {
		t.Fatalf("expected companion script to derive move throttle from configured sample hz")
	}
	if !strings.Contains(body, "if (now - lastMoveSent < moveThrottleMS) return;") {
		t.Fatalf("expected companion script to throttle continuous pointer moves")
	}
}

func TestStateRestoresPersistedOperatorSettingsAndHistory(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.DataDir = t.TempDir()
	cfg.SQLitePath = filepath.Join(cfg.DataDir, "test.db")

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	encryptor, err := security.NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	store, err := storage.NewSQLiteStore(cfg.SQLitePath, encryptor)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if err := store.SaveOperatorState(&operatorstate.State{
		Version: 1,
		RuntimeCodex: operatorstate.RuntimeCodex{
			DefaultProvider: "codex_cli",
			CodexWorkdir:    "/tmp/restored-workdir",
		},
		RuntimeTranscription: operatorstate.RuntimeTranscription{
			Mode:        "faster_whisper",
			Model:       "small",
			Device:      "cpu",
			ComputeType: "int8",
		},
		Audio: operatorstate.Audio{
			Mode:          audio.ModePushToTalk,
			InputDeviceID: "default",
			Muted:         true,
			Paused:        false,
			LevelMin:      0.05,
			LevelMax:      0.9,
		},
	}); err != nil {
		t.Fatalf("save operator state: %v", err)
	}

	now := time.Now().UTC()
	sess := &session.Session{
		ID:               "sess-9",
		TargetWindow:     "Browser Preview",
		TargetURL:        "https://example.com/app",
		Status:           session.StatusPaused,
		CreatedAt:        now.Add(-time.Minute),
		UpdatedAt:        now,
		ApprovalRequired: true,
		Approved:         true,
		Feedback:         []session.FeedbackEvt{{ID: "evt-10", RawTranscript: "bigger", NormalizedText: "bigger"}},
	}
	if err := store.UpsertSession(sess); err != nil {
		t.Fatalf("upsert session: %v", err)
	}
	if err := store.SaveCanonicalPackage(&session.CanonicalPackage{
		SessionID:   sess.ID,
		Summary:     "bigger",
		GeneratedAt: now,
		ChangeRequests: []session.ChangeReq{{
			EventID:  "evt-10",
			Summary:  "bigger",
			Category: "unclear_needs_review",
			Priority: "medium",
		}},
	}); err != nil {
		t.Fatalf("save canonical package: %v", err)
	}

	srv := newRestoredTestServerWithStore(t, cfg, store, encryptor)

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("state failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	sessionPayload, _ := payload["session"].(map[string]any)
	if got := sessionPayload["id"]; got != "sess-9" {
		t.Fatalf("expected restored current session sess-9, got %v", got)
	}
	runtimeCodex, _ := payload["runtime_codex"].(map[string]any)
	if got := runtimeCodex["codex_workdir"]; got != "/tmp/restored-workdir" {
		t.Fatalf("expected restored codex workdir, got %v", got)
	}
	runtimeSTT, _ := payload["runtime_transcription"].(map[string]any)
	if got := runtimeSTT["mode"]; got != "faster_whisper" {
		t.Fatalf("expected restored stt mode faster_whisper, got %v", got)
	}
	audioPayload, _ := payload["audio"].(map[string]any)
	audioState, _ := audioPayload["state"].(map[string]any)
	if got := audioState["mode"]; got != audio.ModePushToTalk {
		t.Fatalf("expected restored audio mode %q, got %v", audio.ModePushToTalk, got)
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"codex_cli"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-restored-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("expected restored approved package to preview, got %d %s", previewRec.Code, previewRec.Body.String())
	}
}

func TestAuditExportRequiresLogsCapability(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.ControlCapabilities = []string{"read"}
	srv := newTestServer(t, cfg)
	if err := srv.audit.Write(audit.Event{Type: "session_started", SessionID: "sess-1"}); err != nil {
		t.Fatalf("write audit event: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/audit/export?limit=10", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected read capability to imply logs access, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if count, _ := payload["count"].(float64); count < 1 {
		t.Fatalf("expected exported audit events, got %#v", payload)
	}

	cfg.ControlCapabilities = []string{"config"}
	srv = newTestServer(t, cfg)
	req = httptest.NewRequest(http.MethodGet, "/api/audit/export?limit=10", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden without logs/read capability, got %d", rec.Code)
	}
}

func TestConfigExportRequiresConfigReadOrReadCapability(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.ControlCapabilities = []string{"logs"}
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/config/export", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden without config_read/read capability, got %d", rec.Code)
	}

	cfg.ControlCapabilities = []string{"config_read"}
	srv = newTestServer(t, cfg)
	req = httptest.NewRequest(http.MethodGet, "/api/config/export", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected config_read capability to allow export, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode config export response: %v", err)
	}
	if got := payload["config_path"]; got == "" {
		t.Fatalf("expected config_path in export payload, got %#v", payload)
	}
	configTOML, _ := payload["config_toml"].(string)
	if !strings.Contains(configTOML, "[runtime_codex]") || !strings.Contains(configTOML, "[prompts]") {
		t.Fatalf("expected TOML config export, got %q", configTOML)
	}
}

func TestIndexIncludesFloatingComposerControl(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<link rel="icon" href="/favicon.ico" sizes="any" />`) {
		t.Fatalf("expected index to declare the Knit favicon")
	}
	if !strings.Contains(body, `class="hero-brand"`) || !strings.Contains(body, `/docs/assets/knit-mark.png`) || !strings.Contains(body, `Local-first multimodal AI feedback runtime`) {
		t.Fatalf("expected index hero to include the Knit logo and branding copy")
	}
	if !strings.Contains(body, "Open popout composer") {
		t.Fatalf("expected index to include the popout composer option")
	}
	if !strings.Contains(body, ".step-header-toolbar {") || !strings.Contains(body, "justify-content: flex-end;") {
		t.Fatalf("expected index to preserve step header toolbar styling for other workflow sections")
	}
	if !strings.Contains(body, `id="captureOptionActions"`) || !strings.Contains(body, `id="captureOptionGrid"`) {
		t.Fatalf("expected step 2 to render separate action and option containers")
	}
	if !strings.Contains(body, ".capture-option-grid {") || !strings.Contains(body, "grid-template-columns: repeat(3, minmax(0, 1fr));") {
		t.Fatalf("expected step 2 options to render side by side across three desktop columns")
	}
	if !strings.Contains(body, `id="captureOptionExtension"`) || !strings.Contains(body, `>Easy<`) || !strings.Contains(body, `Chrome Extension`) {
		t.Fatalf("expected step 2 to render the easy Chrome Extension path")
	}
	if !strings.Contains(body, `onclick="openDocsBrowser('getting_started.md')"`) || !strings.Contains(body, `>Install guide</span>`) {
		t.Fatalf("expected Chrome Extension option card to link directly to the extension install guide")
	}
	if !strings.Contains(body, `id="captureOptionComposer"`) || !strings.Contains(body, `>Intermediate<`) || !strings.Contains(body, `Popout Composer`) || !strings.Contains(body, `Open popout composer`) {
		t.Fatalf("expected step 2 to render the intermediate popout composer path")
	}
	if !strings.Contains(body, `id="captureOptionMainUI"`) || !strings.Contains(body, `Main UI`) || !strings.Contains(body, `Main UI Interface`) {
		t.Fatalf("expected step 2 to render the main UI fallback path")
	}
	if !strings.Contains(body, `id="captureAgentNotice"`) || !strings.Contains(body, `Current coding agent:`) || !strings.Contains(body, `<code>knit.toml</code>`) {
		t.Fatalf("expected step 2 to show the current coding agent note and where to change it")
	}
	if !strings.Contains(body, `onclick="openAgentSettingsFromNotice(event)"`) || !strings.Contains(body, `Change it in <a href="#" onclick="openAgentSettingsFromNotice(event)">Settings</a> → Agent`) {
		t.Fatalf("expected step 2 coding-agent notice to link directly into agent settings")
	}
	if !strings.Contains(body, `id="currentVersionLabel"`) || !strings.Contains(body, `id="manualUpdateCheckBtn"`) || !strings.Contains(body, `Check for updates`) {
		t.Fatalf("expected hero to expose the current version and manual update check action")
	}
	if !strings.Contains(body, `id="updateBanner"`) || !strings.Contains(body, `View release notes`) || !strings.Contains(body, `dismissUpdateBanner()`) {
		t.Fatalf("expected index to include a dismissible update banner")
	}
	if !strings.Contains(body, `function maybeRunStartupUpdateCheck()`) || !strings.Contains(body, `checkForUpdates(false)`) || !strings.Contains(body, `dismissed_update_version`) {
		t.Fatalf("expected index to include startup update-check logic with dismiss persistence")
	}
	if strings.Contains(body, `id="captureOptionExtension" class="capture-option-card">`) && strings.Contains(body, `id="captureOptionComposer" class="capture-option-card">`) {
		extensionSectionIndex := strings.Index(body, `id="captureOptionExtension" class="capture-option-card">`)
		composerSectionIndex := strings.Index(body, `id="captureOptionComposer" class="capture-option-card">`)
		extensionCloseIndex := strings.Index(body[extensionSectionIndex:composerSectionIndex], `</section>`)
		if extensionSectionIndex == -1 || composerSectionIndex == -1 || extensionCloseIndex == -1 {
			t.Fatalf("expected Chrome Extension option card to close before the Popout Composer option begins")
		}
	}
	writtenNoteIndex := strings.Index(body, `id="transcript"`)
	extensionTokenIndex := strings.Index(body, `id="captureOptionExtension"`)
	popoutComposerIndex := strings.Index(body, `id="captureOptionComposer"`)
	docsLibraryIndex := strings.Index(body, `id="openDocsLibraryBtn"`)
	settingsIndex := strings.Index(body, `onclick="openCaptureSettingsModal()"`)
	captureOptionActionsIndex := strings.Index(body, `id="captureOptionActions"`)
	captureOptionGridIndex := strings.Index(body, `id="captureOptionGrid"`)
	if extensionTokenIndex == -1 || popoutComposerIndex == -1 || docsLibraryIndex == -1 || settingsIndex == -1 || writtenNoteIndex == -1 || captureOptionActionsIndex == -1 || captureOptionGridIndex == -1 || captureOptionActionsIndex > captureOptionGridIndex || docsLibraryIndex > captureOptionGridIndex || settingsIndex > captureOptionGridIndex || extensionTokenIndex < captureOptionGridIndex || popoutComposerIndex < captureOptionGridIndex || writtenNoteIndex < captureOptionGridIndex {
		t.Fatalf("expected docs/settings outside the option grid and the three capture paths inside it")
	}
	if !strings.Contains(body, "Capture, review, and send") || !strings.Contains(body, "Queue and delivery") {
		t.Fatalf("expected main workflow to combine capture with review/send and reserve step 3 for delivery progress")
	}
	if !strings.Contains(body, "align-items: start;") || !strings.Contains(body, "aspect-ratio: 1 / 1;") {
		t.Fatalf("expected hero metric cards to stay square instead of stretching vertically")
	}
	if strings.Contains(body, "Browser Companion (Pointer/Click/Hover)") {
		t.Fatalf("did not expect a standalone browser companion panel")
	}
	if !strings.Contains(body, "Start review") || !strings.Contains(body, "⏸ Pause") || !strings.Contains(body, "⏹ Stop") {
		t.Fatalf("expected index to include labeled session transport controls")
	}
	if !strings.Contains(body, "aria-label=\"Pause Capture\" disabled hidden") || !strings.Contains(body, "aria-label=\"Stop Session\" disabled hidden") {
		t.Fatalf("expected pause and stop controls to start hidden until a session is active")
	}
	if strings.Contains(body, "const sessionInProgress = hasSession && captureState !== 'inactive';") {
		t.Fatalf("did not expect index to gate session transport visibility on capture state")
	}
	if !strings.Contains(body, "function renderExtensionPairings()") || !strings.Contains(body, "escapePreviewHTML(label)") || strings.Contains(body, "escapeHTML(label)") {
		t.Fatalf("expected extension pairing rendering to use the shared preview escaping helper")
	}
	if !strings.Contains(body, "function hasLiveSession()") || !strings.Contains(body, "const status = currentSessionStatus();") {
		t.Fatalf("expected index to distinguish live sessions from stopped or submitted ones")
	}
	if !strings.Contains(body, "sessionPlayBtnEl.disabled = hasSession;") || !strings.Contains(body, "A live review session is already active. Stop it before starting another one.") {
		t.Fatalf("expected index to block duplicate review starts only while a live session is active")
	}
	if !strings.Contains(body, "sessionPlayBtnEl.hidden = hasSession;") {
		t.Fatalf("expected session start control to hide whenever a review session already exists")
	}
	if strings.Contains(body, "Review active") || strings.Contains(body, "Review paused") {
		t.Fatalf("did not expect active session status copy to remain in the primary session control area")
	}
	if !strings.Contains(body, "sessionPauseResumeBtnEl.hidden = !hasSession;") || !strings.Contains(body, "sessionStopBtnEl.hidden = !hasSession;") {
		t.Fatalf("expected pause and stop controls to show whenever a review session exists")
	}
	if strings.Contains(body, "Ready state") {
		t.Fatalf("did not expect a standalone ready-state card in the delivery section")
	}
	if !strings.Contains(body, "id=\"deliveryBadge\"") || !strings.Contains(body, "renderQueueStateCard(") || !strings.Contains(body, "renderSubmissionStateCard(") {
		t.Fatalf("expected delivery section to render concise status plus detailed queue and submission summaries")
	}
	if !strings.Contains(body, "<strong>Current run</strong>") || !strings.Contains(body, "<summary>Recent runs</summary>") {
		t.Fatalf("expected delivery section to use clearer current-run and history labels")
	}
	if !strings.Contains(body, ".submit-attempt-indicator") || !strings.Contains(body, "function submitAttemptIndicatorState(attempt)") || !strings.Contains(body, "function renderSubmitAttemptIndicator(attempt)") {
		t.Fatalf("expected recent-run cards to expose a top-right success or failure indicator")
	}
	if !strings.Contains(body, "No live work log yet. Work activity appears here after the adapter starts writing logs.") || !strings.Contains(body, "No agent commentary yet. Plain-language progress updates appear here when the agent explains what it is doing.") {
		t.Fatalf("expected delivery section to split live output into work and commentary lanes")
	}
	if !strings.Contains(body, "overflow-wrap: anywhere;") || !strings.Contains(body, "word-break: break-word;") {
		t.Fatalf("expected index preformatted panels to wrap long unbroken log lines instead of stretching the layout")
	}
	if !strings.Contains(body, "<strong>Running:</strong>") || !strings.Contains(body, "<strong>Waiting:</strong>") || !strings.Contains(body, "<strong>Destination:</strong>") {
		t.Fatalf("expected delivery section summaries to expose running, waiting, and destination details")
	}
	if !strings.Contains(body, "submitStateEl.textContent = 'No active run.';") || strings.Contains(body, "const latestAttempt = attempts[0] || null;") {
		t.Fatalf("expected current-run panel to clear once no attempt is actively running")
	}
	if !strings.Contains(body, "function requestPreviewText(attempt)") || !strings.Contains(body, "<strong>Request:</strong>") || !strings.Contains(body, "request_preview") {
		t.Fatalf("expected delivery section summaries to expose request preview snippets for queued and running attempts")
	}
	if !strings.Contains(body, "function submitAttemptOutputText(attempt)") || !strings.Contains(body, "function hydrateSubmitAttemptOutputs()") || !strings.Contains(body, "function renderSubmitAttemptOutput(attempt)") || !strings.Contains(body, "splitLiveAgentOutputForDisplay(output)") || !strings.Contains(body, "Agent summary") || !strings.Contains(body, "function providerDestinationLabel(provider)") || !strings.Contains(body, "function notifySubmitRecoveryNotices(notices)") || !strings.Contains(body, "No work log captured for this run.") || !strings.Contains(body, "No agent commentary captured for this run.") || !strings.Contains(body, "<summary>Raw JSON</summary>") {
		t.Fatalf("expected recent-run history to expose an agent summary alongside work-log and agent-commentary panes with optional raw JSON")
	}
	if !strings.Contains(body, "function submitAttemptNeedsAttention(attempt)") || !strings.Contains(body, "function submitAttemptOutcomeListItem(attempt)") || !strings.Contains(body, "<strong>Result:</strong>") {
		t.Fatalf("expected main UI recent-run history to expose explicit no-op and blocked-run outcomes")
	}
	if strings.Contains(body, "const useTail = status !== 'in_progress' && status !== 'queued';") || strings.Contains(body, "&tail=1") {
		t.Fatalf("did not expect recent-run previews to tail adapter logs; they should start at the beginning so startup/system output remains visible")
	}
	if !strings.Contains(body, "function submitAttemptWorkspaceListItem(attempt)") || !strings.Contains(body, "<strong>Workspace used:</strong>") {
		t.Fatalf("expected delivery section to expose the actual workspace used for each submit attempt")
	}
	if !strings.Contains(body, "function cancelSubmitAttempt(attemptID)") || !strings.Contains(body, "function rerunSubmitAttempt(attemptID)") || !strings.Contains(body, "/api/session/attempt/cancel") || !strings.Contains(body, "/api/session/attempt/rerun") || !strings.Contains(body, "Remove from queue") || !strings.Contains(body, "Stop request") || !strings.Contains(body, "Rerun request with current settings") {
		t.Fatalf("expected main UI to expose rerun and stop controls for queued and running submit attempts")
	}
	if !strings.Contains(body, "const txt = await res.text();") || !strings.Contains(body, "if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));") || !strings.Contains(body, "function handleStateRefreshFailure(message)") || !strings.Contains(body, "Main UI stopped refreshing: ") {
		t.Fatalf("expected main UI state refresh failures to surface visibly instead of silently freezing the page")
	}
	if !strings.Contains(body, "function snapshotSubmitAttemptOpenState()") || !strings.Contains(body, "function restoreSubmitAttemptOpenState()") || !strings.Contains(body, "data-submit-attempt-raw-json") {
		t.Fatalf("expected recent-run history refreshes to preserve expanded raw JSON panels")
	}
	if !strings.Contains(body, "submitAttemptRawJSONScrollTopByID = new Map()") || !strings.Contains(body, "data-submit-attempt-raw-json-body") || !strings.Contains(body, "rawJSONPre.scrollTop = scrollTop") {
		t.Fatalf("expected recent-run history refreshes to preserve raw JSON scroll position as well as open state")
	}
	if strings.Contains(body, "id=\"providerSelect\"") {
		t.Fatalf("did not expect provider select in Review And Submit panel")
	}
	if !strings.Contains(body, "id=\"agentDefaultProvider\"") {
		t.Fatalf("expected provider selection inside runtime modal")
	}
	if !strings.Contains(body, "Profile maps to your local Codex config.toml profile.") {
		t.Fatalf("expected index runtime modal to explain Codex profile/config.toml usage")
	}
	if strings.Contains(body, "codexOptionsLoaded && !codexOptionsAttempted") {
		t.Fatalf("did not expect main UI to auto-load Codex options on refresh")
	}
	if !strings.Contains(body, "knit_ui_settings_v1") || !strings.Contains(body, "initPersistentSettings()") {
		t.Fatalf("expected index UI localStorage persistence hooks")
	}
	if !strings.Contains(body, "capture_guide_open") {
		t.Fatalf("expected capture guide open/closed state to be persisted")
	}
	if !strings.Contains(body, "togglePauseResume()") {
		t.Fatalf("expected index to include pause/resume toggle handler")
	}
	if !strings.Contains(body, "id=\"captureInputValuesToggle\"") || !strings.Contains(body, "Capture typed values for replay") {
		t.Fatalf("expected index to expose replay value-capture consent toggle")
	}
	if !strings.Contains(body, "Enabled by default for new sessions.") {
		t.Fatalf("expected index replay-value copy to explain the new default")
	}
	if strings.Contains(body, "id=\"sensitiveCaptureBadges\"") {
		t.Fatalf("did not expect step 2 to show always-visible sensitive-capture badges")
	}
	if !strings.Contains(body, "renderSensitiveCaptureBadges()") {
		t.Fatalf("expected index to keep sensitive-capture badge rendering logic for settings-driven state")
	}
	if !strings.Contains(body, "What will be sent") || !strings.Contains(body, "renderDisclosureSummary(preview)") {
		t.Fatalf("expected index preview to render a send-disclosure summary")
	}
	if !strings.Contains(body, "togglePreviewReplayRedaction()") || !strings.Contains(body, "togglePreviewVideoDelivery()") || !strings.Contains(body, "redact_replay_values") || !strings.Contains(body, "omit_video_clips") {
		t.Fatalf("expected index preview/send flow to expose one-click delivery actions")
	}
	if !strings.Contains(body, "function exportReplayBundle(eventID, format)") || !strings.Contains(body, "Export replay JSON") || !strings.Contains(body, "Export Playwright script") {
		t.Fatalf("expected index preview to expose replay exports")
	}
	if !strings.Contains(body, "const latestLoggedAttempt = attempts.find(a => {") || !strings.Contains(body, "if (String(a.status || '') === 'queued') return false;") {
		t.Fatalf("expected live agent output polling to fall back to the latest completed local log when a run finishes quickly")
	}
	if !strings.Contains(body, "allow_large_inline_media: !!allowLargeInlineMediaEl?.checked") {
		t.Fatalf("expected preview and submit flows to carry the user-selected large-inline-media decision")
	}
	if !strings.Contains(body, "Make clip smaller to send") || !strings.Contains(body, "/api/session/feedback/clip?event_id=") {
		t.Fatalf("expected main UI preview flow to support resizing oversized clips")
	}
	if !strings.Contains(body, "id=\"themeToggleBtn\"") || !strings.Contains(body, "toggleTheme()") {
		t.Fatalf("expected index to include a light/dark theme toggle")
	}
	if !strings.Contains(body, "class=\"hero-header\"") {
		t.Fatalf("expected theme toggle to be rendered in the hero header")
	}
	if strings.Index(body, "id=\"themeToggleBtn\"") > strings.Index(body, "Capture what should change. Tell your agent.") {
		t.Fatalf("expected theme toggle to render above the hero headline")
	}
	if strings.Contains(body, ".theme-toggle {\n      position: fixed;") {
		t.Fatalf("did not expect theme toggle to remain fixed outside the hero grid")
	}
	if !strings.Contains(body, "setUISetting('theme'") || !strings.Contains(body, "applyTheme(normalizeTheme(uiSettings.theme || 'light'))") {
		t.Fatalf("expected index theme preference to persist in localStorage")
	}
	if !strings.Contains(body, ".hidden,\n    [hidden] { display:none !important; }") {
		t.Fatalf("expected index CSS to enforce hidden-state display for transport controls and other toggled elements")
	}
	if !strings.Contains(body, "🗑️ <span>Delete session</span>") || !strings.Contains(body, "🗑️🗑️ <span>Delete all data</span>") {
		t.Fatalf("expected index to include labeled delete controls")
	}
	if strings.Contains(body, "Kill Capture") {
		t.Fatalf("did not expect kill capture to remain in primary controls")
	}
	if !strings.Contains(body, "openFloatingComposerPopup()") {
		t.Fatalf("expected index to include popup composer handler")
	}
	if strings.Contains(body, "id=\"approveBtn\"") {
		t.Fatalf("did not expect standalone approve button in review controls")
	}
	if !strings.Contains(body, "id=\"previewBtn\"") || !strings.Contains(body, "id=\"submitBtn\"") || !strings.Contains(body, "id=\"openLogBtn\"") {
		t.Fatalf("expected capture flow to include preview, submit, and open-log actions")
	}
	if !strings.Contains(body, "class=\"main-ui-action-grid\"") || !strings.Contains(body, "class=\"secondary icon-btn toolbar-button main-ui-icon-button\"") {
		t.Fatalf("expected main UI capture actions to render as compact icon buttons")
	}
	if !strings.Contains(body, "id=\"audioNoteBtn\" class=\"secondary icon-btn toolbar-button main-ui-icon-button\"") || !strings.Contains(body, "aria-label=\"Record audio note\">🎙️</button>") {
		t.Fatalf("expected main UI audio note control to render as an icon-only button with a tooltip")
	}
	if !strings.Contains(body, "id=\"previewBtn\" class=\"secondary icon-btn toolbar-button main-ui-icon-button\"") || !strings.Contains(body, "aria-label=\"Preview request\">👁</button>") {
		t.Fatalf("expected main UI preview control to render as an icon-only button with a tooltip")
	}
	if strings.Contains(body, "id=\"agentPromptSettingsBtn\"") {
		t.Fatalf("did not expect a standalone agent prompt settings button in the main send controls")
	}
	if !strings.Contains(body, "onclick=\"openCodexRuntimeModal()\"") || !strings.Contains(body, "id=\"deliveryPromptSection\"") || !strings.Contains(body, "id=\"deliveryIntentProfile\"") || !strings.Contains(body, "id=\"deliveryInstructionText\"") {
		t.Fatalf("expected main UI to expose delivery prompt controls inside agent runtime settings")
	}
	if !strings.Contains(body, "function renderCaptureAgentNotice()") {
		t.Fatalf("expected main UI to render a dynamic current-agent notice in step 2")
	}
	if !strings.Contains(body, "function openAgentSettingsFromNotice(event)") || !strings.Contains(body, "agentDefaultProviderEl?.focus();") {
		t.Fatalf("expected main UI coding-agent notice link to open agent settings and focus the provider selector")
	}
	if !strings.Contains(body, "intent_profile: intentProfile") || !strings.Contains(body, "instruction_text: instructionText") {
		t.Fatalf("expected main preview/submit requests to carry delivery intent fields")
	}
	if !strings.Contains(body, "id=\"openDocsLibraryBtn\"") || !strings.Contains(body, "id=\"openDocsLibrarySettingsBtn\"") || !strings.Contains(body, "function openDocsBrowser(name)") || !strings.Contains(body, "/docs") {
		t.Fatalf("expected main UI to expose the docs library directly and from settings")
	}
	if !strings.Contains(body, "onclick=\"previewPayload()\"") || !strings.Contains(body, "onclick=\"openLastLog()\"") {
		t.Fatalf("expected restored main capture controls to use the existing preview and open-log handlers")
	}
	if !strings.Contains(body, "looksLikeLocalAttemptLogRef(") || !strings.Contains(body, "live log unavailable for this adapter.") || !strings.Contains(body, "id=\"liveSubmitCommentary\"") || !strings.Contains(body, "function splitLiveAgentOutputForDisplay(raw)") || !strings.Contains(body, "function renderLiveAgentOutput()") {
		t.Fatalf("expected live adapter log polling to split work logs from agent commentary and skip non-local adapter refs")
	}
	if !strings.Contains(body, "function activeSubmitAttemptForLog()") || !strings.Contains(body, "if (runningAttempt) return runningAttempt;") || !strings.Contains(body, "const latestLoggedAttempt = attempts.find(a => {") {
		t.Fatalf("expected live adapter log polling to continue through completion for the active attempt")
	}
	if !strings.Contains(body, "popup=yes") {
		t.Fatalf("expected popup composer window features to request popup mode")
	}
	if !strings.Contains(body, "More capture options") || !strings.Contains(body, "Start voice commands") {
		t.Fatalf("expected index to include voice command controls")
	}
	if !strings.Contains(body, "Capture Guide") || !strings.Contains(body, "Choose the repository in <strong>Workspace</strong>") || !strings.Contains(body, "Text-only notes are valid.") {
		t.Fatalf("expected index to include capture guide instructions")
	}
	if strings.Contains(body, "id=\"captureGuideStatus\"") || strings.Contains(body, "id=\"platformRuntimeState\"") || strings.Contains(body, "id=\"composerSupportState\"") || strings.Contains(body, "Session started: ") || strings.Contains(body, "Platform runtime: ") || strings.Contains(body, "Composer popup uses window.open") {
		t.Fatalf("did not expect the capture guide sidebar to include runtime/status diagnostic blocks")
	}
	if !strings.Contains(body, "id=\"captureGuideSidebar\"") {
		t.Fatalf("expected index to include right-side capture guide sidebar")
	}
	if !strings.Contains(body, "class=\"capture-guide-header\"") || !strings.Contains(body, "class=\"danger icon-btn capture-guide-close\"") {
		t.Fatalf("expected capture guide close control to be rendered in the sidebar header corner")
	}
	if !strings.Contains(body, "id=\"guideInfoBtn\" class=\"capture-guide-toggle hidden\"") || !strings.Contains(body, "data-guide-icon=\"capture\"") {
		t.Fatalf("expected index to include modern capture-guide icon control to reopen the sidebar")
	}
	if strings.Contains(body, "id=\"guideInfoBtn\" class=\"icon-btn") {
		t.Fatalf("did not expect guide info icon to render with button chrome class")
	}
	if !strings.Contains(body, "class=\"capture-guide-title\"") || !strings.Contains(body, "class=\"capture-guide-title-mark\"") {
		t.Fatalf("expected capture guide header to render the updated icon treatment")
	}
	if !strings.Contains(body, "id=\"appToast\"") || !strings.Contains(body, "showToast('Connect browser link copied')") {
		t.Fatalf("expected index to include toast feedback for connect-browser copy")
	}
	if !strings.Contains(body, `showToast('Request submitted to ' + destination + ' for "`) {
		t.Fatalf("expected index to show a toast when submission succeeds")
	}
	if !strings.Contains(body, "function notifySubmitAttemptTransitions(attempts)") || !strings.Contains(body, "showToast(submitAttemptToastMessage(attempt), status === 'failed' || submitAttemptNeedsAttention(attempt))") {
		t.Fatalf("expected index to show one-shot completion toasts for submit attempt state transitions")
	}
	if !strings.Contains(body, "closeCaptureGuideSidebar()") || !strings.Contains(body, "openCaptureGuideSidebar()") {
		t.Fatalf("expected index to include capture guide open/close handlers")
	}
	if !strings.Contains(body, "setCaptureGuideSidebarOpen(true)") {
		t.Fatalf("expected index to open capture guide sidebar on first load")
	}
	if strings.Contains(body, "<summary>Capture Guide (Step By Step)</summary>") {
		t.Fatalf("did not expect legacy in-flow capture guide details panel")
	}
	if strings.Contains(body, "Open Floating Composer") {
		t.Fatalf("did not expect deprecated floating composer button")
	}
	if !strings.Contains(body, "onclick=\"openAudioControlsModal()\"") {
		t.Fatalf("expected feedback note to expose audio controls modal trigger")
	}
	if !strings.Contains(body, "id=\"audioNoteBtn\"") || !strings.Contains(body, "startAudioNoteCapture") || !strings.Contains(body, "finishAudioNoteCapture") {
		t.Fatalf("expected index audio note capture to support explicit start/stop recording")
	}
	if !strings.Contains(body, "audioNoteBtnEl.textContent = '■';") || !strings.Contains(body, "audioNoteBtnEl.textContent = '🎙️';") {
		t.Fatalf("expected main UI audio note control to swap icon-only states during recording")
	}
	if !strings.Contains(body, "const recordingActive = isNoteRecordingActive();") || !strings.Contains(body, "previewBtnEl.disabled = recordingActive || !sess || feedbackCount === 0;") || !strings.Contains(body, "submitBtnEl.disabled = recordingActive || submitInFlight || !sess || feedbackCount === 0;") {
		t.Fatalf("expected index preview and submit buttons to disable while a note recording is active")
	}
	if strings.Contains(body, "box-shadow: 0 1px 0 rgba(255,255,255,0.9) inset;") {
		t.Fatalf("did not expect index buttons to render a white inset highlight")
	}
	if strings.Contains(body, "recordAudioNote(4)") {
		t.Fatalf("did not expect index audio note recording to remain hard-limited to 4 seconds")
	}
	if !strings.Contains(body, "Advanced session details") {
		t.Fatalf("expected connect-this-review step to tuck manual session fields behind advanced disclosure")
	}
	if !strings.Contains(body, "DEFAULT_TARGET_WINDOW = 'Browser Review'") {
		t.Fatalf("expected index to use a safe default target window label")
	}
	if !strings.Contains(body, "syncSessionDetailInputsFromState()") {
		t.Fatalf("expected index to sync session details from companion/session state")
	}
	assertAllRenderedButtonsHaveTooltips(t, body)
	if !strings.Contains(body, "🎙️ <span>Audio</span>") {
		t.Fatalf("expected feedback note to expose labeled audio controls trigger")
	}
	if !strings.Contains(body, "onclick=\"openCaptureSettingsModal()\"") || !strings.Contains(body, "⚙️ <span>Settings</span>") {
		t.Fatalf("expected feedback note to expose a gear settings trigger")
	}
	if !strings.Contains(body, "id=\"captureSettingsModal\"") {
		t.Fatalf("expected index to include capture settings modal container")
	}
	if strings.Contains(body, "Save written note") {
		t.Fatalf("did not expect legacy save-written-note label in capture actions")
	}
	if strings.Contains(body, "id=\"pttQuickBtn\"") || strings.Contains(body, "Hold to talk") {
		t.Fatalf("did not expect redundant PTT quick hold button in feedback note")
	}
	if strings.Contains(body, "Hold Push-To-Talk") {
		t.Fatalf("did not expect legacy hold-to-talk button in audio controls modal")
	}
	if !strings.Contains(body, "onclick=\"openWorkspaceModal()\"") || !strings.Contains(body, "📁 <span>Workspace</span>") {
		t.Fatalf("expected feedback note to expose workspace modal trigger")
	}
	if !strings.Contains(body, "onclick=\"copyCompanionSnippet()\"") || !strings.Contains(body, "🔗 <span>Connect browser</span>") {
		t.Fatalf("expected feedback note to expose companion snippet trigger")
	}
	if !strings.Contains(body, "id=\"save\" data-testid=\"settings-save\" class=\"primary primary-action\"") || !strings.Contains(body, ">Save settings</button>") {
		t.Fatalf("expected environment profiles to expose an explicit save settings primary action")
	}
	if !strings.Contains(body, ".primary-action {") || !strings.Contains(body, "min-height: 60px;") || !strings.Contains(body, "background: #072b29;") || !strings.Contains(body, ".primary-action:focus-visible {") {
		t.Fatalf("expected settings save action to render with a larger high-contrast primary style")
	}
	if !strings.Contains(body, "onclick=\"openVideoCaptureModal()\"") || !strings.Contains(body, "🎥 <span>Video</span>") {
		t.Fatalf("expected feedback note to expose video capture settings trigger")
	}
	if !strings.Contains(body, "id=\"videoNoteBtn\"") || !strings.Contains(body, "aria-label=\"Record video note\">🎥</button>") {
		t.Fatalf("expected feedback note to expose inline video note recording as an icon-only button")
	}
	if !strings.Contains(body, "videoNoteBtnEl.textContent = '■';") || !strings.Contains(body, "videoNoteBtnEl.textContent = '🎥';") {
		t.Fatalf("expected main UI video note control to swap icon-only states during recording")
	}
	if !strings.Contains(body, "onclick=\"openCodexRuntimeModal()\"") || !strings.Contains(body, "🤖 <span>Agent</span>") {
		t.Fatalf("expected feedback note to expose agent runtime trigger")
	}
	if !strings.Contains(body, "id=\"workspaceModal\"") {
		t.Fatalf("expected index to include workspace modal container")
	}
	if !strings.Contains(body, "id=\"audioControlsModal\"") {
		t.Fatalf("expected index to include audio controls modal container")
	}
	if !strings.Contains(body, "id=\"openTranscriptionFromAudioBtn\"") || !strings.Contains(body, ">⚙️<") {
		t.Fatalf("expected audio controls modal to include nested gear transcription runtime trigger")
	}
	if !strings.Contains(body, "openTranscriptionRuntimeFromAudioModal()") {
		t.Fatalf("expected audio controls modal to include nested transcription runtime handler")
	}
	if !strings.Contains(body, "id=\"transcriptionRuntimeModal\"") {
		t.Fatalf("expected index to include transcription runtime modal container")
	}
	if !strings.Contains(body, "id=\"videoCaptureModal\"") {
		t.Fatalf("expected index to include video capture modal container")
	}
	if !strings.Contains(body, "id=\"codexRuntimeModal\"") {
		t.Fatalf("expected index to include codex runtime modal container")
	}
	if !strings.Contains(body, "onCodexRuntimeModalBackdrop(event)") {
		t.Fatalf("expected index to include codex runtime modal backdrop handler")
	}
	if !strings.Contains(body, "id=\"codexCliSection\"") || !strings.Contains(body, "id=\"codexAPISection\"") || !strings.Contains(body, "id=\"claudeAPISection\"") || !strings.Contains(body, "id=\"codexCLIDefaultsSection\"") {
		t.Fatalf("expected index runtime modal to include provider-specific runtime sections")
	}
	if !strings.Contains(body, "Default submit adapter") || !strings.Contains(body, "CLI command") || !strings.Contains(body, "Approval policy") {
		t.Fatalf("expected index runtime modal inputs to include visible labels")
	}
	if !strings.Contains(body, `<option value="claude_api">claude_api</option>`) || !strings.Contains(body, `id="claudeAPIBaseURL"`) || !strings.Contains(body, `id="claudeAPIModel"`) {
		t.Fatalf("expected index runtime modal to expose claude_api controls")
	}
	if strings.Contains(body, "Use Codex CLI default (no override)") || !strings.Contains(body, "danger-full-access</code> sandbox and <code>never</code> approval") || !strings.Contains(body, "Allow the coding agent to make changes by switching Sandbox to danger-full-access") {
		t.Fatalf("expected index runtime modal to show Knit-owned sandbox and approval defaults")
	}
	if !strings.Contains(body, "scheduleCodexRuntimeApply") || !strings.Contains(body, "syncCodexRuntimeModeUI") {
		t.Fatalf("expected index runtime modal to auto-save and toggle fields by provider")
	}
	if strings.Contains(body, "Apply Runtime") {
		t.Fatalf("expected stale apply-runtime button to be removed from index runtime modal")
	}
	if !strings.Contains(body, "id=\"clipIncludeAudio\" type=\"checkbox\" checked") {
		t.Fatalf("expected video capture modal to default clip microphone audio to enabled")
	}
	if !strings.Contains(body, "id=\"videoQualityProfile\"") || !strings.Contains(body, "value=\"smaller\"") || !strings.Contains(body, "value=\"balanced\"") || !strings.Contains(body, "value=\"detail\"") {
		t.Fatalf("expected video capture modal to expose user-selectable video quality profiles")
	}
	if !strings.Contains(body, "id=\"allowLargeInlineMedia\" type=\"checkbox\"") || !strings.Contains(body, "Allow large inline media when needed") {
		t.Fatalf("expected video capture modal to expose explicit large-inline-media opt-in")
	}
	if !strings.Contains(body, "lower video quality, use a screenshot instead, or explicitly allow large inline media before sending") {
		t.Fatalf("expected video capture modal to explain the user choices when transport size becomes an issue")
	}
	if !strings.Contains(body, "onVideoCaptureModalBackdrop(event)") {
		t.Fatalf("expected index to include video capture modal backdrop handler")
	}
	if !strings.Contains(body, "id=\"sttMode\"") {
		t.Fatalf("expected index transcription runtime modal to include stt mode control")
	}
	if !strings.Contains(body, "id=\"sttModeHelp\"") || !strings.Contains(body, "id=\"sttConnectionRow\"") || !strings.Contains(body, "id=\"sttFasterWhisperRow\"") || !strings.Contains(body, "id=\"sttCommandRow\"") {
		t.Fatalf("expected index transcription runtime modal to include mode-specific field groups")
	}
	if !strings.Contains(body, "<select id=\"sttFasterWhisperModel\"") || !strings.Contains(body, "<option value=\"large-v3-turbo\">large-v3-turbo</option>") || !strings.Contains(body, "<select id=\"sttDevice\"") || !strings.Contains(body, "<option value=\"cpu\">cpu</option>") || !strings.Contains(body, "<select id=\"sttComputeType\"") || !strings.Contains(body, "<option value=\"int8\">int8</option>") {
		t.Fatalf("expected index faster-whisper controls to use model/device/compute dropdowns")
	}
	if !strings.Contains(body, "id=\"sttLanguage\"") || !strings.Contains(body, "pattern=\"[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8}){0,2}\"") || !strings.Contains(body, "maxlength=\"2048\"") || !strings.Contains(body, "type=\"number\" min=\"1\" max=\"600\"") {
		t.Fatalf("expected index transcription runtime inputs to include client-side safeguards")
	}
	if !strings.Contains(body, "syncSTTRuntimeModeUI") {
		t.Fatalf("expected index transcription runtime modal to include mode-based field rendering")
	}
	if !strings.Contains(body, "scheduleTranscriptionRuntimeApply") {
		t.Fatalf("expected index transcription runtime modal to auto-apply changes")
	}
	if strings.Contains(body, "Apply Transcription Runtime") {
		t.Fatalf("expected stale apply-transcription-runtime button to be removed")
	}
	if !strings.Contains(body, "scheduleAudioConfigApply") {
		t.Fatalf("expected index to include auto-applied audio config behavior")
	}
	if strings.Contains(body, "Apply Audio Config") {
		t.Fatalf("expected stale apply-audio-config button to be removed")
	}
	if !strings.Contains(body, "switch Audio mode to always_on") {
		t.Fatalf("expected index to include cross-tab push-to-talk guidance")
	}
	if !strings.Contains(body, "await approveSession(true, 'preview')") || !strings.Contains(body, "await approveSession(true, 'submit')") {
		t.Fatalf("expected preview/submit actions to auto-prepare approval snapshot")
	}
	if !strings.Contains(body, "editPreviewNote(") || !strings.Contains(body, "deletePreviewNote(") {
		t.Fatalf("expected index preview UI to support editing and deleting change requests")
	}
	if !strings.Contains(body, "Test Mic (10s)") {
		t.Fatalf("expected index to include 10-second mic test control")
	}
	if !strings.Contains(body, "micTestMeterFill") {
		t.Fatalf("expected index to include live mic test meter")
	}
	if !strings.Contains(body, "id=\"audioLevelState\" class=\"hidden\"") {
		t.Fatalf("expected index mic audio level line to stay hidden outside mic test")
	}
	if strings.Contains(body, "Lock snapshot") || strings.Contains(body, "Unlock snapshot") {
		t.Fatalf("did not expect snapshot lock controls to remain in the main capture UI")
	}
	if strings.Contains(body, "title=\"Add written note\"") || strings.Contains(body, "title=\"Discard last note\">↺ <span>Discard last note</span></button>") {
		t.Fatalf("did not expect separate note action buttons in the main capture UI")
	}
	if !strings.Contains(body, "flushTypedNoteDraft('preview')") || !strings.Contains(body, "flushTypedNoteDraft('send')") {
		t.Fatalf("expected preview/send flow to absorb main-page written-note drafts automatically")
	}
	if !strings.Contains(body, "Laser pointer mode") {
		t.Fatalf("expected index to include laser pointer mode control")
	}
	if !strings.Contains(body, "syncLaserModeForVideo") {
		t.Fatalf("expected index to synchronize laser mode when video capture is active")
	}
	if strings.Contains(body, "Recording audio note...") {
		t.Fatalf("did not expect recording progress indication in main control plane")
	}
	if !strings.Contains(body, "More capture options") || !strings.Contains(body, "Apply") {
		t.Fatalf("expected index to include review mode control")
	}
	if strings.Contains(body, "experiment id (optional)") {
		t.Fatalf("did not expect experiment id field in single-user MVP UI")
	}
	if strings.Contains(body, "variant (optional, e.g. A/B)") {
		t.Fatalf("did not expect variant field in single-user MVP UI")
	}
	if strings.Contains(body, "reviewer name") {
		t.Fatalf("did not expect reviewer name field in single-user MVP UI")
	}
	if strings.Contains(body, "collaborative review note (optional)") {
		t.Fatalf("did not expect collaborative review note field in single-user MVP UI")
	}
	if strings.Contains(body, "Add Review Note") {
		t.Fatalf("did not expect collaborative review note action in single-user MVP UI")
	}
	if !strings.Contains(body, "requireCompanionFor('enable visual capture')") {
		t.Fatalf("expected index to require companion before visual capture")
	}
	if !strings.Contains(body, "requireCompanionFor('capture snapshots')") {
		t.Fatalf("expected index to require companion before snapshot capture")
	}
	if !strings.Contains(body, "requireCompanionFor('record video clips')") {
		t.Fatalf("expected index to require companion before clip capture")
	}
	if !strings.Contains(body, "async function freezeFrame()") || !strings.Contains(body, "function unfreezeFrame()") {
		t.Fatalf("expected index to include explicit freeze/unfreeze helpers for annotation flow")
	}
	if !strings.Contains(body, "Hotkey: <code>Ctrl+Shift+S</code> captures manual screenshot.") || !strings.Contains(body, "captureManualScreenshot();") {
		t.Fatalf("expected index to advertise and wire the manual screenshot hotkey")
	}
}

func TestIndexRecentRunsExposeStatusIndicatorHelpers(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, ".submit-attempt-indicator") ||
		!strings.Contains(body, "function submitAttemptIndicatorState(attempt)") ||
		!strings.Contains(body, "function renderSubmitAttemptIndicator(attempt)") ||
		!strings.Contains(body, "function submitAttemptCardShouldStartOpen(attempt, index)") ||
		!strings.Contains(body, "data-submit-attempt-output=\"work\"") ||
		!strings.Contains(body, "function shouldStickScroll(el, threshold = 24)") ||
		!strings.Contains(body, "class=\"status-card submit-attempt-card\"") {
		t.Fatalf("expected index recent runs to expose collapsible cards, scroll helpers, and status indicators")
	}
}

func TestIndexRecentRunsUseStructuredHeaderLayout(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, ".submit-attempt-header {") ||
		!strings.Contains(body, ".submit-attempt-meta-top {") ||
		!strings.Contains(body, ".submit-attempt-action-row {") {
		t.Fatalf("expected main UI recent runs to include dedicated header layout styles")
	}
	if !strings.Contains(body, "submit-attempt-header") ||
		!strings.Contains(body, "submit-attempt-main") ||
		!strings.Contains(body, "submit-attempt-meta") ||
		!strings.Contains(body, "submit-attempt-status-line") ||
		!strings.Contains(body, "submit-attempt-time") {
		t.Fatalf("expected main UI recent runs to render title, status, time, indicator, and actions in structured header slots")
	}
}

func TestIndexRefreshKeepsRuntimeStateInScopeForLiveOutputPolling(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	refreshStart := strings.Index(body, "async function refresh() {")
	if refreshStart == -1 {
		t.Fatalf("expected main UI refresh function to be rendered")
	}
	refreshBody := body[refreshStart:]
	tryStart := strings.Index(refreshBody, "  try {")
	if tryStart == -1 {
		t.Fatalf("expected refresh function to contain a try block")
	}
	preTry := refreshBody[:tryStart]
	if !strings.Contains(preTry, "let rc = currentState?.runtime_codex || {};") ||
		!strings.Contains(preTry, "let preservingRuntimeDraft = codexRuntimeDirty || codexRuntimeApplying;") {
		t.Fatalf("expected refresh to hoist runtime state before the try block so live polling survives fetch failures")
	}
	if !strings.Contains(refreshBody, "rc = currentState.runtime_codex || {};") {
		t.Fatalf("expected refresh to update the runtime snapshot after state loads")
	}
	if strings.Contains(refreshBody, "const rc = currentState.runtime_codex || {};") ||
		strings.Contains(refreshBody, "const preservingRuntimeDraft = codexRuntimeDirty || codexRuntimeApplying;") {
		t.Fatalf("did not expect refresh to redeclare runtime state inside the try block")
	}
	if !strings.Contains(refreshBody, "refreshActiveSubmitLog();") {
		t.Fatalf("expected refresh to continue into live submit-log polling")
	}
}

func TestIndexThemeToggleRendersInHeroHeader(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `class="hero-header"`) {
		t.Fatalf("expected theme toggle to render in the hero header")
	}
	if !strings.Contains(body, `id="themeToggleBtn"`) {
		t.Fatalf("expected theme toggle button in index HTML")
	}
	if strings.Index(body, `id="themeToggleBtn"`) > strings.Index(body, "Capture what should change. Tell your agent.") {
		t.Fatalf("expected theme toggle to render above the hero headline")
	}
	if !strings.Contains(body, ".hero-header .theme-toggle") {
		t.Fatalf("expected mobile styles to preserve the theme toggle width in the hero header")
	}
}

func TestIndexIncludesModernCaptureGuideIcon(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `id="guideInfoBtn" class="capture-guide-toggle hidden"`) {
		t.Fatalf("expected capture guide reopen control")
	}
	if !strings.Contains(body, `data-guide-icon="capture"`) {
		t.Fatalf("expected capture guide reopen control to render the modern svg icon")
	}
	if !strings.Contains(body, `class="capture-guide-title"`) || !strings.Contains(body, `class="capture-guide-title-mark"`) {
		t.Fatalf("expected capture guide sidebar header to render the updated icon treatment")
	}
}

func TestFaviconRouteServesIcon(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "image") {
		t.Fatalf("expected image content type, got %q", got)
	}
	if len(rec.Body.Bytes()) == 0 {
		t.Fatalf("expected favicon response body")
	}
}

func TestCaptureSettingsLockedBlocksAudioAndTranscriptionConfig(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.CaptureSettingsLocked = true
	srv := newTestServer(t, cfg)

	audioReq := httptest.NewRequest(http.MethodPost, "/api/audio/config", bytes.NewReader([]byte(`{"mode":"push_to_talk"}`)))
	audioReq.Header.Set("Content-Type", "application/json")
	addAuth(audioReq, cfg.ControlToken, true, "nonce-audio-lock")
	audioRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(audioRec, audioReq)
	if audioRec.Code != http.StatusLocked {
		t.Fatalf("expected audio config to be locked, got %d: %s", audioRec.Code, audioRec.Body.String())
	}

	sttReq := httptest.NewRequest(http.MethodPost, "/api/runtime/transcription", bytes.NewReader([]byte(`{"mode":"local"}`)))
	sttReq.Header.Set("Content-Type", "application/json")
	addAuth(sttReq, cfg.ControlToken, true, "nonce-stt-lock")
	sttRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(sttRec, sttReq)
	if sttRec.Code != http.StatusLocked {
		t.Fatalf("expected runtime transcription to be locked, got %d: %s", sttRec.Code, sttRec.Body.String())
	}
}

func TestProviderAllowlistBlocksPreviewAndSubmit(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.AllowedSubmitProviders = []string{"codex_cli"}
	srv := newTestServer(t, cfg)

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{ID: "evt-1", RawTranscript: "fix button", NormalizedText: "fix button"}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	if _, err := srv.sessions.Approve("summary"); err != nil {
		t.Fatalf("approve session: %v", err)
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"claude_cli"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-preview-blocked")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusForbidden {
		t.Fatalf("expected payload preview provider block, got %d: %s", previewRec.Code, previewRec.Body.String())
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"claude_cli"}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "nonce-submit-blocked")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusForbidden {
		t.Fatalf("expected submit provider block, got %d: %s", submitRec.Code, submitRec.Body.String())
	}
}

func TestPreviewAndSubmitRejectApprovedPackageWithoutInput(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-empty-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-empty-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-empty-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusConflict {
		t.Fatalf("expected empty preview to be rejected, got %d: %s", previewRec.Code, previewRec.Body.String())
	}
	if !strings.Contains(previewRec.Body.String(), "capture at least one note") {
		t.Fatalf("expected empty preview error message, got %q", previewRec.Body.String())
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "nonce-empty-submit")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusConflict {
		t.Fatalf("expected empty submit to be rejected, got %d: %s", submitRec.Code, submitRec.Body.String())
	}
	if !strings.Contains(submitRec.Body.String(), "capture at least one note") {
		t.Fatalf("expected empty submit error message, got %q", submitRec.Body.String())
	}
}

func TestClaudeAPIPreviewAndSubmitAreAvailable(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.AllowRemoteSubmission = true
	cfg.AllowedSubmitProviders = []string{"claude_api"}
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-token")
	t.Setenv("KNIT_CLAUDE_API_MODEL", "claude-test-model")
	srv := newTestServer(t, cfg)

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{ID: "evt-1", RawTranscript: "tighten the spacing", NormalizedText: "tighten the spacing"}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	if _, err := srv.sessions.Approve("summary"); err != nil {
		t.Fatalf("approve session: %v", err)
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"claude_api"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-preview-claude-api")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("expected claude_api preview to succeed, got %d: %s", previewRec.Code, previewRec.Body.String())
	}
	var previewPayload payloadPreviewResponse
	if err := json.Unmarshal(previewRec.Body.Bytes(), &previewPayload); err != nil {
		t.Fatalf("decode preview payload: %v", err)
	}
	if previewPayload.Provider != "claude_api" {
		t.Fatalf("expected preview provider claude_api, got %#v", previewPayload.Provider)
	}
	providerPayload, _ := previewPayload.Payload.(map[string]any)
	if providerPayload["model"] != "claude-test-model" {
		t.Fatalf("expected preview payload model override, got %#v", providerPayload["model"])
	}
	if providerPayload["max_tokens"] != float64(4096) && providerPayload["max_tokens"] != 4096 {
		t.Fatalf("expected preview payload max_tokens, got %#v", providerPayload["max_tokens"])
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"claude_api"}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "nonce-submit-claude-api")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("expected claude_api submit to be accepted, got %d: %s", submitRec.Code, submitRec.Body.String())
	}
	var submitPayload map[string]any
	if err := json.Unmarshal(submitRec.Body.Bytes(), &submitPayload); err != nil {
		t.Fatalf("decode submit payload: %v", err)
	}
	if got := submitPayload["provider"]; got != "claude_api" {
		t.Fatalf("expected submit provider claude_api, got %#v", got)
	}
}

func TestFloatingComposerEndpointRequiresAuthAndRenders(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	unauthReq := httptest.NewRequest(http.MethodGet, "/floating-composer", nil)
	unauthRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", unauthRec.Code)
	}

	authReq := httptest.NewRequest(http.MethodGet, "/floating-composer", nil)
	addAuth(authReq, cfg.ControlToken, false, "")
	authRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", authRec.Code, authRec.Body.String())
	}
	body := authRec.Body.String()
	if !strings.Contains(body, "Compact Composer") {
		t.Fatalf("expected floating composer page content")
	}
	if !strings.Contains(body, "id=\"talkOnlyBtn\"") || !strings.Contains(body, "id=\"snapshotTalkBtn\"") || !strings.Contains(body, "id=\"videoTalkBtn\"") {
		t.Fatalf("expected floating composer to expose compact capture buttons")
	}
	if !strings.Contains(body, "title=\"Talk only. Record a voice note without a screenshot.\"") || !strings.Contains(body, "title=\"Snapshot plus voice. Capture a snapshot and record a voice note.\"") || !strings.Contains(body, "title=\"Video plus voice. Record a video clip with your voice.\"") {
		t.Fatalf("expected compact capture buttons to explain themselves with tooltips")
	}
	if !strings.Contains(body, "id=\"previewPayloadBtn\"") || !strings.Contains(body, ">Submit<") {
		t.Fatalf("expected floating composer to include inline preview/send actions")
	}
	if strings.Contains(body, "id=\"submitTextBtn\"") {
		t.Fatalf("expected floating composer to avoid a separate save button for typed notes")
	}
	if !strings.Contains(body, "flushTypedNoteDraftFC('preview')") || !strings.Contains(body, "flushTypedNoteDraftFC('send')") {
		t.Fatalf("expected floating composer preview/send flow to absorb typed-note drafts automatically")
	}
	if !strings.Contains(body, "clearSubmittedPreviewFC()") || !strings.Contains(body, "Request queued. Capture another note, then preview the next request here.") {
		t.Fatalf("expected floating composer to clear the preview surface after a successful submit")
	}
	if !strings.Contains(body, "id=\"runtimeGuideLine\"") || !strings.Contains(body, "function renderRuntimeGuideFC(data)") {
		t.Fatalf("expected floating composer to include runtime guidance for supported OSs")
	}
	if !strings.Contains(body, "id=\"fcPayloadPreview\"") || !strings.Contains(body, "Preview the request here before sending it to the agent.") {
		t.Fatalf("expected floating composer to include a payload preview surface")
	}
	if !strings.Contains(body, "id=\"fcCaptureInputValuesToggle\"") || !strings.Contains(body, "Capture typed values for replay") {
		t.Fatalf("expected floating composer settings to expose replay value-capture consent toggle")
	}
	if !strings.Contains(body, "id=\"fcAllowLargeInlineMediaToggle\"") || !strings.Contains(body, "Allow large inline media when needed") {
		t.Fatalf("expected floating composer settings to expose large-media send control")
	}
	if !strings.Contains(body, "<strong>Composer Settings</strong>") {
		t.Fatalf("expected composer settings modal in floating composer HTML")
	}
	if strings.Contains(body, "</div>\n      <label class=\"small\" style=\"display:flex;align-items:center;gap:.4rem;\" title=\"Include typed form values in the replay bundle for this session. Enabled by default for new sessions.\">\n        <input id=\"fcCaptureInputValuesToggle\" type=\"checkbox\" onchange=\"toggleReplayValueCaptureFC()\" />\n        Capture typed values for replay\n      </label>\n      <div class=\"compact-status-row single\">") {
		t.Fatalf("did not expect replay value-capture toggle to remain inline in the compact composer")
	}
	if !strings.Contains(body, "function exportReplayBundleFC(eventID, format)") || !strings.Contains(body, "Replay JSON") || !strings.Contains(body, "Playwright") {
		t.Fatalf("expected floating composer preview to expose replay exports")
	}
	if !strings.Contains(body, "id=\"fcSensitiveCaptureBadges\"") || !strings.Contains(body, "id=\"fcStatusRadiator\"") || !strings.Contains(body, "renderSensitiveCaptureBadgesFC()") || !strings.Contains(body, "renderDisclosureSummaryFC(preview)") {
		t.Fatalf("expected floating composer to expose compact status radiator badges and send disclosure summary")
	}
	if !strings.Contains(body, "togglePreviewReplayRedactionFC()") || !strings.Contains(body, "togglePreviewVideoDeliveryFC()") || !strings.Contains(body, "redact_replay_values") || !strings.Contains(body, "omit_video_clips") {
		t.Fatalf("expected floating composer preview/send flow to expose one-click delivery actions")
	}
	if !strings.Contains(body, "id=\"fcLiveSubmitLog\"") || !strings.Contains(body, "id=\"fcLiveSubmitCommentary\"") || !strings.Contains(body, "refreshActiveSubmitLogFC()") || !strings.Contains(body, "hasOpenableSubmitLogFC()") || !strings.Contains(body, "function splitLiveAgentOutputForDisplayFC(raw)") || !strings.Contains(body, "function renderLiveAgentOutputFC()") || !strings.Contains(body, "function shouldStickScrollFC(el, threshold = 24)") || !strings.Contains(body, "Agent summary:") || !strings.Contains(body, "id=\"fcSubmitHistory\"") || !strings.Contains(body, "function hydrateSubmitAttemptOutputsFC()") || !strings.Contains(body, "function renderSubmitHistoryFC()") || !strings.Contains(body, "function snapshotSubmitHistoryStateFC()") || !strings.Contains(body, "class=\"status-card submit-attempt-card-compact\"") || !strings.Contains(body, "function rerunSubmitAttemptFC(attemptID)") || !strings.Contains(body, "/api/session/attempt/rerun") || !strings.Contains(body, "function providerDestinationLabelFC(provider)") || !strings.Contains(body, "function notifySubmitRecoveryNoticesFC(notices)") {
		t.Fatalf("expected floating composer to include split live output polling, agent summaries, compact recent runs, rerun controls, and recovery notices")
	}
	if !strings.Contains(body, ".submit-attempt-indicator") || !strings.Contains(body, "function submitAttemptIndicatorStateFC(attempt)") || !strings.Contains(body, "function renderSubmitAttemptIndicatorFC(attempt)") {
		t.Fatalf("expected floating recent-run cards to expose a top-right success or failure indicator")
	}
	if !strings.Contains(body, "function activeSubmitAttemptForLogFC()") || !strings.Contains(body, "if (runningAttempt) return runningAttempt;") {
		t.Fatalf("expected floating composer live log polling to stick to the active running attempt before the execution log path arrives")
	}
	if strings.Contains(body, "const useTail = status !== 'in_progress' && status !== 'queued';") || strings.Contains(body, "&tail=1") {
		t.Fatalf("did not expect floating recent-run previews to tail adapter logs; they should start at the beginning so startup/system output remains visible")
	}
	if !strings.Contains(body, "overflow-wrap: anywhere;") || !strings.Contains(body, "word-break: break-word;") {
		t.Fatalf("expected floating composer preformatted panels to wrap long unbroken log lines instead of stretching the layout")
	}
	if !strings.Contains(body, "function submitAttemptWorkspaceTextFC(attempt)") || !strings.Contains(body, "<strong>Workspace used:</strong>") {
		t.Fatalf("expected floating composer recent runs to expose the actual workspace used")
	}
	assertAllRenderedButtonsHaveTooltips(t, body)
	if !strings.Contains(body, "id=\"composerSettingsBtn\"") || !strings.Contains(body, "id=\"composerSettingsModal\"") || !strings.Contains(body, "Composer Settings") {
		t.Fatalf("expected floating composer to move configuration behind a gear-driven settings modal")
	}
	if !strings.Contains(body, "id=\"openDocsLibraryBtnFC\"") || !strings.Contains(body, "function openDocsBrowserFC(name)") || !strings.Contains(body, "/docs") {
		t.Fatalf("expected floating composer settings to expose the docs library in a new tab")
	}
	if !strings.Contains(body, "class=\"header-icon-plain\"") || !strings.Contains(body, "class=\"danger header-icon-plain\"") {
		t.Fatalf("expected floating composer header icons to render without full button outlines")
	}
	if !strings.Contains(body, "Workspace and agent configuration stay behind this gear menu") || !strings.Contains(body, "id=\"openWorkspaceBtn\"") || !strings.Contains(body, "id=\"openCodexRuntimeBtn\"") {
		t.Fatalf("expected floating composer settings modal to expose workspace and agent actions")
	}
	if !strings.Contains(body, "id=\"copyCompanionBtn\"") || !strings.Contains(body, "title=\"Connect browser. Copy the Browser Companion snippet.\"") {
		t.Fatalf("expected floating composer to include compact companion snippet action")
	}
	if !strings.Contains(body, "function captureBlockedReasonFC(kind = 'audio')") || !strings.Contains(body, "Choose a workspace first.") || !strings.Contains(body, "Start a session on the main Knit page first.") {
		t.Fatalf("expected floating composer to explain missing workspace/session prerequisites for capture")
	}
	if !strings.Contains(body, "id=\"openVideoCaptureBtn\"") || !strings.Contains(body, "🎥 Video tools") {
		t.Fatalf("expected floating composer settings modal to expose video tools behind the gear menu")
	}
	if !strings.Contains(body, "id=\"openAudioControlsBtn\"") || !strings.Contains(body, "🎚️ Audio controls") {
		t.Fatalf("expected floating composer settings modal to expose audio controls behind the gear menu")
	}
	if !strings.Contains(body, "id=\"fcAgentDefaultProvider\"") {
		t.Fatalf("expected floating composer runtime modal to include provider selector")
	}
	if !strings.Contains(body, "Profile maps to your local Codex config.toml profile.") {
		t.Fatalf("expected floating composer runtime modal to explain Codex profile/config.toml usage")
	}
	if strings.Contains(body, "fcCodexOptionsLoaded && !fcCodexOptionsAttempted") {
		t.Fatalf("did not expect floating composer to auto-load Codex options on refresh")
	}
	if !strings.Contains(body, "id=\"fcCodexCliSection\"") || !strings.Contains(body, "id=\"fcCodexAPISection\"") || !strings.Contains(body, "id=\"fcClaudeAPISection\"") || !strings.Contains(body, "id=\"fcCodexCLIDefaultsSection\"") {
		t.Fatalf("expected floating composer runtime modal to include provider-specific runtime sections")
	}
	if !strings.Contains(body, "Default submit adapter") || !strings.Contains(body, "CLI command") || !strings.Contains(body, "Approval policy") {
		t.Fatalf("expected floating composer runtime modal inputs to include visible labels")
	}
	if !strings.Contains(body, `<option value="claude_api">claude_api</option>`) || !strings.Contains(body, `id="fcClaudeAPIBaseURL"`) || !strings.Contains(body, `id="fcClaudeAPIModel"`) {
		t.Fatalf("expected floating composer runtime modal to expose claude_api controls")
	}
	if strings.Contains(body, "Use Codex CLI default (no override)") || !strings.Contains(body, "danger-full-access</code> sandbox and <code>never</code> approval") || !strings.Contains(body, "Allow the coding agent to make changes by switching Sandbox to danger-full-access") {
		t.Fatalf("expected floating composer runtime modal to show Knit-owned sandbox and approval defaults")
	}
	if !strings.Contains(body, "scheduleCodexRuntimeApplyFC") || !strings.Contains(body, "syncFCCodexRuntimeModeUI") {
		t.Fatalf("expected floating composer runtime modal to auto-save and toggle fields by provider")
	}
	if strings.Contains(body, "Apply Runtime") {
		t.Fatalf("expected stale apply-runtime button to be removed from floating composer runtime modal")
	}
	if !strings.Contains(body, "id=\"fcThemeToggleBtn\"") || !strings.Contains(body, "toggleThemeFC()") {
		t.Fatalf("expected floating composer to include a light/dark theme toggle")
	}
	if !strings.Contains(body, "id=\"fcToast\"") || !strings.Contains(body, "showToastFC('Connect browser link copied')") {
		t.Fatalf("expected floating composer to include toast feedback for connect-browser copy")
	}
	if !strings.Contains(body, `showToastFC('Request submitted to ' + destination + ' for "`) {
		t.Fatalf("expected floating composer to show a toast when submission succeeds")
	}
	if !strings.Contains(body, "function notifySubmitAttemptTransitionsFC(attempts)") || !strings.Contains(body, "showToastFC(submitAttemptToastMessageFC(attempt), status === 'failed' || submitAttemptNeedsAttentionFC(attempt))") {
		t.Fatalf("expected floating composer to show one-shot completion toasts for submit attempt state transitions")
	}
	if !strings.Contains(body, "function submitAttemptNeedsAttentionFC(attempt)") || !strings.Contains(body, "renderSubmitAttemptOutcomeFC(attempt)") || !strings.Contains(body, "<strong>Result:</strong>") {
		t.Fatalf("expected floating composer history to expose explicit no-op and blocked-run outcomes")
	}
	if !strings.Contains(body, "setFCSetting('theme'") || !strings.Contains(body, "applyThemeFC(normalizeThemeFC(fcSettings.theme || 'light'))") {
		t.Fatalf("expected floating composer theme preference to persist in localStorage")
	}
	if !strings.Contains(body, "knit_ui_settings_v1") || !strings.Contains(body, "initFCPersistentSettings()") {
		t.Fatalf("expected floating composer localStorage persistence hooks")
	}
	if strings.Contains(body, "id=\"approveBtn\"") {
		t.Fatalf("did not expect standalone approve button in floating composer")
	}
	if !strings.Contains(body, "id=\"queueLine\"") || !strings.Contains(body, "id=\"runtimeGuideLine\"") || !strings.Contains(body, "class=\"status-chip\"") {
		t.Fatalf("expected floating composer to move queue and runtime state into compact status chips")
	}
	if !strings.Contains(body, "id=\"fcPreviewDetails\"") || !strings.Contains(body, "<summary>Preview request</summary>") || strings.Contains(body, "<details id=\"fcPreviewDetails\" class=\"preview-card compact-preview\" open>") {
		t.Fatalf("expected floating composer preview to be available on demand rather than open by default")
	}
	if strings.Contains(body, "compact-status-row") {
		t.Fatalf("did not expect old inline status rows to remain in the floating composer layout")
	}
	if !strings.Contains(body, "id=\"toggleTextEditorBtn\"") || !strings.Contains(body, "Type note. Show the typed note field.") {
		t.Fatalf("expected floating composer to include compact text editor toggle tooltip")
	}
	if !strings.Contains(body, "startAudioNoteCaptureFC") || !strings.Contains(body, "finishAudioNoteCaptureFC") || !strings.Contains(body, "Recording voice note") {
		t.Fatalf("expected floating composer voice notes to support explicit start/stop recording")
	}
	if !strings.Contains(body, "startVideoNoteCaptureFC") || !strings.Contains(body, "finishVideoNoteCaptureFC") || !strings.Contains(body, "Stop recording video note") {
		t.Fatalf("expected floating composer video notes to support explicit start/stop recording")
	}
	if !strings.Contains(body, "function updatePreviewSubmitButtonsFC()") || !strings.Contains(body, "const blocked = !!inFlight || !!recording;") {
		t.Fatalf("expected floating composer preview and submit buttons to disable while recording")
	}
	if !strings.Contains(body, "finalizeVideoNoteCaptureFC") || !strings.Contains(body, "Video sharing ended. Finalizing your video note...") {
		t.Fatalf("expected floating composer video notes to finalize automatically when browser sharing ends")
	}
	if !strings.Contains(body, "ensureFeedbackPresentFC()") || !strings.Contains(body, "syncPreviewSessionStateFC(note?.session)") {
		t.Fatalf("expected floating composer video preview/send flow to recover from stale local state after note creation")
	}
	if strings.Contains(body, "recordAudio(4)") || strings.Contains(body, "Keep talking until the timer finishes.") {
		t.Fatalf("did not expect floating composer voice notes to remain timer-limited")
	}
	if strings.Contains(body, "recordVideoAndAudioNoteBundle(6)") {
		t.Fatalf("did not expect floating composer video notes to remain hard-limited to 6 seconds")
	}
	if !strings.Contains(body, "<textarea") || !strings.Contains(body, "id=\"transcript\"") || !strings.Contains(body, "hidden") {
		t.Fatalf("expected floating composer text editor to be hidden by default")
	}
	if !strings.Contains(body, "toggleTextEditorFC()") {
		t.Fatalf("expected floating composer to include text editor toggle handler")
	}
	if !strings.Contains(body, "ensureTextEditorOpenFC()") || !strings.Contains(body, "Typed note field opened. Enter your note, then click Snapshot + typed note again to capture it.") {
		t.Fatalf("expected floating composer snapshot typed-note action to open the text editor before capture")
	}
	if !strings.Contains(body, "Open the typed note field if needed, then capture a snapshot with the current note.") || !strings.Contains(body, "async function submitTextNoteWithSnapshot() {\n  ensureTextEditorOpenFC();") {
		t.Fatalf("expected floating composer snapshot typed-note action to visibly open the editor before capture")
	}
	if !strings.Contains(body, "openComposerSettingsModalFC()") || !strings.Contains(body, "closeComposerSettingsModalFC()") || !strings.Contains(body, "syncComposerSettingsSummaryFC()") {
		t.Fatalf("expected floating composer to wire settings modal open/close and summary sync")
	}
	if !strings.Contains(body, "if (!configLocked && !selectedWorkspace)") {
		t.Fatalf("expected floating composer to only auto-prompt workspace when unset")
	}
	if !strings.Contains(body, "id=\"startLiveVideoBtn\"") || !strings.Contains(body, "startLiveVideoFC()") {
		t.Fatalf("expected floating composer video modal to expose start live video control")
	}
	if !strings.Contains(body, "id=\"stopLiveVideoBtn\"") || !strings.Contains(body, "stopLiveVideoFC()") {
		t.Fatalf("expected floating composer video modal to expose stop live video control")
	}
	if !strings.Contains(body, "id=\"fcLivePreview\"") || !strings.Contains(body, "id=\"fcLiveVideoState\"") {
		t.Fatalf("expected floating composer video modal to include live preview elements")
	}
	if !strings.Contains(body, "Capture Snapshot") {
		t.Fatalf("expected floating snapshot capture control")
	}
	if !strings.Contains(body, "Recorder idle.") {
		t.Fatalf("expected floating recorder status indicator")
	}
	if !strings.Contains(body, "id=\"fcAudioLevelState\" class=\"small hidden\"") {
		t.Fatalf("expected floating mic audio level line to stay hidden outside mic test")
	}
	if !strings.Contains(body, "id=\"audioControlsModal\"") {
		t.Fatalf("expected floating composer to include audio controls modal")
	}
	if !strings.Contains(body, "openAudioControlsModal()") {
		t.Fatalf("expected floating composer to include audio controls trigger")
	}
	if !strings.Contains(body, "id=\"openAudioControlsBtn\"") || !strings.Contains(body, "🎚️ Audio controls") {
		t.Fatalf("expected floating composer to expose audio controls from the gear menu")
	}
	if strings.Contains(body, "id=\"fcPttQuickBtn\"") || strings.Contains(body, "PTT Hold to talk") {
		t.Fatalf("did not expect redundant PTT quick hold button in floating composer")
	}
	if strings.Contains(body, "Hold Push-To-Talk") {
		t.Fatalf("did not expect legacy hold-to-talk button in floating audio controls modal")
	}
	if !strings.Contains(body, "fcAudioMode") {
		t.Fatalf("expected floating composer modal to include audio mode controls")
	}
	if !strings.Contains(body, "id=\"openWorkspaceBtn\"") || !strings.Contains(body, "Open workspace settings") {
		t.Fatalf("expected floating composer settings modal to include workspace trigger")
	}
	if !strings.Contains(body, "id=\"workspaceModal\"") || !strings.Contains(body, "fcWorkspaceDir") {
		t.Fatalf("expected floating composer to include workspace modal and directory field")
	}
	if !strings.Contains(body, "id=\"openTranscriptionFromAudioBtnFC\"") || !strings.Contains(body, ">⚙️<") {
		t.Fatalf("expected floating audio controls modal to include nested gear transcription runtime trigger")
	}
	if !strings.Contains(body, "openTranscriptionRuntimeFromAudioModalFC()") {
		t.Fatalf("expected floating audio controls modal to include nested transcription runtime handler")
	}
	if !strings.Contains(body, "id=\"transcriptionRuntimeModal\"") {
		t.Fatalf("expected floating composer to include transcription runtime modal container")
	}
	if !strings.Contains(body, "id=\"videoCaptureModal\"") {
		t.Fatalf("expected floating composer to include video capture modal container")
	}
	if !strings.Contains(body, "id=\"codexRuntimeModal\"") {
		t.Fatalf("expected floating composer to include codex runtime modal container")
	}
	if !strings.Contains(body, "openCodexRuntimeModalFC()") {
		t.Fatalf("expected floating composer to include codex runtime modal trigger")
	}
	if !strings.Contains(body, "openVideoCaptureModalFC()") {
		t.Fatalf("expected floating composer to include video capture modal trigger")
	}
	if !strings.Contains(body, "fcSttMode") {
		t.Fatalf("expected floating composer transcription runtime modal to include stt mode control")
	}
	if !strings.Contains(body, "id=\"fcSttModeHelp\"") || !strings.Contains(body, "id=\"fcSttConnectionRow\"") || !strings.Contains(body, "id=\"fcSttFasterWhisperRow\"") || !strings.Contains(body, "id=\"fcSttCommandRow\"") {
		t.Fatalf("expected floating composer transcription runtime modal to include mode-specific field groups")
	}
	if !strings.Contains(body, "<select id=\"fcSttFasterWhisperModel\"") || !strings.Contains(body, "<option value=\"large-v3-turbo\">large-v3-turbo</option>") || !strings.Contains(body, "<select id=\"fcSttDevice\"") || !strings.Contains(body, "<option value=\"cpu\">cpu</option>") || !strings.Contains(body, "<select id=\"fcSttComputeType\"") || !strings.Contains(body, "<option value=\"int8\">int8</option>") {
		t.Fatalf("expected floating faster-whisper controls to use model/device/compute dropdowns")
	}
	if !strings.Contains(body, "id=\"fcSttLanguage\"") || !strings.Contains(body, "pattern=\"[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8}){0,2}\"") || !strings.Contains(body, "maxlength=\"2048\"") || !strings.Contains(body, "type=\"number\" min=\"1\" max=\"600\"") {
		t.Fatalf("expected floating transcription runtime inputs to include client-side safeguards")
	}
	if !strings.Contains(body, "syncFCSTTRuntimeModeUI") {
		t.Fatalf("expected floating composer transcription runtime modal to include mode-based field rendering")
	}
	if !strings.Contains(body, "scheduleTranscriptionRuntimeApplyFC") {
		t.Fatalf("expected floating composer transcription runtime modal to auto-apply changes")
	}
	if strings.Contains(body, "Apply Transcription Runtime") {
		t.Fatalf("expected stale floating apply-transcription-runtime button to be removed")
	}
	if !strings.Contains(body, "requireCompanionFC('capture snapshots')") {
		t.Fatalf("expected floating composer to require companion before snapshot capture")
	}
	if !strings.Contains(body, "requireCompanionFC('record video clips')") {
		t.Fatalf("expected floating composer to require companion before video clip capture")
	}
	if !strings.Contains(body, "previewPayloadFC()") || !strings.Contains(body, "submitSessionFC()") || !strings.Contains(body, "openLastLogFC()") {
		t.Fatalf("expected floating composer to include full review/send actions")
	}
	if strings.Contains(body, "id=\"fcAgentPromptSettingsBtn\"") {
		t.Fatalf("did not expect a standalone floating agent prompt settings button in the send controls")
	}
	if !strings.Contains(body, "openCodexRuntimeModalFC()") || !strings.Contains(body, "id=\"fcDeliveryPromptSection\"") || !strings.Contains(body, "id=\"fcDeliveryIntentProfile\"") || !strings.Contains(body, "id=\"fcDeliveryInstructionText\"") {
		t.Fatalf("expected floating composer to expose delivery prompt controls inside agent runtime settings")
	}
	if !strings.Contains(body, "editPreviewNoteFC(") || !strings.Contains(body, "deletePreviewNoteFC(") {
		t.Fatalf("expected floating composer preview UI to support editing and deleting change requests")
	}
	if !strings.Contains(body, "Make clip smaller to send") || !strings.Contains(body, "/api/session/feedback/clip?event_id=") {
		t.Fatalf("expected floating composer preview UI to support resizing oversized clips")
	}
}

func TestFloatingComposerRecentRunsExposeStatusIndicatorHelpers(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/floating-composer", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, ".submit-attempt-indicator") ||
		!strings.Contains(body, "function submitAttemptIndicatorStateFC(attempt)") ||
		!strings.Contains(body, "function renderSubmitAttemptIndicatorFC(attempt)") ||
		!strings.Contains(body, "function submitAttemptCardShouldStartOpenFC(attempt, index)") ||
		!strings.Contains(body, "data-submit-attempt-output=\"output\"") ||
		!strings.Contains(body, "function shouldStickScrollFC(el, threshold = 24)") {
		t.Fatalf("expected floating composer recent runs to expose compact cards, scroll helpers, and status indicators")
	}
}

func TestMainUIIncludesSubmitAttemptDeepLinkFocusHooks(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/?attempt_id=attempt-123", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	required := []string{
		`function requestedSubmitAttemptIDFromLocation()`,
		`searchParams.get('attempt_id')`,
		`function focusSubmitAttemptFromLocation()`,
		`focused-attempt`,
		`window.addEventListener('popstate', focusSubmitAttemptFromLocation);`,
	}
	for _, fragment := range required {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected main UI to include deep-link focus hook %q", fragment)
		}
	}
}

func TestDocsViewEndpointReturnsLocalDocContent(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/docs/view?name=getting_started", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := payload["name"]; got != "GETTING_STARTED.md" {
		t.Fatalf("expected GETTING_STARTED.md, got %v", got)
	}
	if got := payload["label"]; got != "Getting Started" {
		t.Fatalf("expected Getting Started label, got %v", got)
	}
	content, _ := payload["content"].(string)
	if !strings.Contains(content, "# Getting Started") {
		t.Fatalf("expected getting started markdown content, got %q", content)
	}
	if !strings.Contains(content, ":::tabs") || !strings.Contains(content, "@tab npm") {
		t.Fatalf("expected getting started markdown to include tabbed install/start content, got %q", content)
	}
	if !strings.Contains(content, "![Main UI Browser Extension section") || !strings.Contains(content, "/docs/assets/browser-extension-pairing-code.png") {
		t.Fatalf("expected getting started markdown to include browser extension screenshots, got %q", content)
	}
	contentHTML, _ := payload["content_html"].(string)
	if !strings.Contains(contentHTML, "<h1 id=\"getting-started\">Getting Started</h1>") || !strings.Contains(contentHTML, "<h2 id=\"what-runs\">What Runs</h2>") || !strings.Contains(contentHTML, "class=\"doc-tabs\"") || !strings.Contains(contentHTML, "class=\"doc-image\"") || !strings.Contains(contentHTML, "/docs/assets/browser-extension-pairing-code.png") || !strings.Contains(contentHTML, "@chadsly/knit") || !strings.Contains(contentHTML, "<pre class=\"doc-code\"><code>") {
		t.Fatalf("expected rendered markdown html, got %q", contentHTML)
	}
	headings, _ := payload["headings"].([]any)
	if len(headings) < 2 {
		t.Fatalf("expected heading outline metadata, got %#v", payload["headings"])
	}
	path, _ := payload["path"].(string)
	if !strings.HasSuffix(path, filepath.Join("docs", "GETTING_STARTED.md")) {
		t.Fatalf("expected docs path suffix, got %q", path)
	}
}

func TestDocsCatalogEndpointReturnsAvailableDocs(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/docs/catalog", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	docs, _ := payload["docs"].([]any)
	if len(docs) < 2 {
		t.Fatalf("expected at least two docs, got %#v", payload["docs"])
	}
}

func TestDocsBrowserEndpointRequiresAuthAndRenders(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	unauthReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	unauthRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated docs browser request to fail with 401, got %d", unauthRec.Code)
	}

	authReq := httptest.NewRequest(http.MethodGet, "/docs?token="+url.QueryEscape(cfg.ControlToken), nil)
	authRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", authRec.Code, authRec.Body.String())
	}
	body := authRec.Body.String()
	if !strings.Contains(body, "id=\"docsCatalog\"") || !strings.Contains(body, "id=\"docsOutline\"") || !strings.Contains(body, "id=\"docsOpenCurrentTabBtn\"") || !strings.Contains(body, "function initDocTabs()") {
		t.Fatalf("expected docs browser to include the catalog, outline, and article viewer")
	}
}

func TestDocsAssetRouteServesScreenshot(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/docs/assets/browser-extension-pairing-code.png", nil)
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "image/png") {
		t.Fatalf("expected image/png content type, got %q", got)
	}
	if len(rec.Body.Bytes()) == 0 {
		t.Fatalf("expected asset response body")
	}
}

func TestDocsViewEndpointRejectsUnknownDoc(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/docs/view?name=missing", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStateIncludesAudioAndCaptureHealth(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := payload["audio"].(map[string]any); !ok {
		t.Fatalf("expected audio state payload")
	}
	if _, ok := payload["capture_sources"].(map[string]any); !ok {
		t.Fatalf("expected capture_sources payload")
	}
}

func TestAudioConfigAndLevelEndpoints(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	cfgReq := httptest.NewRequest(http.MethodPost, "/api/audio/config", bytes.NewReader([]byte(`{"mode":"always_on","input_device_id":"dev-1","muted":true}`)))
	cfgReq.Header.Set("Content-Type", "application/json")
	addAuth(cfgReq, cfg.ControlToken, true, "nonce-audio-config")
	cfgRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(cfgRec, cfgReq)
	if cfgRec.Code != http.StatusOK {
		t.Fatalf("audio config failed: %d %s", cfgRec.Code, cfgRec.Body.String())
	}

	levelReq := httptest.NewRequest(http.MethodPost, "/api/audio/level", bytes.NewReader([]byte(`{"level":0.5}`)))
	levelReq.Header.Set("Content-Type", "application/json")
	addAuth(levelReq, cfg.ControlToken, true, "nonce-audio-level")
	levelRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(levelRec, levelReq)
	if levelRec.Code != http.StatusOK {
		t.Fatalf("audio level failed: %d %s", levelRec.Code, levelRec.Body.String())
	}
}

func TestAudioDevicesEndpointSupportsSelection(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	devReq := httptest.NewRequest(http.MethodPost, "/api/audio/devices", bytes.NewReader([]byte(`{"devices":[{"id":"usb-mic","label":"USB Mic"}]}`)))
	devReq.Header.Set("Content-Type", "application/json")
	addAuth(devReq, cfg.ControlToken, true, "nonce-audio-devices")
	devRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(devRec, devReq)
	if devRec.Code != http.StatusOK {
		t.Fatalf("audio devices failed: %d %s", devRec.Code, devRec.Body.String())
	}

	cfgReq := httptest.NewRequest(http.MethodPost, "/api/audio/config", bytes.NewReader([]byte(`{"input_device_id":"usb-mic","mode":"push_to_talk"}`)))
	cfgReq.Header.Set("Content-Type", "application/json")
	addAuth(cfgReq, cfg.ControlToken, true, "nonce-audio-device-select")
	cfgRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(cfgRec, cfgReq)
	if cfgRec.Code != http.StatusOK {
		t.Fatalf("audio config failed: %d %s", cfgRec.Code, cfgRec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(cfgRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	state, _ := payload["state"].(map[string]any)
	if got := state["input_device_id"]; got != "usb-mic" {
		t.Fatalf("expected selected device id usb-mic, got %#v", got)
	}
}

func TestCaptureSourceEndpointUpdatesReducedCapabilities(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/capture/source", bytes.NewReader([]byte(`{"source":"screen","status":"unavailable","reason":"permission denied"}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-capture-src")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("capture source update failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	reduced, _ := payload["reduced_capabilities"].([]any)
	if len(reduced) == 0 {
		t.Fatalf("expected reduced capabilities entry after unavailable source")
	}
}

func TestDiscardLastFeedbackEndpoint(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-discard")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"first note","normalized":"first note","pointer_x":10,"pointer_y":20,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "nonce-feedback-discard")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}

	discardReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/discard-last", bytes.NewReader([]byte(`{}`)))
	discardReq.Header.Set("Content-Type", "application/json")
	addAuth(discardReq, cfg.ControlToken, true, "nonce-discard-last")
	discardRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(discardRec, discardReq)
	if discardRec.Code != http.StatusOK {
		t.Fatalf("discard failed: %d %s", discardRec.Code, discardRec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(discardRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode discard response: %v", err)
	}
	if got := payload["discarded_event_id"]; got == "" || got == nil {
		t.Fatalf("expected discarded_event_id, got %#v", got)
	}
	sessionObj, _ := payload["session"].(map[string]any)
	feedback, _ := sessionObj["feedback"].([]any)
	if len(feedback) != 0 {
		t.Fatalf("expected feedback list to be empty after discard, got %d", len(feedback))
	}
}

func TestStateIncludesPointerSampleRateAndVideoMode(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.PointerSampleHz = 60
	cfg.VideoMode = "continuous"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := payload["pointer_sample_hz"]; got != float64(60) {
		t.Fatalf("expected pointer_sample_hz=60, got %#v", got)
	}
	if got := payload["video_mode"]; got != "continuous" {
		t.Fatalf("expected video_mode=continuous, got %#v", got)
	}
	if _, ok := payload["runtime_transcription"].(map[string]any); !ok {
		t.Fatalf("expected runtime_transcription payload")
	}
	if _, ok := payload["native_capture_modules"].([]any); !ok {
		t.Fatalf("expected native_capture_modules payload")
	}
	platformProfile, _ := payload["platform_profile"].(map[string]any)
	if platformProfile["goos"] != runtime.GOOS {
		t.Fatalf("expected platform_profile.goos=%q, got %#v", runtime.GOOS, platformProfile["goos"])
	}
	if _, ok := platformProfile["permissions"].([]any); !ok {
		t.Fatalf("expected platform_profile.permissions payload")
	}
}

func TestConfigImportBlockedWhenLocked(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.ConfigLocked = true
	srv := newTestServer(t, cfg)

	body := []byte(`{"profile":"personal_local_dev"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-1")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusLocked {
		t.Fatalf("expected %d, got %d: %s", http.StatusLocked, rec.Code, rec.Body.String())
	}
}

func TestStateDefaultsExposeManagedFasterWhisper(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := payload["transcription_mode"]; got != "faster_whisper" {
		t.Fatalf("expected transcription_mode=faster_whisper, got %#v", got)
	}
	if got := payload["transcription_provider"]; got != "managed_faster_whisper_stt" {
		t.Fatalf("expected transcription_provider=managed_faster_whisper_stt, got %#v", got)
	}
	rt, _ := payload["runtime_transcription"].(map[string]any)
	if got := rt["mode"]; got != "faster_whisper" {
		t.Fatalf("expected runtime_transcription.mode=faster_whisper, got %#v", got)
	}
	autoStart, _ := payload["auto_start"].(map[string]any)
	if enabled, _ := autoStart["enabled"].(bool); enabled {
		t.Fatalf("expected auto-start disabled by default")
	}
}

func TestConfigImportCanEnableAutoStart(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	body := []byte(`{"config":{"auto_start_enabled":true}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-autostart-import")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if !srv.currentConfig().AutoStartEnabled {
		t.Fatalf("expected config auto-start enabled after import")
	}
	status := srv.autoStart.Status()
	if !status.Registered || !status.Enabled {
		t.Fatalf("expected auto-start manager to register entry, got %#v", status)
	}

	stateReq := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(stateReq, cfg.ControlToken, false, "")
	stateRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("state failed: %d %s", stateRec.Code, stateRec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stateRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode state response: %v", err)
	}
	autoStart, _ := payload["auto_start"].(map[string]any)
	if enabled, _ := autoStart["enabled"].(bool); !enabled {
		t.Fatalf("expected auto-start state enabled, got %#v", autoStart)
	}
}

func TestConfigImportCanEnableLockAndThenRejectFurtherChanges(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	firstReq := httptest.NewRequest(http.MethodPost, "/api/config/import", bytes.NewReader([]byte(`{"profile":"enterprise_managed_workstation"}`)))
	firstReq.Header.Set("Content-Type", "application/json")
	addAuth(firstReq, cfg.ControlToken, true, "nonce-2")
	firstRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(firstRec, firstReq)

	if firstRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", firstRec.Code, firstRec.Body.String())
	}
	if !srv.currentConfig().ConfigLocked {
		t.Fatalf("expected config to be locked after enterprise profile import")
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/config/import", bytes.NewReader([]byte(`{"profile":"personal_local_dev"}`)))
	secondReq.Header.Set("Content-Type", "application/json")
	addAuth(secondReq, cfg.ControlToken, true, "nonce-3")
	secondRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusLocked {
		t.Fatalf("expected %d after lock, got %d: %s", http.StatusLocked, secondRec.Code, secondRec.Body.String())
	}
}

func TestMutationRequiresNonceAndRejectsReplay(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	noNonceReq := httptest.NewRequest(http.MethodPost, "/api/capture/kill", bytes.NewReader([]byte(`{}`)))
	noNonceReq.Header.Set("Content-Type", "application/json")
	addAuth(noNonceReq, cfg.ControlToken, false, "")
	noNonceRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noNonceRec, noNonceReq)
	if noNonceRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing nonce/timestamp, got %d", noNonceRec.Code)
	}

	nonce := "nonce-replay"
	firstReq := httptest.NewRequest(http.MethodPost, "/api/capture/kill", bytes.NewReader([]byte(`{}`)))
	firstReq.Header.Set("Content-Type", "application/json")
	addAuth(firstReq, cfg.ControlToken, true, nonce)
	firstRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid nonce request, got %d: %s", firstRec.Code, firstRec.Body.String())
	}

	replayReq := httptest.NewRequest(http.MethodPost, "/api/capture/kill", bytes.NewReader([]byte(`{}`)))
	replayReq.Header.Set("Content-Type", "application/json")
	addAuth(replayReq, cfg.ControlToken, true, nonce)
	replayRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(replayRec, replayReq)
	if replayRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for replay nonce, got %d", replayRec.Code)
	}
}

func TestCapabilityGateBlocksCaptureActions(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.ControlCapabilities = []string{"read"}
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-cap")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for capture action with read-only capability, got %d", startRec.Code)
	}

	stateReq := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(stateReq, cfg.ControlToken, false, "")
	stateRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for read endpoint, got %d", stateRec.Code)
	}
}

func TestSessionStartIncludesConfiguredMetadata(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.LocalProfile = "enterprise_managed_workstation"
	cfg.EnvironmentName = "staging"
	cfg.BuildID = "build-xyz"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-meta")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var sess session.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &sess); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if sess.Profile != cfg.LocalProfile || sess.Environment != cfg.EnvironmentName || sess.BuildID != cfg.BuildID {
		t.Fatalf("unexpected session metadata: profile=%q env=%q build=%q", sess.Profile, sess.Environment, sess.BuildID)
	}
}

func TestRemoteSubmissionDisabledByPolicy(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.AllowRemoteSubmission = false
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-remote")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start session failed: %d %s", startRec.Code, startRec.Body.String())
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"Fix this bug","normalized":"Fix this bug","pointer_x":100,"pointer_y":80,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "nonce-feedback-remote")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-approve-remote")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"codex_api"}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "nonce-submit-remote")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when remote submissions are disabled, got %d: %s", submitRec.Code, submitRec.Body.String())
	}

	claudeSubmitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"claude_api"}`)))
	claudeSubmitReq.Header.Set("Content-Type", "application/json")
	addAuth(claudeSubmitReq, cfg.ControlToken, true, "nonce-submit-claude-remote")
	claudeSubmitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(claudeSubmitRec, claudeSubmitReq)
	if claudeSubmitRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when claude_api remote submissions are disabled, got %d: %s", claudeSubmitRec.Code, claudeSubmitRec.Body.String())
	}
}

func TestRemoteTranscriptionDisabledByPolicy(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.TranscriptionMode = "remote"
	cfg.AllowRemoteSTT = false
	srv := newTestServerWithSTT(t, cfg, fakeRemoteSTTProvider{})

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-stt")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start session failed: %d %s", startRec.Code, startRec.Body.String())
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fw, err := writer.CreateFormFile("audio", "note.webm")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := fw.Write([]byte("fake-audio")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	addAuth(req, cfg.ControlToken, true, "nonce-note-stt")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when remote stt is disabled, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitAndPreviewUseConfiguredDefaultProvider(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	t.Setenv("KNIT_DEFAULT_PROVIDER", "claude_cli")
	t.Setenv("KNIT_CLAUDE_CLI_ADAPTER_CMD", `echo '{"run_id":"claude-run","status":"accepted","ref":"claude-ref"}'`)
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-default-provider-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"Fix button color","normalized":"Fix button color","pointer_x":100,"pointer_y":80,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "nonce-default-provider-feedback")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-default-provider-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-default-provider-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}
	var previewPayload payloadPreviewResponse
	if err := json.Unmarshal(previewRec.Body.Bytes(), &previewPayload); err != nil {
		t.Fatalf("decode preview payload: %v", err)
	}
	if previewPayload.Provider != "claude_cli" {
		t.Fatalf("expected preview provider claude_cli, got %#v", previewPayload.Provider)
	}
	if len(previewPayload.Preview.Notes) != 1 {
		t.Fatalf("expected one preview note, got %d", len(previewPayload.Preview.Notes))
	}
	if previewPayload.Preview.Notes[0].Text != "Fix button color" {
		t.Fatalf("expected preview note text, got %#v", previewPayload.Preview.Notes[0].Text)
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "nonce-default-provider-submit")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("submit failed: %d %s", submitRec.Code, submitRec.Body.String())
	}
	var submitPayload map[string]any
	if err := json.Unmarshal(submitRec.Body.Bytes(), &submitPayload); err != nil {
		t.Fatalf("decode submit payload: %v", err)
	}
	if got := submitPayload["provider"]; got != "claude_cli" {
		t.Fatalf("expected submit provider claude_cli, got %#v", got)
	}
}

func TestPreviewFeedbackCanEditAndDeleteNotes(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-preview-edit-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	for i, text := range []string{"first change", "second change"} {
		req := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"`+text+`","normalized":"`+text+`","pointer_x":10,"pointer_y":12,"window":"Browser Preview"}`)))
		req.Header.Set("Content-Type", "application/json")
		addAuth(req, cfg.ControlToken, true, "nonce-preview-edit-feedback-"+strconv.Itoa(i))
		rec := httptest.NewRecorder()
		srv.httpSrv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("feedback %d failed: %d %s", i, rec.Code, rec.Body.String())
		}
	}

	curr := srv.sessions.Current()
	if curr == nil || len(curr.Feedback) != 2 {
		t.Fatalf("expected two feedback events, got %#v", curr)
	}
	firstID := curr.Feedback[0].ID
	secondID := curr.Feedback[1].ID

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-preview-edit-approve-1")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/update-text", bytes.NewReader([]byte(`{"event_id":"`+firstID+`","text":"edited change request"}`)))
	updateReq.Header.Set("Content-Type", "application/json")
	addAuth(updateReq, cfg.ControlToken, true, "nonce-preview-edit-update")
	updateRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update text failed: %d %s", updateRec.Code, updateRec.Body.String())
	}
	var updatePayload map[string]any
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updatePayload); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	updatedSession, _ := updatePayload["session"].(map[string]any)
	if approved, _ := updatedSession["approved"].(bool); approved {
		t.Fatalf("expected approval to clear after text update")
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-preview-edit-preview-fail")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected preview to require re-approval after edit, got %d: %s", previewRec.Code, previewRec.Body.String())
	}

	approveReq = httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-preview-edit-approve-2")
	approveRec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("re-approve after edit failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/delete", bytes.NewReader([]byte(`{"event_id":"`+secondID+`"}`)))
	deleteReq.Header.Set("Content-Type", "application/json")
	addAuth(deleteReq, cfg.ControlToken, true, "nonce-preview-edit-delete")
	deleteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete feedback failed: %d %s", deleteRec.Code, deleteRec.Body.String())
	}
	var deletePayload map[string]any
	if err := json.Unmarshal(deleteRec.Body.Bytes(), &deletePayload); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if got, _ := deletePayload["deleted_event_id"].(string); got != secondID {
		t.Fatalf("expected deleted event %q, got %q", secondID, got)
	}

	approveReq = httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-preview-edit-approve-3")
	approveRec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("re-approve after delete failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	previewReq = httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-preview-edit-preview-final")
	previewRec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("final preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}
	var previewPayload payloadPreviewResponse
	if err := json.Unmarshal(previewRec.Body.Bytes(), &previewPayload); err != nil {
		t.Fatalf("decode final preview: %v", err)
	}
	if len(previewPayload.Preview.Notes) != 1 {
		t.Fatalf("expected one note after delete, got %d", len(previewPayload.Preview.Notes))
	}
	if got := previewPayload.Preview.Notes[0].Text; got != "edited change request" {
		t.Fatalf("expected edited preview text, got %q", got)
	}
}

func TestPayloadPreviewExcludesPreviouslySubmittedRequests(t *testing.T) {
	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"preview-submitted","status":"accepted","ref":"/tmp/preview-submitted.log"}'`)
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-preview-submitted-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	firstFeedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"first submitted change","normalized":"first submitted change","pointer_x":10,"pointer_y":12,"window":"Browser Preview"}`)))
	firstFeedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(firstFeedbackReq, cfg.ControlToken, true, "nonce-preview-submitted-feedback-1")
	firstFeedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(firstFeedbackRec, firstFeedbackReq)
	if firstFeedbackRec.Code != http.StatusOK {
		t.Fatalf("first feedback failed: %d %s", firstFeedbackRec.Code, firstFeedbackRec.Body.String())
	}

	firstApproveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	firstApproveReq.Header.Set("Content-Type", "application/json")
	addAuth(firstApproveReq, cfg.ControlToken, true, "nonce-preview-submitted-approve-1")
	firstApproveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(firstApproveRec, firstApproveReq)
	if firstApproveRec.Code != http.StatusOK {
		t.Fatalf("first approve failed: %d %s", firstApproveRec.Code, firstApproveRec.Body.String())
	}

	firstSubmitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	firstSubmitReq.Header.Set("Content-Type", "application/json")
	addAuth(firstSubmitReq, cfg.ControlToken, true, "nonce-preview-submitted-submit-1")
	firstSubmitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(firstSubmitRec, firstSubmitReq)
	if firstSubmitRec.Code != http.StatusAccepted {
		t.Fatalf("first submit failed: %d %s", firstSubmitRec.Code, firstSubmitRec.Body.String())
	}
	var firstSubmitPayload map[string]any
	if err := json.Unmarshal(firstSubmitRec.Body.Bytes(), &firstSubmitPayload); err != nil {
		t.Fatalf("decode first submit response: %v", err)
	}
	firstAttemptID, _ := firstSubmitPayload["attempt_id"].(string)
	if firstAttemptID == "" {
		t.Fatalf("expected first attempt id")
	}
	_ = waitForAttemptStatus(t, srv, cfg.ControlToken, firstAttemptID, "submitted", 3*time.Second)

	secondFeedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"second pending change","normalized":"second pending change","pointer_x":10,"pointer_y":12,"window":"Browser Preview"}`)))
	secondFeedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(secondFeedbackReq, cfg.ControlToken, true, "nonce-preview-submitted-feedback-2")
	secondFeedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(secondFeedbackRec, secondFeedbackReq)
	if secondFeedbackRec.Code != http.StatusOK {
		t.Fatalf("second feedback failed: %d %s", secondFeedbackRec.Code, secondFeedbackRec.Body.String())
	}

	secondApproveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	secondApproveReq.Header.Set("Content-Type", "application/json")
	addAuth(secondApproveReq, cfg.ControlToken, true, "nonce-preview-submitted-approve-2")
	secondApproveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(secondApproveRec, secondApproveReq)
	if secondApproveRec.Code != http.StatusOK {
		t.Fatalf("second approve failed: %d %s", secondApproveRec.Code, secondApproveRec.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-preview-submitted-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}

	var previewPayload payloadPreviewResponse
	if err := json.Unmarshal(previewRec.Body.Bytes(), &previewPayload); err != nil {
		t.Fatalf("decode preview payload: %v", err)
	}
	if previewPayload.Preview.ChangeRequestCount != 1 {
		t.Fatalf("expected one prepared request after previous submit, got %d", previewPayload.Preview.ChangeRequestCount)
	}
	if len(previewPayload.Preview.Notes) != 1 {
		t.Fatalf("expected one preview note after previous submit, got %d", len(previewPayload.Preview.Notes))
	}
	if got := previewPayload.Preview.Notes[0].Text; got != "second pending change" {
		t.Fatalf("expected only the new request in preview, got %q", got)
	}
}

func TestReplayPreviewAndExportRespectValueCaptureSetting(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-replay-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var started map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := started["id"].(string)
	if enabled, _ := started["capture_input_values"].(bool); !enabled {
		t.Fatalf("expected replay typed-value capture to default on for new sessions")
	}

	settingsReq := httptest.NewRequest(http.MethodPost, "/api/session/replay/settings", bytes.NewReader([]byte(`{"capture_input_values":true}`)))
	settingsReq.Header.Set("Content-Type", "application/json")
	addAuth(settingsReq, cfg.ControlToken, true, "nonce-replay-settings")
	settingsRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(settingsRec, settingsReq)
	if settingsRec.Code != http.StatusOK {
		t.Fatalf("replay settings failed: %d %s", settingsRec.Code, settingsRec.Body.String())
	}

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{
		"session_id":"`+sessionID+`",
		"x":250,
		"y":92,
		"event_type":"input",
		"window":"Browser Preview",
		"url":"https://example.com/app/editor?draft=secret",
		"route":"/app/editor",
		"target_tag":"input",
		"target_id":"headline",
		"target_test_id":"headline-field",
		"target_role":"textbox",
		"target_label":"Headline",
		"target_selector":"#headline",
		"input_type":"text",
		"value":"Make the primary button larger",
		"value_captured":true,
		"dom":{"tag":"input","id":"headline","test_id":"headline-field","role":"textbox","label":"Headline","selector":"#headline","text_preview":"","attributes":{"placeholder":"Headline"}},
		"console":[{"level":"error","message":"Validation failed"}],
		"network":[{"kind":"fetch","method":"POST","url":"https://example.com/api/save?token=secret","status":500,"ok":false,"duration_ms":120}]
	}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "nonce-replay-pointer")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusOK {
		t.Fatalf("pointer failed: %d %s", pointerRec.Code, pointerRec.Body.String())
	}

	noteBody, noteCT := multipartNoteBody(t, "Reproduce the save failure", nil)
	noteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", noteBody)
	noteReq.Header.Set("Content-Type", noteCT)
	addAuth(noteReq, cfg.ControlToken, true, "nonce-replay-note")
	noteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusOK {
		t.Fatalf("feedback note failed: %d %s", noteRec.Code, noteRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":"Reproduce the save failure"}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-replay-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"codex_cli"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-replay-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}
	var previewPayload payloadPreviewResponse
	if err := json.Unmarshal(previewRec.Body.Bytes(), &previewPayload); err != nil {
		t.Fatalf("decode preview payload: %v", err)
	}
	if len(previewPayload.Preview.Notes) != 1 {
		t.Fatalf("expected one preview note, got %d", len(previewPayload.Preview.Notes))
	}
	note := previewPayload.Preview.Notes[0]
	if note.ReplayValueMode != "opt_in" {
		t.Fatalf("expected replay value mode opt_in, got %#v", note.ReplayValueMode)
	}
	if note.ReplayStepCount == 0 || len(note.ReplaySteps) == 0 {
		t.Fatalf("expected replay steps in preview, got %#v", note)
	}
	if !strings.Contains(note.ReplaySteps[0], "Make the primary button larger") {
		t.Fatalf("expected replay preview to include captured value, got %#v", note.ReplaySteps)
	}
	if !strings.Contains(note.PlaywrightScript, ".fill(\"Make the primary button larger\")") {
		t.Fatalf("expected playwright preview script to include fill step, got %q", note.PlaywrightScript)
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/api/session/replay/export?event_id="+url.QueryEscape(note.EventID)+"&format=json", nil)
	addAuth(exportReq, cfg.ControlToken, false, "")
	exportRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(exportRec, exportReq)
	if exportRec.Code != http.StatusOK {
		t.Fatalf("replay export failed: %d %s", exportRec.Code, exportRec.Body.String())
	}
	if !strings.Contains(exportRec.Body.String(), "\"value\": \"Make the primary button larger\"") {
		t.Fatalf("expected replay json export to include captured value, got %s", exportRec.Body.String())
	}
	if strings.Contains(exportRec.Body.String(), "draft=secret") {
		t.Fatalf("expected replay json export to sanitize query strings, got %s", exportRec.Body.String())
	}

	pkg, err := srv.sessions.ApprovedPackage()
	if err != nil {
		t.Fatalf("approved package: %v", err)
	}
	if len(pkg.ChangeRequests) != 1 || pkg.ChangeRequests[0].Replay == nil || len(pkg.ChangeRequests[0].Replay.Exports) != 2 {
		t.Fatalf("expected replay exports in approved package, got %#v", pkg.ChangeRequests)
	}
	if len(pkg.Artifacts) < 2 {
		t.Fatalf("expected replay artifacts in package, got %#v", pkg.Artifacts)
	}
}

func TestReplayExportRedactsValuesByDefault(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-replay-redacted-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var started map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := started["id"].(string)

	settingsReq := httptest.NewRequest(http.MethodPost, "/api/session/replay/settings", bytes.NewReader([]byte(`{"capture_input_values":false}`)))
	settingsReq.Header.Set("Content-Type", "application/json")
	addAuth(settingsReq, cfg.ControlToken, true, "nonce-replay-redacted-settings")
	settingsRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(settingsRec, settingsReq)
	if settingsRec.Code != http.StatusOK {
		t.Fatalf("replay settings disable failed: %d %s", settingsRec.Code, settingsRec.Body.String())
	}

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{
		"session_id":"`+sessionID+`",
		"x":250,
		"y":92,
		"event_type":"input",
		"window":"Browser Preview",
		"url":"https://example.com/app/login",
		"route":"/app/login",
		"target_tag":"input",
		"target_id":"email",
		"target_label":"Email address",
		"target_selector":"#email",
		"input_type":"text",
		"value":"hidden@example.com",
		"value_captured":true
	}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "nonce-replay-redacted-pointer")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusOK {
		t.Fatalf("pointer failed: %d %s", pointerRec.Code, pointerRec.Body.String())
	}

	noteBody, noteCT := multipartNoteBody(t, "Reproduce the login issue", nil)
	noteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", noteBody)
	noteReq.Header.Set("Content-Type", noteCT)
	addAuth(noteReq, cfg.ControlToken, true, "nonce-replay-redacted-note")
	noteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusOK {
		t.Fatalf("feedback note failed: %d %s", noteRec.Code, noteRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":"Reproduce the login issue"}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-replay-redacted-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"codex_cli"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-replay-redacted-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}
	var previewPayload payloadPreviewResponse
	if err := json.Unmarshal(previewRec.Body.Bytes(), &previewPayload); err != nil {
		t.Fatalf("decode preview payload: %v", err)
	}
	note := previewPayload.Preview.Notes[0]
	if note.ReplayValueMode != "redacted" {
		t.Fatalf("expected replay value mode redacted, got %#v", note.ReplayValueMode)
	}
	if strings.Contains(strings.Join(note.ReplaySteps, "\n"), "hidden@example.com") {
		t.Fatalf("expected preview replay steps to redact default values, got %#v", note.ReplaySteps)
	}
	if !strings.Contains(note.PlaywrightScript, "typed value was redacted") {
		t.Fatalf("expected redaction comment in playwright script, got %q", note.PlaywrightScript)
	}
}

func TestSanitizeCompanionInputValueMasksObviousSecrets(t *testing.T) {
	tests := []struct {
		name         string
		evt          companion.PointerEvent
		wantValue    string
		wantRedacted bool
	}{
		{
			name: "password omitted",
			evt: companion.PointerEvent{
				InputType:     "password",
				TargetID:      "password",
				TargetLabel:   "Password",
				Value:         "super-secret",
				ValueCaptured: true,
			},
			wantValue:    "",
			wantRedacted: true,
		},
		{
			name: "token masked",
			evt: companion.PointerEvent{
				InputType:     "text",
				TargetID:      "api_token",
				TargetLabel:   "API token",
				Value:         "tok_live_1234567890",
				ValueCaptured: true,
			},
			wantValue:    "tok_...7890",
			wantRedacted: false,
		},
		{
			name: "card masked",
			evt: companion.PointerEvent{
				InputType:     "text",
				TargetID:      "credit_card",
				TargetLabel:   "Credit card number",
				Value:         "4242 4242 4242 4242",
				ValueCaptured: true,
			},
			wantValue:    "**** **** **** 4242",
			wantRedacted: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotValue, gotRedacted := sanitizeCompanionInputValue(tc.evt)
			if gotValue != tc.wantValue || gotRedacted != tc.wantRedacted {
				t.Fatalf("sanitizeCompanionInputValue() = (%q, %v), want (%q, %v)", gotValue, gotRedacted, tc.wantValue, tc.wantRedacted)
			}
		})
	}
}

func TestFeedbackNoteWithAudioBlockedWhenMuted(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServerWithSTT(t, cfg, fakeRemoteSTTProvider{})

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-muted")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start session failed: %d %s", startRec.Code, startRec.Body.String())
	}

	muteReq := httptest.NewRequest(http.MethodPost, "/api/audio/config", bytes.NewReader([]byte(`{"muted":true}`)))
	muteReq.Header.Set("Content-Type", "application/json")
	addAuth(muteReq, cfg.ControlToken, true, "nonce-mute")
	muteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(muteRec, muteReq)
	if muteRec.Code != http.StatusOK {
		t.Fatalf("mute config failed: %d %s", muteRec.Code, muteRec.Body.String())
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fw, err := writer.CreateFormFile("audio", "note.webm")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := fw.Write([]byte("fake-audio")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	addAuth(req, cfg.ControlToken, true, "nonce-note-muted")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 when muted, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanionPointerPersistsGroundingFieldsInState(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-grounding")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var sess map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &sess); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := sess["id"].(string)
	if sessionID == "" {
		t.Fatalf("missing session id")
	}

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{
		"session_id":"`+sessionID+`",
		"x":612,
		"y":384,
		"event_type":"move",
		"window":"Browser Preview",
		"url":"https://example.com/app",
		"route":"/app/settings",
		"target_tag":"button",
		"target_id":"save",
		"target_test_id":"settings-save",
		"target_label":"Save Settings",
		"target_selector":"#save",
		"dom":{"tag":"button","id":"save","test_id":"settings-save","label":"Save Settings","selector":"#save","text_preview":"Save Settings"},
		"console":[{"level":"warn","message":"Save button emitted a warning"}],
		"network":[{"kind":"fetch","method":"POST","url":"https://example.com/api/save?token=secret","status":500,"ok":false,"duration_ms":812}]
	}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "nonce-pointer-grounding")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusOK {
		t.Fatalf("pointer failed: %d %s", pointerRec.Code, pointerRec.Body.String())
	}

	stateReq := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(stateReq, cfg.ControlToken, false, "")
	stateRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("state failed: %d %s", stateRec.Code, stateRec.Body.String())
	}
	var state map[string]any
	if err := json.Unmarshal(stateRec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state response: %v", err)
	}
	latest, _ := state["pointer_latest"].(map[string]any)
	if got := latest["target_selector"]; got != "#save" {
		t.Fatalf("expected target_selector=#save, got %#v", got)
	}
	if got := latest["target_test_id"]; got != "settings-save" {
		t.Fatalf("expected target_test_id=settings-save, got %#v", got)
	}
	dom, _ := latest["dom"].(map[string]any)
	if got := dom["selector"]; got != "#save" {
		t.Fatalf("expected dom selector=#save, got %#v", got)
	}
	consoleEntries, _ := latest["console"].([]any)
	if len(consoleEntries) != 1 {
		t.Fatalf("expected one console entry, got %#v", latest["console"])
	}
	networkEntries, _ := latest["network"].([]any)
	if len(networkEntries) != 1 {
		t.Fatalf("expected one network entry, got %#v", latest["network"])
	}
	firstNetwork, _ := networkEntries[0].(map[string]any)
	if got := firstNetwork["url"]; got != "https://example.com/api/save" {
		t.Fatalf("expected sanitized network url without query string, got %#v", got)
	}
}

func TestSessionStartDefaultsTargetWindowWhenBlank(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"","target_url":""}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-start-default-window")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var sess map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &sess); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if got := sess["target_window"]; got != "Browser Review" {
		t.Fatalf("expected default target window Browser Review, got %#v", got)
	}
}

func TestPurgeSessionResetsCaptureState(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-purge-reset")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	purgeReq := httptest.NewRequest(http.MethodPost, "/api/purge/session", bytes.NewReader([]byte(`{}`)))
	purgeReq.Header.Set("Content-Type", "application/json")
	addAuth(purgeReq, cfg.ControlToken, true, "nonce-purge-reset")
	purgeRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(purgeRec, purgeReq)
	if purgeRec.Code != http.StatusOK {
		t.Fatalf("purge failed: %d %s", purgeRec.Code, purgeRec.Body.String())
	}

	stateReq := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(stateReq, cfg.ControlToken, false, "")
	stateRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("state failed: %d %s", stateRec.Code, stateRec.Body.String())
	}

	var state map[string]any
	if err := json.Unmarshal(stateRec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state response: %v", err)
	}
	if got := state["capture_state"]; got != "inactive" {
		t.Fatalf("expected capture_state inactive after purge, got %#v", got)
	}
	if got := state["session"]; got != nil {
		t.Fatalf("expected no active session after purge, got %#v", got)
	}
	sources, _ := state["capture_sources"].(map[string]any)
	for _, source := range []string{"screen", "microphone", "companion"} {
		entry, _ := sources[source].(map[string]any)
		if got := entry["status"]; got != "unavailable" {
			t.Fatalf("expected %s source unavailable after purge, got %#v", source, got)
		}
	}
}

func TestCompanionPointerUpdatesSessionTargetMetadata(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"","target_url":""}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-companion-target")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var sess map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &sess); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := sess["id"].(string)
	if sessionID == "" {
		t.Fatalf("missing session id")
	}

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{
		"session_id":"`+sessionID+`",
		"x":612,
		"y":384,
		"event_type":"move",
		"window":"Ruddur - Home",
		"url":"https://example.com/app",
		"route":"/app"
	}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "nonce-pointer-target-update")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusOK {
		t.Fatalf("pointer failed: %d %s", pointerRec.Code, pointerRec.Body.String())
	}

	stateReq := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(stateReq, cfg.ControlToken, false, "")
	stateRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("state failed: %d %s", stateRec.Code, stateRec.Body.String())
	}
	var state map[string]any
	if err := json.Unmarshal(stateRec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state response: %v", err)
	}
	curr, _ := state["session"].(map[string]any)
	if got := curr["target_window"]; got != "Ruddur - Home" {
		t.Fatalf("expected session target window to update from companion, got %#v", got)
	}
	if got := curr["target_url"]; got != "https://example.com/app" {
		t.Fatalf("expected session target url to update from companion, got %#v", got)
	}
}

func TestCompanionPointerRejectsEventsOutsideApprovedTargetScope(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-pointer-scope")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var sess map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &sess); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := sess["id"].(string)
	if sessionID == "" {
		t.Fatalf("missing session id")
	}

	validPointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{
		"session_id":"`+sessionID+`",
		"x":111,
		"y":222,
		"event_type":"move",
		"window":"Browser Preview",
		"url":"https://example.com/app",
		"route":"/app"
	}`)))
	validPointerReq.Header.Set("Content-Type", "application/json")
	addAuth(validPointerReq, cfg.ControlToken, true, "nonce-pointer-valid")
	validPointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(validPointerRec, validPointerReq)
	if validPointerRec.Code != http.StatusOK {
		t.Fatalf("valid pointer failed: %d %s", validPointerRec.Code, validPointerRec.Body.String())
	}

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{
		"session_id":"`+sessionID+`",
		"x":640,
		"y":360,
		"event_type":"move",
		"window":"Browser Preview",
		"url":"https://other.example.net/app",
		"route":"/app"
	}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "nonce-pointer-invalid-scope")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusForbidden {
		t.Fatalf("expected pointer scope rejection, got %d: %s", pointerRec.Code, pointerRec.Body.String())
	}

	stateReq := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(stateReq, cfg.ControlToken, false, "")
	stateRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("state failed: %d %s", stateRec.Code, stateRec.Body.String())
	}
	var state map[string]any
	if err := json.Unmarshal(stateRec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state response: %v", err)
	}
	latest, _ := state["pointer_latest"].(map[string]any)
	if latest["url"] != "https://example.com/app" {
		t.Fatalf("expected rejected pointer event to leave latest scoped sample intact, got %#v", latest["url"])
	}
}

func TestCompanionPointerAcceptsLoopbackAliasesForSameTargetScope(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"http://localhost:3000/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-pointer-loopback")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var sess map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &sess); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := sess["id"].(string)
	if sessionID == "" {
		t.Fatalf("missing session id")
	}

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{
		"session_id":"`+sessionID+`",
		"x":320,
		"y":180,
		"event_type":"extension_context",
		"window":"Browser Preview",
		"url":"http://127.0.0.1:3000/app",
		"route":"/app"
	}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "nonce-pointer-loopback")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusOK {
		t.Fatalf("expected loopback alias pointer to be accepted, got %d: %s", pointerRec.Code, pointerRec.Body.String())
	}
}

func TestFeedbackNoteSuppressesOutOfScopeVisualArtifacts(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-scope")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var sess map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &sess); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := sess["id"].(string)
	if sessionID == "" {
		t.Fatalf("missing session id")
	}

	// Inject an out-of-scope pointer sample to verify screenshot/clip suppression on note capture.
	srv.privilegedCapture.AddPointer(companion.PointerEvent{
		SessionID: sessionID,
		X:         8,
		Y:         12,
		Window:    "Browser Preview",
		URL:       "https://evil.example.net/app",
		Route:     "/app",
		EventType: "move",
		Timestamp: time.Now().UTC(),
	})

	noteBody, noteCT := multipartNoteBody(t, "This should still record note text", tinyPNG(t))
	noteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", noteBody)
	noteReq.Header.Set("Content-Type", noteCT)
	addAuth(noteReq, cfg.ControlToken, true, "nonce-note-scope")
	noteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusOK {
		t.Fatalf("note failed: %d %s", noteRec.Code, noteRec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(noteRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode note response: %v", err)
	}
	if got := payload["screenshot_ref"]; got != "" {
		t.Fatalf("expected screenshot to be suppressed for out-of-scope pointer, got %#v", got)
	}
}

func TestFeedbackClipRejectedOutsideTargetScope(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-clip-scope")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var sess map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &sess); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := sess["id"].(string)
	if sessionID == "" {
		t.Fatalf("missing session id")
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"Track this","normalized":"Track this","pointer_x":10,"pointer_y":20,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "nonce-feedback-clip-scope")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}
	var curr session.Session
	if err := json.Unmarshal(feedbackRec.Body.Bytes(), &curr); err != nil {
		t.Fatalf("decode feedback response: %v", err)
	}
	if len(curr.Feedback) == 0 {
		t.Fatalf("expected feedback event")
	}
	eventID := curr.Feedback[len(curr.Feedback)-1].ID
	if eventID == "" {
		t.Fatalf("expected event id")
	}

	// Inject an out-of-scope pointer sample to force clip rejection.
	srv.privilegedCapture.AddPointer(companion.PointerEvent{
		SessionID: sessionID,
		X:         15,
		Y:         25,
		Window:    "Browser Preview",
		URL:       "https://other.example.net/path",
		Route:     "/path",
		EventType: "move",
		Timestamp: time.Now().UTC(),
	})

	clipBody, clipCT := multipartClipBody(t, eventID, []byte("fake-webm-clip"))
	clipReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/clip", clipBody)
	clipReq.Header.Set("Content-Type", clipCT)
	addAuth(clipReq, cfg.ControlToken, true, "nonce-clip-scope")
	clipRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(clipRec, clipReq)
	if clipRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for out-of-scope clip, got %d: %s", clipRec.Code, clipRec.Body.String())
	}
}

func TestFeedbackClipPersistsVideoMetadataAndLatency(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-clip-meta")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"Attach clip metadata","normalized":"Attach clip metadata","pointer_x":10,"pointer_y":20,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "nonce-feedback-clip-meta")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}
	var curr session.Session
	if err := json.Unmarshal(feedbackRec.Body.Bytes(), &curr); err != nil {
		t.Fatalf("decode feedback response: %v", err)
	}
	eventID := curr.Feedback[len(curr.Feedback)-1].ID
	startedAt := time.Now().UTC().Add(-5 * time.Second).Format(time.RFC3339Nano)
	endedAt := time.Now().UTC().Format(time.RFC3339Nano)
	clipFields := map[string]string{
		"video_scope":           "selected-region",
		"video_region_x":        "12",
		"video_region_y":        "24",
		"video_region_w":        "320",
		"video_region_h":        "180",
		"video_codec":           "video/webm;codecs=vp9",
		"video_has_audio":       "1",
		"video_pointer_overlay": "1",
		"video_window":          "Browser Preview",
		"clip_started_at":       startedAt,
		"clip_ended_at":         endedAt,
		"clip_duration_ms":      "5000",
	}
	clipBody, clipCT := multipartClipBodyWithFields(t, eventID, []byte("fake-webm-clip"), clipFields)
	clipReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/clip", clipBody)
	clipReq.Header.Set("Content-Type", clipCT)
	addAuth(clipReq, cfg.ControlToken, true, "nonce-clip-meta")
	clipRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(clipRec, clipReq)
	if clipRec.Code != http.StatusOK {
		t.Fatalf("clip attach failed: %d %s", clipRec.Code, clipRec.Body.String())
	}

	stateReq := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(stateReq, cfg.ControlToken, false, "")
	stateRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("state failed: %d %s", stateRec.Code, stateRec.Body.String())
	}
	var state map[string]any
	if err := json.Unmarshal(stateRec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	sess, _ := state["session"].(map[string]any)
	events, _ := sess["feedback"].([]any)
	if len(events) == 0 {
		t.Fatalf("expected feedback events in session")
	}
	last, _ := events[len(events)-1].(map[string]any)
	video, _ := last["video"].(map[string]any)
	if video["scope"] != "selected-region" {
		t.Fatalf("expected video scope selected-region, got %#v", video["scope"])
	}
	if video["codec"] != "video/webm;codecs=vp9" {
		t.Fatalf("expected video codec metadata, got %#v", video["codec"])
	}
	if got, _ := video["has_audio"].(bool); !got {
		t.Fatalf("expected has_audio=true in video metadata")
	}
	if got, _ := video["pointer_overlay"].(bool); !got {
		t.Fatalf("expected pointer_overlay=true in video metadata")
	}
	latency, _ := state["latency_metrics"].(map[string]any)
	clipLatency, _ := latency["feedback_clip_attach_ms"].(map[string]any)
	if count, _ := clipLatency["count"].(float64); count < 1 {
		t.Fatalf("expected feedback_clip_attach_ms latency count >= 1, got %#v", clipLatency["count"])
	}
}

func TestPayloadPreviewIncludesInlineArtifactDataForTransmission(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-inline-artifacts")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var sess map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &sess); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := sess["id"].(string)
	if sessionID == "" {
		t.Fatalf("missing session id")
	}

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{
		"session_id":"`+sessionID+`",
		"x":220,
		"y":140,
		"event_type":"move",
		"window":"Browser Preview",
		"url":"https://example.com/app",
		"route":"/app"
	}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "nonce-inline-pointer")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusOK {
		t.Fatalf("pointer failed: %d %s", pointerRec.Code, pointerRec.Body.String())
	}

	noteBody, noteCT := multipartNoteBody(t, "Attach inline clip for agent transmission", tinyPNG(t))
	noteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", noteBody)
	noteReq.Header.Set("Content-Type", noteCT)
	addAuth(noteReq, cfg.ControlToken, true, "nonce-inline-note")
	noteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusOK {
		t.Fatalf("note failed: %d %s", noteRec.Code, noteRec.Body.String())
	}
	var noteResp map[string]any
	if err := json.Unmarshal(noteRec.Body.Bytes(), &noteResp); err != nil {
		t.Fatalf("decode note response: %v", err)
	}
	eventID, _ := noteResp["event_id"].(string)
	if eventID == "" {
		t.Fatalf("expected event id")
	}

	clipFields := map[string]string{
		"video_scope":           "full-window",
		"video_codec":           "video/webm;codecs=vp9",
		"video_has_audio":       "1",
		"video_pointer_overlay": "1",
		"clip_duration_ms":      "5000",
	}
	clipBody, clipCT := multipartClipBodyWithFields(t, eventID, []byte("fake-webm-clip"), clipFields)
	clipReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/clip", clipBody)
	clipReq.Header.Set("Content-Type", clipCT)
	addAuth(clipReq, cfg.ControlToken, true, "nonce-inline-clip")
	clipRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(clipRec, clipReq)
	if clipRec.Code != http.StatusOK {
		t.Fatalf("clip attach failed: %d %s", clipRec.Code, clipRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":"Inline clip summary"}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-inline-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-inline-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}

	var preview payloadPreviewResponse
	if err := json.Unmarshal(previewRec.Body.Bytes(), &preview); err != nil {
		t.Fatalf("decode typed preview response: %v", err)
	}
	if preview.Preview.Disclosure.Destination != "codex_cli" || preview.Preview.Disclosure.RequestTextCount != 1 {
		t.Fatalf("expected disclosure summary for preview payload, got %#v", preview.Preview.Disclosure)
	}
	if preview.Preview.Disclosure.TypedValuesStatus != "included" {
		t.Fatalf("expected typed values disclosure to show included, got %#v", preview.Preview.Disclosure)
	}
	if preview.Preview.Disclosure.VideosSent != 1 || preview.Preview.Disclosure.ScreenshotsSent != 1 {
		t.Fatalf("expected media disclosure counts, got %#v", preview.Preview.Disclosure)
	}

	var payload map[string]any
	if err := json.Unmarshal(previewRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode preview response: %v", err)
	}
	providerPayload, _ := payload["payload"].(map[string]any)
	pkg, _ := providerPayload["package"].(map[string]any)
	artifacts, _ := pkg["artifacts"].([]any)
	if len(artifacts) < 2 {
		t.Fatalf("expected screenshot and video artifacts in provider payload, got %#v", artifacts)
	}

	var sawInlineVideo bool
	for _, raw := range artifacts {
		artifact, _ := raw.(map[string]any)
		if artifact["kind"] != "video" {
			continue
		}
		if artifact["event_id"] != eventID {
			t.Fatalf("expected video artifact to stay linked to event %q, got %#v", eventID, artifact["event_id"])
		}
		if mime := artifact["mime_type"]; mime != "video/webm" {
			t.Fatalf("expected video mime type, got %#v", mime)
		}
		inlineData, _ := artifact["inline_data_url"].(string)
		if !strings.HasPrefix(inlineData, "data:video/webm;base64,") {
			t.Fatalf("expected inline video data URL, got %q", inlineData)
		}
		sawInlineVideo = true
	}
	if !sawInlineVideo {
		t.Fatalf("expected inline video artifact in provider payload, got %#v", artifacts)
	}
}

func TestLargeInlineMediaRequiresExplicitUserDecision(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-large-inline")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var sess map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &sess); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := sess["id"].(string)
	if sessionID == "" {
		t.Fatalf("missing session id")
	}

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{
		"session_id":"`+sessionID+`",
		"x":220,
		"y":140,
		"event_type":"move",
		"window":"Browser Preview",
		"url":"https://example.com/app",
		"route":"/app"
	}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "nonce-large-inline-pointer")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusOK {
		t.Fatalf("pointer failed: %d %s", pointerRec.Code, pointerRec.Body.String())
	}

	noteBody, noteCT := multipartNoteBody(t, "Oversized clip should require confirmation", tinyPNG(t))
	noteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", noteBody)
	noteReq.Header.Set("Content-Type", noteCT)
	addAuth(noteReq, cfg.ControlToken, true, "nonce-large-inline-note")
	noteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusOK {
		t.Fatalf("note failed: %d %s", noteRec.Code, noteRec.Body.String())
	}
	var noteResp map[string]any
	if err := json.Unmarshal(noteRec.Body.Bytes(), &noteResp); err != nil {
		t.Fatalf("decode note response: %v", err)
	}
	eventID, _ := noteResp["event_id"].(string)
	if eventID == "" {
		t.Fatalf("expected event id")
	}

	largeClip := bytes.Repeat([]byte("v"), int(maxInlineVideoBytes)+1)
	clipFields := map[string]string{
		"video_scope":      "full-window",
		"video_codec":      "video/webm;codecs=vp9",
		"clip_duration_ms": "5000",
	}
	clipBody, clipCT := multipartClipBodyWithFields(t, eventID, largeClip, clipFields)
	clipReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/clip", clipBody)
	clipReq.Header.Set("Content-Type", clipCT)
	addAuth(clipReq, cfg.ControlToken, true, "nonce-large-inline-clip")
	clipRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(clipRec, clipReq)
	if clipRec.Code != http.StatusOK {
		t.Fatalf("clip attach failed: %d %s", clipRec.Code, clipRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":"Large clip summary"}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-large-inline-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-large-inline-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}
	var preview payloadPreviewResponse
	if err := json.Unmarshal(previewRec.Body.Bytes(), &preview); err != nil {
		t.Fatalf("decode preview response: %v", err)
	}
	if len(preview.Preview.Warnings) == 0 || !strings.Contains(strings.Join(preview.Preview.Warnings, "\n"), "over the default send limit") {
		t.Fatalf("expected oversized clip warning, got %#v", preview.Preview.Warnings)
	}
	if preview.Preview.Disclosure.VideosOmitted != 1 || preview.Preview.Disclosure.VideosSent != 0 {
		t.Fatalf("expected disclosure to show omitted oversized clip, got %#v", preview.Preview.Disclosure)
	}
	if len(preview.Preview.Notes) != 1 {
		t.Fatalf("expected one preview note, got %#v", preview.Preview.Notes)
	}
	note := preview.Preview.Notes[0]
	if note.EventID != eventID {
		t.Fatalf("expected preview note to stay linked to %q, got %#v", eventID, note.EventID)
	}
	if note.VideoTransmissionState != "omitted_due_to_limit" {
		t.Fatalf("expected oversized clip transmission state, got %#v", note.VideoTransmissionState)
	}
	if note.VideoSizeBytes != int64(len(largeClip)) {
		t.Fatalf("expected preview note to expose clip size %d, got %d", len(largeClip), note.VideoSizeBytes)
	}
	if note.VideoSendLimitBytes != maxInlineVideoBytes {
		t.Fatalf("expected preview note to expose send limit %d, got %d", maxInlineVideoBytes, note.VideoSendLimitBytes)
	}
	if !note.HasVideo {
		t.Fatalf("expected preview note to report an attached clip")
	}

	previewReq = httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"cli","omit_video_event_ids":["`+eventID+`"]}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-large-inline-preview-snapshot")
	previewRec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("snapshot preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}
	if err := json.Unmarshal(previewRec.Body.Bytes(), &preview); err != nil {
		t.Fatalf("decode snapshot preview response: %v", err)
	}
	if len(preview.Preview.Notes) != 1 || preview.Preview.Notes[0].VideoTransmissionState != "omitted_by_user" {
		t.Fatalf("expected snapshot path to omit the selected clip, got %#v", preview.Preview.Notes)
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "nonce-large-inline-submit")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusConflict {
		t.Fatalf("expected submit to require explicit decision for oversized clip, got %d: %s", submitRec.Code, submitRec.Body.String())
	}

	approveReq = httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":"Large clip summary"}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-large-inline-approve-snapshot")
	approveRec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("re-approve for snapshot submit failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	submitReq = httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"cli","omit_video_event_ids":["`+eventID+`"]}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "nonce-large-inline-submit-snapshot")
	submitRec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("expected snapshot submit to succeed, got %d: %s", submitRec.Code, submitRec.Body.String())
	}

}

func TestFeedbackClipFetchReturnsStoredClipForCurrentSession(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-clip-fetch-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	noteBody, noteCT := multipartNoteBody(t, "Clip fetch", tinyPNG(t))
	noteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", noteBody)
	noteReq.Header.Set("Content-Type", noteCT)
	addAuth(noteReq, cfg.ControlToken, true, "nonce-clip-fetch-note")
	noteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusOK {
		t.Fatalf("note failed: %d %s", noteRec.Code, noteRec.Body.String())
	}
	var noteResp map[string]any
	if err := json.Unmarshal(noteRec.Body.Bytes(), &noteResp); err != nil {
		t.Fatalf("decode note response: %v", err)
	}
	eventID, _ := noteResp["event_id"].(string)
	if eventID == "" {
		t.Fatalf("expected event id")
	}

	clipFields := map[string]string{
		"video_scope":      "full-window",
		"video_codec":      "video/webm;codecs=vp9",
		"clip_duration_ms": "5000",
	}
	clipBytes := []byte("stored-webm-clip")
	clipBody, clipCT := multipartClipBodyWithFields(t, eventID, clipBytes, clipFields)
	clipReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/clip", clipBody)
	clipReq.Header.Set("Content-Type", clipCT)
	addAuth(clipReq, cfg.ControlToken, true, "nonce-clip-fetch-attach")
	clipRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(clipRec, clipReq)
	if clipRec.Code != http.StatusOK {
		t.Fatalf("clip attach failed: %d %s", clipRec.Code, clipRec.Body.String())
	}

	fetchReq := httptest.NewRequest(http.MethodGet, "/api/session/feedback/clip?event_id="+url.QueryEscape(eventID), nil)
	addAuth(fetchReq, cfg.ControlToken, false, "")
	fetchRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(fetchRec, fetchReq)
	if fetchRec.Code != http.StatusOK {
		t.Fatalf("clip fetch failed: %d %s", fetchRec.Code, fetchRec.Body.String())
	}
	if got := fetchRec.Header().Get("Content-Type"); got != "video/webm" {
		t.Fatalf("expected clip content type video/webm, got %q", got)
	}
	if got := fetchRec.Header().Get("X-Knit-Video-Codec"); got != "video/webm;codecs=vp9" {
		t.Fatalf("expected clip codec header, got %q", got)
	}
	if !bytes.Equal(fetchRec.Body.Bytes(), clipBytes) {
		t.Fatalf("expected stored clip payload, got %q", fetchRec.Body.Bytes())
	}
}

func TestPayloadPreviewSupportsPreviewOnlyReplayRedactionAndVideoOmission(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-preview-transform-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var started map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := started["id"].(string)

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{
		"session_id":"`+sessionID+`",
		"x":220,
		"y":140,
		"event_type":"input",
		"window":"Browser Preview",
		"url":"https://example.com/app",
		"route":"/app",
		"target_tag":"input",
		"target_id":"headline",
		"target_selector":"#headline",
		"value":"Secret-ish headline copy",
		"value_captured":true
	}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "nonce-preview-transform-pointer")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusOK {
		t.Fatalf("pointer failed: %d %s", pointerRec.Code, pointerRec.Body.String())
	}

	noteBody, noteCT := multipartNoteBody(t, "Transform preview delivery", tinyPNG(t))
	noteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", noteBody)
	noteReq.Header.Set("Content-Type", noteCT)
	addAuth(noteReq, cfg.ControlToken, true, "nonce-preview-transform-note")
	noteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusOK {
		t.Fatalf("note failed: %d %s", noteRec.Code, noteRec.Body.String())
	}
	var noteResp map[string]any
	if err := json.Unmarshal(noteRec.Body.Bytes(), &noteResp); err != nil {
		t.Fatalf("decode note response: %v", err)
	}
	eventID, _ := noteResp["event_id"].(string)
	if eventID == "" {
		t.Fatalf("expected event id")
	}

	clipFields := map[string]string{
		"video_scope":      "full-window",
		"video_codec":      "video/webm;codecs=vp9",
		"clip_duration_ms": "5000",
	}
	clipBody, clipCT := multipartClipBodyWithFields(t, eventID, []byte("fake-webm-clip"), clipFields)
	clipReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/clip", clipBody)
	clipReq.Header.Set("Content-Type", clipCT)
	addAuth(clipReq, cfg.ControlToken, true, "nonce-preview-transform-clip")
	clipRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(clipRec, clipReq)
	if clipRec.Code != http.StatusOK {
		t.Fatalf("clip attach failed: %d %s", clipRec.Code, clipRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":"Transform preview delivery"}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-preview-transform-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"cli","redact_replay_values":true,"omit_video_clips":true}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-preview-transform-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}
	var preview payloadPreviewResponse
	if err := json.Unmarshal(previewRec.Body.Bytes(), &preview); err != nil {
		t.Fatalf("decode preview response: %v", err)
	}
	if preview.Preview.Disclosure.TypedValuesStatus != "redacted" {
		t.Fatalf("expected preview-only replay redaction, got %#v", preview.Preview.Disclosure)
	}
	if preview.Preview.Disclosure.VideosSent != 0 || preview.Preview.Disclosure.VideosOmitted != 1 {
		t.Fatalf("expected preview-only video omission, got %#v", preview.Preview.Disclosure)
	}
	if len(preview.Preview.Notes) != 1 {
		t.Fatalf("expected one preview note, got %d", len(preview.Preview.Notes))
	}
	note := preview.Preview.Notes[0]
	if note.ReplayValueMode != "redacted" {
		t.Fatalf("expected replay value mode redacted, got %#v", note.ReplayValueMode)
	}
	if strings.Contains(strings.Join(note.ReplaySteps, "\n"), "Secret-ish headline copy") {
		t.Fatalf("expected preview-only replay steps to redact typed values, got %#v", note.ReplaySteps)
	}
	if !strings.Contains(note.PlaywrightScript, "typed value was redacted") {
		t.Fatalf("expected redacted playwright script, got %q", note.PlaywrightScript)
	}
	if note.VideoDataURL != "" {
		t.Fatalf("expected preview-only video omission to remove clip from preview, got %q", note.VideoDataURL)
	}

	var payload map[string]any
	if err := json.Unmarshal(previewRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode preview payload map: %v", err)
	}
	providerPayload, _ := payload["payload"].(map[string]any)
	pkg, _ := providerPayload["package"].(map[string]any)
	changeRequests, _ := pkg["change_requests"].([]any)
	if len(changeRequests) != 1 {
		t.Fatalf("expected one change request in provider payload, got %#v", pkg["change_requests"])
	}
	changeReq, _ := changeRequests[0].(map[string]any)
	replay, _ := changeReq["replay"].(map[string]any)
	if replay["value_capture_mode"] != "redacted" {
		t.Fatalf("expected provider payload replay mode redacted, got %#v", replay)
	}
	replaySteps, _ := replay["steps"].([]any)
	if len(replaySteps) == 0 {
		t.Fatalf("expected replay steps in provider payload")
	}
	firstStep, _ := replaySteps[0].(map[string]any)
	if value := firstStep["value"]; value != "" && value != nil {
		t.Fatalf("expected transmitted replay step value to be redacted, got %#v", firstStep)
	}
	artifacts, _ := pkg["artifacts"].([]any)
	for _, raw := range artifacts {
		artifact, _ := raw.(map[string]any)
		if artifact["kind"] != "video" {
			continue
		}
		if artifact["transmission_status"] != "omitted_by_user" {
			t.Fatalf("expected video artifact omitted_by_user, got %#v", artifact)
		}
		if ref := artifact["ref"]; ref != "" && ref != nil {
			t.Fatalf("expected omitted video artifact ref to be cleared, got %#v", artifact)
		}
		if inline := artifact["inline_data_url"]; inline != "" && inline != nil {
			t.Fatalf("expected omitted video artifact to have no inline data, got %#v", artifact)
		}
	}
}

func TestStateLatencyMetricsIncludePointerAndFeedbackNote(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-latency")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var started map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := started["id"].(string)

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{
		"session_id":"`+sessionID+`",
		"x":612,
		"y":384,
		"event_type":"move",
		"window":"Browser Preview",
		"url":"https://example.com/app",
		"route":"/app"
	}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "nonce-pointer-latency")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusOK {
		t.Fatalf("pointer failed: %d %s", pointerRec.Code, pointerRec.Body.String())
	}

	noteBody, noteCT := multipartNoteBody(t, "Measure note latency", tinyPNG(t))
	noteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", noteBody)
	noteReq.Header.Set("Content-Type", noteCT)
	addAuth(noteReq, cfg.ControlToken, true, "nonce-note-latency")
	noteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusOK {
		t.Fatalf("note failed: %d %s", noteRec.Code, noteRec.Body.String())
	}

	stateReq := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(stateReq, cfg.ControlToken, false, "")
	stateRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("state failed: %d %s", stateRec.Code, stateRec.Body.String())
	}
	var state map[string]any
	if err := json.Unmarshal(stateRec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	latency, _ := state["latency_metrics"].(map[string]any)
	pointerLatency, _ := latency["pointer_ingest_ms"].(map[string]any)
	noteLatency, _ := latency["feedback_note_ms"].(map[string]any)
	if count, _ := pointerLatency["count"].(float64); count < 1 {
		t.Fatalf("expected pointer latency samples, got %#v", pointerLatency["count"])
	}
	if count, _ := noteLatency["count"].(float64); count < 1 {
		t.Fatalf("expected note latency samples, got %#v", noteLatency["count"])
	}
}

func TestRuntimeCodexConfigEndpointUpdatesRuntimeState(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	body := []byte(`{
		"default_provider":"claude_api",
		"cli_adapter_cmd":"/tmp/knit-codex-cli-adapter.sh",
		"cli_timeout_seconds":300,
		"claude_cli_adapter_cmd":"/tmp/knit-claude-cli-adapter.sh",
		"claude_cli_timeout_seconds":450,
		"opencode_cli_adapter_cmd":"/tmp/knit-opencode-cli-adapter.sh",
		"opencode_cli_timeout_seconds":700,
		"submit_execution_mode":"parallel",
		"codex_workdir":"/Users/chadsylvester/SW_Dev/Knit",
		"codex_sandbox":"workspace-write",
		"codex_approval_policy":"on-request",
		"codex_reasoning_effort":"high",
		"openai_base_url":"https://api.openai.com",
		"codex_api_timeout_seconds":120,
		"openai_org_id":"org-test",
		"openai_project_id":"proj-test",
		"anthropic_base_url":"https://api.anthropic.com",
		"claude_api_timeout_seconds":95,
		"claude_api_model":"claude-3-7-sonnet-latest",
		"delivery_intent_profile":"create_jira_tickets",
		"implement_changes_prompt":"Implement exactly what the package requests.",
		"create_jira_tickets_prompt":"Create Jira tickets from this package.",
		"post_submit_rebuild_cmd":"echo rebuild",
		"post_submit_verify_cmd":"echo verify",
		"post_submit_timeout_seconds":240,
		"codex_skip_git_repo_check":true
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/codex", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-runtime-codex")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime codex response: %v", err)
	}
	rt, _ := payload["runtime_codex"].(map[string]any)
	if got := rt["codex_workdir"]; got != "/Users/chadsylvester/SW_Dev/Knit" {
		t.Fatalf("expected codex workdir in runtime state, got %#v", got)
	}
	if got := rt["codex_sandbox"]; got != "workspace-write" {
		t.Fatalf("expected codex sandbox in runtime state, got %#v", got)
	}
	if got := rt["default_provider"]; got != "claude_api" {
		t.Fatalf("expected default provider in runtime state, got %#v", got)
	}
	if got := rt["submit_execution_mode"]; got != "parallel" {
		t.Fatalf("expected submit mode in runtime state, got %#v", got)
	}
	if got := rt["anthropic_base_url"]; got != "https://api.anthropic.com" {
		t.Fatalf("expected anthropic base url in runtime state, got %#v", got)
	}
	if got := rt["claude_api_timeout_seconds"]; got != "95" {
		t.Fatalf("expected Claude API timeout in runtime state, got %#v", got)
	}
	if got := rt["claude_api_model"]; got != "claude-3-7-sonnet-latest" {
		t.Fatalf("expected Claude API model in runtime state, got %#v", got)
	}
	if got := rt["delivery_intent_profile"]; got != "create_jira_tickets" {
		t.Fatalf("expected delivery intent profile in runtime state, got %#v", got)
	}
	if got := rt["implement_changes_prompt"]; got != "Implement exactly what the package requests." {
		t.Fatalf("expected implement prompt in runtime state, got %#v", got)
	}
	if got := rt["post_submit_timeout_seconds"]; got != "240" {
		t.Fatalf("expected post-submit timeout in runtime state, got %#v", got)
	}
}

func TestRuntimeCodexConfigEndpointDefaultsSandboxAndApproval(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/codex", bytes.NewReader([]byte(`{"codex_workdir":"/tmp/repo"}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-runtime-defaults")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime codex response: %v", err)
	}
	rt, _ := payload["runtime_codex"].(map[string]any)
	if got := rt["codex_sandbox"]; got != operatorstate.DefaultLocalCodingSandbox {
		t.Fatalf("expected sandbox default %q, got %#v", operatorstate.DefaultLocalCodingSandbox, got)
	}
	if got := rt["codex_approval_policy"]; got != operatorstate.DefaultLocalCodingApproval {
		t.Fatalf("expected approval default %q, got %#v", operatorstate.DefaultLocalCodingApproval, got)
	}
}

func TestRuntimeCodexConfigAppliesWorkspaceToCLIChildEnv(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)
	t.Setenv("KNIT_CODEX_WORKDIR", "/tmp/old-workspace")

	workdirFile := filepath.Join(t.TempDir(), "workdir.txt")
	command := `sh -lc 'printf "%s" "$KNIT_CODEX_WORKDIR" > "` + workdirFile + `"; echo "{\"run_id\":\"cli-workdir\",\"status\":\"accepted\",\"ref\":\"` + workdirFile + `\"}"'`
	if runtime.GOOS == "windows" {
		command = `powershell -Command "$p='` + strings.ReplaceAll(workdirFile, `\`, `\\`) + `'; Set-Content -Path $p -NoNewline -Value $env:KNIT_CODEX_WORKDIR; Write-Output '{\"run_id\":\"cli-workdir\",\"status\":\"accepted\",\"ref\":\"` + strings.ReplaceAll(workdirFile, `\`, `\\`) + `\"}'"`
	}

	runtimePayload, err := json.Marshal(map[string]any{
		"cli_adapter_cmd": command,
		"codex_workdir":   "/tmp/intent-manager",
	})
	if err != nil {
		t.Fatalf("marshal runtime payload: %v", err)
	}
	runtimeReq := httptest.NewRequest(http.MethodPost, "/api/runtime/codex", bytes.NewReader(runtimePayload))
	runtimeReq.Header.Set("Content-Type", "application/json")
	addAuth(runtimeReq, cfg.ControlToken, true, "nonce-runtime-workdir")
	runtimeRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(runtimeRec, runtimeReq)
	if runtimeRec.Code != http.StatusOK {
		t.Fatalf("runtime codex setup failed: %d %s", runtimeRec.Code, runtimeRec.Body.String())
	}

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-runtime-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"Fix this bug","normalized":"Fix this bug","pointer_x":100,"pointer_y":80,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "nonce-runtime-feedback")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-runtime-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "nonce-runtime-submit")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("submit failed: %d %s", submitRec.Code, submitRec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(submitRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	attemptID, _ := payload["attempt_id"].(string)
	if attemptID == "" {
		t.Fatalf("expected attempt_id in submit response")
	}
	_ = waitForAttemptStatus(t, srv, cfg.ControlToken, attemptID, "submitted", 3*time.Second)

	got, err := os.ReadFile(workdirFile)
	if err != nil {
		t.Fatalf("read child workdir file: %v", err)
	}
	if string(got) != "/tmp/intent-manager" {
		t.Fatalf("expected child process workdir env to be updated, got %q", string(got))
	}
	if gotEnv := os.Getenv("KNIT_CODEX_WORKDIR"); gotEnv != "/tmp/intent-manager" {
		t.Fatalf("expected daemon process env updated, got %q", gotEnv)
	}
}

func TestRuntimeCodexConfigBlockedWhenLocked(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.ConfigLocked = true
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/codex", bytes.NewReader([]byte(`{"codex_workdir":"/tmp"}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-runtime-locked")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusLocked {
		t.Fatalf("expected %d, got %d: %s", http.StatusLocked, rec.Code, rec.Body.String())
	}
}

func TestRuntimeCodexConfigCanUnsetCLITimeout(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/codex", bytes.NewReader([]byte(`{"cli_timeout_seconds":0}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-runtime-timeout-clear")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime codex response: %v", err)
	}
	rt, _ := payload["runtime_codex"].(map[string]any)
	if got := rt["cli_timeout_seconds"]; got != "" {
		t.Fatalf("expected cli timeout cleared in runtime state, got %#v", got)
	}
}

func TestRuntimeTranscriptionConfigEndpointSwitchesProvider(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	body := []byte(`{
		"mode":"local",
		"local_command":"printf transcribed text",
		"timeout_seconds":45
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/transcription", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-runtime-stt")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	stateReq := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(stateReq, cfg.ControlToken, false, "")
	stateRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("state expected 200, got %d: %s", stateRec.Code, stateRec.Body.String())
	}
	var state map[string]any
	if err := json.Unmarshal(stateRec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if got := state["transcription_mode"]; got != "local" {
		t.Fatalf("expected state transcription_mode=local, got %#v", got)
	}
	rt, _ := state["runtime_transcription"].(map[string]any)
	if got := rt["local_command"]; got != "printf transcribed text" {
		t.Fatalf("expected local command in runtime_transcription state, got %#v", got)
	}
}

func TestRuntimeTranscriptionRejectsInvalidLocalCommand(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	body := []byte("{\"mode\":\"local\",\"local_command\":\"printf hello\\nrm -rf /\"}")
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/transcription", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-runtime-stt-invalid-command")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid local command, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRuntimeTranscriptionRejectsInvalidLanguage(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	t.Setenv("KNIT_FASTER_WHISPER_LANGUAGE", "")
	srv := newTestServer(t, cfg)

	body := []byte("{\"mode\":\"faster_whisper\",\"language\":\"english!!!\"}")
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/transcription", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-runtime-stt-invalid-language")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid language, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRuntimeTranscriptionNormalizesInvalidFasterWhisperModel(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	body := []byte("{\"mode\":\"faster_whisper\",\"model\":\"gpt-4o-mini-transcribe\",\"device\":\"cpu\",\"compute_type\":\"int8\"}")
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/transcription", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-runtime-stt-invalid-fw-model")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for normalized faster-whisper model, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	rt, _ := payload["runtime_transcription"].(map[string]any)
	if got := rt["mode"]; got != "faster_whisper" {
		t.Fatalf("expected response runtime_transcription.mode=%q, got %#v", "faster_whisper", got)
	}
	if got := rt["model"]; got != transcription.DefaultFasterWhisperModel() {
		t.Fatalf("expected response runtime_transcription.model=%q, got %#v", transcription.DefaultFasterWhisperModel(), got)
	}
}

func TestRuntimeTranscriptionRejectsOversizedTimeout(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	t.Setenv("KNIT_LOCAL_STT_TIMEOUT_SECONDS", "")
	srv := newTestServer(t, cfg)

	body := []byte("{\"mode\":\"local\",\"local_command\":\"printf hello\",\"timeout_seconds\":601}")
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/transcription", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-runtime-stt-invalid-timeout")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized timeout, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRuntimeTranscriptionHealthEndpoint(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	applyReq := httptest.NewRequest(http.MethodPost, "/api/runtime/transcription", bytes.NewReader([]byte(`{
		"mode":"local",
		"local_command":"printf hello"
	}`)))
	applyReq.Header.Set("Content-Type", "application/json")
	addAuth(applyReq, cfg.ControlToken, true, "nonce-runtime-stt-apply")
	applyRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(applyRec, applyReq)
	if applyRec.Code != http.StatusOK {
		t.Fatalf("apply stt expected 200, got %d: %s", applyRec.Code, applyRec.Body.String())
	}

	healthReq := httptest.NewRequest(http.MethodGet, "/api/runtime/transcription/health", nil)
	addAuth(healthReq, cfg.ControlToken, false, "")
	healthRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("health expected 200, got %d: %s", healthRec.Code, healthRec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(healthRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if got := payload["status"]; got != "ok" {
		t.Fatalf("expected health status ok, got %#v", got)
	}
}

func TestRuntimeTranscriptionConfigBlockedWhenLocked(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.ConfigLocked = true
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/transcription", bytes.NewReader([]byte(`{"mode":"local"}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-runtime-stt-locked")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusLocked {
		t.Fatalf("expected %d, got %d: %s", http.StatusLocked, rec.Code, rec.Body.String())
	}
}

func TestRuntimeCodexOptionsEndpointUsesFetcher(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	prev := codexOptionsFetcher
	codexOptionsFetcher = func(_ context.Context) (codexRuntimeOptions, error) {
		return codexRuntimeOptions{
			Source:           "codex_cli",
			Models:           []codexModelOption{{Model: "gpt-5.3-codex", DisplayName: "gpt-5.3-codex", IsDefault: true}},
			ReasoningEfforts: []string{"low", "medium", "high"},
			DefaultModel:     "gpt-5.3-codex",
			DefaultReasoning: "medium",
		}, nil
	}
	t.Cleanup(func() { codexOptionsFetcher = prev })

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/codex/options", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode options response: %v", err)
	}
	if payload["source"] != "codex_cli" {
		t.Fatalf("expected codex_cli source, got: %#v", payload["source"])
	}
	models, _ := payload["models"].([]any)
	if len(models) == 0 {
		t.Fatalf("expected models in response")
	}
}

func TestFSListEndpointReturnsDirectories(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "repo-a"), 0o755); err != nil {
		t.Fatalf("mkdir repo-a: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatalf("mkdir hidden: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/fs/list?path="+url.QueryEscape(root), nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		CurrentPath string `json:"current_path"`
		Dirs        []struct {
			Name string `json:"name"`
		} `json:"dirs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode fs list response: %v", err)
	}
	if payload.CurrentPath == "" {
		t.Fatalf("expected current path")
	}
	found := false
	hiddenFound := false
	for _, d := range payload.Dirs {
		if d.Name == "repo-a" {
			found = true
		}
		if d.Name == ".hidden" {
			hiddenFound = true
		}
	}
	if !found {
		t.Fatalf("expected repo-a directory in listing")
	}
	if hiddenFound {
		t.Fatalf("did not expect hidden directories in listing")
	}
}

func TestFSPickDirEndpointUsesOverrideForHeadlessTests(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	dir := t.TempDir()
	t.Setenv("KNIT_PICKDIR_OVERRIDE", dir)

	req := httptest.NewRequest(http.MethodPost, "/api/fs/pickdir", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-pick-dir")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode pickdir response: %v", err)
	}
	if payload["path"] == "" {
		t.Fatalf("expected path in pickdir response")
	}
}

func TestOpenLastLogUsesCurrentSessionVersionReference(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	logPath := filepath.Join(t.TempDir(), "knit-codex-test.log")
	if err := os.WriteFile(logPath, []byte("ok"), 0o600); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{
		RawTranscript:   "fix color",
		NormalizedText:  "fix color",
		Pointer:         session.PointerCtx{X: 10, Y: 20, Window: "Browser Preview"},
		ScreenshotRef:   "shot-1",
		VisualTargetRef: "button#save",
	}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if err := srv.sessions.MarkSubmitted(logPath); err != nil {
		t.Fatalf("mark submitted: %v", err)
	}

	var opened string
	prevOpen := openLocalPath
	openLocalPath = func(path string) error {
		opened = path
		return nil
	}
	t.Cleanup(func() { openLocalPath = prevOpen })

	req := httptest.NewRequest(http.MethodPost, "/api/session/open-last-log", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-open-log")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if opened != logPath {
		t.Fatalf("expected opened path %q, got %q", logPath, opened)
	}
}

func TestOpenLastLogPrefersLatestAttemptExecutionRef(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	srv.sessions.Start("Browser Preview", "https://example.com")

	f, err := os.CreateTemp(t.TempDir(), "knit-codex-attempt-open-*.log")
	if err != nil {
		t.Fatalf("create temp attempt log: %v", err)
	}
	logPath := f.Name()
	_ = f.Close()
	if err := os.WriteFile(logPath, []byte("attempt log"), 0o600); err != nil {
		t.Fatalf("write attempt log: %v", err)
	}

	srv.submitMu.Lock()
	srv.submitAttempts = append([]submitAttempt{{
		AttemptID:    "attempt-open-last-log",
		Status:       "submitted",
		ExecutionRef: logPath,
	}}, srv.submitAttempts...)
	srv.submitMu.Unlock()

	var opened string
	prevOpen := openLocalPath
	openLocalPath = func(path string) error {
		opened = path
		return nil
	}
	t.Cleanup(func() { openLocalPath = prevOpen })

	req := httptest.NewRequest(http.MethodPost, "/api/session/open-last-log", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-open-last-attempt-log")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if opened != logPath {
		t.Fatalf("expected latest attempt log path %q, got %q", logPath, opened)
	}
}

func TestOpenLastLogRejectsNonLocalReference(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{
		RawTranscript:   "fix color",
		NormalizedText:  "fix color",
		Pointer:         session.PointerCtx{X: 10, Y: 20, Window: "Browser Preview"},
		ScreenshotRef:   "shot-1",
		VisualTargetRef: "button#save",
	}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if err := srv.sessions.MarkSubmitted("response:abc123"); err != nil {
		t.Fatalf("mark submitted: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/session/open-last-log", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-open-log-bad")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAttemptLogEndpointReturnsChunk(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	f, err := os.CreateTemp(t.TempDir(), "knit-codex-attempt-*.log")
	if err != nil {
		t.Fatalf("create temp log file: %v", err)
	}
	logPath := f.Name()
	_ = f.Close()
	if err := os.WriteFile(logPath, []byte("line-1\nline-2\n"), 0o600); err != nil {
		t.Fatalf("write temp log file: %v", err)
	}

	srv.submitMu.Lock()
	srv.submitAttempts = append(srv.submitAttempts, submitAttempt{
		AttemptID:    "attempt-log-1",
		Status:       "in_progress",
		ExecutionRef: logPath,
	})
	srv.submitMu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/session/attempt/log?attempt_id=attempt-log-1&offset=0&limit=7", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	chunk, _ := payload["chunk"].(string)
	if chunk == "" {
		t.Fatalf("expected non-empty chunk")
	}
	if chunk != "line-1\n" {
		t.Fatalf("expected first chunk line-1, got %q", chunk)
	}
}

func TestAttemptLogEndpointReturnsTailChunk(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	f, err := os.CreateTemp(t.TempDir(), "knit-codex-tail-*.log")
	if err != nil {
		t.Fatalf("create temp log file: %v", err)
	}
	logPath := f.Name()
	_ = f.Close()
	if err := os.WriteFile(logPath, []byte("head-line\nmiddle-line\nfinal-line\n"), 0o600); err != nil {
		t.Fatalf("write temp log file: %v", err)
	}

	srv.submitMu.Lock()
	srv.submitAttempts = append(srv.submitAttempts, submitAttempt{
		AttemptID:    "attempt-log-tail",
		Status:       "failed",
		ExecutionRef: logPath,
	})
	srv.submitMu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/session/attempt/log?attempt_id=attempt-log-tail&limit=12&tail=1", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	chunk, _ := payload["chunk"].(string)
	if !strings.Contains(chunk, "final-line") {
		t.Fatalf("expected tail chunk to include final line, got %q", chunk)
	}
	truncatedHead, _ := payload["truncated_head"].(bool)
	if !truncatedHead {
		t.Fatalf("expected truncated_head=true for tailed response")
	}
}

func TestAttemptLogEndpointAcceptsMktempStyleLogSuffix(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	logPath := filepath.Join(t.TempDir(), "knit-codex-attempt-123.log3169046175")
	if err := os.WriteFile(logPath, []byte("codex output\n"), 0o600); err != nil {
		t.Fatalf("write temp log file: %v", err)
	}

	srv.submitMu.Lock()
	srv.submitAttempts = append(srv.submitAttempts, submitAttempt{
		AttemptID:    "attempt-log-mktemp",
		Status:       "submitted",
		ExecutionRef: logPath,
	})
	srv.submitMu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/session/attempt/log?attempt_id=attempt-log-mktemp&offset=0&limit=24000&tail=1", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := payload["chunk"]; got != "codex output\n" {
		t.Fatalf("expected mktemp-style log chunk, got %#v", got)
	}
}

func TestAttemptLogEndpointRejectsUnknownAttempt(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/session/attempt/log?attempt_id=missing", nil)
	addAuth(req, cfg.ControlToken, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCancelSubmitAttemptEndpointCancelsQueuedAttempt(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionSeries)
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)
	srv.agents = agents.NewRegistry(&fakeQueueAdapter{name: "fake_queue", delay: 300 * time.Millisecond})

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "first", NormalizedText: "first"}); err != nil {
		t.Fatalf("add first feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve first: %v", err)
	}
	pkg1, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve first package: %v", err)
	}
	first := srv.enqueueSubmitJob("fake_queue", *pkg1, map[string]any{"id": "one"}, agents.DeliveryIntent{}, "test", "test")

	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "second", NormalizedText: "second"}); err != nil {
		t.Fatalf("add second feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve second: %v", err)
	}
	pkg2, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve second package: %v", err)
	}
	second := srv.enqueueSubmitJob("fake_queue", *pkg2, map[string]any{"id": "two"}, agents.DeliveryIntent{}, "test", "test")

	_ = waitForAttemptStatus(t, srv, cfg.ControlToken, first.AttemptID, "in_progress", 3*time.Second)

	req := httptest.NewRequest(http.MethodPost, "/api/session/attempt/cancel", bytes.NewReader([]byte(`{"attempt_id":"`+second.AttemptID+`"}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-cancel-queued-attempt")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	attempt, _ := payload["attempt"].(map[string]any)
	if got := attempt["status"]; got != submitStatusCanceled {
		t.Fatalf("expected canceled attempt in response, got %#v", got)
	}
}

func TestRerunSubmitAttemptEndpointRequeuesWithCurrentWorkspace(t *testing.T) {
	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"rerun-submit","status":"accepted","ref":"/tmp/rerun-submit.log"}'`)
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{
		ID:             "evt-rerun-1",
		RawTranscript:  "Tighten the checkout spacing",
		NormalizedText: "Tighten the checkout spacing",
	}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	pkg, err := srv.sessions.Approve("Tighten the checkout spacing")
	if err != nil {
		t.Fatalf("approve session: %v", err)
	}
	if err := srv.store.SaveCanonicalPackage(pkg); err != nil {
		t.Fatalf("save canonical package: %v", err)
	}

	intent := agents.NormalizeDeliveryIntent(agents.DeliveryIntent{
		Profile:         agents.IntentCreateJira,
		InstructionText: "Turn this into a Jira-ready change request.",
	})
	redactedPkg := redactPackageForTransmission(*pkg)
	providerPayload, err := agents.PreviewProviderPayloadWithConfig("codex_cli", redactedPkg, "", "", intent)
	if err != nil {
		t.Fatalf("preview provider payload: %v", err)
	}
	original := srv.enqueueSubmitJob("codex_cli", redactedPkg, providerPayload, intent, "test", "test")
	_ = waitForAttemptStatus(t, srv, cfg.ControlToken, original.AttemptID, "submitted", 3*time.Second)

	nextWorkspace := filepath.Join(t.TempDir(), "rerun-workspace")
	srv.runtimeMu.Lock()
	srv.runtime.RuntimeCodex.CodexWorkdir = nextWorkspace
	srv.runtimeMu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/session/attempt/rerun", bytes.NewReader([]byte(`{"attempt_id":"`+original.AttemptID+`"}`)))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, cfg.ControlToken, true, "nonce-rerun-attempt")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode rerun response: %v", err)
	}
	attempt, _ := payload["attempt"].(map[string]any)
	rerunAttemptID, _ := attempt["attempt_id"].(string)
	if rerunAttemptID == "" || rerunAttemptID == original.AttemptID {
		t.Fatalf("expected new rerun attempt id, got %#v", attempt["attempt_id"])
	}
	if got := attempt["provider"]; got != "codex_cli" {
		t.Fatalf("expected rerun provider codex_cli, got %#v", got)
	}
	if got := attempt["intent_profile"]; got != agents.IntentCreateJira {
		t.Fatalf("expected rerun to preserve intent profile, got %#v", got)
	}
	if got := attempt["instruction_text"]; got != "Turn this into a Jira-ready change request." {
		t.Fatalf("expected rerun to preserve instruction text, got %#v", got)
	}
	if got := attempt["workdir_used"]; got != nextWorkspace {
		t.Fatalf("expected rerun to use updated workspace %q, got %#v", nextWorkspace, got)
	}

	rerun := waitForAttemptStatus(t, srv, cfg.ControlToken, rerunAttemptID, "submitted", 3*time.Second)
	if got := rerun["workdir_used"]; got != nextWorkspace {
		t.Fatalf("expected completed rerun attempt to use updated workspace %q, got %#v", nextWorkspace, got)
	}
	if got := rerun["request_preview"]; got != original.RequestPreview {
		t.Fatalf("expected rerun request preview %q, got %#v", original.RequestPreview, got)
	}
}

func TestExtensionSessionIncludesRerunAttemptsFromMainUI(t *testing.T) {
	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"rerun-submit","status":"accepted","ref":"/tmp/rerun-submit.log"}'`)
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startPairReq := httptest.NewRequest(http.MethodPost, "/api/extension/pair/start", bytes.NewReader([]byte(`{"name":"Chromium Popup","browser":"chromium"}`)))
	startPairReq.Header.Set("Content-Type", "application/json")
	addAuth(startPairReq, cfg.ControlToken, true, "nonce-ext-rerun-start")
	startPairRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startPairRec, startPairReq)
	if startPairRec.Code != http.StatusOK {
		t.Fatalf("start extension pairing failed: %d %s", startPairRec.Code, startPairRec.Body.String())
	}
	var started map[string]any
	if err := json.Unmarshal(startPairRec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode extension pair start: %v", err)
	}
	pairingCode, _ := started["pairing_code"].(string)
	if pairingCode == "" {
		t.Fatalf("expected pairing code")
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/api/extension/pair/complete", bytes.NewReader([]byte(`{"pairing_code":"`+pairingCode+`","name":"Sidebar","browser":"chrome","platform":"macOS"}`)))
	completeReq.Header.Set("Content-Type", "application/json")
	completeRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("complete extension pairing failed: %d %s", completeRec.Code, completeRec.Body.String())
	}
	var completed map[string]any
	if err := json.Unmarshal(completeRec.Body.Bytes(), &completed); err != nil {
		t.Fatalf("decode extension pair complete: %v", err)
	}
	extensionToken, _ := completed["token"].(string)
	if extensionToken == "" {
		t.Fatalf("expected extension token")
	}

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{
		ID:             "evt-rerun-extension",
		RawTranscript:  "Tighten the checkout spacing",
		NormalizedText: "Tighten the checkout spacing",
	}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	pkg, err := srv.sessions.Approve("Tighten the checkout spacing")
	if err != nil {
		t.Fatalf("approve session: %v", err)
	}
	if err := srv.store.SaveCanonicalPackage(pkg); err != nil {
		t.Fatalf("save canonical package: %v", err)
	}

	intent := agents.NormalizeDeliveryIntent(agents.DeliveryIntent{
		Profile:         agents.IntentImplementChanges,
		InstructionText: "Implement the requested software changes in the current repository.",
	})
	redactedPkg := redactPackageForTransmission(*pkg)
	providerPayload, err := agents.PreviewProviderPayloadWithConfig("codex_cli", redactedPkg, "", "", intent)
	if err != nil {
		t.Fatalf("preview provider payload: %v", err)
	}
	original := srv.enqueueSubmitJob("codex_cli", redactedPkg, providerPayload, intent, "test", "test")
	_ = waitForAttemptStatus(t, srv, cfg.ControlToken, original.AttemptID, "submitted", 3*time.Second)

	rerunReq := httptest.NewRequest(http.MethodPost, "/api/session/attempt/rerun", bytes.NewReader([]byte(`{"attempt_id":"`+original.AttemptID+`"}`)))
	rerunReq.Header.Set("Content-Type", "application/json")
	addAuth(rerunReq, cfg.ControlToken, true, "nonce-rerun-extension")
	rerunRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rerunRec, rerunReq)
	if rerunRec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rerunRec.Code, rerunRec.Body.String())
	}
	var rerunPayload map[string]any
	if err := json.Unmarshal(rerunRec.Body.Bytes(), &rerunPayload); err != nil {
		t.Fatalf("decode rerun response: %v", err)
	}
	rerunAttempt, _ := rerunPayload["attempt"].(map[string]any)
	rerunAttemptID, _ := rerunAttempt["attempt_id"].(string)
	if rerunAttemptID == "" || rerunAttemptID == original.AttemptID {
		t.Fatalf("expected new rerun attempt id, got %#v", rerunAttempt["attempt_id"])
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/extension/session", nil)
	addBearerAuth(sessionReq, extensionToken, false, "")
	sessionRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("extension session failed: %d %s", sessionRec.Code, sessionRec.Body.String())
	}
	var sessionPayload map[string]any
	if err := json.Unmarshal(sessionRec.Body.Bytes(), &sessionPayload); err != nil {
		t.Fatalf("decode extension session payload: %v", err)
	}
	attempts, _ := sessionPayload["submit_attempts"].([]any)
	if len(attempts) < 2 {
		t.Fatalf("expected at least 2 submit attempts, got %#v", sessionPayload["submit_attempts"])
	}
	firstAttempt, _ := attempts[0].(map[string]any)
	if got := firstAttempt["attempt_id"]; got != rerunAttemptID {
		t.Fatalf("expected rerun attempt %q first in extension session, got %#v", rerunAttemptID, got)
	}
	if got := firstAttempt["intent_profile"]; got != agents.IntentImplementChanges {
		t.Fatalf("expected rerun intent profile %q, got %#v", agents.IntentImplementChanges, got)
	}
	if got := firstAttempt["instruction_text"]; got != "Implement the requested software changes in the current repository." {
		t.Fatalf("expected rerun instruction text in extension payload, got %#v", got)
	}
}

func TestSubmitIncludesPostSubmitAutomationResult(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	runtimePayload, err := json.Marshal(map[string]any{
		"cli_adapter_cmd":             `echo '{"run_id":"cli-run","status":"accepted","ref":"/tmp/knit-codex-test.log"}'`,
		"post_submit_rebuild_cmd":     testEchoCommand("rebuilt"),
		"post_submit_verify_cmd":      testEchoCommand("verified"),
		"post_submit_timeout_seconds": 60,
	})
	if err != nil {
		t.Fatalf("marshal runtime payload: %v", err)
	}
	runtimeReq := httptest.NewRequest(http.MethodPost, "/api/runtime/codex", bytes.NewReader(runtimePayload))
	runtimeReq.Header.Set("Content-Type", "application/json")
	addAuth(runtimeReq, cfg.ControlToken, true, "nonce-ps-runtime")
	runtimeRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(runtimeRec, runtimeReq)
	if runtimeRec.Code != http.StatusOK {
		t.Fatalf("runtime codex setup failed: %d %s", runtimeRec.Code, runtimeRec.Body.String())
	}

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-ps-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"Fix this bug","normalized":"Fix this bug","pointer_x":100,"pointer_y":80,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "nonce-ps-feedback")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-ps-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "nonce-ps-submit")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("submit failed: %d %s", submitRec.Code, submitRec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(submitRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	attemptID, _ := payload["attempt_id"].(string)
	if attemptID == "" {
		t.Fatalf("expected attempt_id in submit response")
	}
	attempt := waitForAttemptPostSubmitResult(t, srv, cfg.ControlToken, attemptID, 3*time.Second)
	ps, ok := attempt["post_submit"].(map[string]any)
	if !ok {
		t.Fatalf("expected post_submit result on completed attempt, got %#v", attempt["post_submit"])
	}
	rebuild, _ := ps["rebuild"].(map[string]any)
	verify, _ := ps["verify"].(map[string]any)
	if rebuild["status"] != "success" {
		t.Fatalf("expected rebuild success, got: %#v", rebuild)
	}
	if verify["status"] != "success" {
		t.Fatalf("expected verify success, got: %#v", verify)
	}
}

func waitForAttemptPostSubmitResult(t *testing.T, srv *Server, token, attemptID string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last []map[string]any
	for time.Now().Before(deadline) {
		last = fetchSubmitAttempts(t, srv, token)
		for _, a := range last {
			id, _ := a["attempt_id"].(string)
			if id != attemptID {
				continue
			}
			if _, ok := a["post_submit"].(map[string]any); ok {
				return a
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("attempt %s never exposed post_submit result; last attempts=%#v", attemptID, last)
	return nil
}

func waitForAttemptStatus(t *testing.T, srv *Server, token, attemptID, want string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last []map[string]any
	for time.Now().Before(deadline) {
		last = fetchSubmitAttempts(t, srv, token)
		for _, a := range last {
			id, _ := a["attempt_id"].(string)
			if id != attemptID {
				continue
			}
			status, _ := a["status"].(string)
			if status == want {
				return a
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for attempt %s status=%s; attempts=%v", attemptID, want, last)
	return nil
}

func fetchSubmitAttempts(t *testing.T, srv *Server, token string) []map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(req, token, false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("state failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	raw, _ := payload["submit_attempts"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func testEchoCommand(text string) string {
	if runtime.GOOS == "windows" {
		return "powershell -Command \"Write-Output '" + text + "'\""
	}
	return "printf '" + text + "'"
}

func newTestServer(t testing.TB, cfg config.Config) *Server {
	return newTestServerWithSTT(t, cfg, nil)
}

func newTestServerWithSTT(t testing.TB, cfg config.Config, stt transcription.Provider) *Server {
	t.Helper()
	dir := strings.TrimSpace(cfg.DataDir)
	defaultCfg := config.Default()
	defaultDataDir := strings.TrimSpace(defaultCfg.DataDir)
	if dir == "" || dir == defaultDataDir {
		dir = t.TempDir()
		cfg.DataDir = dir
	}
	sqlitePath := strings.TrimSpace(cfg.SQLitePath)
	defaultSQLitePath := strings.TrimSpace(defaultCfg.SQLitePath)
	if sqlitePath == "" || sqlitePath == defaultSQLitePath {
		cfg.SQLitePath = filepath.Join(dir, "test.db")
	}
	if cfg.ControlToken == "" {
		cfg.ControlToken = "test-token"
	}

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	encryptor, err := security.NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	store, err := storage.NewSQLiteStore(cfg.SQLitePath, encryptor)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	artifactStore, err := storage.NewArtifactStore(filepath.Join(dir, "artifacts"), encryptor)
	if err != nil {
		t.Fatalf("new artifact store: %v", err)
	}
	auditLogger, err := audit.NewLogger(dir, encryptor, "")
	if err != nil {
		t.Fatalf("new audit logger: %v", err)
	}
	captureManager := capture.NewManager()
	pointerTracker := companion.NewTracker(360)
	audioController := audio.NewController()
	autoStart := newTestAutoStartManager(t, dir)

	return New(
		cfg,
		session.NewService(),
		privileged.NewCaptureBroker(captureManager, pointerTracker, audioController),
		auditLogger,
		agents.NewRegistry(
			agents.NewCodexAPIAdapterFromEnv(),
			agents.NewCLIAdapterFromEnv(),
			agents.NewClaudeCLIAdapterFromEnv(),
			agents.NewOpenCodeCLIAdapterFromEnv(),
		),
		store,
		artifactStore,
		autoStart,
		stt,
	)
}

func newRestoredTestServerWithStore(t testing.TB, cfg config.Config, store storage.Store, encryptor *security.Encryptor) *Server {
	t.Helper()
	audioController := audio.NewController()
	if persisted, err := store.LoadOperatorState(); err == nil && persisted != nil {
		cfg = operatorstate.Apply(cfg, audioController, persisted)
	}
	sessions := session.NewService()
	if history, err := store.ListSessions(); err == nil && len(history) > 0 {
		var approvedPkg *session.CanonicalPackage
		if history[0] != nil && history[0].Approved {
			approvedPkg, _ = store.LoadLatestCanonicalPackage(history[0].ID)
		}
		sessions.Bootstrap(history, approvedPkg)
	}
	artifactStore, err := storage.NewArtifactStore(filepath.Join(cfg.DataDir, "artifacts"), encryptor)
	if err != nil {
		t.Fatalf("new artifact store: %v", err)
	}
	auditLogger, err := audit.NewLogger(cfg.DataDir, encryptor, "")
	if err != nil {
		t.Fatalf("new audit logger: %v", err)
	}
	captureManager := capture.NewManager()
	pointerTracker := companion.NewTracker(360)
	autoStart := newTestAutoStartManager(t, cfg.DataDir)
	return New(
		cfg,
		sessions,
		privileged.NewCaptureBroker(captureManager, pointerTracker, audioController),
		auditLogger,
		agents.NewRegistry(
			agents.NewCodexAPIAdapterFromEnv(),
			agents.NewCLIAdapterFromEnv(),
			agents.NewClaudeCLIAdapterFromEnv(),
			agents.NewOpenCodeCLIAdapterFromEnv(),
		),
		store,
		artifactStore,
		autoStart,
		transcription.NewProviderFromEnv(cfg.TranscriptionMode),
	)
}

func TestExtensionPairingIssuesScopedTokenAndSupportsSessionFlow(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startPairReq := httptest.NewRequest(http.MethodPost, "/api/extension/pair/start", bytes.NewReader([]byte(`{"name":"Chromium Popup","browser":"chromium"}`)))
	startPairReq.Header.Set("Content-Type", "application/json")
	addAuth(startPairReq, cfg.ControlToken, true, "nonce-ext-pair-start")
	startPairRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startPairRec, startPairReq)
	if startPairRec.Code != http.StatusOK {
		t.Fatalf("start extension pairing failed: %d %s", startPairRec.Code, startPairRec.Body.String())
	}
	var started map[string]any
	if err := json.Unmarshal(startPairRec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode pairing start: %v", err)
	}
	pairingCode, _ := started["pairing_code"].(string)
	if pairingCode == "" {
		t.Fatalf("expected pairing code")
	}
	if !regexp.MustCompile(`^[A-Z0-9]+$`).MatchString(pairingCode) {
		t.Fatalf("expected pairing code to be alphanumeric only, got %q", pairingCode)
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/api/extension/pair/complete", bytes.NewReader([]byte(`{"pairing_code":"`+pairingCode+`","name":"Popup","browser":"chrome","platform":"macOS"}`)))
	completeReq.Header.Set("Content-Type", "application/json")
	completeRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("complete extension pairing failed: %d %s", completeRec.Code, completeRec.Body.String())
	}
	var completed map[string]any
	if err := json.Unmarshal(completeRec.Body.Bytes(), &completed); err != nil {
		t.Fatalf("decode pairing complete: %v", err)
	}
	extensionToken, _ := completed["token"].(string)
	if extensionToken == "" {
		t.Fatalf("expected extension token")
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/extension/session", nil)
	addBearerAuth(sessionReq, extensionToken, false, "")
	sessionRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("extension session failed: %d %s", sessionRec.Code, sessionRec.Body.String())
	}

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Extension","target_url":"https://example.com/ext"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addBearerAuth(startReq, extensionToken, true, "nonce-ext-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("extension session start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	configReq := httptest.NewRequest(http.MethodPost, "/api/runtime/codex", bytes.NewReader([]byte(`{"default_provider":"cli"}`)))
	configReq.Header.Set("Content-Type", "application/json")
	addBearerAuth(configReq, extensionToken, true, "nonce-ext-config")
	configRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(configRec, configReq)
	if configRec.Code != http.StatusForbidden {
		t.Fatalf("expected extension token runtime codex update to be forbidden, got %d %s", configRec.Code, configRec.Body.String())
	}

	stateReq := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	addAuth(stateReq, cfg.ControlToken, false, "")
	stateRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("state failed: %d %s", stateRec.Code, stateRec.Body.String())
	}
	var statePayload map[string]any
	if err := json.Unmarshal(stateRec.Body.Bytes(), &statePayload); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	pairings, _ := statePayload["extension_pairings"].([]any)
	if len(pairings) != 1 {
		t.Fatalf("expected one extension pairing in state, got %#v", statePayload["extension_pairings"])
	}

	pairingID, _ := started["pairing_id"].(string)
	revokeReq := httptest.NewRequest(http.MethodPost, "/api/extension/pair/revoke", bytes.NewReader([]byte(`{"pairing_id":"`+pairingID+`"}`)))
	revokeReq.Header.Set("Content-Type", "application/json")
	addAuth(revokeReq, cfg.ControlToken, true, "nonce-ext-revoke")
	revokeRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("revoke extension pairing failed: %d %s", revokeRec.Code, revokeRec.Body.String())
	}

	sessionReq = httptest.NewRequest(http.MethodGet, "/api/extension/session", nil)
	addBearerAuth(sessionReq, extensionToken, false, "")
	sessionRec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked extension token to fail auth, got %d %s", sessionRec.Code, sessionRec.Body.String())
	}
}

func TestExtensionSubmitAppearsInMainUIState(t *testing.T) {
	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"ext-submit-visible","status":"accepted","ref":"/tmp/ext-submit-visible.log"}'`)
	t.Setenv("KNIT_DEFAULT_PROVIDER", "codex_cli")
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startPairReq := httptest.NewRequest(http.MethodPost, "/api/extension/pair/start", bytes.NewReader([]byte(`{"name":"Chromium Popup","browser":"chromium"}`)))
	startPairReq.Header.Set("Content-Type", "application/json")
	addAuth(startPairReq, cfg.ControlToken, true, "nonce-ext-pair-start-submit")
	startPairRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startPairRec, startPairReq)
	if startPairRec.Code != http.StatusOK {
		t.Fatalf("start extension pairing failed: %d %s", startPairRec.Code, startPairRec.Body.String())
	}
	var started map[string]any
	if err := json.Unmarshal(startPairRec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode pairing start: %v", err)
	}
	pairingCode, _ := started["pairing_code"].(string)
	if pairingCode == "" {
		t.Fatalf("expected pairing code")
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/api/extension/pair/complete", bytes.NewReader([]byte(`{"pairing_code":"`+pairingCode+`","name":"Sidebar","browser":"chrome","platform":"macOS"}`)))
	completeReq.Header.Set("Content-Type", "application/json")
	completeRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("complete extension pairing failed: %d %s", completeRec.Code, completeRec.Body.String())
	}
	var completed map[string]any
	if err := json.Unmarshal(completeRec.Body.Bytes(), &completed); err != nil {
		t.Fatalf("decode pairing complete: %v", err)
	}
	extensionToken, _ := completed["token"].(string)
	if extensionToken == "" {
		t.Fatalf("expected extension token")
	}

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Extension","target_url":"https://example.com/ext-submit"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addBearerAuth(startReq, extensionToken, true, "nonce-ext-start-submit")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("extension session start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	noteBody, noteCT := multipartNoteBody(t, "Move this widget higher.", tinyPNG(t))
	noteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", noteBody)
	noteReq.Header.Set("Content-Type", noteCT)
	addBearerAuth(noteReq, extensionToken, true, "nonce-ext-note-submit")
	noteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusOK {
		t.Fatalf("extension feedback note failed: %d %s", noteRec.Code, noteRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":"Move this widget higher."}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addBearerAuth(approveReq, extensionToken, true, "nonce-ext-approve-submit")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("extension approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":""}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addBearerAuth(submitReq, extensionToken, true, "nonce-ext-submit")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("extension submit failed: %d %s", submitRec.Code, submitRec.Body.String())
	}
	var submitPayload map[string]any
	if err := json.Unmarshal(submitRec.Body.Bytes(), &submitPayload); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	attemptID, _ := submitPayload["attempt_id"].(string)
	if attemptID == "" {
		t.Fatalf("expected attempt id in extension submit response")
	}
	if got, _ := submitPayload["provider"].(string); got != "codex_cli" {
		t.Fatalf("expected codex_cli provider, got %#v", submitPayload["provider"])
	}

	attempt := waitForAttemptStatus(t, srv, cfg.ControlToken, attemptID, "submitted", 3*time.Second)
	if got, _ := attempt["source"].(string); got != "browser_extension" {
		t.Fatalf("expected browser_extension source, got %#v", attempt["source"])
	}
	if got, _ := attempt["actor"].(string); !strings.HasPrefix(got, "extension:") {
		t.Fatalf("expected extension actor, got %#v", attempt["actor"])
	}
}

func TestPreviewAndSubmitCarryDeliveryIntent(t *testing.T) {
	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"intent-submit","status":"accepted","ref":"/tmp/intent-submit.log"}'`)
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-intent-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"Need a clearer billing summary","normalized":"Need a clearer billing summary","window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "nonce-intent-feedback")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":"Need a clearer billing summary"}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "nonce-intent-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	intentBody := []byte(`{"provider":"cli","intent_profile":"create_jira_tickets","instruction_text":"Create Jira-ready tickets for this feedback and group them by owning team."}`)
	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader(intentBody))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "nonce-intent-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}
	var preview payloadPreviewResponse
	if err := json.Unmarshal(previewRec.Body.Bytes(), &preview); err != nil {
		t.Fatalf("decode preview payload: %v", err)
	}
	if preview.Preview.IntentProfile != agents.IntentCreateJira || preview.Preview.IntentLabel != "Create Jira tickets" {
		t.Fatalf("expected preview to carry selected intent, got %#v", preview.Preview)
	}
	if preview.Preview.InstructionText != "Create Jira-ready tickets for this feedback and group them by owning team." {
		t.Fatalf("expected preview to carry instruction text, got %#v", preview.Preview.InstructionText)
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader(intentBody))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "nonce-intent-submit")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("submit failed: %d %s", submitRec.Code, submitRec.Body.String())
	}
	var submitPayload map[string]any
	if err := json.Unmarshal(submitRec.Body.Bytes(), &submitPayload); err != nil {
		t.Fatalf("decode submit payload: %v", err)
	}
	if got := submitPayload["intent_profile"]; got != agents.IntentCreateJira {
		t.Fatalf("expected submit response to carry intent profile, got %#v", got)
	}
	if got := submitPayload["intent_label"]; got != "Create Jira tickets" {
		t.Fatalf("expected submit response to carry intent label, got %#v", got)
	}

	attemptID, _ := submitPayload["attempt_id"].(string)
	attempt := waitForAttemptStatus(t, srv, cfg.ControlToken, attemptID, "submitted", 3*time.Second)
	if got := attempt["intent_profile"]; got != agents.IntentCreateJira {
		t.Fatalf("expected attempt intent profile, got %#v", got)
	}
	if got := attempt["intent_label"]; got != "Create Jira tickets" {
		t.Fatalf("expected attempt intent label, got %#v", got)
	}
	if got := attempt["instruction_text"]; got != "Create Jira-ready tickets for this feedback and group them by owning team." {
		t.Fatalf("expected attempt instruction text, got %#v", got)
	}
}

func TestExtensionPairingPersistsAcrossServerRestart(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.DataDir = t.TempDir()
	cfg.SQLitePath = filepath.Join(cfg.DataDir, "test.db")

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	encryptor, err := security.NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	store, err := storage.NewSQLiteStore(cfg.SQLitePath, encryptor)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if err := store.SaveOperatorState(&operatorstate.State{
		Version: 1,
		Extensions: operatorstate.ExtensionState{
			Pairings: []operatorstate.ExtensionPairing{{
				ID:           "ext-restored",
				Name:         "Restored popup",
				Browser:      "chromium",
				Platform:     "macOS",
				Capabilities: []string{"read", "capture", "submit"},
				TokenHash:    hashToken("paired-token"),
				CreatedAt:    time.Now().UTC(),
			}},
		},
	}); err != nil {
		t.Fatalf("save operator state: %v", err)
	}

	srv := newRestoredTestServerWithStore(t, cfg, store, encryptor)
	req := httptest.NewRequest(http.MethodGet, "/api/extension/session", nil)
	addBearerAuth(req, "paired-token", false, "")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected restored extension token to authenticate, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestExtensionSessionIncludesCommonSubmitFailureOutcomesForBrowserExtension(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startPairReq := httptest.NewRequest(http.MethodPost, "/api/extension/pair/start", bytes.NewReader([]byte(`{"name":"Chromium Popup","browser":"chromium"}`)))
	startPairReq.Header.Set("Content-Type", "application/json")
	addAuth(startPairReq, cfg.ControlToken, true, "nonce-ext-pair-start-outcomes")
	startPairRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startPairRec, startPairReq)
	if startPairRec.Code != http.StatusOK {
		t.Fatalf("start extension pairing failed: %d %s", startPairRec.Code, startPairRec.Body.String())
	}
	var started map[string]any
	if err := json.Unmarshal(startPairRec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode pairing start: %v", err)
	}
	pairingCode, _ := started["pairing_code"].(string)
	if pairingCode == "" {
		t.Fatalf("expected pairing code")
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/api/extension/pair/complete", bytes.NewReader([]byte(`{"pairing_code":"`+pairingCode+`","name":"Popup","browser":"chrome","platform":"macOS"}`)))
	completeReq.Header.Set("Content-Type", "application/json")
	completeRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("complete extension pairing failed: %d %s", completeRec.Code, completeRec.Body.String())
	}
	var completed map[string]any
	if err := json.Unmarshal(completeRec.Body.Bytes(), &completed); err != nil {
		t.Fatalf("decode pairing complete: %v", err)
	}
	extensionToken, _ := completed["token"].(string)
	if extensionToken == "" {
		t.Fatalf("expected extension token")
	}

	srv.submitMu.Lock()
	srv.submitAttempts = []submitAttempt{
		{
			AttemptID:      "attempt-no-input",
			Status:         "submitted",
			OutcomeCode:    submitOutcomeNoInput,
			OutcomeTitle:   "No input",
			OutcomeMessage: "Knit submitted this run without any captured change requests or artifacts, so the coding agent had nothing to change.",
		},
		{
			AttemptID:      "attempt-trusted-directory",
			Status:         "failed",
			OutcomeCode:    submitOutcomeTrustedDir,
			OutcomeTitle:   "Trusted directory required",
			OutcomeMessage: "Go back to Capture, Review, and Send, open Settings, then check Workspace first. If the wrong repository is selected, choose the correct workspace for this project and rerun. If the workspace is already correct, open Settings > Agent and switch Sandbox to danger-full-access before rerunning. Workspace used: /tmp/ruddur.",
		},
		{
			AttemptID:      "attempt-wrong-workspace",
			Status:         "submitted",
			OutcomeCode:    submitOutcomeWrongWorkspace,
			OutcomeTitle:   "Wrong workspace",
			OutcomeMessage: "Go back to Capture, Review, and Send, open Settings > Workspace, and choose the repository that matches this request before rerunning. Workspace used: /tmp/ruddur.",
		},
		{
			AttemptID:      "attempt-read-only",
			Status:         "submitted",
			OutcomeCode:    submitOutcomeReadOnly,
			OutcomeTitle:   "Read-only",
			OutcomeMessage: "Go back to Capture, Review, and Send, open Settings > Agent, and switch Sandbox to danger-full-access before rerunning.",
		},
	}
	srv.submitMu.Unlock()

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/extension/session", nil)
	addBearerAuth(sessionReq, extensionToken, false, "")
	sessionRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("extension session failed: %d %s", sessionRec.Code, sessionRec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(sessionRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode extension session payload: %v", err)
	}
	attempts, _ := payload["submit_attempts"].([]any)
	if len(attempts) != 4 {
		t.Fatalf("expected 4 submit attempts, got %#v", payload["submit_attempts"])
	}

	want := map[string]struct {
		code     string
		title    string
		contains string
	}{
		"attempt-no-input":          {code: submitOutcomeNoInput, title: "No input", contains: "had nothing to change"},
		"attempt-trusted-directory": {code: submitOutcomeTrustedDir, title: "Trusted directory required", contains: "open Settings, then check Workspace first"},
		"attempt-wrong-workspace":   {code: submitOutcomeWrongWorkspace, title: "Wrong workspace", contains: "open Settings > Workspace"},
		"attempt-read-only":         {code: submitOutcomeReadOnly, title: "Read-only", contains: "open Settings > Agent"},
	}
	for _, raw := range attempts {
		attempt, _ := raw.(map[string]any)
		id, _ := attempt["attempt_id"].(string)
		exp, ok := want[id]
		if !ok {
			t.Fatalf("unexpected attempt in extension payload: %#v", attempt)
		}
		if got := attempt["outcome_code"]; got != exp.code {
			t.Fatalf("expected outcome_code %q for %s, got %#v", exp.code, id, got)
		}
		if got := attempt["outcome_title"]; got != exp.title {
			t.Fatalf("expected outcome_title %q for %s, got %#v", exp.title, id, got)
		}
		message, _ := attempt["outcome_message"].(string)
		if !strings.Contains(message, exp.contains) {
			t.Fatalf("expected outcome_message for %s to contain %q, got %q", id, exp.contains, message)
		}
	}
}

func newTestAutoStartManager(t testing.TB, dir string) *platform.AutoStartManager {
	t.Helper()
	execPath := filepath.Join(dir, "knit-test-binary")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("write test executable: %v", err)
	}
	return platform.NewAutoStartManagerForTest(runtime.GOOS, "Knit", execPath, nil, dir, filepath.Join(dir, ".config"), filepath.Join(dir, "AppData", "Roaming"))
}

func addAuth(req *http.Request, token string, mutation bool, nonce string) {
	if token != "" {
		req.Header.Set("X-Knit-Token", token)
	}
	if mutation {
		if nonce == "" {
			nonce = "nonce-default"
		}
		req.Header.Set("X-Knit-Nonce", nonce)
		req.Header.Set("X-Knit-Timestamp", strconv.FormatInt(time.Now().UTC().UnixMilli(), 10))
	}
}

func addBearerAuth(req *http.Request, token string, mutation bool, nonce string) {
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if mutation {
		if nonce == "" {
			nonce = "nonce-default"
		}
		req.Header.Set("X-Knit-Nonce", nonce)
		req.Header.Set("X-Knit-Timestamp", strconv.FormatInt(time.Now().UTC().UnixMilli(), 10))
	}
}

func assertAllRenderedButtonsHaveTooltips(t *testing.T, body string) {
	t.Helper()

	buttonTagPattern := regexp.MustCompile(`(?s)<button\b[^>]*>`)
	buttonTags := buttonTagPattern.FindAllString(body, -1)
	if len(buttonTags) == 0 {
		t.Fatalf("expected at least one rendered button")
	}
	for _, tag := range buttonTags {
		if !strings.Contains(tag, `title="`) {
			t.Fatalf("expected button tag to include title attribute: %s", tag)
		}
	}
}

type fakeRemoteSTTProvider struct{}

func (fakeRemoteSTTProvider) Name() string     { return "fake-remote-stt" }
func (fakeRemoteSTTProvider) Mode() string     { return "remote" }
func (fakeRemoteSTTProvider) Endpoint() string { return "https://api.openai.com" }
func (fakeRemoteSTTProvider) Transcribe(_ context.Context, _ string) (string, error) {
	return "transcript", nil
}

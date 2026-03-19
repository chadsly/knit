package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"knit/internal/agents"
	"knit/internal/config"
)

func TestE2EWorkflowCaptureToSubmission(t *testing.T) {
	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"e2e-capture-submit","status":"accepted","ref":"/tmp/e2e-capture-submit.log"}'`)
	cfg := config.Default()
	cfg.ControlToken = "e2e-token"
	cfg.AllowRemoteSubmission = true
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "e2e-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start session failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var started map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := started["id"].(string)
	if sessionID == "" {
		t.Fatalf("expected session id")
	}

	settingsReq := httptest.NewRequest(http.MethodPost, "/api/session/replay/settings", bytes.NewReader([]byte(`{"capture_input_values":true}`)))
	settingsReq.Header.Set("Content-Type", "application/json")
	addAuth(settingsReq, cfg.ControlToken, true, "e2e-replay-settings")
	settingsRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(settingsRec, settingsReq)
	if settingsRec.Code != http.StatusOK {
		t.Fatalf("replay settings failed: %d %s", settingsRec.Code, settingsRec.Body.String())
	}

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{"session_id":"`+sessionID+`","x":612,"y":384,"event_type":"input","window":"Browser Preview","url":"https://example.com/app?draft=secret","route":"/app","target_tag":"button","target_id":"save","target_test_id":"settings-save","target_role":"button","target_label":"Save Settings","target_selector":"#save","input_type":"text","value":"Save CTA","value_captured":true,"dom":{"tag":"button","id":"save","test_id":"settings-save","label":"Save Settings","selector":"#save","text_preview":"Save Settings"},"console":[{"level":"warn","message":"Save button repainted twice"}],"network":[{"kind":"fetch","method":"POST","url":"https://example.com/api/save?token=secret","status":500,"ok":false,"duration_ms":812}]}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "e2e-pointer")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusOK {
		t.Fatalf("pointer event failed: %d %s", pointerRec.Code, pointerRec.Body.String())
	}

	noteBody, noteCT := multipartNoteBody(t, "This button should be larger", tinyPNG(t))
	noteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", noteBody)
	noteReq.Header.Set("Content-Type", noteCT)
	addAuth(noteReq, cfg.ControlToken, true, "e2e-note")
	noteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusOK {
		t.Fatalf("feedback note failed: %d %s", noteRec.Code, noteRec.Body.String())
	}
	var noteResp map[string]any
	if err := json.Unmarshal(noteRec.Body.Bytes(), &noteResp); err != nil {
		t.Fatalf("decode note response: %v", err)
	}
	eventID, _ := noteResp["event_id"].(string)
	if eventID == "" {
		t.Fatalf("expected event id from note capture")
	}

	clipBody, clipCT := multipartClipBody(t, eventID, []byte("fake-webm-clip"))
	clipReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/clip", clipBody)
	clipReq.Header.Set("Content-Type", clipCT)
	addAuth(clipReq, cfg.ControlToken, true, "e2e-clip")
	clipRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(clipRec, clipReq)
	if clipRec.Code != http.StatusOK {
		t.Fatalf("attach clip failed: %d %s", clipRec.Code, clipRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":"Increase primary action prominence"}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "e2e-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "e2e-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("payload preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}
	var previewPayload payloadPreviewResponse
	if err := json.Unmarshal(previewRec.Body.Bytes(), &previewPayload); err != nil {
		t.Fatalf("decode preview payload: %v", err)
	}
	if previewPayload.Provider != "codex_cli" {
		t.Fatalf("expected codex_cli preview provider, got %q", previewPayload.Provider)
	}
	if previewPayload.Preview.Summary != "Increase primary action prominence" {
		t.Fatalf("expected approved summary in preview, got %q", previewPayload.Preview.Summary)
	}
	if len(previewPayload.Preview.Notes) != 1 {
		t.Fatalf("expected one preview note, got %d", len(previewPayload.Preview.Notes))
	}
	notePreview := previewPayload.Preview.Notes[0]
	if notePreview.Text != "Increase primary action prominence" {
		t.Fatalf("expected preview note text to match approved summary, got %q", notePreview.Text)
	}
	if notePreview.DOMSummary == "" || !strings.Contains(notePreview.DOMSummary, "#save") {
		t.Fatalf("expected preview dom summary, got %#v", notePreview.DOMSummary)
	}
	if len(notePreview.Console) != 1 || !strings.Contains(notePreview.Console[0], "Save button repainted twice") {
		t.Fatalf("expected preview console context, got %#v", notePreview.Console)
	}
	if len(notePreview.Network) != 1 || !strings.Contains(notePreview.Network[0], "https://example.com/api/save") {
		t.Fatalf("expected preview network context, got %#v", notePreview.Network)
	}
	if notePreview.PointerEventCount == 0 {
		t.Fatalf("expected preview pointer event count")
	}
	if notePreview.ReplayValueMode != "opt_in" || notePreview.ReplayStepCount == 0 || !strings.Contains(strings.Join(notePreview.ReplaySteps, "\n"), "Save CTA") {
		t.Fatalf("expected replay bundle preview with opted-in values, got %#v", notePreview)
	}
	if !strings.Contains(notePreview.PlaywrightScript, ".fill(\"Save CTA\")") {
		t.Fatalf("expected preview playwright script, got %q", notePreview.PlaywrightScript)
	}
	if !strings.HasPrefix(notePreview.ScreenshotDataURL, "data:image/") {
		t.Fatalf("expected screenshot data URL, got %q", notePreview.ScreenshotDataURL)
	}
	if !strings.HasPrefix(notePreview.VideoDataURL, "data:video/webm;base64,") {
		t.Fatalf("expected video data URL, got %q", notePreview.VideoDataURL)
	}

	editReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/update-text", bytes.NewReader([]byte(`{"event_id":"`+notePreview.EventID+`","text":"Increase the primary action size and contrast"}`)))
	editReq.Header.Set("Content-Type", "application/json")
	addAuth(editReq, cfg.ControlToken, true, "e2e-preview-edit")
	editRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(editRec, editReq)
	if editRec.Code != http.StatusOK {
		t.Fatalf("edit preview text failed: %d %s", editRec.Code, editRec.Body.String())
	}

	approveReq = httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "e2e-approve-edited")
	approveRec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("re-approve after preview edit failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	previewReq = httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "e2e-preview-edited")
	previewRec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("edited payload preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}
	if err := json.Unmarshal(previewRec.Body.Bytes(), &previewPayload); err != nil {
		t.Fatalf("decode edited preview payload: %v", err)
	}
	if len(previewPayload.Preview.Notes) != 1 {
		t.Fatalf("expected one edited preview note, got %d", len(previewPayload.Preview.Notes))
	}
	if got := previewPayload.Preview.Notes[0].Text; got != "Increase the primary action size and contrast" {
		t.Fatalf("expected edited preview text, got %q", got)
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "e2e-submit")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("submit failed: %d %s", submitRec.Code, submitRec.Body.String())
	}
	var submitPayload map[string]any
	if err := json.Unmarshal(submitRec.Body.Bytes(), &submitPayload); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	attemptID, _ := submitPayload["attempt_id"].(string)
	if attemptID == "" {
		t.Fatalf("expected attempt id in submit response")
	}
	_ = waitForAttemptStatus(t, srv, cfg.ControlToken, attemptID, "submitted", 3*time.Second)
	if pkg, err := srv.store.LoadLatestCanonicalPackage(sessionID); err == nil {
		if len(pkg.Artifacts) < 4 {
			t.Fatalf("expected canonical package artifacts to include replay exports, got %#v", pkg.Artifacts)
		}
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/api/session/history", nil)
	addAuth(historyReq, cfg.ControlToken, false, "")
	historyRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(historyRec, historyReq)
	if historyRec.Code != http.StatusOK {
		t.Fatalf("history failed: %d %s", historyRec.Code, historyRec.Body.String())
	}

	var sessions []map[string]any
	if err := json.Unmarshal(historyRec.Body.Bytes(), &sessions); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatalf("expected at least one session in history")
	}
	if sessions[0]["status"] != "submitted" {
		t.Fatalf("expected submitted status, got %v", sessions[0]["status"])
	}
}

func TestE2EWorkflowLifecycleStatePauseResumeStop(t *testing.T) {
	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"e2e-lifecycle","status":"accepted","ref":"/tmp/e2e-lifecycle.log"}'`)
	cfg := config.Default()
	cfg.ControlToken = "e2e-token-lifecycle"
	cfg.AllowRemoteSubmission = true
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/lifecycle"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "e2e-life-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start session failed: %d %s", startRec.Code, startRec.Body.String())
	}

	state := fetchE2EState(t, srv, cfg.ControlToken)
	if state["capture_state"] != "active" {
		t.Fatalf("expected active capture state after start, got %#v", state["capture_state"])
	}
	sessionObj, _ := state["session"].(map[string]any)
	if sessionObj["target_url"] != "https://example.com/lifecycle" {
		t.Fatalf("expected state to include target url, got %#v", sessionObj["target_url"])
	}
	if profile, _ := state["platform_profile"].(map[string]any); profile["supported"] != true {
		t.Fatalf("expected supported platform profile, got %#v", profile)
	}
	if runtimeGuide, _ := state["runtime_platform"].(map[string]any); runtimeGuide["host_target"] == "" {
		t.Fatalf("expected runtime platform metadata, got %#v", runtimeGuide)
	}

	pauseReq := httptest.NewRequest(http.MethodPost, "/api/session/pause", bytes.NewReader([]byte(`{}`)))
	pauseReq.Header.Set("Content-Type", "application/json")
	addAuth(pauseReq, cfg.ControlToken, true, "e2e-life-pause")
	pauseRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pauseRec, pauseReq)
	if pauseRec.Code != http.StatusOK {
		t.Fatalf("pause session failed: %d %s", pauseRec.Code, pauseRec.Body.String())
	}
	state = fetchE2EState(t, srv, cfg.ControlToken)
	if state["capture_state"] != "paused" {
		t.Fatalf("expected paused capture state after pause, got %#v", state["capture_state"])
	}

	resumeReq := httptest.NewRequest(http.MethodPost, "/api/session/resume", bytes.NewReader([]byte(`{}`)))
	resumeReq.Header.Set("Content-Type", "application/json")
	addAuth(resumeReq, cfg.ControlToken, true, "e2e-life-resume")
	resumeRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(resumeRec, resumeReq)
	if resumeRec.Code != http.StatusOK {
		t.Fatalf("resume session failed: %d %s", resumeRec.Code, resumeRec.Body.String())
	}
	state = fetchE2EState(t, srv, cfg.ControlToken)
	if state["capture_state"] != "active" {
		t.Fatalf("expected active capture state after resume, got %#v", state["capture_state"])
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"Lifecycle note","normalized":"Lifecycle note","pointer_x":40,"pointer_y":80,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "e2e-life-feedback")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":"Lifecycle summary"}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "e2e-life-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}
	var pkg map[string]any
	if err := json.Unmarshal(approveRec.Body.Bytes(), &pkg); err != nil {
		t.Fatalf("decode approve response: %v", err)
	}
	if pkg["summary"] != "Lifecycle summary" {
		t.Fatalf("expected approval summary, got %#v", pkg["summary"])
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/api/session/payload/preview", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	previewReq.Header.Set("Content-Type", "application/json")
	addAuth(previewReq, cfg.ControlToken, true, "e2e-life-preview")
	previewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "e2e-life-submit")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("submit failed: %d %s", submitRec.Code, submitRec.Body.String())
	}
	var submitPayload map[string]any
	if err := json.Unmarshal(submitRec.Body.Bytes(), &submitPayload); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	attemptID, _ := submitPayload["attempt_id"].(string)
	if attemptID == "" {
		t.Fatalf("expected attempt id")
	}
	_ = waitForAttemptStatus(t, srv, cfg.ControlToken, attemptID, "submitted", 3*time.Second)

	stopReq := httptest.NewRequest(http.MethodPost, "/api/session/stop", bytes.NewReader([]byte(`{}`)))
	stopReq.Header.Set("Content-Type", "application/json")
	addAuth(stopReq, cfg.ControlToken, true, "e2e-life-stop")
	stopRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(stopRec, stopReq)
	if stopRec.Code != http.StatusOK {
		t.Fatalf("stop session failed: %d %s", stopRec.Code, stopRec.Body.String())
	}
	state = fetchE2EState(t, srv, cfg.ControlToken)
	if state["capture_state"] != "inactive" {
		t.Fatalf("expected inactive capture state after stop, got %#v", state["capture_state"])
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/api/session/history", nil)
	addAuth(historyReq, cfg.ControlToken, false, "")
	historyRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(historyRec, historyReq)
	if historyRec.Code != http.StatusOK {
		t.Fatalf("history failed: %d %s", historyRec.Code, historyRec.Body.String())
	}
	var history []map[string]any
	if err := json.Unmarshal(historyRec.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if len(history) == 0 {
		t.Fatalf("expected history entries")
	}
	if history[0]["status"] != "stopped" {
		t.Fatalf("expected stopped history status after explicit stop, got %#v", history[0]["status"])
	}
}

func TestE2EWorkflowSeriesQueueMultiSubmit(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", "series")
	t.Setenv("KNIT_CLI_ADAPTER_CMD", slowCLIAdapterCommand())

	cfg := config.Default()
	cfg.ControlToken = "e2e-token-queue"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "e2e-queue-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start session failed: %d %s", startRec.Code, startRec.Body.String())
	}

	firstAttempt := queueOneSubmission(t, srv, cfg.ControlToken, "e2e-queue-1", "First queued change")
	secondAttempt := queueOneSubmission(t, srv, cfg.ControlToken, "e2e-queue-2", "Second queued change")

	_ = waitForAttemptStatus(t, srv, cfg.ControlToken, firstAttempt, "submitted", 6*time.Second)
	second := waitForAttemptStatus(t, srv, cfg.ControlToken, secondAttempt, "submitted", 6*time.Second)

	var queueWait int64
	switch v := second["queue_wait_ms"].(type) {
	case float64:
		queueWait = int64(v)
	case int64:
		queueWait = v
	case int:
		queueWait = int64(v)
	}
	if queueWait <= 0 {
		t.Fatalf("expected second queued submission to include queue_wait_ms > 0, got %d", queueWait)
	}
}

func TestE2EWorkflowSeriesQueueCancelQueuedSubmission(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", "series")
	t.Setenv("KNIT_CLI_ADAPTER_CMD", slowCLIAdapterCommand())

	cfg := config.Default()
	cfg.ControlToken = "e2e-token-queue-cancel"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "e2e-queue-cancel-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start session failed: %d %s", startRec.Code, startRec.Body.String())
	}

	firstAttempt := queueOneSubmission(t, srv, cfg.ControlToken, "e2e-queue-cancel-1", "First queued change")
	secondAttempt := queueOneSubmission(t, srv, cfg.ControlToken, "e2e-queue-cancel-2", "Second queued change")

	_ = waitForAttemptStatus(t, srv, cfg.ControlToken, firstAttempt, "in_progress", 4*time.Second)

	cancelReq := httptest.NewRequest(http.MethodPost, "/api/session/attempt/cancel", bytes.NewReader([]byte(`{"attempt_id":"`+secondAttempt+`"}`)))
	cancelReq.Header.Set("Content-Type", "application/json")
	addAuth(cancelReq, cfg.ControlToken, true, "e2e-queue-cancel-stop")
	cancelRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("cancel attempt failed: %d %s", cancelRec.Code, cancelRec.Body.String())
	}

	_ = waitForAttemptStatus(t, srv, cfg.ControlToken, firstAttempt, "submitted", 6*time.Second)
	second := waitForAttemptStatus(t, srv, cfg.ControlToken, secondAttempt, submitStatusCanceled, 4*time.Second)
	if got := second["status"]; got != submitStatusCanceled {
		t.Fatalf("expected canceled status, got %#v", got)
	}
}

func TestE2EWorkflowOfflineDeferredSubmissionResume(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", "series")
	t.Setenv("KNIT_SUBMIT_MAX_ATTEMPTS", "1")
	t.Setenv("KNIT_OFFLINE_RETRY_SECONDS", "1")

	cfg := config.Default()
	cfg.ControlToken = "e2e-token-offline-deferred"
	srv := newTestServer(t, cfg)
	srv.agents = agents.NewRegistry(&fakeQueueAdapter{
		name:          "codex_api",
		remote:        true,
		delay:         10 * time.Millisecond,
		failUntilCall: 1,
		failError:     "dial tcp: lookup api.openai.com: no such host",
	})

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "e2e-offline-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start session failed: %d %s", startRec.Code, startRec.Body.String())
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"Offline defer test","normalized":"Offline defer test","pointer_x":12,"pointer_y":16,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "e2e-offline-feedback")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "e2e-offline-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"codex_api"}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, cfg.ControlToken, true, "e2e-offline-submit")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("submit failed: %d %s", submitRec.Code, submitRec.Body.String())
	}
	var submitPayload map[string]any
	if err := json.Unmarshal(submitRec.Body.Bytes(), &submitPayload); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	attemptID, _ := submitPayload["attempt_id"].(string)
	if attemptID == "" {
		t.Fatalf("expected attempt id")
	}
	attempt := waitForAttemptStatus(t, srv, cfg.ControlToken, attemptID, "submitted", 6*time.Second)
	timeline, _ := attempt["timeline"].([]any)
	foundDeferred := false
	for _, item := range timeline {
		evt, _ := item.(map[string]any)
		if status, _ := evt["status"].(string); status == "deferred_offline" {
			foundDeferred = true
			break
		}
	}
	if !foundDeferred {
		t.Fatalf("expected deferred_offline timeline status before submit completion; timeline=%v", timeline)
	}
}

func TestE2EWorkflowEnhancementMetadataAndLaserMode(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "e2e-token-enhancements"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app","review_mode":"accessibility"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "e2e-enh-start")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start session failed: %d %s", startRec.Code, startRec.Body.String())
	}
	var started map[string]any
	if err := json.Unmarshal(startRec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	sessionID, _ := started["id"].(string)
	if sessionID == "" {
		t.Fatalf("expected session id")
	}

	pointerReq := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader([]byte(`{"session_id":"`+sessionID+`","x":220,"y":140,"event_type":"move","window":"Browser Preview","url":"https://example.com/app","route":"/app","target_tag":"button","target_id":"save","target_test_id":"settings-save","target_label":"Save Settings"}`)))
	pointerReq.Header.Set("Content-Type", "application/json")
	addAuth(pointerReq, cfg.ControlToken, true, "e2e-enh-pointer")
	pointerRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(pointerRec, pointerReq)
	if pointerRec.Code != http.StatusOK {
		t.Fatalf("pointer event failed: %d %s", pointerRec.Code, pointerRec.Body.String())
	}

	reviewModeReq := httptest.NewRequest(http.MethodPost, "/api/session/review-mode", bytes.NewReader([]byte(`{"mode":"accessibility"}`)))
	reviewModeReq.Header.Set("Content-Type", "application/json")
	addAuth(reviewModeReq, cfg.ControlToken, true, "e2e-enh-review-mode")
	reviewModeRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(reviewModeRec, reviewModeReq)
	if reviewModeRec.Code != http.StatusOK {
		t.Fatalf("review mode failed: %d %s", reviewModeRec.Code, reviewModeRec.Body.String())
	}

	reviewReq := httptest.NewRequest(http.MethodPost, "/api/session/review-note", bytes.NewReader([]byte(`{"author":"alice","note":"Check keyboard access and contrast."}`)))
	reviewReq.Header.Set("Content-Type", "application/json")
	addAuth(reviewReq, cfg.ControlToken, true, "e2e-enh-review-note")
	reviewRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(reviewRec, reviewReq)
	if reviewRec.Code != http.StatusOK {
		t.Fatalf("review note failed: %d %s", reviewRec.Code, reviewRec.Body.String())
	}

	laserPath := `[{"x":100,"y":90,"t":"2026-03-09T15:00:00Z"},{"x":160,"y":130,"t":"2026-03-09T15:00:01Z"}]`
	noteFields := map[string]string{
		"review_mode":     "accessibility",
		"experiment_id":   "exp-copy-01",
		"variant":         "B",
		"laser_mode":      "1",
		"laser_path_json": laserPath,
	}
	noteBody, noteCT := multipartNoteBodyWithFields(t, "Increase contrast on the save button", tinyPNG(t), noteFields)
	noteReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", noteBody)
	noteReq.Header.Set("Content-Type", noteCT)
	addAuth(noteReq, cfg.ControlToken, true, "e2e-enh-note")
	noteRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(noteRec, noteReq)
	if noteRec.Code != http.StatusOK {
		t.Fatalf("feedback note failed: %d %s", noteRec.Code, noteRec.Body.String())
	}
	var noteResp map[string]any
	if err := json.Unmarshal(noteRec.Body.Bytes(), &noteResp); err != nil {
		t.Fatalf("decode note response: %v", err)
	}
	sess, _ := noteResp["session"].(map[string]any)
	events, _ := sess["feedback"].([]any)
	if len(events) == 0 {
		t.Fatalf("expected at least one feedback event")
	}
	last, _ := events[len(events)-1].(map[string]any)
	if last["experiment_id"] != "exp-copy-01" {
		t.Fatalf("expected experiment_id on feedback event, got %#v", last["experiment_id"])
	}
	if last["variant"] != "B" {
		t.Fatalf("expected variant on feedback event, got %#v", last["variant"])
	}
	if got, _ := last["laser_mode"].(bool); !got {
		t.Fatalf("expected laser_mode=true on feedback event")
	}
	laserSamples, _ := last["laser_path"].([]any)
	if len(laserSamples) != 2 {
		t.Fatalf("expected 2 laser path samples, got %d", len(laserSamples))
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, cfg.ControlToken, true, "e2e-enh-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}
	var approved map[string]any
	if err := json.Unmarshal(approveRec.Body.Bytes(), &approved); err != nil {
		t.Fatalf("decode approve response: %v", err)
	}
	meta, _ := approved["session_meta"].(map[string]any)
	if meta["review_mode"] != "accessibility" {
		t.Fatalf("expected session_meta.review_mode=accessibility, got %#v", meta["review_mode"])
	}
	reqs, _ := approved["change_requests"].([]any)
	if len(reqs) != 1 {
		t.Fatalf("expected one change request, got %d", len(reqs))
	}
	change, _ := reqs[0].(map[string]any)
	if change["experiment_id"] != "exp-copy-01" {
		t.Fatalf("expected change request experiment_id, got %#v", change["experiment_id"])
	}
	if change["variant"] != "B" {
		t.Fatalf("expected change request variant, got %#v", change["variant"])
	}
	if change["category"] != "accessibility" {
		t.Fatalf("expected accessibility category, got %#v", change["category"])
	}
}

func queueOneSubmission(t *testing.T, srv *Server, token, noncePrefix, note string) string {
	t.Helper()
	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"`+note+`","normalized":"`+note+`","pointer_x":12,"pointer_y":16,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, token, true, noncePrefix+"-feedback")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/session/approve", bytes.NewReader([]byte(`{"summary":""}`)))
	approveReq.Header.Set("Content-Type", "application/json")
	addAuth(approveReq, token, true, noncePrefix+"-approve")
	approveRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", approveRec.Code, approveRec.Body.String())
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/session/submit", bytes.NewReader([]byte(`{"provider":"cli"}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	addAuth(submitReq, token, true, noncePrefix+"-submit")
	submitRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("submit failed: %d %s", submitRec.Code, submitRec.Body.String())
	}
	var submitPayload map[string]any
	if err := json.Unmarshal(submitRec.Body.Bytes(), &submitPayload); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	attemptID, _ := submitPayload["attempt_id"].(string)
	if attemptID == "" {
		t.Fatalf("expected attempt id")
	}
	return attemptID
}

func slowCLIAdapterCommand() string {
	if runtime.GOOS == "windows" {
		return `powershell -Command "Start-Sleep -Milliseconds 800; Write-Output '{\"run_id\":\"queue-run\",\"status\":\"accepted\",\"ref\":\"/tmp/queue.log\"}'"`
	}
	return `sleep 0.8; printf '{"run_id":"queue-run","status":"accepted","ref":"/tmp/queue.log"}'`
}

func multipartNoteBody(t *testing.T, transcript string, screenshot []byte) (*bytes.Buffer, string) {
	return multipartNoteBodyWithFields(t, transcript, screenshot, nil)
}

func multipartNoteBodyWithFields(t *testing.T, transcript string, screenshot []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("raw_transcript", transcript); err != nil {
		t.Fatalf("write raw_transcript: %v", err)
	}
	if err := writer.WriteField("normalized", transcript); err != nil {
		t.Fatalf("write normalized: %v", err)
	}
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write field %s: %v", k, err)
		}
	}
	fw, err := writer.CreateFormFile("screenshot", "frame.png")
	if err != nil {
		t.Fatalf("create screenshot part: %v", err)
	}
	if _, err := fw.Write(screenshot); err != nil {
		t.Fatalf("write screenshot: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	return &body, writer.FormDataContentType()
}

func multipartClipBody(t *testing.T, eventID string, clip []byte) (*bytes.Buffer, string) {
	return multipartClipBodyWithFields(t, eventID, clip, nil)
}

func multipartClipBodyWithFields(t *testing.T, eventID string, clip []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("event_id", eventID); err != nil {
		t.Fatalf("write event_id: %v", err)
	}
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write field %s: %v", k, err)
		}
	}
	fw, err := writer.CreateFormFile("clip", "event.webm")
	if err != nil {
		t.Fatalf("create clip part: %v", err)
	}
	if _, err := fw.Write(clip); err != nil {
		t.Fatalf("write clip: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close clip multipart: %v", err)
	}
	return &body, writer.FormDataContentType()
}

func fetchE2EState(t *testing.T, srv *Server, token string) map[string]any {
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
	return payload
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	encoded := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO6L8WQAAAAASUVORK5CYII="
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode tiny png: %v", err)
	}
	return b
}

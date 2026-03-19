package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"knit/internal/config"
)

func TestParseVoiceCommand(t *testing.T) {
	cases := map[string]string{
		"start session":     voiceCommandStartSession,
		"pause capture":     voiceCommandPauseCapture,
		"capture note":      voiceCommandCaptureNote,
		"freeze screen":     voiceCommandFreezeScreen,
		"submit feedback":   voiceCommandSubmitFeedback,
		"discard last note": voiceCommandDiscardLast,
		"unknown request":   "",
	}
	for in, want := range cases {
		if got := parseVoiceCommand(in); got != want {
			t.Fatalf("parseVoiceCommand(%q)=%q want %q", in, got, want)
		}
	}
}

func TestVoiceCommandDiscardLastNoteViaFeedbackEndpoint(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-voice")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"normal note","normalized":"normal note","pointer_x":1,"pointer_y":1,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "nonce-feedback-voice")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}

	body, ct := voiceCommandMultipart(t, "discard last note")
	req := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", body)
	req.Header.Set("Content-Type", ct)
	addAuth(req, cfg.ControlToken, true, "nonce-voice-discard")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("voice discard failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode voice discard response: %v", err)
	}
	if handled, _ := payload["command_handled"].(bool); !handled {
		t.Fatalf("expected command_handled=true")
	}
	cmdResult, _ := payload["command_result"].(map[string]any)
	sessionObj, _ := cmdResult["session"].(map[string]any)
	feedback, _ := sessionObj["feedback"].([]any)
	if len(feedback) != 0 {
		t.Fatalf("expected no feedback after discard command, got %d", len(feedback))
	}
}

func TestVoiceCommandSubmitFeedbackQueuesAttempt(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"voice-submit","status":"accepted","ref":"/tmp/voice-submit.log"}'`)
	t.Setenv("KNIT_VOICE_COMMAND_PROVIDER", "cli")

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-voice-submit")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	feedbackReq := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader([]byte(`{"raw_transcript":"normal note","normalized":"normal note","pointer_x":1,"pointer_y":1,"window":"Browser Preview"}`)))
	feedbackReq.Header.Set("Content-Type", "application/json")
	addAuth(feedbackReq, cfg.ControlToken, true, "nonce-feedback-voice-submit")
	feedbackRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(feedbackRec, feedbackReq)
	if feedbackRec.Code != http.StatusOK {
		t.Fatalf("feedback failed: %d %s", feedbackRec.Code, feedbackRec.Body.String())
	}

	body, ct := voiceCommandMultipart(t, "submit feedback")
	req := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", body)
	req.Header.Set("Content-Type", ct)
	addAuth(req, cfg.ControlToken, true, "nonce-voice-submit")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("voice submit failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode voice submit response: %v", err)
	}
	cmdResult, _ := payload["command_result"].(map[string]any)
	if got := cmdResult["voice_command"]; got != voiceCommandSubmitFeedback {
		t.Fatalf("expected submit feedback voice command, got %#v", got)
	}
	if attemptID, _ := cmdResult["attempt_id"].(string); attemptID == "" {
		t.Fatalf("expected attempt_id from voice submit command")
	}
}

func TestVoiceCommandLifecycleAndFreezeCommands(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	startReq := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader([]byte(`{"target_window":"Browser Preview","target_url":"https://example.com/app"}`)))
	startReq.Header.Set("Content-Type", "application/json")
	addAuth(startReq, cfg.ControlToken, true, "nonce-start-voice-lifecycle")
	startRec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start failed: %d %s", startRec.Code, startRec.Body.String())
	}

	body, ct := voiceCommandMultipart(t, "pause capture")
	req := httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", body)
	req.Header.Set("Content-Type", ct)
	addAuth(req, cfg.ControlToken, true, "nonce-voice-pause")
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("voice pause failed: %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode voice pause response: %v", err)
	}
	cmdResult, _ := payload["command_result"].(map[string]any)
	if got := cmdResult["voice_command"]; got != voiceCommandPauseCapture {
		t.Fatalf("expected pause voice command, got %#v", got)
	}
	if state := fetchE2EState(t, srv, cfg.ControlToken); state["capture_state"] != "paused" {
		t.Fatalf("expected paused capture state after voice pause, got %#v", state["capture_state"])
	}

	body, ct = voiceCommandMultipart(t, "start session")
	req = httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", body)
	req.Header.Set("Content-Type", ct)
	addAuth(req, cfg.ControlToken, true, "nonce-voice-start")
	rec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("voice start failed: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode voice start response: %v", err)
	}
	cmdResult, _ = payload["command_result"].(map[string]any)
	if got := cmdResult["voice_command"]; got != voiceCommandStartSession {
		t.Fatalf("expected start voice command, got %#v", got)
	}
	if state := fetchE2EState(t, srv, cfg.ControlToken); state["capture_state"] != "active" {
		t.Fatalf("expected active capture state after voice start, got %#v", state["capture_state"])
	}

	body, ct = voiceCommandMultipart(t, "capture note")
	req = httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", body)
	req.Header.Set("Content-Type", ct)
	addAuth(req, cfg.ControlToken, true, "nonce-voice-capture-note")
	rec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("voice capture-note failed: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode voice capture-note response: %v", err)
	}
	cmdResult, _ = payload["command_result"].(map[string]any)
	if got := cmdResult["voice_command"]; got != voiceCommandCaptureNote {
		t.Fatalf("expected capture-note voice command, got %#v", got)
	}
	sessionObj, _ := cmdResult["session"].(map[string]any)
	feedback, _ := sessionObj["feedback"].([]any)
	if len(feedback) != 0 {
		t.Fatalf("expected capture-note command to avoid creating feedback, got %d events", len(feedback))
	}

	body, ct = voiceCommandMultipart(t, "freeze screen")
	req = httptest.NewRequest(http.MethodPost, "/api/session/feedback/note", body)
	req.Header.Set("Content-Type", ct)
	addAuth(req, cfg.ControlToken, true, "nonce-voice-freeze")
	rec = httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("voice freeze failed: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode voice freeze response: %v", err)
	}
	cmdResult, _ = payload["command_result"].(map[string]any)
	if got := cmdResult["voice_command"]; got != voiceCommandFreezeScreen {
		t.Fatalf("expected freeze voice command, got %#v", got)
	}
	if freeze, _ := cmdResult["freeze_screen"].(bool); !freeze {
		t.Fatalf("expected freeze_screen=true in voice command result")
	}
}

func voiceCommandMultipart(t *testing.T, transcript string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("raw_transcript", transcript); err != nil {
		t.Fatalf("write raw transcript: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	return &body, writer.FormDataContentType()
}

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"knit/internal/config"
	"knit/internal/session"
)

func TestDesktopHTTPApprovedDeliveryLifecycle(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "desktop-test-token"
	srv := newTestServer(t, cfg)
	workspace := t.TempDir()
	srv.runtimeMu.Lock()
	srv.runtime.RuntimeCodex.CodexWorkdir = workspace
	srv.runtimeMu.Unlock()
	sequence := 0
	post := func(path string, body any, want int) map[string]any {
		t.Helper()
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
		sequence++
		addAuth(req, cfg.ControlToken, true, fmt.Sprintf("desktop-%d", sequence))
		rec := httptest.NewRecorder()
		srv.httpSrv.Handler.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s: got %d want %d: %s", path, rec.Code, want, rec.Body.String())
		}
		var data map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	connected := post("/api/desktop/panel", map[string]any{"action": "connect", "label": "Shop fixes", "workspace": workspace}, 200)
	binding := connected["result"].(map[string]any)["id"].(string)
	submit := map[string]any{"binding_id": binding, "request_id": "browser-request"}
	post("/api/desktop/submit", submit, 412)
	srv.sessions.Start("Shop", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{ID: "note", RawTranscript: "Fix checkout spacing", NormalizedText: "Fix checkout spacing"}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.sessions.Approve("Fix checkout spacing"); err != nil {
		t.Fatal(err)
	}
	wrong := post("/api/desktop/panel", map[string]any{"action": "connect", "label": "Wrong project", "workspace": t.TempDir()}, 200)["result"].(map[string]any)["id"].(string)
	post("/api/desktop/submit", map[string]any{"binding_id": wrong, "request_id": "wrong-workspace"}, 409)
	if _, err := srv.sessions.ApprovedPackage(); err != nil {
		t.Fatal("workspace rejection consumed approved feedback")
	}
	queued := post("/api/desktop/submit", submit, 202)
	id := queued["delivery_id"].(string)
	repeated := post("/api/desktop/submit", submit, 200)
	if repeated["delivery_id"] != id {
		t.Fatal("retry duplicated delivery")
	}
	if len(srv.submitAttempts) != 0 {
		t.Fatal("desktop request entered direct-exec queue")
	}
	prepared := post("/api/desktop/panel", map[string]any{"action": "prepare", "binding_id": binding, "delivery_id": id}, 200)
	delivery := prepared["result"].(map[string]any)
	if delivery["workspace"] != workspace || delivery["package"] == nil {
		t.Fatal("missing approved payload or workspace")
	}
	post("/api/desktop/panel", map[string]any{"action": "prepare", "binding_id": binding, "delivery_id": id}, 409)
	for _, status := range []string{"working", "delivered", "completed"} {
		post("/api/desktop/panel", map[string]any{"action": "report", "binding_id": binding, "delivery_id": id, "status": status, "result": "Fixed spacing and tested."}, 200)
	}
	state := srv.desktopPublicState()
	encoded, _ := json.Marshal(state)
	if strings.Contains(string(encoded), "\"package\"") || strings.Contains(string(encoded), cfg.ControlToken) {
		t.Fatal("public status leaked payload or secret")
	}
	if state["deliveries"].([]map[string]any)[0]["status"] != "completed" {
		t.Fatal("completion not reflected in public status")
	}
	post("/api/desktop/panel", map[string]any{"action": "disconnect", "binding_id": binding}, 200)
	post("/api/desktop/panel", map[string]any{"action": "heartbeat", "binding_id": binding}, 409)
	t.Run("missing media is blocked without losing approval", func(t *testing.T) {
		if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{ID: "media-note", RawTranscript: "Check this screenshot", NormalizedText: "Check this screenshot", ScreenshotRef: "encrypted-screenshot-ref"}); err != nil {
			t.Fatal(err)
		}
		if _, err := srv.sessions.Approve("Check screenshot"); err != nil {
			t.Fatal(err)
		}
		response := post("/api/desktop/submit", map[string]any{"binding_id": wrong, "request_id": "media-request"}, 409)
		encoded, _ := json.Marshal(response)
		if !strings.Contains(string(encoded), "desktop_submission_blocked") || !strings.Contains(string(encoded), "no longer available") {
			t.Fatal("missing explicit missing-capture failure")
		}
		if _, err := srv.sessions.ApprovedPackage(); err != nil {
			t.Fatal("unsupported media consumed approval")
		}
	})
}

func TestDesktopHTTPRequiresAuthentication(t *testing.T) {
	srv := newTestServer(t, config.Default())
	for _, path := range []string{"/api/desktop", "/api/desktop/panel", "/api/desktop/submit"} {
		method := http.MethodPost
		if path == "/api/desktop" {
			method = http.MethodGet
		}
		req := httptest.NewRequest(method, path, strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		srv.httpSrv.Handler.ServeHTTP(rec, req)
		if rec.Code != 401 {
			t.Fatalf("%s unauthenticated: %d", path, rec.Code)
		}
	}
}

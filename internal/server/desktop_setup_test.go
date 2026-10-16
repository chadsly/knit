package server

import (
	"bytes"
	"encoding/json"
	"knit/internal/config"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestDesktopWorkspaceOnboardingPreservesExistingSettings(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "setup-test"
	cfg.UserConfigPath = filepath.Join(t.TempDir(), "knit.toml")
	srv := newTestServer(t, cfg)
	workspace := t.TempDir()
	connect := func(folder, nonce string, want int) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"action": "connect", "label": "Shop", "workspace": folder, "configure_workspace": true})
		req := httptest.NewRequest(http.MethodPost, "/api/desktop/panel", bytes.NewReader(body))
		addAuth(req, cfg.ControlToken, true, nonce)
		rec := httptest.NewRecorder()
		srv.httpSrv.Handler.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("got %d want %d: %s", rec.Code, want, rec.Body.String())
		}
	}
	original := srv.currentRuntimeCodex()
	connect(workspace, "fresh", 200)
	if srv.currentRuntimeCodex().CodexWorkdir != workspace {
		t.Fatal("workspace not configured")
	}
	if srv.currentRuntimeCodex().CodexSandbox != original.CodexSandbox {
		t.Fatal("Codex permissions changed")
	}
	saved, err := srv.store.LoadOperatorState()
	if err != nil || saved.RuntimeCodex.CodexWorkdir != workspace {
		t.Fatal("workspace not persisted", err)
	}
	if srv.sessions.Current() != nil {
		t.Fatal("panel connect started capture without browser confirmation")
	}
	connect(t.TempDir(), "different", 409)
	connect(filepath.Join(workspace, "missing"), "missing", 409)
	if srv.currentRuntimeCodex().CodexWorkdir != workspace {
		t.Fatal("existing workspace overwritten")
	}
	srv.updateConfig(func() config.Config { locked := srv.currentConfig(); locked.ConfigLocked = true; return locked }())
	connect(workspace, "locked", 409)
}

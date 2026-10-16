package server

import (
	"knit/internal/config"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDesktopBrowserWorkflow(t *testing.T) {
	if os.Getenv("KNIT_DESKTOP_PLAYWRIGHT") != "1" {
		t.Skip("set KNIT_DESKTOP_PLAYWRIGHT=1 with Playwright and Chrome installed")
	}
	cfg := config.Default()
	cfg.ControlToken = "isolated-desktop-browser-test"
	cfg.UserConfigPath = filepath.Join(t.TempDir(), "knit.toml")
	srv := newTestServer(t, cfg)
	workspace := t.TempDir()
	httpServer := httptest.NewServer(srv.httpSrv.Handler)
	defer httpServer.Close()
	script, err := filepath.Abs("../desktopui/verify.cjs")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", script)
	cmd.Env = append(os.Environ(), "KNIT_TEST_URL="+httpServer.URL, "KNIT_TEST_TOKEN="+cfg.ControlToken, "KNIT_TEST_WORKSPACE="+workspace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("browser workflow: %v\n%s", err, output)
	}
	t.Log(string(output))
	if len(srv.submitAttempts) != 0 {
		t.Fatal("desktop workflow launched a CLI attempt")
	}
}

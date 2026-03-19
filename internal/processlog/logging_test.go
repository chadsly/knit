package processlog

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupUsesProvidedDataDir(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "runtime")
	cleanup, logPath, err := Setup("daemon", dataDir)
	if err != nil {
		t.Fatalf("setup logging: %v", err)
	}
	log.Printf("hello from test")
	cleanup()

	if want := filepath.Join(dataDir, "daemon.log"); logPath != want {
		t.Fatalf("expected log path %q, got %q", want, logPath)
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	body := string(b)
	if !strings.Contains(body, "daemon logging initialized") {
		t.Fatalf("expected init message, got %q", body)
	}
	if !strings.Contains(body, "hello from test") {
		t.Fatalf("expected written log entry, got %q", body)
	}
}

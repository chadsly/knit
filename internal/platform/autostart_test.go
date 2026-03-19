package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutoStartManagerEnableDisableWritesLinuxEntry(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "knit-tray")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("write exec: %v", err)
	}
	manager := NewAutoStartManagerForTest("linux", "Knit", execPath, []string{"--tray"}, dir, filepath.Join(dir, ".config"), filepath.Join(dir, "AppData"))

	status, err := manager.Ensure(true)
	if err != nil {
		t.Fatalf("enable auto-start: %v", err)
	}
	if !status.Registered || !status.Enabled {
		t.Fatalf("expected auto-start registered status, got %#v", status)
	}
	body, err := os.ReadFile(status.EntryPath)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if !strings.Contains(string(body), "Exec=\""+execPath+"\" \"--tray\"") {
		t.Fatalf("expected desktop entry to reference executable and args, got %q", string(body))
	}

	status, err = manager.Ensure(false)
	if err != nil {
		t.Fatalf("disable auto-start: %v", err)
	}
	if status.Registered {
		t.Fatalf("expected disabled auto-start to be unregistered, got %#v", status)
	}
	if _, err := os.Stat(filepath.Join(dir, ".config", "autostart", "knit.desktop")); !os.IsNotExist(err) {
		t.Fatalf("expected auto-start entry removed, stat err=%v", err)
	}
}

func TestAutoStartManagerRejectsEphemeralGoRunBinary(t *testing.T) {
	dir := t.TempDir()
	manager := NewAutoStartManagerForTest("linux", "Knit", filepath.Join(dir, "go-build123", "b001", "exe", "tray"), nil, dir, filepath.Join(dir, ".config"), filepath.Join(dir, "AppData"))
	if _, err := manager.Ensure(true); err == nil {
		t.Fatalf("expected go-run style binary path to be rejected")
	}
}

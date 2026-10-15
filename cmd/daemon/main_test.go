package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"knit/internal/config"
)

func TestStartupBannerLinesIncludeWelcomeVersionAndURL(t *testing.T) {
	cfg := config.Default()
	cfg.BuildID = "build-123"
	cfg.HTTPListenAddr = "127.0.0.1:8888"

	lines := startupBannerLines(cfg)
	if len(lines) != 7 {
		t.Fatalf("expected 7 startup lines, got %d: %#v", len(lines), lines)
	}
	if lines[0] != "Welcome to Knit." {
		t.Fatalf("unexpected welcome line: %q", lines[0])
	}
	if lines[1] != "Version: build-123" {
		t.Fatalf("unexpected version line: %q", lines[1])
	}
	if lines[2] != "Open the local UI: http://127.0.0.1:8888" {
		t.Fatalf("unexpected UI line: %q", lines[2])
	}
	if lines[3] != "Browser extension: Knit Browser Composer" {
		t.Fatalf("unexpected extension line: %q", lines[3])
	}
	if lines[4] != "Chrome Web Store: "+chromeWebStoreURL() {
		t.Fatalf("unexpected Chrome Web Store line: %q", lines[4])
	}
	if lines[5] != "Local install: chrome://extensions -> Load unpacked -> extension/chromium" {
		t.Fatalf("unexpected local install line: %q", lines[5])
	}
	if lines[6] != "Pair from the UI: Capture, review, and send -> Chrome Extension" {
		t.Fatalf("unexpected pairing line: %q", lines[6])
	}
}

func TestStartupUIURLUsesLoopbackForWildcardListenAddr(t *testing.T) {
	cfg := config.Default()
	cfg.HTTPListenAddr = "0.0.0.0:7777"

	if got := startupUIURL(cfg); got != "http://127.0.0.1:7777" {
		t.Fatalf("expected wildcard UI URL to use loopback, got %q", got)
	}
}

func TestStartupVersionFallsBackToVersionPinThenDev(t *testing.T) {
	cfg := config.Default()
	cfg.VersionPin = "v0.9.0"
	if got := startupVersion(cfg); got != "v0.9.0" {
		t.Fatalf("expected version pin fallback, got %q", got)
	}
	cfg.VersionPin = ""
	if got := startupVersion(cfg); got != "dev" {
		t.Fatalf("expected dev fallback, got %q", got)
	}
}

func TestMCPStartupBannerWritesManagedTokenToStderrOnly(t *testing.T) {
	cfg := config.Default()
	cfg.HTTPListenAddr = "127.0.0.1:7777"

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW
	t.Cleanup(func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	printMCPStartupBanner(cfg, true, map[string]any{"token": "managed-token-123"})

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	stdout, err := io.ReadAll(stdoutR)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderr, err := io.ReadAll(stderrR)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if strings.TrimSpace(string(stdout)) != "" {
		t.Fatalf("expected no MCP startup banner output on stdout, got %q", string(stdout))
	}
	text := string(stderr)
	if !strings.Contains(text, "Knit MCP connected.") || !strings.Contains(text, "Chrome extension token: managed-token-123") {
		t.Fatalf("expected stderr banner to include token setup text, got %q", text)
	}
}

func TestCompatibleDaemonForMCPAcceptsManagedWorkerDaemon(t *testing.T) {
	ok, reason := compatibleDaemonForMCP(map[string]any{
		"daemon_instance_mode":   daemonInstanceModeManaged,
		"default_execution_mode": executionModeWorker,
		"mcp_attach_ready":       true,
	})
	if !ok || reason != "" {
		t.Fatalf("expected compatible daemon instance, got ok=%v reason=%q", ok, reason)
	}
}

func TestCompatibleDaemonForMCPRejectsNormalDaemon(t *testing.T) {
	ok, reason := compatibleDaemonForMCP(map[string]any{
		"daemon_instance_mode":   "normal",
		"default_execution_mode": "direct_exec",
		"mcp_attach_ready":       false,
	})
	if ok {
		t.Fatalf("expected normal daemon instance to be incompatible")
	}
	if !strings.Contains(reason, "normal mode") {
		t.Fatalf("expected normal-mode incompatibility reason, got %q", reason)
	}
}

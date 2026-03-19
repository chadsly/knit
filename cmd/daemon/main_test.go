package main

import (
	"testing"

	"knit/internal/config"
)

func TestStartupBannerLinesIncludeWelcomeVersionAndURL(t *testing.T) {
	cfg := config.Default()
	cfg.BuildID = "build-123"
	cfg.HTTPListenAddr = "127.0.0.1:8888"

	lines := startupBannerLines(cfg)
	if len(lines) != 3 {
		t.Fatalf("expected 3 startup lines, got %d: %#v", len(lines), lines)
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

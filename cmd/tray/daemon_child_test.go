package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDaemonChildCommandIncludesServerOnlyEnv(t *testing.T) {
	cmd, err := daemonChildCommand()
	if err != nil {
		t.Fatalf("daemonChildCommand error: %v", err)
	}
	found := false
	for _, env := range cmd.Env {
		if env == trayServerOnlyEnv+"=1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("daemon child env missing %s=1", trayServerOnlyEnv)
	}
	if got := strings.TrimSpace(cmd.Path); got == "" {
		t.Fatal("daemon child command path empty")
	}
}

func TestProbeServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	if !probeServer(addr, time.Second) {
		t.Fatal("probeServer should succeed for healthy server")
	}
	if probeServer("127.0.0.1:1", 100*time.Millisecond) {
		t.Fatal("probeServer should fail for unavailable server")
	}
}

func TestTrayDataDirUsesDefaultConfig(t *testing.T) {
	old := os.Getenv("KNIT_DATA_DIR")
	defer os.Setenv("KNIT_DATA_DIR", old)
	if err := os.Setenv("KNIT_DATA_DIR", "./test-tray-data"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if got := trayDataDir(); got != "./test-tray-data" {
		t.Fatalf("trayDataDir=%q want %q", got, "./test-tray-data")
	}
}

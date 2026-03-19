package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"knit/internal/app"
)

const trayServerOnlyEnv = "KNIT_TRAY_SERVER_ONLY"

func runServerOnly() {
	cfg := app.DefaultConfig()
	cleanupLog, logPath, err := setupChildDaemonLogging(cfg.DataDir)
	if err != nil {
		log.Printf("daemon logging setup failed: %v", err)
	} else {
		defer cleanupLog()
		log.Printf("daemon effective data dir: %s", cfg.DataDir)
		log.Printf("daemon runtime log path: %s", logPath)
	}
	defer recoverTrayChildPanic()

	a, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	if err := a.Run(context.Background()); err != nil {
		log.Fatalf("daemon stopped with error: %v", err)
	}
}

func ensureServerRunning(addr string) error {
	if probeServer(addr, 2*time.Second) {
		return nil
	}
	cmd, err := daemonChildCommand()
	if err != nil {
		return err
	}
	if err := startDetachedProcess(cmd); err != nil {
		return fmt.Errorf("start detached daemon child: %w", err)
	}
	if !waitForServer(addr, 12*time.Second) {
		return fmt.Errorf("daemon child did not become healthy at http://%s/healthz", addr)
	}
	return nil
}

func daemonChildCommand() (*exec.Cmd, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve tray executable: %w", err)
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), trayServerOnlyEnv+"=1")
	return cmd, nil
}

func probeServer(addr string, timeout time.Duration) bool {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return false
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get("http://" + addr + "/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func waitForServer(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if probeServer(addr, 2*time.Second) {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return probeServer(addr, 2*time.Second)
}

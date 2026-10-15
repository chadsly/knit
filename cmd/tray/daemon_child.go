package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
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
	if app.ProbeDaemon(addr, 2*time.Second) {
		return nil
	}
	cmd, err := daemonChildCommand()
	if err != nil {
		return err
	}
	if err := startDetachedProcess(cmd); err != nil {
		return fmt.Errorf("start detached daemon child: %w", err)
	}
	if !app.WaitForDaemon(addr, 12*time.Second) {
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
	return app.ProbeDaemon(addr, timeout)
}

func waitForServer(addr string, timeout time.Duration) bool {
	return app.WaitForDaemon(addr, timeout)
}

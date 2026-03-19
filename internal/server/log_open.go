package server

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var openLocalPath = openPathNative

func resolveLocalLogPath(ref string) (string, error) {
	path := strings.TrimSpace(ref)
	if path == "" {
		return "", fmt.Errorf("no submission log reference found")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("submission reference is not a local absolute path")
	}
	clean := filepath.Clean(path)
	base := filepath.Base(clean)
	if !looksLikeLocalAttemptLogBase(base) {
		return "", fmt.Errorf("last submission reference is not a local codex log file")
	}
	info, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("submission log is not accessible: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("submission log reference points to a directory")
	}
	return clean, nil
}

func looksLikeLocalAttemptLogBase(base string) bool {
	base = strings.TrimSpace(base)
	if !strings.HasPrefix(base, "knit-codex-") {
		return false
	}
	return strings.Contains(base, ".log")
}

func openPathNative(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", "", path)
	default:
		if _, err := exec.LookPath("xdg-open"); err != nil {
			return fmt.Errorf("xdg-open is unavailable: %w", err)
		}
		cmd = exec.Command("xdg-open", path)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

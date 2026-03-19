package server

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func pickDirectoryNative() (string, error) {
	if override := strings.TrimSpace(os.Getenv("KNIT_PICKDIR_OVERRIDE")); override != "" {
		return normalizePickedPath(override)
	}
	switch runtime.GOOS {
	case "darwin":
		return runPickDirCommand("osascript", "-e", `POSIX path of (choose folder with prompt "Select Codex workspace directory")`)
	case "windows":
		script := `Add-Type -AssemblyName System.Windows.Forms; $dialog = New-Object System.Windows.Forms.FolderBrowserDialog; $dialog.Description = 'Select Codex workspace directory'; if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { Write-Output $dialog.SelectedPath }`
		return runPickDirCommand("powershell", "-NoProfile", "-Command", script)
	default:
		if _, err := exec.LookPath("zenity"); err == nil {
			return runPickDirCommand("zenity", "--file-selection", "--directory", "--title=Select Codex workspace directory")
		}
		if _, err := exec.LookPath("kdialog"); err == nil {
			return runPickDirCommand("kdialog", "--getexistingdirectory", ".", "Select Codex workspace directory")
		}
		return "", fmt.Errorf("native directory picker is unavailable on this system")
	}
}

func runPickDirCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if strings.Contains(strings.ToLower(text), "cancel") || strings.Contains(strings.ToLower(err.Error()), "cancel") {
			return "", fmt.Errorf("folder selection canceled")
		}
		if text == "" {
			text = err.Error()
		}
		return "", fmt.Errorf("directory picker failed: %s", text)
	}
	if text == "" {
		return "", fmt.Errorf("folder selection canceled")
	}
	return normalizePickedPath(text)
}

func normalizePickedPath(raw string) (string, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return "", fmt.Errorf("empty directory path")
	}
	clean = strings.TrimSuffix(clean, string(filepath.Separator))
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("invalid directory path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("selected path is not accessible: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("selected path is not a directory")
	}
	return abs, nil
}

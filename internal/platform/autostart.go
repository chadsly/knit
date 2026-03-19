package platform

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type AutoStartStatus struct {
	Enabled    bool   `json:"enabled"`
	Supported  bool   `json:"supported"`
	Registered bool   `json:"registered"`
	Backend    string `json:"backend,omitempty"`
	EntryPath  string `json:"entry_path,omitempty"`
	Message    string `json:"message,omitempty"`
}

type AutoStartManager struct {
	mu         sync.RWMutex
	goos       string
	appName    string
	execPath   string
	args       []string
	homeDir    string
	configHome string
	appDataDir string
	status     AutoStartStatus
}

func NewAutoStartManager(appName, execPath string, args []string) *AutoStartManager {
	homeDir, _ := os.UserHomeDir()
	configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if configHome == "" && homeDir != "" {
		configHome = filepath.Join(homeDir, ".config")
	}
	appDataDir := strings.TrimSpace(os.Getenv("APPDATA"))
	if appDataDir == "" && homeDir != "" {
		appDataDir = filepath.Join(homeDir, "AppData", "Roaming")
	}
	return NewAutoStartManagerForTest(runtime.GOOS, appName, execPath, args, homeDir, configHome, appDataDir)
}

func NewAutoStartManagerForTest(goos, appName, execPath string, args []string, homeDir, configHome, appDataDir string) *AutoStartManager {
	return &AutoStartManager{
		goos:       strings.TrimSpace(goos),
		appName:    strings.TrimSpace(appName),
		execPath:   strings.TrimSpace(execPath),
		args:       append([]string(nil), args...),
		homeDir:    strings.TrimSpace(homeDir),
		configHome: strings.TrimSpace(configHome),
		appDataDir: strings.TrimSpace(appDataDir),
		status: AutoStartStatus{
			Enabled:   false,
			Supported: autoStartSupported(goos),
			Backend:   autoStartBackend(goos),
		},
	}
}

func (m *AutoStartManager) Status() AutoStartStatus {
	if m == nil {
		return AutoStartStatus{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *AutoStartManager) Ensure(enabled bool) (AutoStartStatus, error) {
	if m == nil {
		return AutoStartStatus{}, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	status := AutoStartStatus{
		Enabled:   enabled,
		Supported: autoStartSupported(m.goos),
		Backend:   autoStartBackend(m.goos),
	}
	if !status.Supported {
		status.Message = "auto-start is not supported on this operating system"
		m.status = status
		if enabled {
			return status, fmt.Errorf("%s", status.Message)
		}
		return status, nil
	}

	entryPath, err := m.entryPath()
	if err != nil {
		status.Message = err.Error()
		m.status = status
		return status, err
	}
	status.EntryPath = entryPath

	if !enabled {
		if removeErr := os.Remove(entryPath); removeErr != nil && !os.IsNotExist(removeErr) {
			status.Message = removeErr.Error()
			m.status = status
			return status, fmt.Errorf("remove auto-start entry: %w", removeErr)
		}
		status.Message = "auto-start disabled"
		m.status = status
		return status, nil
	}

	if isEphemeralExecutable(m.execPath) {
		status.Message = "auto-start requires a stable built executable path; go run build-cache binaries are not allowed"
		m.status = status
		return status, fmt.Errorf("%s", status.Message)
	}

	payload, err := m.renderEntry()
	if err != nil {
		status.Message = err.Error()
		m.status = status
		return status, err
	}
	defer zeroBuffer(payload)

	if err := os.MkdirAll(filepath.Dir(entryPath), 0o700); err != nil {
		status.Message = err.Error()
		m.status = status
		return status, fmt.Errorf("create auto-start directory: %w", err)
	}
	if err := os.WriteFile(entryPath, payload, 0o600); err != nil {
		status.Message = err.Error()
		m.status = status
		return status, fmt.Errorf("write auto-start entry: %w", err)
	}
	status.Registered = true
	status.Message = "auto-start enabled"
	m.status = status
	return status, nil
}

func (m *AutoStartManager) entryPath() (string, error) {
	switch m.goos {
	case "darwin":
		if m.homeDir == "" {
			return "", fmt.Errorf("home directory is required for macOS auto-start")
		}
		return filepath.Join(m.homeDir, "Library", "LaunchAgents", "com.knit.autostart.plist"), nil
	case "linux":
		if m.configHome == "" {
			return "", fmt.Errorf("config home is required for Linux auto-start")
		}
		return filepath.Join(m.configHome, "autostart", "knit.desktop"), nil
	case "windows":
		if m.appDataDir == "" {
			return "", fmt.Errorf("APPDATA is required for Windows auto-start")
		}
		return filepath.Join(m.appDataDir, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "Knit.cmd"), nil
	default:
		return "", fmt.Errorf("unsupported auto-start platform: %s", m.goos)
	}
}

func (m *AutoStartManager) renderEntry() ([]byte, error) {
	switch m.goos {
	case "darwin":
		return renderLaunchAgentPlist(m.appName, m.execPath, m.args)
	case "linux":
		return renderDesktopEntry(m.appName, m.execPath, m.args), nil
	case "windows":
		return renderWindowsStartupScript(m.execPath, m.args), nil
	default:
		return nil, fmt.Errorf("unsupported auto-start platform: %s", m.goos)
	}
}

func autoStartSupported(goos string) bool {
	switch goos {
	case "darwin", "linux", "windows":
		return true
	default:
		return false
	}
}

func autoStartBackend(goos string) string {
	switch goos {
	case "darwin":
		return "launch_agents"
	case "linux":
		return "xdg_autostart"
	case "windows":
		return "startup_folder"
	default:
		return "unsupported"
	}
}

func isEphemeralExecutable(path string) bool {
	clean := filepath.ToSlash(strings.TrimSpace(path))
	return strings.Contains(clean, "/go-build") || strings.Contains(clean, "\\go-build")
}

func renderLaunchAgentPlist(appName, execPath string, args []string) ([]byte, error) {
	programArgs := append([]string{execPath}, args...)
	var out bytes.Buffer
	out.WriteString(xml.Header)
	out.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	out.WriteString(`<plist version="1.0">` + "\n")
	out.WriteString(`  <dict>` + "\n")
	out.WriteString(`    <key>Label</key>` + "\n")
	out.WriteString(`    <string>com.knit.autostart</string>` + "\n")
	out.WriteString(`    <key>RunAtLoad</key>` + "\n")
	out.WriteString(`    <true/>` + "\n")
	out.WriteString(`    <key>ProgramArguments</key>` + "\n")
	out.WriteString(`    <array>` + "\n")
	for _, arg := range programArgs {
		out.WriteString(`      <string>`)
		if err := xml.EscapeText(&out, []byte(arg)); err != nil {
			return nil, fmt.Errorf("escape launch agent arg: %w", err)
		}
		out.WriteString(`</string>` + "\n")
	}
	out.WriteString(`    </array>` + "\n")
	out.WriteString(`  </dict>` + "\n")
	out.WriteString(`</plist>`)
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func renderDesktopEntry(appName, execPath string, args []string) []byte {
	execLine := quoteDesktopExec(execPath, args)
	return []byte(strings.Join([]string{
		"[Desktop Entry]",
		"Type=Application",
		"Version=1.0",
		"Name=" + firstNonEmpty(appName, "Knit"),
		"Exec=" + execLine,
		"Terminal=false",
		"X-GNOME-Autostart-enabled=true",
		"",
	}, "\n"))
}

func renderWindowsStartupScript(execPath string, args []string) []byte {
	parts := make([]string, 0, 2+len(args))
	parts = append(parts, `@echo off`, `start "" "`+escapeWindowsArg(execPath)+`"`)
	if len(args) > 0 {
		command := parts[len(parts)-1]
		for _, arg := range args {
			command += ` "` + escapeWindowsArg(arg) + `"`
		}
		parts[len(parts)-1] = command
	}
	parts = append(parts, "")
	return []byte(strings.Join(parts, "\r\n"))
}

func quoteDesktopExec(execPath string, args []string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, quoteDesktopArg(execPath))
	for _, arg := range args {
		parts = append(parts, quoteDesktopArg(arg))
	}
	return strings.Join(parts, " ")
}

func quoteDesktopArg(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", "$", "\\$")
	return `"` + replacer.Replace(value) + `"`
}

func escapeWindowsArg(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func zeroBuffer(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

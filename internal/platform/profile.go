package platform

import (
	"os"
	"runtime"
	"strings"
)

type PermissionStatus string

const (
	PermissionStatusUnknown     PermissionStatus = "unknown"
	PermissionStatusRequired    PermissionStatus = "required"
	PermissionStatusUnavailable PermissionStatus = "unavailable"
)

type PermissionSpec struct {
	Name      string           `json:"name"`
	Status    PermissionStatus `json:"status"`
	Required  bool             `json:"required"`
	Prompt    string           `json:"prompt,omitempty"`
	Scope     string           `json:"scope,omitempty"`
	Notes     string           `json:"notes,omitempty"`
	Supported bool             `json:"supported"`
}

type PackagingSpec struct {
	Kind      string `json:"kind"`
	Format    string `json:"format"`
	Generated bool   `json:"generated"`
	Notes     string `json:"notes,omitempty"`
}

type Profile struct {
	GOOS                 string                `json:"goos"`
	DisplayName          string                `json:"display_name"`
	Supported            bool                  `json:"supported"`
	SecureStorageBackend string                `json:"secure_storage_backend"`
	LaunchMode           string                `json:"launch_mode"`
	BrowserFirst         bool                  `json:"browser_first"`
	Permissions          []PermissionSpec      `json:"permissions"`
	Packaging            []PackagingSpec       `json:"packaging"`
	NativeCaptureModules []NativeCaptureModule `json:"native_capture_modules"`
}

type RuntimeGuide struct {
	HostTarget     string `json:"host_target"`
	InstallerHint  string `json:"installer_hint"`
	RuntimeSummary string `json:"runtime_summary"`
	Supported      bool   `json:"supported"`
}

func CurrentProfile() Profile {
	return ProfileForOS(runtime.GOOS)
}

func CurrentRuntimeGuide() RuntimeGuide {
	return RuntimeGuideForOS(runtime.GOOS, runtime.GOARCH)
}

func RuntimeGuideForOS(goos, goarch string) RuntimeGuide {
	profile := ProfileForOS(goos)
	return RuntimeGuide{
		HostTarget:     strings.TrimSpace(goos) + "_" + strings.TrimSpace(goarch),
		InstallerHint:  installerHint(profile),
		RuntimeSummary: runtimeSummary(profile),
		Supported:      profile.Supported,
	}
}

func ProfileForOS(goos string) Profile {
	goos = strings.TrimSpace(strings.ToLower(goos))
	profile := Profile{
		GOOS:         goos,
		DisplayName:  strings.ToUpper(goos),
		Supported:    true,
		LaunchMode:   "native_tray_plus_local_web_ui",
		BrowserFirst: true,
	}

	switch goos {
	case "darwin":
		profile.DisplayName = "macOS"
		profile.SecureStorageBackend = "keychain"
		profile.Permissions = []PermissionSpec{
			permission("microphone", true, "System microphone permission prompt", "device", "Required for audio note capture on macOS."),
			permission("screen_recording", true, "System Screen Recording prompt", "window", "Required for screenshots and clips outside pure browser metadata."),
			permission("accessibility", false, "System Accessibility/Input Monitoring prompt", "desktop", "Needed only for future native pointer/input hooks."),
			permission("secure_storage", true, "OS Keychain access", "credential_store", "Used for the artifact encryption key."),
		}
		profile.Packaging = []PackagingSpec{
			{Kind: "installer", Format: "command", Generated: true, Notes: "Command-based installer wrapper is generated in release packaging."},
			{Kind: "portable", Format: "tar.gz", Generated: true},
		}
	case "windows":
		profile.DisplayName = "Windows"
		profile.SecureStorageBackend = "credential_manager"
		profile.Permissions = []PermissionSpec{
			permission("microphone", true, "Windows privacy prompt for microphone", "device", "Required for audio note capture."),
			permission("graphics_capture", true, "Windows Graphics Capture picker/consent", "window", "Required for screenshots and clips."),
			permission("accessibility", false, "Input monitoring / UI Automation consent", "desktop", "Needed only for future native pointer/input hooks."),
			permission("secure_storage", true, "Credential Manager access", "credential_store", "Used for the artifact encryption key."),
		}
		profile.Packaging = []PackagingSpec{
			{Kind: "installer", Format: "powershell", Generated: true, Notes: "PowerShell installer wrapper is generated in release packaging."},
			{Kind: "portable", Format: "zip", Generated: true},
		}
	case "linux":
		profile.DisplayName = "Linux"
		profile.SecureStorageBackend = "secret_service"
		profile.Permissions = []PermissionSpec{
			permission("microphone", true, "Desktop portal / PulseAudio / PipeWire prompt", "device", "Required for audio note capture."),
			permission("screen_capture", true, "Desktop portal / compositor picker", "window", "Required for screenshots and clips."),
			permission("accessibility", false, "Desktop-specific accessibility/input consent", "desktop", "Needed only for future native pointer/input hooks."),
			permission("secure_storage", true, "Secret Service / keyring access", "credential_store", "Used for the artifact encryption key."),
		}
		profile.Packaging = []PackagingSpec{
			{Kind: "portable", Format: "tar.gz", Generated: true},
			{Kind: "installer", Format: "shell", Generated: true, Notes: "Portable install script is generated in release packaging."},
		}
	default:
		profile.DisplayName = "Unsupported"
		profile.Supported = false
		profile.LaunchMode = "unsupported"
		profile.BrowserFirst = false
		profile.SecureStorageBackend = "unsupported"
		profile.Permissions = []PermissionSpec{
			{Name: "microphone", Status: PermissionStatusUnavailable, Required: true, Supported: false},
			{Name: "screen_capture", Status: PermissionStatusUnavailable, Required: true, Supported: false},
			{Name: "secure_storage", Status: PermissionStatusUnavailable, Required: true, Supported: false},
		}
		profile.Packaging = []PackagingSpec{
			{Kind: "portable", Format: "unsupported", Generated: false, Notes: "No packaging flow defined for this OS."},
		}
	}
	profile.NativeCaptureModules = NativeCaptureModulesForOS(goos)
	return applyPermissionEnvOverrides(profile)
}

func permission(name string, required bool, prompt, scope, notes string) PermissionSpec {
	return PermissionSpec{
		Name:      name,
		Status:    PermissionStatusRequired,
		Required:  required,
		Prompt:    prompt,
		Scope:     scope,
		Notes:     notes,
		Supported: true,
	}
}

func applyPermissionEnvOverrides(profile Profile) Profile {
	for i := range profile.Permissions {
		envKey := "KNIT_PERMISSION_" + strings.ToUpper(strings.ReplaceAll(profile.Permissions[i].Name, "-", "_")) + "_STATUS"
		switch strings.ToLower(strings.TrimSpace(os.Getenv(envKey))) {
		case "":
		case "required", "granted", "available":
			profile.Permissions[i].Status = PermissionStatusRequired
		case "unknown":
			profile.Permissions[i].Status = PermissionStatusUnknown
		case "unavailable", "denied", "blocked":
			profile.Permissions[i].Status = PermissionStatusUnavailable
		}
	}
	return profile
}

func installerHint(profile Profile) string {
	for _, packaging := range profile.Packaging {
		if packaging.Kind == "installer" {
			switch packaging.Format {
			case "command":
				return ".install.command"
			case "powershell":
				return ".install.ps1"
			case "shell":
				return ".install.sh"
			default:
				return packaging.Format
			}
		}
	}
	return "portable archive"
}

func runtimeSummary(profile Profile) string {
	if !profile.Supported {
		return "Unsupported runtime"
	}
	return profile.DisplayName + " runtime: browser-first review, local web UI, and " + installerHint(profile) + " packaging."
}

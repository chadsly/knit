package platform

import "runtime"

type NativeCaptureModule struct {
	Module    string `json:"module"`
	Backend   string `json:"backend"`
	Scope     string `json:"scope"`
	Status    string `json:"status"`
	Supported bool   `json:"supported"`
	Notes     string `json:"notes,omitempty"`
}

// NativeCaptureModules reports the OS-native capture backends the daemon can
// target beyond the browser-first path. Status is "planned" until a backend is
// fully wired into active capture flows.
func NativeCaptureModules() []NativeCaptureModule {
	return NativeCaptureModulesForOS(runtime.GOOS)
}

func NativeCaptureModulesForOS(goos string) []NativeCaptureModule {
	common := []NativeCaptureModule{
		{
			Module:    "active_window_metadata",
			Backend:   "os_window_manager",
			Scope:     "window",
			Status:    "abstracted",
			Supported: true,
			Notes:     "Stable interface boundary for OS window metadata.",
		},
		{
			Module:    "pointer_input",
			Backend:   "os_pointer_hooks",
			Scope:     "window",
			Status:    "abstracted",
			Supported: true,
			Notes:     "Browser companion remains default until native input hooks are wired.",
		},
	}

	switch goos {
	case "darwin":
		return append(common, NativeCaptureModule{
			Module:    "screen_capture",
			Backend:   "screen_capture_kit",
			Scope:     "window",
			Status:    "abstracted",
			Supported: true,
			Notes:     "Requires Screen Recording permission on macOS.",
		})
	case "windows":
		return append(common, NativeCaptureModule{
			Module:    "screen_capture",
			Backend:   "windows_graphics_capture",
			Scope:     "window",
			Status:    "abstracted",
			Supported: true,
			Notes:     "Requires Graphics Capture consent on Windows.",
		})
	case "linux":
		return append(common, NativeCaptureModule{
			Module:    "screen_capture",
			Backend:   "pipewire_x11_wayland",
			Scope:     "window",
			Status:    "abstracted",
			Supported: true,
			Notes:     "Backend choice depends on compositor/session.",
		})
	default:
		return append(common, NativeCaptureModule{
			Module:    "screen_capture",
			Backend:   "unsupported_os",
			Scope:     "window",
			Status:    "unsupported",
			Supported: false,
			Notes:     "No native capture backend defined for this OS.",
		})
	}
}

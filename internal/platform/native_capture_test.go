package platform

import "testing"

func TestNativeCaptureModulesIncludesScreenAndPointer(t *testing.T) {
	modules := NativeCaptureModules()
	if len(modules) == 0 {
		t.Fatalf("expected native capture modules")
	}
	var hasScreen bool
	var hasPointer bool
	for _, m := range modules {
		if m.Module == "screen_capture" {
			hasScreen = true
		}
		if m.Module == "pointer_input" {
			hasPointer = true
		}
	}
	if !hasScreen {
		t.Fatalf("expected screen_capture module")
	}
	if !hasPointer {
		t.Fatalf("expected pointer_input module")
	}
}

func TestNativeCaptureModulesForKnownOSAreAbstracted(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "windows"} {
		modules := NativeCaptureModulesForOS(goos)
		if len(modules) == 0 {
			t.Fatalf("expected modules for %s", goos)
		}
		for _, module := range modules {
			if module.Module == "screen_capture" && module.Status != "abstracted" {
				t.Fatalf("expected abstracted screen module for %s, got %#v", goos, module)
			}
		}
	}
}

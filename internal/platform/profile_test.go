package platform

import "testing"

func TestProfileForKnownOSIncludesPermissionModelAndPackaging(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "windows"} {
		profile := ProfileForOS(goos)
		if !profile.Supported {
			t.Fatalf("expected %s to be supported", goos)
		}
		if profile.SecureStorageBackend == "" || profile.LaunchMode == "" {
			t.Fatalf("expected secure storage and launch mode for %s, got %#v", goos, profile)
		}
		if len(profile.Permissions) < 3 {
			t.Fatalf("expected permission model entries for %s, got %#v", goos, profile.Permissions)
		}
		if len(profile.Packaging) == 0 {
			t.Fatalf("expected packaging entries for %s", goos)
		}
	}
}

func TestProfileForUnsupportedOSMarksUnsupported(t *testing.T) {
	profile := ProfileForOS("plan9")
	if profile.Supported {
		t.Fatalf("expected unsupported profile")
	}
	if profile.LaunchMode != "unsupported" {
		t.Fatalf("expected unsupported launch mode, got %q", profile.LaunchMode)
	}
}

func TestRuntimeGuideForKnownOSIncludesHostTargetAndSummary(t *testing.T) {
	guide := RuntimeGuideForOS("linux", "amd64")
	if guide.HostTarget != "linux_amd64" {
		t.Fatalf("expected linux_amd64 host target, got %q", guide.HostTarget)
	}
	if guide.InstallerHint == "" || guide.RuntimeSummary == "" {
		t.Fatalf("expected runtime guide hints, got %#v", guide)
	}
	if !guide.Supported {
		t.Fatalf("expected supported runtime guide")
	}
}

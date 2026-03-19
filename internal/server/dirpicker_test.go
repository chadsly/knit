package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPickDirectoryNativeUsesOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KNIT_PICKDIR_OVERRIDE", dir)
	picked, err := pickDirectoryNative()
	if err != nil {
		t.Fatalf("pick directory with override: %v", err)
	}
	if picked != dir {
		t.Fatalf("expected picked directory %q, got %q", dir, picked)
	}
}

func TestNormalizePickedPathRejectsFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := normalizePickedPath(file); err == nil {
		t.Fatalf("expected file path to be rejected")
	}
}

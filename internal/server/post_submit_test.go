package server

import (
	"runtime"
	"testing"
)

func TestRunPostSubmitAutomationDisabled(t *testing.T) {
	t.Setenv("KNIT_POST_SUBMIT_REBUILD_CMD", "")
	t.Setenv("KNIT_POST_SUBMIT_VERIFY_CMD", "")
	if got := runPostSubmitAutomation(); got != nil {
		t.Fatalf("expected nil result when disabled, got %#v", got)
	}
}

func TestRunPostSubmitAutomationSkipsVerifyWhenRebuildFails(t *testing.T) {
	t.Setenv("KNIT_POST_SUBMIT_REBUILD_CMD", failingCommand())
	t.Setenv("KNIT_POST_SUBMIT_VERIFY_CMD", successCommand("verified"))
	t.Setenv("KNIT_POST_SUBMIT_TIMEOUT_SECONDS", "20")

	got := runPostSubmitAutomation()
	if got == nil {
		t.Fatalf("expected automation result")
	}
	if got.Rebuild == nil || got.Rebuild.Status != "failed" {
		t.Fatalf("expected rebuild failed, got %#v", got.Rebuild)
	}
	if got.Verify == nil || got.Verify.Status != "skipped" {
		t.Fatalf("expected verify skipped, got %#v", got.Verify)
	}
}

func successCommand(text string) string {
	if runtime.GOOS == "windows" {
		return "powershell -Command \"Write-Output '" + text + "'\""
	}
	return "printf '" + text + "'"
}

func failingCommand() string {
	if runtime.GOOS == "windows" {
		return "powershell -Command \"Write-Output fail; exit 3\""
	}
	return "echo fail && exit 3"
}

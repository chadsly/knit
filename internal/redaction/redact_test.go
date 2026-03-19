package redaction

import (
	"strings"
	"testing"
)

func TestTextRedactsSecrets(t *testing.T) {
	in := "api_key=abc123 password:mysecret token=foo"
	out := Text(in)
	if out == in {
		t.Fatalf("expected redaction to change output")
	}
	if containsPlainSecret(out) {
		t.Fatalf("expected redacted output, got: %s", out)
	}
}

func TestURLAllowedAppliesAllowAndBlocklist(t *testing.T) {
	allow := []string{"example.com", "api.openai.com"}
	block := []string{"forbidden.example.com"}
	if !URLAllowed("https://example.com/app", allow, block) {
		t.Fatalf("expected allowlisted url")
	}
	if URLAllowed("https://forbidden.example.com/app", allow, block) {
		t.Fatalf("expected blocklisted url to be rejected")
	}
	if URLAllowed("https://unknown.com", allow, block) {
		t.Fatalf("expected non-allowlisted url to be rejected when allowlist is set")
	}
}

func TestSensitiveContextDetection(t *testing.T) {
	if !SensitiveContext("Password Reset Form") {
		t.Fatalf("expected password context to be marked sensitive")
	}
	if SensitiveContext("marketing landing page") {
		t.Fatalf("expected non-sensitive context to remain allowed")
	}
}

func containsPlainSecret(text string) bool {
	return strings.Contains(text, "abc123") || strings.Contains(text, "mysecret") || strings.Contains(text, "token=foo")
}

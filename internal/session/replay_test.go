package session

import (
	"strings"
	"testing"
	"time"
)

func TestGeneratePlaywrightScriptUsesCapturedStepsAndSoftAssertions(t *testing.T) {
	now := time.Now().UTC()
	script := GeneratePlaywrightScript("Replay save failure", &ReplayBundle{
		URL:              "https://example.com/settings",
		TargetSelector:   "#save",
		ValueCaptureMode: "opt_in",
		Steps: []ReplayStep{{
			Type:           "focus",
			Timestamp:      now,
			TargetSelector: "#search",
		}, {
			Type:           "input",
			Timestamp:      now.Add(1 * time.Second),
			TargetSelector: "#search",
			ValueCaptured:  true,
			Value:          "button copy",
		}, {
			Type:           "click",
			Timestamp:      now.Add(2 * time.Second),
			TargetSelector: "#save",
			ClickCount:     1,
		}},
		Console: []ConsoleEntry{{
			Level:   "error",
			Message: "Save failed",
		}},
		Network: []NetworkEntry{{
			Method: "POST",
			URL:    "https://example.com/api/save",
			Status: 500,
			OK:     false,
		}},
	})
	if !strings.Contains(script, "await page.goto(\"https://example.com/settings\")") {
		t.Fatalf("expected goto in script, got:\n%s", script)
	}
	if !strings.Contains(script, "await page.locator(\"#search\").first().fill(\"button copy\")") {
		t.Fatalf("expected fill step in script, got:\n%s", script)
	}
	if !strings.Contains(script, "await page.locator(\"#save\").first().click()") {
		t.Fatalf("expected click step in script, got:\n%s", script)
	}
	if !strings.Contains(script, "expect.soft(consoleFailures).toEqual([])") || !strings.Contains(script, "expect.soft(failedResponses).toEqual([])") {
		t.Fatalf("expected soft assertions for captured failures, got:\n%s", script)
	}
}

func TestGeneratePlaywrightScriptCommentsWhenValuesWereRedacted(t *testing.T) {
	script := GeneratePlaywrightScript("", &ReplayBundle{
		Steps: []ReplayStep{{
			Type:           "input",
			TargetSelector: "#email",
			ValueCaptured:  false,
			ValueRedacted:  true,
		}},
	})
	if !strings.Contains(script, "typed value was redacted") {
		t.Fatalf("expected redaction comment in script, got:\n%s", script)
	}
}

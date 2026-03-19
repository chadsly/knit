package session

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func GeneratePlaywrightScript(title string, bundle *ReplayBundle) string {
	if bundle == nil || len(bundle.Steps) == 0 {
		return ""
	}
	steps := cloneReplaySteps(bundle.Steps)
	sort.SliceStable(steps, func(i, j int) bool {
		return steps[i].Timestamp.Before(steps[j].Timestamp)
	})

	name := strings.TrimSpace(title)
	if name == "" {
		name = "reproduce captured browser issue"
	}
	url := strings.TrimSpace(bundle.URL)
	if url == "" {
		for _, step := range steps {
			if candidate := strings.TrimSpace(step.URL); candidate != "" {
				url = candidate
				break
			}
		}
	}

	var b strings.Builder
	b.WriteString("import { test, expect } from '@playwright/test';\n\n")
	b.WriteString("test(" + strconv.Quote(name) + ", async ({ page }) => {\n")

	hasConsoleFailures := false
	for _, entry := range bundle.Console {
		level := strings.ToLower(strings.TrimSpace(entry.Level))
		if level == "error" || level == "warn" {
			hasConsoleFailures = true
			break
		}
	}
	if hasConsoleFailures {
		b.WriteString("  const consoleFailures = [];\n")
		b.WriteString("  page.on('console', msg => {\n")
		b.WriteString("    if (msg.type() === 'error' || msg.type() === 'warning') consoleFailures.push(msg.text());\n")
		b.WriteString("  });\n")
	}

	hasNetworkFailures := false
	for _, entry := range bundle.Network {
		if !entry.OK || entry.Status >= 400 || entry.Status == 0 {
			hasNetworkFailures = true
			break
		}
	}
	if hasNetworkFailures {
		b.WriteString("  const failedResponses = [];\n")
		b.WriteString("  page.on('response', response => {\n")
		b.WriteString("    if (response.status() >= 400) failedResponses.push(`${response.status()} ${response.url()}`);\n")
		b.WriteString("  });\n")
	}

	if url != "" {
		b.WriteString("  await page.goto(" + strconv.Quote(url) + ");\n")
		b.WriteString("  await page.waitForLoadState('domcontentloaded');\n")
	} else {
		b.WriteString("  // No page URL was captured. Start from the relevant page before replaying these steps.\n")
	}

	lastSelector := strings.TrimSpace(bundle.TargetSelector)
	for _, step := range steps {
		if line := playwrightStepLine(step); line != "" {
			b.WriteString("  " + line + "\n")
		} else {
			b.WriteString("  " + replayComment(step) + "\n")
		}
		if selector := replaySelector(step); selector != "" {
			lastSelector = selector
		}
	}

	if lastSelector != "" {
		b.WriteString("  await expect.soft(page.locator(" + strconv.Quote(lastSelector) + ").first()).toBeVisible();\n")
	}
	if hasConsoleFailures {
		b.WriteString("  expect.soft(consoleFailures).toEqual([]);\n")
	}
	if hasNetworkFailures {
		b.WriteString("  expect.soft(failedResponses).toEqual([]);\n")
	}
	b.WriteString("});\n")
	return b.String()
}

func playwrightStepLine(step ReplayStep) string {
	selector := replaySelector(step)
	switch strings.ToLower(strings.TrimSpace(step.Type)) {
	case "click":
		if selector == "" {
			return ""
		}
		opts := []string{}
		if button := replayMouseButton(step.MouseButton); button != "" && button != "left" {
			opts = append(opts, "button: "+strconv.Quote(button))
		}
		if step.ClickCount > 1 {
			opts = append(opts, fmt.Sprintf("clickCount: %d", step.ClickCount))
		}
		if len(opts) > 0 {
			return "await page.locator(" + strconv.Quote(selector) + ").first().click({ " + strings.Join(opts, ", ") + " });"
		}
		return "await page.locator(" + strconv.Quote(selector) + ").first().click();"
	case "focus":
		if selector == "" {
			return ""
		}
		return "await page.locator(" + strconv.Quote(selector) + ").first().focus();"
	case "input", "change":
		if selector == "" {
			return ""
		}
		if step.ValueCaptured {
			return "await page.locator(" + strconv.Quote(selector) + ").first().fill(" + strconv.Quote(step.Value) + ");"
		}
		return ""
	case "keydown":
		if key := replayKeyChord(step); key != "" {
			return "await page.keyboard.press(" + strconv.Quote(key) + ");"
		}
		return ""
	case "scroll":
		return fmt.Sprintf("await page.mouse.wheel(%d, %d);", int(step.ScrollDX), int(step.ScrollDY))
	case "submit":
		if selector == "" {
			return ""
		}
		return "await page.locator(" + strconv.Quote(selector) + ").first().press('Enter');"
	}
	return ""
}

func replayComment(step ReplayStep) string {
	selector := replaySelector(step)
	label := strings.TrimSpace(step.TargetLabel)
	if label == "" {
		label = strings.TrimSpace(step.TargetTag)
	}
	switch strings.ToLower(strings.TrimSpace(step.Type)) {
	case "input", "change":
		if selector == "" {
			return "// Input step captured without a stable selector; locate the field manually."
		}
		if step.ValueRedacted || !step.ValueCaptured {
			return "// Input captured for " + strconv.Quote(selector) + ", but the typed value was redacted because value capture was not enabled."
		}
	case "keyup", "blur", "drag_start", "drag_move", "drag_end":
		if selector != "" {
			return "// Captured " + strings.TrimSpace(step.Type) + " on " + strconv.Quote(selector) + "."
		}
		if label != "" {
			return "// Captured " + strings.TrimSpace(step.Type) + " on " + strconv.Quote(label) + "."
		}
	}
	return "// Captured " + strings.TrimSpace(step.Type) + " step needs manual review."
}

func replaySelector(step ReplayStep) string {
	if selector := strings.TrimSpace(step.TargetSelector); selector != "" {
		return selector
	}
	if testID := strings.TrimSpace(step.TargetTestID); testID != "" {
		return `[data-testid="` + testID + `"]`
	}
	if id := strings.TrimSpace(step.TargetID); id != "" {
		return `#` + id
	}
	return ""
}

func replayMouseButton(button int) string {
	switch button {
	case 1:
		return "middle"
	case 2:
		return "right"
	default:
		return "left"
	}
}

func replayKeyChord(step ReplayStep) string {
	key := strings.TrimSpace(step.Key)
	if key == "" {
		return ""
	}
	parts := make([]string, 0, len(step.Modifiers)+1)
	for _, modifier := range step.Modifiers {
		switch strings.ToLower(strings.TrimSpace(modifier)) {
		case "alt":
			parts = append(parts, "Alt")
		case "control", "ctrl":
			parts = append(parts, "Control")
		case "meta", "cmd", "command":
			parts = append(parts, "Meta")
		case "shift":
			parts = append(parts, "Shift")
		}
	}
	switch strings.ToLower(key) {
	case " ":
		key = "Space"
	case "esc":
		key = "Escape"
	}
	parts = append(parts, key)
	return strings.Join(parts, "+")
}

package companion

import (
	"testing"
	"time"

	"knit/internal/session"
)

func TestTrackerSnapshotIncludesLatestAndPath(t *testing.T) {
	tracker := NewTracker(3)
	now := time.Now().UTC()
	tracker.Add(PointerEvent{SessionID: "sess-1", X: 10, Y: 20, Window: "Browser", EventType: "move", URL: "https://example.com/app", Route: "/app", Timestamp: now.Add(-2 * time.Second)})
	tracker.Add(PointerEvent{
		SessionID:      "sess-1",
		X:              15,
		Y:              25,
		Window:         "Browser",
		EventType:      "click",
		URL:            "https://example.com/app",
		Route:          "/app/settings",
		TargetTag:      "button",
		TargetID:       "save",
		TargetTestID:   "settings-save",
		TargetRole:     "button",
		TargetLabel:    "Save Settings",
		TargetSelector: "#save",
		ClickCount:     1,
		DOM: &session.DOMInspection{
			Tag:         "button",
			ID:          "save",
			TestID:      "settings-save",
			Label:       "Save Settings",
			Selector:    "#save",
			TextPreview: "Save Settings",
		},
		Console: []session.ConsoleEntry{{
			Level:     "warn",
			Message:   "Save button repainted twice",
			Timestamp: now.Add(-1500 * time.Millisecond),
		}},
		Network: []session.NetworkEntry{{
			Kind:       "fetch",
			Method:     "POST",
			URL:        "https://example.com/api/save",
			Status:     500,
			OK:         false,
			DurationMS: 812,
			Timestamp:  now.Add(-1200 * time.Millisecond),
		}},
		Timestamp: now.Add(-1 * time.Second),
	})

	ctx, path := tracker.Snapshot("sess-1")
	if ctx.X != 15 || ctx.Y != 25 {
		t.Fatalf("unexpected pointer snapshot coordinates: %d,%d", ctx.X, ctx.Y)
	}
	if ctx.TargetTag != "button" || ctx.TargetID != "save" {
		t.Fatalf("expected target metadata in snapshot")
	}
	if ctx.URL != "https://example.com/app" || ctx.Route != "/app/settings" {
		t.Fatalf("expected url/route in snapshot, got url=%q route=%q", ctx.URL, ctx.Route)
	}
	if ctx.TargetTestID != "settings-save" || ctx.TargetLabel != "Save Settings" || ctx.TargetSelector != "#save" {
		t.Fatalf("expected enriched target metadata in snapshot")
	}
	if ctx.DOM == nil || ctx.DOM.Selector != "#save" {
		t.Fatalf("expected dom inspection in snapshot")
	}
	if len(ctx.Console) != 1 || ctx.Console[0].Message != "Save button repainted twice" {
		t.Fatalf("expected console context in snapshot, got %#v", ctx.Console)
	}
	if len(ctx.Network) != 1 || ctx.Network[0].Status != 500 {
		t.Fatalf("expected network context in snapshot, got %#v", ctx.Network)
	}
	if len(path) != 2 {
		t.Fatalf("expected path length 2, got %d", len(path))
	}
	if got := path[1].Route; got != "/app/settings" {
		t.Fatalf("expected route in pointer path sample, got %q", got)
	}
	replay := tracker.ReplaySnapshot("sess-1")
	if len(replay) != 1 || replay[0].Type != "click" || replay[0].TargetRole != "button" {
		t.Fatalf("expected replay snapshot to include click step, got %#v", replay)
	}
}

func TestTrackerReplaySnapshotIncludesInteractionVariants(t *testing.T) {
	tracker := NewTracker(8)
	now := time.Now().UTC()
	tracker.Add(PointerEvent{
		SessionID:       "sess-2",
		X:               42,
		Y:               64,
		Window:          "Browser",
		EventType:       "drag_start",
		URL:             "https://example.com/app",
		Route:           "/canvas",
		TargetTag:       "canvas",
		TargetID:        "editor",
		TargetTestID:    "editor-canvas",
		TargetRole:      "application",
		TargetLabel:     "Editor Canvas",
		TargetSelector:  "#editor",
		HoverDurationMS: 240,
		Timestamp:       now.Add(-4 * time.Second),
	})
	tracker.Add(PointerEvent{
		SessionID: "sess-2",
		X:         120,
		Y:         160,
		Window:    "Browser",
		EventType: "drag_end",
		URL:       "https://example.com/app",
		Route:     "/canvas",
		Timestamp: now.Add(-3 * time.Second),
	})
	tracker.Add(PointerEvent{
		SessionID:      "sess-2",
		X:              120,
		Y:              160,
		Window:         "Browser",
		EventType:      "scroll",
		URL:            "https://example.com/app",
		Route:          "/canvas",
		ScrollDY:       320,
		TargetSelector: "#editor",
		Timestamp:      now.Add(-2 * time.Second),
	})
	tracker.Add(PointerEvent{
		SessionID:      "sess-2",
		X:              128,
		Y:              176,
		Window:         "Browser",
		EventType:      "input",
		URL:            "https://example.com/app",
		Route:          "/canvas",
		InputType:      "text",
		Value:          "Rename layer",
		ValueCaptured:  true,
		TargetTag:      "input",
		TargetID:       "layer-name",
		TargetTestID:   "layer-name",
		TargetRole:     "textbox",
		TargetLabel:    "Layer name",
		TargetSelector: "#layer-name",
		DOM: &session.DOMInspection{
			Tag:         "input",
			ID:          "layer-name",
			TestID:      "layer-name",
			Label:       "Layer name",
			Selector:    "#layer-name",
			TextPreview: "Rename layer",
		},
		Timestamp: now.Add(-1 * time.Second),
	})
	tracker.Add(PointerEvent{
		SessionID:    "sess-2",
		X:            128,
		Y:            176,
		Window:       "Browser",
		EventType:    "keydown",
		URL:          "https://example.com/app",
		Route:        "/canvas",
		Key:          "Enter",
		Code:         "Enter",
		Modifiers:    []string{"Meta"},
		TargetRole:   "textbox",
		TargetLabel:  "Layer name",
		TargetTestID: "layer-name",
		Timestamp:    now,
	})

	ctx, path := tracker.Snapshot("sess-2")
	if ctx.TargetTestID != "layer-name" || ctx.TargetLabel != "Layer name" {
		t.Fatalf("expected latest target grounding in snapshot, got %#v", ctx)
	}
	if len(path) != 5 {
		t.Fatalf("expected five pointer samples in path, got %d", len(path))
	}
	if path[0].EventType != "drag_start" || path[1].EventType != "drag_end" || path[2].EventType != "scroll" {
		t.Fatalf("expected drag and scroll events in pointer path, got %#v", path)
	}

	replay := tracker.ReplaySnapshot("sess-2")
	if len(replay) != 5 {
		t.Fatalf("expected five replay steps, got %#v", replay)
	}
	if replay[0].Type != "drag_start" || replay[1].Type != "drag_end" || replay[2].Type != "scroll" {
		t.Fatalf("expected drag and scroll replay steps, got %#v", replay)
	}
	if replay[3].Type != "input" || !replay[3].ValueCaptured || replay[3].Value != "Rename layer" {
		t.Fatalf("expected captured input replay step, got %#v", replay[3])
	}
	if replay[3].DOM == nil || replay[3].DOM.TestID != "layer-name" {
		t.Fatalf("expected dom grounding on input replay step, got %#v", replay[3].DOM)
	}
	if replay[4].Type != "keydown" || replay[4].Key != "Enter" || len(replay[4].Modifiers) != 1 || replay[4].Modifiers[0] != "Meta" {
		t.Fatalf("expected key replay step with modifiers, got %#v", replay[4])
	}
}

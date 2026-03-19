package capture

import "testing"

func TestManagerStateTransitions(t *testing.T) {
	m := NewManager()
	if m.State() != StateInactive {
		t.Fatalf("expected inactive initial state")
	}
	m.Start()
	if m.State() != StateActive {
		t.Fatalf("expected active state")
	}
	m.Pause()
	if m.State() != StatePaused {
		t.Fatalf("expected paused state")
	}
	m.Stop()
	if m.State() != StateInactive {
		t.Fatalf("expected inactive after stop")
	}
}

func TestSourceStatusAndReducedCapabilities(t *testing.T) {
	m := NewManager()
	m.SetSourceStatus("microphone", "available", "")
	m.SetSourceStatus("screen", "degraded", "permission denied")
	m.SetSourceStatus("companion", "unavailable", "not attached")

	sources := m.SourceStatuses()
	if sources["microphone"].Status != "available" {
		t.Fatalf("expected microphone available")
	}
	if sources["screen"].Status != "degraded" {
		t.Fatalf("expected screen degraded")
	}
	if sources["companion"].Status != "unavailable" {
		t.Fatalf("expected companion unavailable")
	}

	reduced := m.ReducedCapabilities()
	if len(reduced) != 2 {
		t.Fatalf("expected two reduced capabilities, got %d (%v)", len(reduced), reduced)
	}
}

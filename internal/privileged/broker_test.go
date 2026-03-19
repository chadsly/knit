package privileged

import (
	"testing"

	"knit/internal/audio"
	"knit/internal/capture"
	"knit/internal/companion"
)

func TestCaptureBrokerReturnsDefensiveCopies(t *testing.T) {
	manager := capture.NewManager()
	manager.SetSourceStatus("screen", "available", "ready")
	tracker := companion.NewTracker(8)
	controller := audio.NewController()
	controller.SetDevices([]audio.Device{{ID: "mic-a", Label: "Mic A"}})

	broker := NewCaptureBroker(manager, tracker, controller)

	sources := broker.SourceStatuses()
	sources["screen"] = capture.SourceState{Status: "degraded"}
	if got := broker.SourceStatuses()["screen"].Status; got != "available" {
		t.Fatalf("expected source status copy, got %q", got)
	}

	devices := broker.AudioDevices()
	devices[0].Label = "mutated"
	if got := broker.AudioDevices()[0].Label; got != "Mic A" {
		t.Fatalf("expected audio devices copy, got %q", got)
	}
}

package companion

import (
	"testing"
	"time"
)

func BenchmarkTrackerAddAndSnapshot(b *testing.B) {
	tracker := NewTracker(512)
	base := time.Now().UTC()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Add(PointerEvent{
			SessionID: "sess-bench",
			X:         i % 1024,
			Y:         (i * 3) % 768,
			EventType: "move",
			Window:    "Browser Preview",
			URL:       "https://example.com/app",
			Route:     "/app",
			ScrollDY:  float64(i % 7),
			TargetTag: "button",
			TargetID:  "save",
			Timestamp: base.Add(time.Duration(i) * time.Millisecond),
		})
		if i%8 == 0 {
			_, _ = tracker.Snapshot("sess-bench")
		}
	}
}

package server

import (
	"testing"
	"time"
)

func TestLatencyBookObserveAndSnapshot(t *testing.T) {
	book := newLatencyBook(32)
	book.observe("pointer_ingest_ms", 1*time.Millisecond)
	book.observe("pointer_ingest_ms", 2*time.Millisecond)
	book.observe("pointer_ingest_ms", 3*time.Millisecond)

	snap := book.snapshot()
	got, ok := snap["pointer_ingest_ms"]
	if !ok {
		t.Fatalf("expected pointer_ingest_ms metric")
	}
	if got.Count != 3 {
		t.Fatalf("expected 3 samples, got %d", got.Count)
	}
	if got.P95MS <= 0 || got.MaxMS <= 0 || got.LastMS <= 0 {
		t.Fatalf("expected positive latency summary values, got %#v", got)
	}
}

func TestLatencyTrackerRespectsWindowCapacity(t *testing.T) {
	tr := newLatencyTracker(10)
	for i := 0; i < 12; i++ {
		tr.observe(time.Duration(i+1) * time.Millisecond)
	}
	snap := tr.snapshot()
	if snap.Count != 10 {
		t.Fatalf("expected capped sample window of 10, got %d", snap.Count)
	}
	if snap.MaxMS < 12 {
		t.Fatalf("expected max to include most recent sample, got %f", snap.MaxMS)
	}
}

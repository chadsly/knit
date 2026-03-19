package server

import (
	"sort"
	"sync"
	"time"
)

type latencySummary struct {
	Count  int     `json:"count"`
	P50MS  float64 `json:"p50_ms"`
	P95MS  float64 `json:"p95_ms"`
	P99MS  float64 `json:"p99_ms"`
	MaxMS  float64 `json:"max_ms"`
	LastMS float64 `json:"last_ms"`
}

type latencyTracker struct {
	mu      sync.Mutex
	samples []float64
	idx     int
	filled  bool
}

func newLatencyTracker(capacity int) *latencyTracker {
	if capacity < 8 {
		capacity = 8
	}
	return &latencyTracker{samples: make([]float64, capacity)}
}

func (l *latencyTracker) observe(d time.Duration) {
	ms := float64(d.Microseconds()) / 1000.0
	if ms < 0 {
		ms = 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.samples[l.idx] = ms
	l.idx++
	if l.idx >= len(l.samples) {
		l.idx = 0
		l.filled = true
	}
}

func (l *latencyTracker) snapshot() latencySummary {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := l.idx
	if l.filled {
		n = len(l.samples)
	}
	if n == 0 {
		return latencySummary{}
	}
	out := make([]float64, 0, n)
	if l.filled {
		out = append(out, l.samples...)
	} else {
		out = append(out, l.samples[:n]...)
	}
	sort.Float64s(out)
	max := out[len(out)-1]
	lastIdx := l.idx - 1
	if lastIdx < 0 {
		lastIdx = len(l.samples) - 1
	}
	last := l.samples[lastIdx]
	return latencySummary{
		Count:  len(out),
		P50MS:  percentile(out, 0.50),
		P95MS:  percentile(out, 0.95),
		P99MS:  percentile(out, 0.99),
		MaxMS:  max,
		LastMS: last,
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := int(float64(len(sorted)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

type latencyBook struct {
	mu       sync.RWMutex
	capacity int
	items    map[string]*latencyTracker
}

func newLatencyBook(capacity int) *latencyBook {
	return &latencyBook{
		capacity: capacity,
		items:    map[string]*latencyTracker{},
	}
}

func (b *latencyBook) observe(name string, d time.Duration) {
	if b == nil || name == "" {
		return
	}
	b.mu.Lock()
	tracker, ok := b.items[name]
	if !ok {
		tracker = newLatencyTracker(b.capacity)
		b.items[name] = tracker
	}
	b.mu.Unlock()
	tracker.observe(d)
}

func (b *latencyBook) snapshot() map[string]latencySummary {
	if b == nil {
		return map[string]latencySummary{}
	}
	b.mu.RLock()
	keys := make([]string, 0, len(b.items))
	trackers := make(map[string]*latencyTracker, len(b.items))
	for k, v := range b.items {
		keys = append(keys, k)
		trackers[k] = v
	}
	b.mu.RUnlock()
	sort.Strings(keys)
	out := make(map[string]latencySummary, len(keys))
	for _, k := range keys {
		out[k] = trackers[k].snapshot()
	}
	return out
}

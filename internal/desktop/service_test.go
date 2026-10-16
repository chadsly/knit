package desktop

import (
	"errors"
	"knit/internal/session"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	raw  []byte
	fail bool
}

func (m *memoryStore) LoadDesktopState() ([]byte, error) { return append([]byte(nil), m.raw...), nil }
func (m *memoryStore) SaveDesktopState(raw []byte) error {
	if m.fail {
		return errors.New("disk unavailable")
	}
	m.raw = append([]byte(nil), raw...)
	return nil
}
func samplePackage() session.CanonicalPackage {
	return session.CanonicalPackage{SessionID: "session-1", Summary: "Fix spacing", ChangeRequests: []session.ChangeReq{{Summary: "Fix spacing"}}}
}

func TestDurableDesktopLifecycle(t *testing.T) {
	store := &memoryStore{}
	s := New(store)
	b, err := s.Connect("Checkout fixes", "/workspace/shop")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.Queue(b.ID, "request-1", "/workspace/shop", samplePackage())
	if err != nil {
		t.Fatal(err)
	}
	if d.Workspace != b.Workspace {
		t.Fatal("delivery lost its intended workspace")
	}
	repeated, err := s.Queue(b.ID, "request-1", "/workspace/shop", samplePackage())
	if err != nil || repeated.ID != d.ID {
		t.Fatal("request was duplicated")
	}
	s = New(store)
	state, err := s.Heartbeat(b.ID)
	if err != nil || len(state.Deliveries) != 1 {
		t.Fatalf("lost queued delivery: %v", err)
	}
	if _, err = s.Prepare(b.ID, d.ID); err != nil {
		t.Fatal(err)
	}
	s = New(store)
	if _, err = s.Prepare(b.ID, d.ID); err == nil {
		t.Fatal("uncertain send was replayed after restart")
	}
	if _, err = s.Report(b.ID, d.ID, "completed", "done"); err == nil {
		t.Fatal("sending was mistaken for completed")
	}
	if _, err = s.Report(b.ID, d.ID, "delivered", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Report(b.ID, d.ID, "working", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Report(b.ID, d.ID, "completed", "Updated checkout spacing and ran tests."); err != nil {
		t.Fatal(err)
	}
	state, _ = New(store).Snapshot()
	if state.Deliveries[0].Status != "completed" {
		t.Fatal("completion was not durable")
	}
	if err = s.PurgeSession("session-1"); err != nil {
		t.Fatal(err)
	}
	state, _ = s.Snapshot()
	if len(state.Deliveries) != 0 {
		t.Fatal("purged session feedback retained")
	}
}

func TestDesktopScopeAndFailureSafety(t *testing.T) {
	store := &memoryStore{}
	s := New(store)
	b, _ := s.Connect("Current conversation", "/workspace/shop")
	other, _ := s.Connect("Other conversation", "/workspace/shop")
	if _, err := s.Queue(b.ID, "r", "/workspace/other", samplePackage()); err == nil {
		t.Fatal("workspace mismatch accepted")
	}
	if _, err := s.Queue(b.ID, "r", "/workspace/shop", session.CanonicalPackage{}); err == nil {
		t.Fatal("empty input accepted")
	}
	d, _ := s.Queue(b.ID, "r", "/workspace/shop", samplePackage())
	if _, err := s.Prepare(other.ID, d.ID); err == nil {
		t.Fatal("another panel read feedback")
	}
	store.fail = true
	if _, err := s.Prepare(b.ID, d.ID); err == nil {
		t.Fatal("write failure was swallowed")
	}
	store.fail = false
	state, _ := s.Snapshot()
	if state.Deliveries[0].Status != "queued" {
		t.Fatal("failed write changed persisted state")
	}
	if err := s.Disconnect(b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prepare(b.ID, d.ID); err == nil {
		t.Fatal("revoked panel received feedback")
	}
	state, _ = s.Snapshot()
	if state.Deliveries[0].Status != "needs_attention" {
		t.Fatal("disconnection must retain feedback for attention")
	}
}

func TestDesktopPrepareIsAtomic(t *testing.T) {
	s := New(&memoryStore{})
	b, _ := s.Connect("Current conversation", "/workspace/shop")
	d, _ := s.Queue(b.ID, "r", "/workspace/shop", samplePackage())
	var wg sync.WaitGroup
	success := make(chan bool, 12)
	for range 12 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.Prepare(b.ID, d.ID); success <- err == nil }()
	}
	wg.Wait()
	close(success)
	count := 0
	for ok := range success {
		if ok {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one payload preparation, got %d", count)
	}
}

func TestDesktopStructuredRetention(t *testing.T) {
	s := New(&memoryStore{})
	b, _ := s.Connect("Shop", "/workspace/shop")
	if _, err := s.Queue(b.ID, "r", b.Workspace, samplePackage()); err != nil {
		t.Fatal(err)
	}
	if err := s.PurgeBefore(time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	state, _ := s.Snapshot()
	if len(state.Deliveries) != 1 {
		t.Fatal("unexpired feedback was removed")
	}
	if err := s.PurgeBefore(time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	state, _ = s.Snapshot()
	if len(state.Deliveries) != 0 || len(state.Bindings) != 1 {
		t.Fatal("retention must remove feedback, not connections")
	}
}

func TestCaptureScopeAndExplicitDeliveryConsent(t *testing.T) {
	store := &memoryStore{}
	s := New(store)
	b, _ := s.Connect("Shop", "/workspace/shop")
	other, _ := s.Connect("Other", "/workspace/shop")
	if b.AutomaticDelivery {
		t.Fatal("automatic delivery must require opt-in")
	}
	if _, err := s.SetAutomaticDelivery(b.ID, true); err != nil {
		t.Fatal(err)
	}
	state, _ := New(store).Snapshot()
	if !state.Bindings[0].AutomaticDelivery {
		t.Fatal("consent not durable")
	}
	pkg := samplePackage()
	pkg.Artifacts = []session.ArtifactRef{{Kind: "screenshot", Ref: "encrypted-snapshot"}}
	d, _ := s.Queue(b.ID, "media", b.Workspace, pkg)
	if _, err := s.CaptureReference(b.ID, d.ID, 0); err == nil {
		t.Fatal("capture read before delivery")
	}
	if _, err := s.Prepare(b.ID, d.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Report(b.ID, d.ID, "working", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CaptureReference(other.ID, d.ID, 0); err == nil {
		t.Fatal("other connection read capture")
	}
	if _, err := s.CaptureReference(b.ID, d.ID, -1); err == nil {
		t.Fatal("negative capture index accepted")
	}
	if ref, err := s.CaptureReference(b.ID, d.ID, 0); err != nil || ref.Ref != "encrypted-snapshot" {
		t.Fatal("approved capture unavailable", err)
	}
	if _, err := s.SetAutomaticDelivery(b.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := s.Disconnect(b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CaptureReference(b.ID, d.ID, 0); err == nil {
		t.Fatal("revoked connection read capture")
	}
}

func TestDesktopLateHostAcknowledgementPreservesProgress(t *testing.T) {
	s := New(&memoryStore{})
	b, _ := s.Connect("Checkout", "/workspace/shop")
	d, _ := s.Queue(b.ID, "race", b.Workspace, samplePackage())
	if _, err := s.Prepare(b.ID, d.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Report(b.ID, d.ID, "working", ""); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Report(b.ID, d.ID, "delivered", ""); err != nil || got.Status != "working" {
		t.Fatalf("late acknowledgement overwrote progress: %+v %v", got, err)
	}
	if _, err := s.Report(b.ID, d.ID, "completed", "Fixed checkout and tested."); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Report(b.ID, d.ID, "delivered", ""); err != nil || got.Status != "completed" || got.Result == "" {
		t.Fatalf("late acknowledgement overwrote result: %+v %v", got, err)
	}
}

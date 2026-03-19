package session

import (
	"encoding/json"
	"testing"
)

func TestCanonicalPackageUnmarshalNormalizesNullSlices(t *testing.T) {
	payload := []byte(`{
		"session_id":"sess-timeout",
		"session_meta":{"target_window":""},
		"change_requests":null,
		"artifacts":null,
		"generated_at":"2026-03-11T12:28:01.190171Z"
	}`)

	var pkg CanonicalPackage
	if err := json.Unmarshal(payload, &pkg); err != nil {
		t.Fatalf("unmarshal canonical package: %v", err)
	}
	if pkg.ChangeRequests == nil {
		t.Fatalf("expected change requests to normalize to empty slice")
	}
	if pkg.Artifacts == nil {
		t.Fatalf("expected artifacts to normalize to empty slice")
	}
	if len(pkg.ChangeRequests) != 0 || len(pkg.Artifacts) != 0 {
		t.Fatalf("expected empty normalized slices, got %d change requests and %d artifacts", len(pkg.ChangeRequests), len(pkg.Artifacts))
	}
}

func TestCanonicalPackageMarshalEmitsEmptySlices(t *testing.T) {
	pkg := CanonicalPackage{
		SessionID:   "sess-timeout",
		SessionMeta: SessionMeta{TargetWindow: ""},
	}

	b, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("marshal canonical package: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("decode marshaled canonical package: %v", err)
	}
	if _, ok := decoded["change_requests"].([]any); !ok {
		t.Fatalf("expected change_requests to marshal as array, got %#v", decoded["change_requests"])
	}
	if _, ok := decoded["artifacts"].([]any); !ok {
		t.Fatalf("expected artifacts to marshal as array, got %#v", decoded["artifacts"])
	}
}

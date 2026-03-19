package session

import (
	"testing"
	"time"
)

func TestSessionLifecycleAndApproval(t *testing.T) {
	svc := NewService()

	sess := svc.StartWithMeta("Browser Preview", "https://localhost:3000", "personal_local_dev", "local-dev", "build-1")
	if sess == nil || sess.ID == "" {
		t.Fatalf("expected started session")
	}
	if sess.Profile != "personal_local_dev" || sess.Environment != "local-dev" || sess.BuildID != "build-1" {
		t.Fatalf("expected profile/env/build metadata on session")
	}
	if !sess.CaptureInputValues {
		t.Fatalf("expected replay typed-value capture enabled by default for new sessions")
	}

	_, err := svc.AddFeedback(FeedbackEvt{RawTranscript: "make button bigger", NormalizedText: "Increase primary button size"})
	if err != nil {
		t.Fatalf("add feedback: %v", err)
	}

	pkg, err := svc.Approve("")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if len(pkg.ChangeRequests) != 1 {
		t.Fatalf("expected 1 change request, got %d", len(pkg.ChangeRequests))
	}
	if pkg.SessionMeta.Profile != "personal_local_dev" {
		t.Fatalf("expected profile in canonical metadata")
	}
	if pkg.ChangeRequests[0].Category == "" || pkg.ChangeRequests[0].Priority == "" {
		t.Fatalf("expected classification fields in canonical change request")
	}

	if err := svc.MarkSubmitted("session:abc123"); err != nil {
		t.Fatalf("mark submitted: %v", err)
	}

	curr := svc.Current()
	if curr.Status != StatusSubmitted {
		t.Fatalf("expected submitted status, got %s", curr.Status)
	}
	if curr.Approved {
		t.Fatalf("expected approval to be consumed after submit")
	}
}

func TestCategoryAndAmbiguityInference(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")

	_, err := svc.AddFeedback(FeedbackEvt{
		RawTranscript:   "This is slow and this should feel faster",
		NormalizedText:  "This is slow and this should feel faster",
		VisualTargetRef: "",
		Pointer: PointerCtx{
			Window: "Preview",
		},
	})
	if err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	pkg, err := svc.Approve("")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if len(pkg.ChangeRequests) != 1 {
		t.Fatalf("expected 1 change request, got %d", len(pkg.ChangeRequests))
	}
	req := pkg.ChangeRequests[0]
	if req.Category != "performance_perceived_responsiveness" {
		t.Fatalf("expected performance category, got %s", req.Category)
	}
	if len(req.Ambiguities) == 0 {
		t.Fatalf("expected ambiguity markers when pronoun used without visual target")
	}
	if len(req.Assumptions) == 0 {
		t.Fatalf("expected assumptions when screenshot/visual target are missing")
	}
}

func TestReserveApprovedPackageConsumesApproval(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")

	if _, err := svc.AddFeedback(FeedbackEvt{RawTranscript: "first", NormalizedText: "first"}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	if _, err := svc.Approve(""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	pkg, err := svc.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve approved package: %v", err)
	}
	if pkg == nil || len(pkg.ChangeRequests) != 1 {
		t.Fatalf("expected reserved package with one change request")
	}
	if _, err := svc.ApprovedPackage(); err == nil {
		t.Fatalf("expected approved package to be unavailable after reserve")
	}
	curr := svc.Current()
	if curr.Approved {
		t.Fatalf("expected session approval consumed after reserve")
	}
	if curr.Feedback[0].Disposition != DispositionQueued {
		t.Fatalf("expected reserved feedback to move to queued, got %s", curr.Feedback[0].Disposition)
	}
}

func TestMarkSubmittedForOnlyMarksSelectedEvents(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")

	if _, err := svc.AddFeedback(FeedbackEvt{RawTranscript: "first", NormalizedText: "first"}); err != nil {
		t.Fatalf("add first feedback: %v", err)
	}
	if _, err := svc.Approve(""); err != nil {
		t.Fatalf("approve first: %v", err)
	}
	firstPkg, err := svc.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve first package: %v", err)
	}

	if _, err := svc.AddFeedback(FeedbackEvt{RawTranscript: "second", NormalizedText: "second"}); err != nil {
		t.Fatalf("add second feedback: %v", err)
	}
	if _, err := svc.Approve(""); err != nil {
		t.Fatalf("approve second: %v", err)
	}

	firstEventID := firstPkg.ChangeRequests[0].EventID
	if firstEventID == "" {
		t.Fatalf("expected first event id")
	}
	if err := svc.MarkSubmittedFor("session:first", []string{firstEventID}); err != nil {
		t.Fatalf("mark submitted for: %v", err)
	}

	curr := svc.Current()
	if len(curr.Feedback) != 2 {
		t.Fatalf("expected two feedback events")
	}
	if curr.Feedback[0].Disposition != DispositionSubmitted {
		t.Fatalf("expected first feedback submitted, got %s", curr.Feedback[0].Disposition)
	}
	if curr.Feedback[1].Disposition != DispositionApproved {
		t.Fatalf("expected second feedback disposition unchanged, got %s", curr.Feedback[1].Disposition)
	}
}

func TestApproveSkipsPreviouslySubmittedFeedback(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")

	if _, err := svc.AddFeedback(FeedbackEvt{RawTranscript: "first", NormalizedText: "first"}); err != nil {
		t.Fatalf("add first feedback: %v", err)
	}
	if _, err := svc.Approve(""); err != nil {
		t.Fatalf("approve first: %v", err)
	}
	firstPkg, err := svc.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve first package: %v", err)
	}
	if err := svc.MarkSubmittedFor("session:first", []string{firstPkg.ChangeRequests[0].EventID}); err != nil {
		t.Fatalf("mark submitted for first package: %v", err)
	}

	if _, err := svc.AddFeedback(FeedbackEvt{RawTranscript: "second", NormalizedText: "second"}); err != nil {
		t.Fatalf("add second feedback: %v", err)
	}
	pkg, err := svc.Approve("")
	if err != nil {
		t.Fatalf("approve second: %v", err)
	}
	if len(pkg.ChangeRequests) != 1 {
		t.Fatalf("expected only one unsubmitted change request, got %d", len(pkg.ChangeRequests))
	}
	if got := pkg.ChangeRequests[0].Summary; got != "second" {
		t.Fatalf("expected second change request only, got %q", got)
	}

	curr := svc.Current()
	if curr.Feedback[0].Disposition != DispositionSubmitted {
		t.Fatalf("expected first feedback to remain submitted, got %s", curr.Feedback[0].Disposition)
	}
}

func TestDiscardLastFeedbackRemovesLastEventAndClearsApproval(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")

	if _, err := svc.AddFeedback(FeedbackEvt{RawTranscript: "first", NormalizedText: "first"}); err != nil {
		t.Fatalf("add first feedback: %v", err)
	}
	if _, err := svc.AddFeedback(FeedbackEvt{RawTranscript: "second", NormalizedText: "second"}); err != nil {
		t.Fatalf("add second feedback: %v", err)
	}
	if _, err := svc.Approve(""); err != nil {
		t.Fatalf("approve: %v", err)
	}

	curr, discarded, err := svc.DiscardLastFeedback()
	if err != nil {
		t.Fatalf("discard last feedback: %v", err)
	}
	if discarded == nil || discarded.RawTranscript != "second" {
		t.Fatalf("expected second feedback to be discarded, got %#v", discarded)
	}
	if len(curr.Feedback) != 1 {
		t.Fatalf("expected one feedback remaining, got %d", len(curr.Feedback))
	}
	if curr.Approved {
		t.Fatalf("expected approval to be cleared after discard")
	}
}

func TestUpdateFeedbackTextClearsApprovalAndUsesNewText(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")

	curr, err := svc.AddFeedback(FeedbackEvt{RawTranscript: "old copy", NormalizedText: "old copy"})
	if err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	eventID := curr.Feedback[0].ID
	if _, err := svc.Approve(""); err != nil {
		t.Fatalf("approve: %v", err)
	}

	updated, evt, err := svc.UpdateFeedbackText(eventID, "new copy")
	if err != nil {
		t.Fatalf("update feedback text: %v", err)
	}
	if evt == nil || evt.NormalizedText != "new copy" || evt.RawTranscript != "new copy" {
		t.Fatalf("expected updated feedback text, got %#v", evt)
	}
	if updated.Approved {
		t.Fatalf("expected approval to be cleared after edit")
	}
	if _, err := svc.Approve(""); err != nil {
		t.Fatalf("re-approve: %v", err)
	}
	pkg, err := svc.ApprovedPackage()
	if err != nil {
		t.Fatalf("approved package: %v", err)
	}
	if got := pkg.ChangeRequests[0].Summary; got != "new copy" {
		t.Fatalf("expected updated change request text, got %q", got)
	}
}

func TestDeleteFeedbackRemovesMatchingEventAndClearsApproval(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")

	curr, err := svc.AddFeedback(FeedbackEvt{RawTranscript: "first", NormalizedText: "first"})
	if err != nil {
		t.Fatalf("add first feedback: %v", err)
	}
	firstID := curr.Feedback[0].ID
	curr, err = svc.AddFeedback(FeedbackEvt{RawTranscript: "second", NormalizedText: "second"})
	if err != nil {
		t.Fatalf("add second feedback: %v", err)
	}
	secondID := curr.Feedback[1].ID
	if _, err := svc.Approve(""); err != nil {
		t.Fatalf("approve: %v", err)
	}

	updated, deleted, err := svc.DeleteFeedback(firstID)
	if err != nil {
		t.Fatalf("delete feedback: %v", err)
	}
	if deleted == nil || deleted.ID != firstID {
		t.Fatalf("expected first feedback to be deleted, got %#v", deleted)
	}
	if updated.Approved {
		t.Fatalf("expected approval to be cleared after delete")
	}
	if len(updated.Feedback) != 1 || updated.Feedback[0].ID != secondID {
		t.Fatalf("expected only second feedback to remain, got %#v", updated.Feedback)
	}
}

func TestBootstrapRestoresHistoryCurrentAndSequence(t *testing.T) {
	svc := NewService()
	now := time.Now().UTC()
	sessions := []*Session{
		{
			ID:               "sess-1",
			TargetWindow:     "Preview",
			Status:           StatusSubmitted,
			CreatedAt:        now.Add(-2 * time.Hour),
			UpdatedAt:        now.Add(-90 * time.Minute),
			ApprovalRequired: true,
			Approved:         false,
			Feedback:         []FeedbackEvt{{ID: "evt-2", RawTranscript: "first", NormalizedText: "first"}},
			ReviewNotes:      []ReviewNote{{ID: "rev-3", Author: "chad", Note: "note", CreatedAt: now.Add(-89 * time.Minute)}},
		},
		{
			ID:               "sess-4",
			TargetWindow:     "Preview",
			Status:           StatusPaused,
			CreatedAt:        now.Add(-30 * time.Minute),
			UpdatedAt:        now.Add(-10 * time.Minute),
			ApprovalRequired: true,
			Approved:         true,
			Feedback:         []FeedbackEvt{{ID: "evt-5", RawTranscript: "second", NormalizedText: "second"}},
		},
	}
	pkg := &CanonicalPackage{
		SessionID:   "sess-4",
		Summary:     "second",
		GeneratedAt: now.Add(-9 * time.Minute),
		ChangeRequests: []ChangeReq{{
			EventID:  "evt-5",
			Summary:  "second",
			Category: "unclear_needs_review",
			Priority: "medium",
		}},
	}

	svc.Bootstrap(sessions, pkg)

	curr := svc.Current()
	if curr == nil || curr.ID != "sess-4" {
		t.Fatalf("expected latest session restored as current, got %#v", curr)
	}
	if got := svc.History(); len(got) != 2 || got[0].ID != "sess-1" || got[1].ID != "sess-4" {
		t.Fatalf("unexpected restored history ordering: %#v", got)
	}
	if restored, err := svc.ApprovedPackage(); err != nil || restored == nil || restored.SessionID != "sess-4" {
		t.Fatalf("expected approved package restored, got %#v err=%v", restored, err)
	}
	next := svc.Start("Preview", "https://example.com")
	if next.ID != "sess-6" {
		t.Fatalf("expected sequence to continue at sess-6, got %q", next.ID)
	}
}

func TestAccessibilityReviewModeInfluencesClassification(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")
	if _, err := svc.SetReviewMode("accessibility"); err != nil {
		t.Fatalf("set review mode: %v", err)
	}
	if _, err := svc.AddFeedback(FeedbackEvt{
		RawTranscript:  "Make this easier to read",
		NormalizedText: "Make this easier to read",
		ReviewMode:     "accessibility",
	}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	pkg, err := svc.Approve("")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if len(pkg.ChangeRequests) != 1 {
		t.Fatalf("expected 1 change request")
	}
	if pkg.ChangeRequests[0].Category != "accessibility" {
		t.Fatalf("expected accessibility category, got %s", pkg.ChangeRequests[0].Category)
	}
	if pkg.SessionMeta.ReviewMode != "accessibility" {
		t.Fatalf("expected accessibility session review mode in package metadata")
	}
}

func TestAddReviewNote(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")

	curr, note, err := svc.AddReviewNote("alice", "Looks good, but spacing is off.")
	if err != nil {
		t.Fatalf("add review note: %v", err)
	}
	if note == nil || note.Author != "alice" {
		t.Fatalf("expected review note author alice, got %#v", note)
	}
	if len(curr.ReviewNotes) != 1 {
		t.Fatalf("expected one review note, got %d", len(curr.ReviewNotes))
	}
}

func TestUpdateTargetContext(t *testing.T) {
	svc := NewService()
	svc.Start("Browser Review", "")

	curr, changed, err := svc.UpdateTargetContext("Ruddur - Home", "https://localhost:3000/")
	if err != nil {
		t.Fatalf("update target context: %v", err)
	}
	if !changed {
		t.Fatalf("expected target context to change")
	}
	if curr.TargetWindow != "Ruddur - Home" || curr.TargetURL != "https://localhost:3000/" {
		t.Fatalf("expected updated target metadata, got %#v", curr)
	}

	curr, changed, err = svc.UpdateTargetContext("", "")
	if err != nil {
		t.Fatalf("noop update target context: %v", err)
	}
	if changed {
		t.Fatalf("expected noop target context update")
	}
	if curr.TargetWindow != "Ruddur - Home" || curr.TargetURL != "https://localhost:3000/" {
		t.Fatalf("expected target metadata to stay unchanged, got %#v", curr)
	}
}

func TestFeedbackExperimentFieldsFlowIntoCanonicalPackage(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")

	if _, err := svc.AddFeedback(FeedbackEvt{
		RawTranscript:  "Variant B header should be shorter",
		NormalizedText: "Variant B header should be shorter",
		ExperimentID:   "exp-header-copy",
		Variant:        "B",
	}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	pkg, err := svc.Approve("")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	req := pkg.ChangeRequests[0]
	if req.ExperimentID != "exp-header-copy" || req.Variant != "B" {
		t.Fatalf("expected experiment metadata in change request, got %#v", req)
	}
}

func TestFeedbackInteractionContextFlowsIntoCanonicalPackage(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")

	if _, err := svc.AddFeedback(FeedbackEvt{
		RawTranscript:  "Save button throws a console warning and 500s",
		NormalizedText: "Save button throws a console warning and 500s",
		Pointer: PointerCtx{
			X:              120,
			Y:              48,
			Window:         "Preview",
			URL:            "https://example.com/settings",
			Route:          "/settings",
			TargetTag:      "button",
			TargetID:       "save",
			TargetTestID:   "settings-save",
			TargetLabel:    "Save Settings",
			TargetSelector: "#save",
			DOM: &DOMInspection{
				Tag:         "button",
				ID:          "save",
				TestID:      "settings-save",
				Label:       "Save Settings",
				Selector:    "#save",
				TextPreview: "Save Settings",
			},
			Console: []ConsoleEntry{{
				Level:   "error",
				Message: "Save failed with HTTP 500",
			}},
			Network: []NetworkEntry{{
				Kind:       "fetch",
				Method:     "POST",
				URL:        "https://example.com/api/save",
				Status:     500,
				OK:         false,
				DurationMS: 920,
			}},
		},
		PointerPath: []PointerSample{{
			X:         118,
			Y:         45,
			EventType: "move",
		}, {
			X:         120,
			Y:         48,
			EventType: "click",
		}},
		Replay: &ReplayBundle{
			URL:              "https://example.com/settings",
			Route:            "/settings",
			TargetSelector:   "#save",
			ValueCaptureMode: "redacted",
			Steps: []ReplayStep{{
				Type:           "click",
				URL:            "https://example.com/settings",
				Route:          "/settings",
				TargetSelector: "#save",
				TargetLabel:    "Save Settings",
			}},
		},
		VisualTargetRef: "button#save | #save",
	}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}

	pkg, err := svc.Approve("")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	req := pkg.ChangeRequests[0]
	if req.Pointer.TargetSelector != "#save" || req.VisualTargetRef != "button#save | #save" {
		t.Fatalf("expected pointer context in change request, got %#v", req)
	}
	if req.Pointer.DOM == nil || req.Pointer.DOM.TestID != "settings-save" {
		t.Fatalf("expected dom inspection in change request, got %#v", req.Pointer.DOM)
	}
	if len(req.Pointer.Console) != 1 || req.Pointer.Console[0].Level != "error" {
		t.Fatalf("expected console context in change request, got %#v", req.Pointer.Console)
	}
	if len(req.Pointer.Network) != 1 || req.Pointer.Network[0].Status != 500 {
		t.Fatalf("expected network context in change request, got %#v", req.Pointer.Network)
	}
	if len(req.PointerPath) != 2 || req.PointerPath[1].EventType != "click" {
		t.Fatalf("expected pointer path in change request, got %#v", req.PointerPath)
	}
	if req.Replay == nil || len(req.Replay.Steps) != 1 || req.Replay.Steps[0].TargetSelector != "#save" {
		t.Fatalf("expected replay bundle in change request, got %#v", req.Replay)
	}
}

func TestBugFirstTriagePolicyEscalatesBugPriority(t *testing.T) {
	t.Setenv("KNIT_TRIAGE_POLICY", "bug_first")

	svc := NewService()
	svc.Start("Preview", "https://example.com")

	if _, err := svc.AddFeedback(FeedbackEvt{
		RawTranscript:  "The save button fails when clicked",
		NormalizedText: "The save button fails when clicked",
	}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	pkg, err := svc.Approve("")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if len(pkg.ChangeRequests) != 1 {
		t.Fatalf("expected 1 change request")
	}
	req := pkg.ChangeRequests[0]
	if req.Category != "bug_defect" {
		t.Fatalf("expected bug_defect category, got %s", req.Category)
	}
	if req.Priority != "high" {
		t.Fatalf("expected high priority under bug_first policy, got %s", req.Priority)
	}
}

func TestAttachClipStoresVideoMetadata(t *testing.T) {
	svc := NewService()
	svc.Start("Preview", "https://example.com")
	curr, err := svc.AddFeedback(FeedbackEvt{RawTranscript: "clip me", NormalizedText: "clip me"})
	if err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	if len(curr.Feedback) == 0 {
		t.Fatalf("expected feedback event")
	}
	eventID := curr.Feedback[0].ID
	start := time.Now().UTC().Add(-5 * time.Second)
	end := time.Now().UTC()
	meta := &VideoMetadata{
		Scope:          "selected-region",
		Window:         "Browser Preview",
		RegionX:        12,
		RegionY:        24,
		RegionW:        320,
		RegionH:        180,
		Codec:          "video/webm;codecs=vp9",
		HasAudio:       true,
		PointerOverlay: true,
		StartedAt:      &start,
		EndedAt:        &end,
		DurationMS:     5000,
	}
	updated, err := svc.AttachClip(eventID, "clip-ref", meta)
	if err != nil {
		t.Fatalf("attach clip: %v", err)
	}
	got := updated.Feedback[0].Video
	if got == nil {
		t.Fatalf("expected video metadata on event")
	}
	if got.Codec != meta.Codec || !got.HasAudio || !got.PointerOverlay {
		t.Fatalf("unexpected video metadata: %#v", got)
	}
	if got.DurationMS != 5000 || got.RegionW != 320 || got.RegionH != 180 {
		t.Fatalf("unexpected video dimensions/duration metadata: %#v", got)
	}
}

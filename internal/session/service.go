package session

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNoSession         = errors.New("no active session")
	ErrReviewNotApproved = errors.New("session not approved")
)

type Service struct {
	mu               sync.RWMutex
	current          *Session
	history          []*Session
	sequence         int64
	lastApprovedPack *CanonicalPackage
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Bootstrap(sessions []*Session, approved *CanonicalPackage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(sessions) == 0 {
		s.current = nil
		s.history = nil
		s.lastApprovedPack = nil
		s.sequence = 0
		return
	}

	ordered := make([]*Session, 0, len(sessions))
	for _, sess := range sessions {
		if sess == nil {
			continue
		}
		ordered = append(ordered, cloneSession(sess))
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].UpdatedAt.Before(ordered[j].UpdatedAt)
	})

	var seq int64
	for _, sess := range ordered {
		seq = maxSeq(seq, parseSequence(sess.ID))
		for _, evt := range sess.Feedback {
			seq = maxSeq(seq, parseSequence(evt.ID))
		}
		for _, note := range sess.ReviewNotes {
			seq = maxSeq(seq, parseSequence(note.ID))
		}
	}

	s.history = ordered
	s.current = cloneSession(ordered[len(ordered)-1])
	s.sequence = seq
	s.lastApprovedPack = nil
	if s.current != nil && s.current.Approved && approved != nil && approved.SessionID == s.current.ID {
		s.lastApprovedPack = cloneCanonicalPackage(approved)
	}
}

func (s *Service) Start(targetWindow, targetURL string) *Session {
	return s.StartWithMeta(targetWindow, targetURL, "", "", "")
}

func (s *Service) StartWithMeta(targetWindow, targetURL, profile, environment, buildID string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.sequence++
	sess := &Session{
		ID:                 fmt.Sprintf("sess-%d", s.sequence),
		Profile:            strings.TrimSpace(profile),
		Environment:        strings.TrimSpace(environment),
		BuildID:            strings.TrimSpace(buildID),
		ReviewMode:         "",
		TargetWindow:       targetWindow,
		TargetURL:          targetURL,
		Status:             StatusActive,
		CreatedAt:          now,
		UpdatedAt:          now,
		ApprovalRequired:   true,
		Approved:           false,
		CaptureInputValues: true,
		Feedback:           []FeedbackEvt{},
		ReviewNotes:        []ReviewNote{},
	}
	s.current = sess
	s.history = append(s.history, sess)
	s.lastApprovedPack = nil
	return cloneSession(sess)
}

func (s *Service) SetReviewMode(mode string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, ErrNoSession
	}
	mode = normalizeReviewMode(mode)
	s.current.ReviewMode = mode
	s.current.Approved = false
	s.current.UpdatedAt = time.Now().UTC()
	s.lastApprovedPack = nil
	return cloneSession(s.current), nil
}

func (s *Service) UpdateTargetContext(targetWindow, targetURL string) (*Session, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, false, ErrNoSession
	}
	nextWindow := strings.TrimSpace(targetWindow)
	nextURL := strings.TrimSpace(targetURL)
	changed := false
	if nextWindow != "" && nextWindow != s.current.TargetWindow {
		s.current.TargetWindow = nextWindow
		changed = true
	}
	if nextURL != "" && nextURL != s.current.TargetURL {
		s.current.TargetURL = nextURL
		changed = true
	}
	if !changed {
		return cloneSession(s.current), false, nil
	}
	s.current.UpdatedAt = time.Now().UTC()
	return cloneSession(s.current), true, nil
}

func (s *Service) AddReviewNote(author, note string) (*Session, *ReviewNote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, nil, ErrNoSession
	}
	author = strings.TrimSpace(author)
	note = strings.TrimSpace(note)
	if author == "" || note == "" {
		return nil, nil, fmt.Errorf("review note author and note are required")
	}
	s.sequence++
	entry := ReviewNote{
		ID:        fmt.Sprintf("rev-%d", s.sequence),
		Author:    author,
		Note:      note,
		CreatedAt: time.Now().UTC(),
	}
	s.current.ReviewNotes = append(s.current.ReviewNotes, entry)
	s.current.UpdatedAt = time.Now().UTC()
	return cloneSession(s.current), &entry, nil
}

func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return ErrNoSession
	}
	s.current.Status = StatusStopped
	s.current.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return ErrNoSession
	}
	s.current.Status = StatusPaused
	s.current.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return ErrNoSession
	}
	s.current.Status = StatusActive
	s.current.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) Current() *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSession(s.current)
}

func (s *Service) History() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Session, 0, len(s.history))
	for _, sess := range s.history {
		out = append(out, cloneSession(sess))
	}
	return out
}

func (s *Service) DropCurrent() *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	dropped := cloneSession(s.current)
	s.current = nil
	s.lastApprovedPack = nil
	return dropped
}

func (s *Service) ResetAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = nil
	s.history = nil
	s.lastApprovedPack = nil
	s.sequence = 0
}

func (s *Service) AddFeedback(evt FeedbackEvt) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, ErrNoSession
	}
	s.current.UpdatedAt = time.Now().UTC()
	s.sequence++
	evt.ID = fmt.Sprintf("evt-%d", s.sequence)
	evt.Timestamp = time.Now().UTC()
	if evt.EndTime.IsZero() {
		evt.EndTime = evt.Timestamp
	}
	if evt.StartTime.IsZero() {
		evt.StartTime = evt.Timestamp
	}
	if evt.StartTime.After(evt.EndTime) {
		evt.StartTime = evt.EndTime
	}
	evt.Disposition = DispositionPending
	s.current.Approved = false
	s.lastApprovedPack = nil
	s.current.Feedback = append(s.current.Feedback, evt)
	return cloneSession(s.current), nil
}

func (s *Service) SetCaptureInputValues(enabled bool) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, ErrNoSession
	}
	s.current.CaptureInputValues = enabled
	s.current.UpdatedAt = time.Now().UTC()
	return cloneSession(s.current), nil
}

func (s *Service) DiscardLastFeedback() (*Session, *FeedbackEvt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, nil, ErrNoSession
	}
	if len(s.current.Feedback) == 0 {
		return cloneSession(s.current), nil, nil
	}
	lastIdx := len(s.current.Feedback) - 1
	discarded := s.current.Feedback[lastIdx]
	discarded.Disposition = DispositionDiscarded
	s.current.Feedback = s.current.Feedback[:lastIdx]
	s.current.Approved = false
	s.current.UpdatedAt = time.Now().UTC()
	s.lastApprovedPack = nil
	return cloneSession(s.current), &discarded, nil
}

func (s *Service) UpdateFeedbackText(eventID, text string) (*Session, *FeedbackEvt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, nil, ErrNoSession
	}
	eventID = strings.TrimSpace(eventID)
	text = strings.TrimSpace(text)
	if eventID == "" {
		return nil, nil, fmt.Errorf("event_id is required")
	}
	if text == "" {
		return nil, nil, fmt.Errorf("text is required")
	}
	for i := range s.current.Feedback {
		if s.current.Feedback[i].ID != eventID {
			continue
		}
		s.current.Feedback[i].RawTranscript = text
		s.current.Feedback[i].NormalizedText = text
		s.current.Feedback[i].ApprovedInterpret = ""
		if s.current.Feedback[i].Disposition == DispositionApproved {
			s.current.Feedback[i].Disposition = DispositionPending
		}
		s.current.Approved = false
		s.current.UpdatedAt = time.Now().UTC()
		s.lastApprovedPack = nil
		updated := s.current.Feedback[i]
		return cloneSession(s.current), &updated, nil
	}
	return nil, nil, fmt.Errorf("feedback event not found: %s", eventID)
}

func (s *Service) DeleteFeedback(eventID string) (*Session, *FeedbackEvt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, nil, ErrNoSession
	}
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return nil, nil, fmt.Errorf("event_id is required")
	}
	for i := range s.current.Feedback {
		if s.current.Feedback[i].ID != eventID {
			continue
		}
		deleted := s.current.Feedback[i]
		deleted.Disposition = DispositionDiscarded
		s.current.Feedback = append(s.current.Feedback[:i], s.current.Feedback[i+1:]...)
		s.current.Approved = false
		s.current.UpdatedAt = time.Now().UTC()
		s.lastApprovedPack = nil
		return cloneSession(s.current), &deleted, nil
	}
	return nil, nil, fmt.Errorf("feedback event not found: %s", eventID)
}

func (s *Service) Approve(summary string) (*CanonicalPackage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, ErrNoSession
	}
	s.current.Approved = true
	s.current.UpdatedAt = time.Now().UTC()

	changeCount := 0
	for _, evt := range s.current.Feedback {
		if evt.Disposition == DispositionQueued || evt.Disposition == DispositionSubmitted || evt.Disposition == DispositionDiscarded {
			continue
		}
		changeCount++
	}

	cp := &CanonicalPackage{
		SessionID: s.current.ID,
		SessionMeta: SessionMeta{
			Profile:      s.current.Profile,
			ReviewMode:   s.current.ReviewMode,
			TargetWindow: s.current.TargetWindow,
			TargetURL:    s.current.TargetURL,
			Environment:  s.current.Environment,
			BuildID:      s.current.BuildID,
		},
		ChangeRequests: make([]ChangeReq, 0, changeCount),
		Artifacts:      []ArtifactRef{},
		GeneratedAt:    time.Now().UTC(),
	}
	summaries := make([]string, 0, changeCount)
	for i, evt := range s.current.Feedback {
		if evt.Disposition == DispositionQueued || evt.Disposition == DispositionSubmitted || evt.Disposition == DispositionDiscarded {
			continue
		}
		reqSummary := evt.NormalizedText
		if reqSummary == "" {
			reqSummary = evt.RawTranscript
		}
		if summary != "" {
			reqSummary = summary
		}
		s.current.Feedback[i].Disposition = DispositionApproved
		s.current.Feedback[i].ApprovedInterpret = reqSummary
		category := inferCategory(reqSummary, evt, s.current.ReviewMode)
		priority := inferPriority(reqSummary, category)
		ambiguities := inferAmbiguities(reqSummary, evt.VisualTargetRef)
		changeAssumptions := inferAssumptions(evt)
		affected := inferAffectedArea(evt)
		summaries = append(summaries, reqSummary)
		cp.ChangeRequests = append(cp.ChangeRequests, ChangeReq{
			EventID:         evt.ID,
			Summary:         reqSummary,
			Category:        category,
			Priority:        priority,
			ReviewMode:      evt.ReviewMode,
			ExperimentID:    evt.ExperimentID,
			Variant:         evt.Variant,
			Assumptions:     changeAssumptions,
			Ambiguities:     ambiguities,
			AffectedArea:    affected,
			Pointer:         clonePointerCtx(evt.Pointer),
			PointerPath:     clonePointerSamples(evt.PointerPath),
			VisualTargetRef: evt.VisualTargetRef,
			Replay:          cloneReplayBundle(evt.Replay),
		})
		if evt.ScreenshotRef != "" {
			cp.Artifacts = append(cp.Artifacts, ArtifactRef{Kind: "screenshot", Ref: evt.ScreenshotRef, EventID: evt.ID})
		}
		if evt.AudioRef != "" {
			cp.Artifacts = append(cp.Artifacts, ArtifactRef{Kind: "audio", Ref: evt.AudioRef, EventID: evt.ID})
		}
		if evt.VideoClipRef != "" {
			cp.Artifacts = append(cp.Artifacts, ArtifactRef{Kind: "video", Ref: evt.VideoClipRef, EventID: evt.ID})
		}
	}
	cp.Summary = buildPackageSummary(summaries)
	s.lastApprovedPack = cp
	return cp, nil
}

func (s *Service) ReplaceApprovedPackage(pkg *CanonicalPackage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return ErrNoSession
	}
	if !s.current.Approved {
		return ErrReviewNotApproved
	}
	if pkg == nil {
		return fmt.Errorf("approved package is required")
	}
	s.lastApprovedPack = cloneCanonicalPackage(pkg)
	return nil
}

func (s *Service) ApprovedPackage() (*CanonicalPackage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.current == nil {
		return nil, ErrNoSession
	}
	if !s.current.Approved || s.lastApprovedPack == nil {
		return nil, ErrReviewNotApproved
	}
	return cloneCanonicalPackage(s.lastApprovedPack), nil
}

func (s *Service) ReserveApprovedPackage() (*CanonicalPackage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, ErrNoSession
	}
	if !s.current.Approved || s.lastApprovedPack == nil {
		return nil, ErrReviewNotApproved
	}
	cp := cloneCanonicalPackage(s.lastApprovedPack)
	queuedIDs := make(map[string]struct{}, len(cp.ChangeRequests))
	for _, req := range cp.ChangeRequests {
		if id := strings.TrimSpace(req.EventID); id != "" {
			queuedIDs[id] = struct{}{}
		}
	}
	for i := range s.current.Feedback {
		if _, ok := queuedIDs[s.current.Feedback[i].ID]; !ok {
			continue
		}
		if s.current.Feedback[i].Disposition == DispositionPending || s.current.Feedback[i].Disposition == DispositionApproved {
			s.current.Feedback[i].Disposition = DispositionQueued
		}
	}
	// Approval is consumed when a submission is queued; each submit requires explicit approval.
	s.current.Approved = false
	s.current.UpdatedAt = time.Now().UTC()
	s.lastApprovedPack = nil
	return cp, nil
}

func (s *Service) AttachClipRef(eventID, clipRef string) (*Session, error) {
	return s.AttachClip(eventID, clipRef, nil)
}

func (s *Service) AttachClip(eventID, clipRef string, meta *VideoMetadata) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil, ErrNoSession
	}
	for i := range s.current.Feedback {
		if s.current.Feedback[i].ID == eventID {
			s.current.Feedback[i].VideoClipRef = clipRef
			if meta != nil {
				copyMeta := *meta
				if meta.StartedAt != nil {
					v := *meta.StartedAt
					copyMeta.StartedAt = &v
				}
				if meta.EndedAt != nil {
					v := *meta.EndedAt
					copyMeta.EndedAt = &v
				}
				s.current.Feedback[i].Video = &copyMeta
			}
			s.current.UpdatedAt = time.Now().UTC()
			s.current.Approved = false
			s.lastApprovedPack = nil
			return cloneSession(s.current), nil
		}
	}
	return nil, fmt.Errorf("feedback event not found: %s", eventID)
}

func (s *Service) MarkSubmitted(versionRef string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return ErrNoSession
	}
	if !s.current.Approved {
		return ErrReviewNotApproved
	}
	s.markSubmittedLocked(versionRef, nil)
	return nil
}

func (s *Service) MarkSubmittedFor(versionRef string, eventIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return ErrNoSession
	}
	s.markSubmittedLocked(versionRef, eventIDs)
	return nil
}

func (s *Service) markSubmittedLocked(versionRef string, eventIDs []string) {
	idSet := map[string]struct{}{}
	for _, id := range eventIDs {
		if strings.TrimSpace(id) == "" {
			continue
		}
		idSet[id] = struct{}{}
	}
	filtered := len(idSet) > 0

	s.current.VersionReference = versionRef
	s.current.Status = StatusSubmitted
	s.current.UpdatedAt = time.Now().UTC()
	// Approval is consumed by a successful submission; next submit requires explicit re-approval.
	s.current.Approved = false
	s.lastApprovedPack = nil
	for i := range s.current.Feedback {
		if filtered {
			if _, ok := idSet[s.current.Feedback[i].ID]; !ok {
				continue
			}
		}
		if s.current.Feedback[i].Disposition == DispositionPending || s.current.Feedback[i].Disposition == DispositionApproved || s.current.Feedback[i].Disposition == DispositionQueued {
			s.current.Feedback[i].Disposition = DispositionSubmitted
		}
	}
}

func cloneSession(in *Session) *Session {
	if in == nil {
		return nil
	}
	out := *in
	out.Feedback = make([]FeedbackEvt, len(in.Feedback))
	for i, evt := range in.Feedback {
		out.Feedback[i] = evt
		out.Feedback[i].Pointer = clonePointerCtx(evt.Pointer)
		out.Feedback[i].PointerPath = clonePointerSamples(evt.PointerPath)
		out.Feedback[i].Replay = cloneReplayBundle(evt.Replay)
		out.Feedback[i].LaserPath = clonePointerSamples(evt.LaserPath)
		if evt.Video != nil {
			v := *evt.Video
			if evt.Video.StartedAt != nil {
				ts := *evt.Video.StartedAt
				v.StartedAt = &ts
			}
			if evt.Video.EndedAt != nil {
				ts := *evt.Video.EndedAt
				v.EndedAt = &ts
			}
			out.Feedback[i].Video = &v
		}
	}
	out.ReviewNotes = append([]ReviewNote(nil), in.ReviewNotes...)
	return &out
}

func cloneCanonicalPackage(in *CanonicalPackage) *CanonicalPackage {
	if in == nil {
		return nil
	}
	cp := *in
	cp.ChangeRequests = make([]ChangeReq, len(in.ChangeRequests))
	for i, req := range in.ChangeRequests {
		cp.ChangeRequests[i] = req
		cp.ChangeRequests[i].Pointer = clonePointerCtx(req.Pointer)
		cp.ChangeRequests[i].PointerPath = clonePointerSamples(req.PointerPath)
		cp.ChangeRequests[i].Replay = cloneReplayBundle(req.Replay)
		cp.ChangeRequests[i].Assumptions = append([]string(nil), req.Assumptions...)
		cp.ChangeRequests[i].Ambiguities = append([]string(nil), req.Ambiguities...)
		cp.ChangeRequests[i].AffectedArea = append([]string(nil), req.AffectedArea...)
	}
	cp.Artifacts = append([]ArtifactRef(nil), in.Artifacts...)
	return &cp
}

func clonePointerCtx(in PointerCtx) PointerCtx {
	out := in
	out.DOM = cloneDOMInspection(in.DOM)
	out.Console = cloneConsoleEntries(in.Console)
	out.Network = cloneNetworkEntries(in.Network)
	return out
}

func clonePointerSamples(in []PointerSample) []PointerSample {
	return append([]PointerSample(nil), in...)
}

func cloneDOMInspection(in *DOMInspection) *DOMInspection {
	if in == nil {
		return nil
	}
	out := *in
	if len(in.Attributes) > 0 {
		out.Attributes = make(map[string]string, len(in.Attributes))
		for k, v := range in.Attributes {
			out.Attributes[k] = v
		}
	}
	return &out
}

func cloneConsoleEntries(in []ConsoleEntry) []ConsoleEntry {
	return append([]ConsoleEntry(nil), in...)
}

func cloneNetworkEntries(in []NetworkEntry) []NetworkEntry {
	return append([]NetworkEntry(nil), in...)
}

func cloneReplayBundle(in *ReplayBundle) *ReplayBundle {
	if in == nil {
		return nil
	}
	out := *in
	out.PointerPath = clonePointerSamples(in.PointerPath)
	out.Steps = cloneReplaySteps(in.Steps)
	out.DOM = cloneDOMInspection(in.DOM)
	out.Console = cloneConsoleEntries(in.Console)
	out.Network = cloneNetworkEntries(in.Network)
	out.Exports = append([]ReplayExport(nil), in.Exports...)
	return &out
}

func cloneReplaySteps(in []ReplayStep) []ReplayStep {
	if len(in) == 0 {
		return nil
	}
	out := make([]ReplayStep, len(in))
	for i, step := range in {
		out[i] = step
		out[i].Modifiers = append([]string(nil), step.Modifiers...)
		out[i].DOM = cloneDOMInspection(step.DOM)
	}
	return out
}

func parseSequence(id string) int64 {
	id = strings.TrimSpace(id)
	if id == "" {
		return 0
	}
	idx := strings.LastIndexByte(id, '-')
	if idx < 0 || idx >= len(id)-1 {
		return 0
	}
	var seq int64
	for _, r := range id[idx+1:] {
		if r < '0' || r > '9' {
			return 0
		}
		seq = (seq * 10) + int64(r-'0')
	}
	return seq
}

func maxSeq(a, b int64) int64 {
	if b > a {
		return b
	}
	return a
}

func buildPackageSummary(summaries []string) string {
	if len(summaries) == 0 {
		return ""
	}
	if len(summaries) == 1 {
		return summaries[0]
	}
	return fmt.Sprintf("%d requested changes: %s", len(summaries), strings.Join(summaries, "; "))
}

func inferCategory(summary string, evt FeedbackEvt, sessionReviewMode string) string {
	if mode := normalizeReviewMode(evt.ReviewMode); mode == "accessibility" {
		return "accessibility"
	}
	if normalizeReviewMode(sessionReviewMode) == "accessibility" {
		return "accessibility"
	}
	s := strings.ToLower(summary)
	switch {
	case containsAny(s, "a11y", "accessibility", "screen reader", "keyboard", "contrast", "aria"):
		return "accessibility"
	case containsAny(s, "slow", "lag", "latency", "performance", "jank"):
		return "performance_perceived_responsiveness"
	case containsAny(s, "bug", "broken", "error", "crash", "fails", "doesn't work", "does not work"):
		return "bug_defect"
	case containsAny(s, "copy", "wording", "text", "label", "spelling", "typo"):
		return "content_copy"
	case containsAny(s, "workflow", "step", "flow", "journey"):
		return "workflow"
	case containsAny(s, "click", "hover", "focus", "transition", "interaction", "modal"):
		return "interaction_behavior"
	case containsAny(s, "layout", "spacing", "align", "padding", "margin", "position"):
		return "layout"
	case containsAny(s, "color", "font", "style", "theme", "visual", "design"):
		return "visual_design"
	default:
		base := "unclear_needs_review"
		switch triagePolicy() {
		case "accessibility_first":
			if containsAny(s, "read", "contrast", "keyboard", "screen reader") {
				return "accessibility"
			}
		case "workflow_first":
			if containsAny(s, "step", "flow", "journey", "path") {
				return "workflow"
			}
		case "performance_first":
			if containsAny(s, "delay", "slow", "lag", "load") {
				return "performance_perceived_responsiveness"
			}
		}
		return base
	}
}

func inferPriority(summary, category string) string {
	s := strings.ToLower(summary)
	policy := triagePolicy()
	switch {
	case containsAny(s, "critical", "blocker", "urgent", "cannot", "can't", "security"):
		return "high"
	case containsAny(s, "minor", "nit", "nice to have", "polish"):
		return "low"
	case policy == "accessibility_first" && category == "accessibility":
		return "high"
	case policy == "performance_first" && category == "performance_perceived_responsiveness":
		return "high"
	case policy == "bug_first" && category == "bug_defect":
		return "high"
	default:
		return "medium"
	}
}

func triagePolicy() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("KNIT_TRIAGE_POLICY")))
	switch v {
	case "default", "accessibility_first", "workflow_first", "performance_first", "bug_first":
		return v
	default:
		return "default"
	}
}

func normalizeReviewMode(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "", "general":
		return ""
	case "accessibility":
		return "accessibility"
	default:
		return ""
	}
}

func inferAmbiguities(summary, target string) []string {
	ambiguities := []string{}
	s := strings.ToLower(summary)
	if containsAny(s, "this", "that", "it", "here") && strings.TrimSpace(target) == "" {
		ambiguities = append(ambiguities, "Pronoun reference without an identified visual target.")
	}
	return ambiguities
}

func inferAssumptions(evt FeedbackEvt) []string {
	assumptions := []string{}
	if strings.TrimSpace(evt.VisualTargetRef) == "" && strings.TrimSpace(evt.Pointer.Window) != "" {
		assumptions = append(assumptions, "Assumed target scope from active pointer window due missing UI element reference.")
	}
	if evt.ScreenshotRef == "" {
		assumptions = append(assumptions, "No screenshot available for this event.")
	}
	return assumptions
}

func inferAffectedArea(evt FeedbackEvt) []string {
	areas := []string{}
	if strings.TrimSpace(evt.VisualTargetRef) != "" {
		areas = append(areas, evt.VisualTargetRef)
	}
	if strings.TrimSpace(evt.Pointer.Window) != "" {
		areas = append(areas, evt.Pointer.Window)
	}
	return areas
}

func containsAny(text string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(text, n) {
			return true
		}
	}
	return false
}

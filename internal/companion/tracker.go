package companion

import (
	"sync"
	"time"

	"knit/internal/session"
)

type PointerEvent struct {
	SessionID       string                 `json:"session_id"`
	X               int                    `json:"x"`
	Y               int                    `json:"y"`
	HoverDurationMS int64                  `json:"hover_duration_ms"`
	EventType       string                 `json:"event_type"`
	Window          string                 `json:"window"`
	URL             string                 `json:"url,omitempty"`
	Route           string                 `json:"route,omitempty"`
	TargetTag       string                 `json:"target_tag,omitempty"`
	TargetID        string                 `json:"target_id,omitempty"`
	TargetTestID    string                 `json:"target_test_id,omitempty"`
	TargetRole      string                 `json:"target_role,omitempty"`
	TargetLabel     string                 `json:"target_label,omitempty"`
	TargetSelector  string                 `json:"target_selector,omitempty"`
	DOM             *session.DOMInspection `json:"dom,omitempty"`
	Console         []session.ConsoleEntry `json:"console,omitempty"`
	Network         []session.NetworkEntry `json:"network,omitempty"`
	MouseButton     int                    `json:"mouse_button,omitempty"`
	ClickCount      int                    `json:"click_count,omitempty"`
	Key             string                 `json:"key,omitempty"`
	Code            string                 `json:"code,omitempty"`
	Modifiers       []string               `json:"modifiers,omitempty"`
	InputType       string                 `json:"input_type,omitempty"`
	Value           string                 `json:"value,omitempty"`
	ValueCaptured   bool                   `json:"value_captured,omitempty"`
	ValueRedacted   bool                   `json:"value_redacted,omitempty"`
	ScrollDX        float64                `json:"scroll_dx,omitempty"`
	ScrollDY        float64                `json:"scroll_dy,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
}

type Tracker struct {
	mu      sync.RWMutex
	latest  map[string]PointerEvent
	history map[string][]PointerEvent
	replay  map[string][]session.ReplayStep
	max     int
}

func NewTracker(maxEvents int) *Tracker {
	if maxEvents < 1 {
		maxEvents = 300
	}
	return &Tracker{
		latest:  map[string]PointerEvent{},
		history: map[string][]PointerEvent{},
		replay:  map[string][]session.ReplayStep{},
		max:     maxEvents,
	}
}

func (t *Tracker) Add(evt PointerEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	t.latest[evt.SessionID] = evt
	h := append(t.history[evt.SessionID], evt)
	if len(h) > t.max {
		h = h[len(h)-t.max:]
	}
	t.history[evt.SessionID] = h
	if replayStep, ok := replayStepFromEvent(evt); ok {
		replay := append(t.replay[evt.SessionID], replayStep)
		replayMax := t.max * 2
		if replayMax < 1 {
			replayMax = 1
		}
		if len(replay) > replayMax {
			replay = replay[len(replay)-replayMax:]
		}
		t.replay[evt.SessionID] = replay
	}
}

func (t *Tracker) Snapshot(sessionID string) (session.PointerCtx, []session.PointerSample) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	latest := t.latest[sessionID]
	ctx := session.PointerCtx{
		X:               latest.X,
		Y:               latest.Y,
		HoverDurationMS: latest.HoverDurationMS,
		Window:          latest.Window,
		URL:             latest.URL,
		Route:           latest.Route,
		TargetTag:       latest.TargetTag,
		TargetID:        latest.TargetID,
		TargetTestID:    latest.TargetTestID,
		TargetLabel:     latest.TargetLabel,
		TargetSelector:  latest.TargetSelector,
		DOM:             cloneDOMInspection(latest.DOM),
		Console:         append([]session.ConsoleEntry(nil), latest.Console...),
		Network:         append([]session.NetworkEntry(nil), latest.Network...),
	}

	h := t.history[sessionID]
	path := make([]session.PointerSample, 0, len(h))
	for _, v := range h {
		path = append(path, session.PointerSample{
			X:         v.X,
			Y:         v.Y,
			EventType: v.EventType,
			ScrollDX:  v.ScrollDX,
			ScrollDY:  v.ScrollDY,
			Route:     v.Route,
			Timestamp: v.Timestamp,
		})
	}
	return ctx, path
}

func cloneDOMInspection(in *session.DOMInspection) *session.DOMInspection {
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

func (t *Tracker) ReplaySnapshot(sessionID string) []session.ReplayStep {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return cloneReplaySteps(t.replay[sessionID])
}

func replayStepFromEvent(evt PointerEvent) (session.ReplayStep, bool) {
	eventType := evt.EventType
	switch eventType {
	case "move", "context_sync", "":
		return session.ReplayStep{}, false
	}
	return session.ReplayStep{
		Type:           eventType,
		Timestamp:      evt.Timestamp,
		URL:            evt.URL,
		Route:          evt.Route,
		X:              evt.X,
		Y:              evt.Y,
		ScrollDX:       evt.ScrollDX,
		ScrollDY:       evt.ScrollDY,
		MouseButton:    evt.MouseButton,
		ClickCount:     evt.ClickCount,
		Key:            evt.Key,
		Code:           evt.Code,
		Modifiers:      append([]string(nil), evt.Modifiers...),
		InputType:      evt.InputType,
		Value:          evt.Value,
		ValueCaptured:  evt.ValueCaptured,
		ValueRedacted:  evt.ValueRedacted,
		TargetTag:      evt.TargetTag,
		TargetID:       evt.TargetID,
		TargetTestID:   evt.TargetTestID,
		TargetRole:     evt.TargetRole,
		TargetLabel:    evt.TargetLabel,
		TargetSelector: evt.TargetSelector,
		DOM:            cloneDOMInspection(evt.DOM),
	}, true
}

func cloneReplaySteps(in []session.ReplayStep) []session.ReplayStep {
	if len(in) == 0 {
		return nil
	}
	out := make([]session.ReplayStep, len(in))
	for i, step := range in {
		out[i] = step
		out[i].Modifiers = append([]string(nil), step.Modifiers...)
		out[i].DOM = cloneDOMInspection(step.DOM)
	}
	return out
}

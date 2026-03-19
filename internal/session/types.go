package session

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusStopped   Status = "stopped"
	StatusSubmitted Status = "submitted"
)

type Disposition string

const (
	DispositionPending   Disposition = "pending"
	DispositionApproved  Disposition = "approved"
	DispositionQueued    Disposition = "queued"
	DispositionDiscarded Disposition = "discarded"
	DispositionSubmitted Disposition = "submitted"
)

type Session struct {
	ID                 string        `json:"id"`
	Profile            string        `json:"profile,omitempty"`
	Environment        string        `json:"environment,omitempty"`
	BuildID            string        `json:"build_id,omitempty"`
	ReviewMode         string        `json:"review_mode,omitempty"`
	TargetWindow       string        `json:"target_window"`
	TargetURL          string        `json:"target_url,omitempty"`
	VersionReference   string        `json:"version_reference,omitempty"`
	Status             Status        `json:"status"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	ApprovalRequired   bool          `json:"approval_required"`
	Approved           bool          `json:"approved"`
	CaptureInputValues bool          `json:"capture_input_values,omitempty"`
	Feedback           []FeedbackEvt `json:"feedback"`
	ReviewNotes        []ReviewNote  `json:"review_notes,omitempty"`
}

type FeedbackEvt struct {
	ID                string          `json:"id"`
	Timestamp         time.Time       `json:"timestamp"`
	StartTime         time.Time       `json:"start_time,omitempty"`
	EndTime           time.Time       `json:"end_time,omitempty"`
	RawTranscript     string          `json:"raw_transcript"`
	NormalizedText    string          `json:"normalized_text"`
	Pointer           PointerCtx      `json:"pointer"`
	PointerPath       []PointerSample `json:"pointer_path,omitempty"`
	AudioRef          string          `json:"audio_ref,omitempty"`
	ScreenshotRef     string          `json:"screenshot_ref,omitempty"`
	VideoClipRef      string          `json:"video_clip_ref,omitempty"`
	Video             *VideoMetadata  `json:"video,omitempty"`
	VisualTargetRef   string          `json:"visual_target_ref,omitempty"`
	Replay            *ReplayBundle   `json:"replay,omitempty"`
	Confidence        float64         `json:"confidence"`
	Ambiguity         string          `json:"ambiguity,omitempty"`
	Disposition       Disposition     `json:"disposition"`
	ApprovedInterpret string          `json:"approved_interpretation,omitempty"`
	ReviewMode        string          `json:"review_mode,omitempty"`
	ExperimentID      string          `json:"experiment_id,omitempty"`
	Variant           string          `json:"variant,omitempty"`
	LaserMode         bool            `json:"laser_mode,omitempty"`
	LaserPath         []PointerSample `json:"laser_path,omitempty"`
}

type VideoMetadata struct {
	Scope          string     `json:"scope,omitempty"`
	Window         string     `json:"window,omitempty"`
	RegionX        int        `json:"region_x,omitempty"`
	RegionY        int        `json:"region_y,omitempty"`
	RegionW        int        `json:"region_w,omitempty"`
	RegionH        int        `json:"region_h,omitempty"`
	Codec          string     `json:"codec,omitempty"`
	HasAudio       bool       `json:"has_audio,omitempty"`
	PointerOverlay bool       `json:"pointer_overlay,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	DurationMS     int64      `json:"duration_ms,omitempty"`
}

type ReviewNote struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type PointerCtx struct {
	X               int            `json:"x"`
	Y               int            `json:"y"`
	HoverDurationMS int64          `json:"hover_duration_ms"`
	Window          string         `json:"window"`
	URL             string         `json:"url,omitempty"`
	Route           string         `json:"route,omitempty"`
	TargetTag       string         `json:"target_tag,omitempty"`
	TargetID        string         `json:"target_id,omitempty"`
	TargetTestID    string         `json:"target_test_id,omitempty"`
	TargetLabel     string         `json:"target_label,omitempty"`
	TargetSelector  string         `json:"target_selector,omitempty"`
	DOM             *DOMInspection `json:"dom,omitempty"`
	Console         []ConsoleEntry `json:"console,omitempty"`
	Network         []NetworkEntry `json:"network,omitempty"`
}

type PointerSample struct {
	X         int       `json:"x"`
	Y         int       `json:"y"`
	EventType string    `json:"event_type,omitempty"`
	ScrollDX  float64   `json:"scroll_dx,omitempty"`
	ScrollDY  float64   `json:"scroll_dy,omitempty"`
	Route     string    `json:"route,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type DOMInspection struct {
	Tag         string            `json:"tag,omitempty"`
	ID          string            `json:"id,omitempty"`
	TestID      string            `json:"test_id,omitempty"`
	Role        string            `json:"role,omitempty"`
	Label       string            `json:"label,omitempty"`
	Selector    string            `json:"selector,omitempty"`
	TextPreview string            `json:"text_preview,omitempty"`
	OuterHTML   string            `json:"outer_html,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

type ConsoleEntry struct {
	Level     string    `json:"level,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type NetworkEntry struct {
	Kind       string    `json:"kind,omitempty"`
	Method     string    `json:"method,omitempty"`
	URL        string    `json:"url,omitempty"`
	Status     int       `json:"status,omitempty"`
	OK         bool      `json:"ok,omitempty"`
	DurationMS int64     `json:"duration_ms,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type ReplayBundle struct {
	URL              string          `json:"url,omitempty"`
	Route            string          `json:"route,omitempty"`
	TargetTag        string          `json:"target_tag,omitempty"`
	TargetID         string          `json:"target_id,omitempty"`
	TargetTestID     string          `json:"target_test_id,omitempty"`
	TargetLabel      string          `json:"target_label,omitempty"`
	TargetSelector   string          `json:"target_selector,omitempty"`
	ValueCaptureMode string          `json:"value_capture_mode,omitempty"`
	PointerPath      []PointerSample `json:"pointer_path,omitempty"`
	Steps            []ReplayStep    `json:"steps,omitempty"`
	DOM              *DOMInspection  `json:"dom,omitempty"`
	Console          []ConsoleEntry  `json:"console,omitempty"`
	Network          []NetworkEntry  `json:"network,omitempty"`
	PlaywrightScript string          `json:"playwright_script,omitempty"`
	Exports          []ReplayExport  `json:"exports,omitempty"`
}

type ReplayStep struct {
	Type           string         `json:"type,omitempty"`
	Timestamp      time.Time      `json:"timestamp"`
	URL            string         `json:"url,omitempty"`
	Route          string         `json:"route,omitempty"`
	X              int            `json:"x,omitempty"`
	Y              int            `json:"y,omitempty"`
	ScrollDX       float64        `json:"scroll_dx,omitempty"`
	ScrollDY       float64        `json:"scroll_dy,omitempty"`
	MouseButton    int            `json:"mouse_button,omitempty"`
	ClickCount     int            `json:"click_count,omitempty"`
	Key            string         `json:"key,omitempty"`
	Code           string         `json:"code,omitempty"`
	Modifiers      []string       `json:"modifiers,omitempty"`
	InputType      string         `json:"input_type,omitempty"`
	Value          string         `json:"value,omitempty"`
	ValueCaptured  bool           `json:"value_captured,omitempty"`
	ValueRedacted  bool           `json:"value_redacted,omitempty"`
	TargetTag      string         `json:"target_tag,omitempty"`
	TargetID       string         `json:"target_id,omitempty"`
	TargetTestID   string         `json:"target_test_id,omitempty"`
	TargetRole     string         `json:"target_role,omitempty"`
	TargetLabel    string         `json:"target_label,omitempty"`
	TargetSelector string         `json:"target_selector,omitempty"`
	DOM            *DOMInspection `json:"dom,omitempty"`
}

type ReplayExport struct {
	Kind     string `json:"kind"`
	Ref      string `json:"ref,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type CanonicalPackage struct {
	SessionID      string        `json:"session_id"`
	Summary        string        `json:"summary,omitempty"`
	SessionMeta    SessionMeta   `json:"session_meta"`
	ChangeRequests []ChangeReq   `json:"change_requests"`
	Artifacts      []ArtifactRef `json:"artifacts"`
	GeneratedAt    time.Time     `json:"generated_at"`
}

func (p *CanonicalPackage) normalizeSlices() {
	if p == nil {
		return
	}
	if p.ChangeRequests == nil {
		p.ChangeRequests = []ChangeReq{}
	}
	if p.Artifacts == nil {
		p.Artifacts = []ArtifactRef{}
	}
}

func (p *CanonicalPackage) UnmarshalJSON(data []byte) error {
	type canonicalPackageAlias CanonicalPackage
	var decoded canonicalPackageAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*p = CanonicalPackage(decoded)
	p.normalizeSlices()
	return nil
}

func (p CanonicalPackage) MarshalJSON() ([]byte, error) {
	type canonicalPackageAlias CanonicalPackage
	p.normalizeSlices()
	return json.Marshal(canonicalPackageAlias(p))
}

type SessionMeta struct {
	Profile      string `json:"profile,omitempty"`
	ReviewMode   string `json:"review_mode,omitempty"`
	TargetWindow string `json:"target_window"`
	TargetURL    string `json:"target_url,omitempty"`
	Environment  string `json:"environment,omitempty"`
	BuildID      string `json:"build_id,omitempty"`
}

type ChangeReq struct {
	EventID         string          `json:"event_id"`
	Summary         string          `json:"summary"`
	Category        string          `json:"category"`
	Priority        string          `json:"priority"`
	ReviewMode      string          `json:"review_mode,omitempty"`
	ExperimentID    string          `json:"experiment_id,omitempty"`
	Variant         string          `json:"variant,omitempty"`
	Assumptions     []string        `json:"assumptions,omitempty"`
	Ambiguities     []string        `json:"ambiguities,omitempty"`
	AffectedArea    []string        `json:"affected_area,omitempty"`
	Pointer         PointerCtx      `json:"pointer,omitempty"`
	PointerPath     []PointerSample `json:"pointer_path,omitempty"`
	VisualTargetRef string          `json:"visual_target_ref,omitempty"`
	Replay          *ReplayBundle   `json:"replay,omitempty"`
}

type ArtifactRef struct {
	Kind               string `json:"kind"`
	Ref                string `json:"ref"`
	EventID            string `json:"event_id,omitempty"`
	MIMEType           string `json:"mime_type,omitempty"`
	SizeBytes          int64  `json:"size_bytes,omitempty"`
	InlineDataURL      string `json:"inline_data_url,omitempty"`
	TransmissionStatus string `json:"transmission_status,omitempty"`
	TransmissionNote   string `json:"transmission_note,omitempty"`
}

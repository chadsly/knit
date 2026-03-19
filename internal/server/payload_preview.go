package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"knit/internal/agents"
	"knit/internal/session"
)

type payloadPreviewResponse struct {
	Provider string                 `json:"provider"`
	Payload  any                    `json:"payload"`
	Preview  renderedPayloadPreview `json:"preview"`
}

type renderedPayloadPreview struct {
	Summary            string                `json:"summary,omitempty"`
	ChangeRequestCount int                   `json:"change_request_count"`
	IntentProfile      string                `json:"intent_profile,omitempty"`
	IntentLabel        string                `json:"intent_label,omitempty"`
	InstructionText    string                `json:"instruction_text,omitempty"`
	CustomInstructions string                `json:"custom_instructions,omitempty"`
	Notes              []renderedPreviewNote `json:"notes"`
	Warnings           []string              `json:"warnings,omitempty"`
	Disclosure         renderedDisclosure    `json:"disclosure"`
}

type renderedDisclosure struct {
	Destination        string `json:"destination,omitempty"`
	RequestTextCount   int    `json:"request_text_count"`
	TypedValuesStatus  string `json:"typed_values_status,omitempty"`
	ScreenshotsSent    int    `json:"screenshots_sent,omitempty"`
	ScreenshotsOmitted int    `json:"screenshots_omitted,omitempty"`
	VideosSent         int    `json:"videos_sent,omitempty"`
	VideosOmitted      int    `json:"videos_omitted,omitempty"`
	AudioSent          int    `json:"audio_sent,omitempty"`
	AudioOmitted       int    `json:"audio_omitted,omitempty"`
}

type renderedPreviewNote struct {
	EventID                string   `json:"event_id"`
	Text                   string   `json:"text"`
	Target                 string   `json:"target,omitempty"`
	ReviewMode             string   `json:"review_mode,omitempty"`
	DOMSummary             string   `json:"dom_summary,omitempty"`
	Console                []string `json:"console,omitempty"`
	Network                []string `json:"network,omitempty"`
	PointerEventCount      int      `json:"pointer_event_count,omitempty"`
	ReplayStepCount        int      `json:"replay_step_count,omitempty"`
	ReplayValueMode        string   `json:"replay_value_mode,omitempty"`
	ReplaySteps            []string `json:"replay_steps,omitempty"`
	PlaywrightScript       string   `json:"playwright_script,omitempty"`
	ScreenshotDataURL      string   `json:"screenshot_data_url,omitempty"`
	VideoDataURL           string   `json:"video_data_url,omitempty"`
	AudioDataURL           string   `json:"audio_data_url,omitempty"`
	HasAudio               bool     `json:"has_audio,omitempty"`
	HasScreenshot          bool     `json:"has_screenshot,omitempty"`
	HasVideo               bool     `json:"has_video,omitempty"`
	VideoDurationMS        int64    `json:"video_duration_ms,omitempty"`
	VideoCodec             string   `json:"video_codec,omitempty"`
	VideoScope             string   `json:"video_scope,omitempty"`
	VideoWindow            string   `json:"video_window,omitempty"`
	VideoHasAudio          bool     `json:"video_has_audio,omitempty"`
	VideoPointerOverlay    bool     `json:"video_pointer_overlay,omitempty"`
	VideoTransmissionState string   `json:"video_transmission_status,omitempty"`
	VideoTransmissionNote  string   `json:"video_transmission_note,omitempty"`
	VideoSizeBytes         int64    `json:"video_size_bytes,omitempty"`
	VideoSendLimitBytes    int64    `json:"video_send_limit_bytes,omitempty"`
}

func (s *Server) buildRenderedPayloadPreview(pkg *session.CanonicalPackage, curr *session.Session, provider string, intent agents.DeliveryIntent) renderedPayloadPreview {
	intent = agents.NormalizeDeliveryIntent(intent)
	preview := renderedPayloadPreview{
		Summary:            strings.TrimSpace(pkg.Summary),
		ChangeRequestCount: len(pkg.ChangeRequests),
		IntentProfile:      intent.Profile,
		IntentLabel:        intent.Label(),
		InstructionText:    intent.InstructionText,
		CustomInstructions: intent.CustomInstructions,
		Notes:              make([]renderedPreviewNote, 0, len(pkg.ChangeRequests)),
		Disclosure: renderedDisclosure{
			Destination:      strings.TrimSpace(provider),
			RequestTextCount: len(pkg.ChangeRequests),
		},
	}
	preview.Disclosure.TypedValuesStatus = previewTypedValuesStatus(pkg)
	applyArtifactDisclosure(&preview.Disclosure, pkg)
	eventByID := make(map[string]session.FeedbackEvt, len(curr.Feedback))
	for _, evt := range curr.Feedback {
		eventByID[evt.ID] = evt
	}
	artifactByEventKind := make(map[string]session.ArtifactRef, len(pkg.Artifacts))
	for _, artifact := range pkg.Artifacts {
		key := strings.TrimSpace(artifact.EventID) + ":" + strings.TrimSpace(artifact.Kind)
		if key == ":" {
			continue
		}
		artifactByEventKind[key] = artifact
	}
	for _, change := range pkg.ChangeRequests {
		evt, ok := eventByID[change.EventID]
		if !ok {
			preview.Warnings = append(preview.Warnings, fmt.Sprintf("%s event unavailable in current session", change.EventID))
			continue
		}
		text := strings.TrimSpace(evt.ApprovedInterpret)
		if candidate := strings.TrimSpace(change.Summary); candidate != "" {
			text = candidate
		}
		if text == "" {
			text = strings.TrimSpace(evt.NormalizedText)
		}
		if text == "" {
			text = strings.TrimSpace(evt.RawTranscript)
		}
		replay := change.Replay
		if replay == nil {
			replay = evt.Replay
		}
		note := renderedPreviewNote{
			EventID:             evt.ID,
			Text:                text,
			Target:              strings.TrimSpace(evt.VisualTargetRef),
			ReviewMode:          strings.TrimSpace(evt.ReviewMode),
			DOMSummary:          previewDOMSummary(evt.Pointer.DOM),
			Console:             previewConsoleLines(evt.Pointer.Console),
			Network:             previewNetworkLines(evt.Pointer.Network),
			PointerEventCount:   len(evt.PointerPath),
			ReplayStepCount:     previewReplayStepCount(replay),
			ReplayValueMode:     previewReplayValueMode(replay),
			ReplaySteps:         previewReplaySteps(replay),
			PlaywrightScript:    previewPlaywrightScript(replay),
			HasAudio:            strings.TrimSpace(evt.AudioRef) != "",
			HasScreenshot:       strings.TrimSpace(evt.ScreenshotRef) != "",
			HasVideo:            strings.TrimSpace(evt.VideoClipRef) != "",
			VideoDurationMS:     0,
			VideoSendLimitBytes: transmissionInlineLimit("video"),
		}
		if evt.Video != nil && evt.Video.DurationMS > 0 {
			note.VideoDurationMS = evt.Video.DurationMS
			note.VideoCodec = strings.TrimSpace(evt.Video.Codec)
			note.VideoScope = strings.TrimSpace(evt.Video.Scope)
			note.VideoWindow = strings.TrimSpace(evt.Video.Window)
			note.VideoHasAudio = evt.Video.HasAudio
			note.VideoPointerOverlay = evt.Video.PointerOverlay
		}
		if artifact, ok := artifactByEventKind[evt.ID+":screenshot"]; ok {
			note.ScreenshotDataURL = strings.TrimSpace(artifact.InlineDataURL)
		}
		if artifact, ok := artifactByEventKind[evt.ID+":video"]; ok {
			note.VideoDataURL = strings.TrimSpace(artifact.InlineDataURL)
			note.VideoTransmissionState = strings.TrimSpace(artifact.TransmissionStatus)
			note.VideoTransmissionNote = strings.TrimSpace(artifact.TransmissionNote)
			note.VideoSizeBytes = artifact.SizeBytes
		}
		if artifact, ok := artifactByEventKind[evt.ID+":audio"]; ok {
			note.AudioDataURL = strings.TrimSpace(artifact.InlineDataURL)
		}
		preview.Notes = append(preview.Notes, note)
	}
	sort.Strings(preview.Warnings)
	return preview
}

func previewTypedValuesStatus(pkg *session.CanonicalPackage) string {
	if pkg == nil || len(pkg.ChangeRequests) == 0 {
		return "not_applicable"
	}
	seenIncluded := false
	seenRedacted := false
	for _, change := range pkg.ChangeRequests {
		mode := ""
		if change.Replay != nil {
			mode = strings.TrimSpace(change.Replay.ValueCaptureMode)
		}
		switch mode {
		case "opt_in", "captured":
			seenIncluded = true
		case "redacted":
			seenRedacted = true
		}
	}
	switch {
	case seenIncluded && seenRedacted:
		return "mixed"
	case seenIncluded:
		return "included"
	case seenRedacted:
		return "redacted"
	default:
		return "not_applicable"
	}
}

func applyArtifactDisclosure(disclosure *renderedDisclosure, transmissionPkg *session.CanonicalPackage) {
	if disclosure == nil || transmissionPkg == nil {
		return
	}
	for _, artifact := range transmissionPkg.Artifacts {
		switch strings.TrimSpace(artifact.Kind) {
		case "screenshot":
			if strings.HasPrefix(strings.TrimSpace(artifact.TransmissionStatus), "omitted") {
				disclosure.ScreenshotsOmitted++
			} else if strings.TrimSpace(artifact.InlineDataURL) != "" {
				disclosure.ScreenshotsSent++
			}
		case "video":
			if strings.HasPrefix(strings.TrimSpace(artifact.TransmissionStatus), "omitted") {
				disclosure.VideosOmitted++
			} else if strings.TrimSpace(artifact.InlineDataURL) != "" {
				disclosure.VideosSent++
			}
		case "audio":
			if strings.HasPrefix(strings.TrimSpace(artifact.TransmissionStatus), "omitted") {
				disclosure.AudioOmitted++
			} else if strings.TrimSpace(artifact.InlineDataURL) != "" {
				disclosure.AudioSent++
			}
		}
	}
}

func mergePayloadPreviewWarnings(preview renderedPayloadPreview, extra []string) renderedPayloadPreview {
	if len(extra) == 0 {
		return preview
	}
	preview.Warnings = append(preview.Warnings, extra...)
	sort.Strings(preview.Warnings)
	return preview
}

func previewDOMSummary(dom *session.DOMInspection) string {
	if dom == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if tag := strings.TrimSpace(dom.Tag); tag != "" {
		parts = append(parts, tag)
	}
	if id := strings.TrimSpace(dom.ID); id != "" {
		parts = append(parts, "#"+id)
	}
	if testID := strings.TrimSpace(dom.TestID); testID != "" {
		parts = append(parts, "data-testid="+testID)
	}
	if label := strings.TrimSpace(dom.Label); label != "" {
		parts = append(parts, "label="+label)
	}
	return strings.Join(parts, " ")
}

func previewConsoleLines(entries []session.ConsoleEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		line := strings.TrimSpace(strings.ToUpper(entry.Level))
		if msg := strings.TrimSpace(entry.Message); msg != "" {
			if line != "" {
				line += ": "
			}
			line += msg
		}
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func previewNetworkLines(entries []session.NetworkEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts := []string{}
		if method := strings.TrimSpace(entry.Method); method != "" {
			parts = append(parts, method)
		}
		if status := entry.Status; status > 0 {
			parts = append(parts, fmt.Sprintf("%d", status))
		}
		if rawURL := strings.TrimSpace(entry.URL); rawURL != "" {
			parts = append(parts, rawURL)
		}
		if len(parts) > 0 {
			out = append(out, strings.Join(parts, " "))
		}
	}
	return out
}

func previewReplayStepCount(replay *session.ReplayBundle) int {
	if replay == nil {
		return 0
	}
	return len(replay.Steps)
}

func previewReplayValueMode(replay *session.ReplayBundle) string {
	if replay == nil {
		return ""
	}
	return strings.TrimSpace(replay.ValueCaptureMode)
}

func previewReplaySteps(replay *session.ReplayBundle) []string {
	if replay == nil || len(replay.Steps) == 0 {
		return nil
	}
	limit := len(replay.Steps)
	if limit > 8 {
		limit = 8
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		step := replay.Steps[i]
		line := strings.TrimSpace(step.Type)
		target := strings.TrimSpace(step.TargetSelector)
		if target == "" {
			target = strings.TrimSpace(step.TargetLabel)
		}
		if target == "" {
			target = strings.TrimSpace(step.TargetTag)
		}
		if target != "" {
			if line != "" {
				line += " "
			}
			line += target
		}
		if step.ValueRedacted {
			line += " [value redacted]"
		} else if step.ValueCaptured && strings.TrimSpace(step.Value) != "" {
			line += " = " + strings.TrimSpace(step.Value)
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func previewPlaywrightScript(replay *session.ReplayBundle) string {
	if replay == nil {
		return ""
	}
	return strings.TrimSpace(replay.PlaywrightScript)
}

func (s *Server) artifactRefToDataURL(ref string) (string, error) {
	payload, err := s.artifacts.Load(ref)
	if err != nil {
		return "", err
	}
	mimeType := inferArtifactMIMEType(ref, payload)
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(payload), nil
}

func inferArtifactMIMEType(ref string, payload []byte) string {
	base := strings.ToLower(filepath.Base(ref))
	trimmed := strings.TrimSuffix(base, ".enc")
	switch strings.TrimPrefix(filepath.Ext(trimmed), ".") {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webm":
		if strings.HasPrefix(trimmed, "audio_") {
			return "audio/webm"
		}
		return "video/webm"
	case "mp4":
		return "video/mp4"
	case "wav":
		return "audio/wav"
	case "mp3":
		return "audio/mpeg"
	case "m4a":
		return "audio/mp4"
	case "ogg", "oga":
		return "audio/ogg"
	}
	detected := http.DetectContentType(payload)
	if detected == "" {
		return "application/octet-stream"
	}
	return detected
}

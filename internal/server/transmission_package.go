package server

import (
	"encoding/base64"
	"fmt"
	"strings"

	"knit/internal/session"
)

type transmissionOptions struct {
	AllowLargeInline   bool
	RedactReplayValues bool
	OmitVideoClips     bool
	OmitVideoEventIDs  map[string]struct{}
}

const (
	maxInlineScreenshotBytes int64 = 1 << 20
	maxInlineAudioBytes      int64 = 2 << 20
	maxInlineVideoBytes      int64 = 4 << 20
)

func transmissionInlineLimit(kind string) int64 {
	switch strings.TrimSpace(kind) {
	case "screenshot":
		return maxInlineScreenshotBytes
	case "audio":
		return maxInlineAudioBytes
	case "video":
		return maxInlineVideoBytes
	default:
		return 0
	}
}

func humanArtifactLimit(limit int64) string {
	if limit <= 0 {
		return "no limit"
	}
	if limit%(1<<20) == 0 {
		return fmt.Sprintf("%d MB", limit>>(20))
	}
	if limit%(1<<10) == 0 {
		return fmt.Sprintf("%d KB", limit>>(10))
	}
	return fmt.Sprintf("%d bytes", limit)
}

func humanArtifactSize(size int64) string {
	if size >= 1<<20 {
		return fmt.Sprintf("%.1f MB", float64(size)/float64(1<<20))
	}
	if size >= 1<<10 {
		return fmt.Sprintf("%.1f KB", float64(size)/float64(1<<10))
	}
	return fmt.Sprintf("%d bytes", size)
}

func artifactDecisionWarning(kind, eventID string, size, limit int64) string {
	label := strings.TrimSpace(kind)
	if label == "" {
		label = "media"
	}
	if strings.TrimSpace(eventID) == "" {
		return fmt.Sprintf("%s is %s, over the default send limit of %s. Lower video quality, use a screenshot instead, or allow large inline media before submitting.", label, humanArtifactSize(size), humanArtifactLimit(limit))
	}
	return fmt.Sprintf("%s for %s is %s, over the default send limit of %s. Lower video quality, use a screenshot instead, or allow large inline media before submitting.", label, eventID, humanArtifactSize(size), humanArtifactLimit(limit))
}

func redactReplayBundleForTransmission(summary string, replay *session.ReplayBundle) *session.ReplayBundle {
	if replay == nil {
		return nil
	}
	out := cloneReplayBundle(replay)
	out.ValueCaptureMode = "redacted"
	for i := range out.Steps {
		out.Steps[i].Value = ""
		out.Steps[i].ValueCaptured = false
		if strings.EqualFold(strings.TrimSpace(out.Steps[i].Type), "input") || strings.EqualFold(strings.TrimSpace(out.Steps[i].Type), "change") {
			out.Steps[i].ValueRedacted = true
		}
	}
	out.PlaywrightScript = session.GeneratePlaywrightScript(summary, out)
	return out
}

func cloneCanonicalPackageForTransmission(pkg *session.CanonicalPackage) *session.CanonicalPackage {
	if pkg == nil {
		return nil
	}
	out := *pkg
	out.ChangeRequests = append([]session.ChangeReq(nil), pkg.ChangeRequests...)
	for i := range out.ChangeRequests {
		out.ChangeRequests[i].PointerPath = cloneReplayPointerPathForTransmission(out.ChangeRequests[i].PointerPath)
		out.ChangeRequests[i].AffectedArea = append([]string(nil), out.ChangeRequests[i].AffectedArea...)
		out.ChangeRequests[i].Assumptions = append([]string(nil), out.ChangeRequests[i].Assumptions...)
		out.ChangeRequests[i].Ambiguities = append([]string(nil), out.ChangeRequests[i].Ambiguities...)
		out.ChangeRequests[i].Replay = cloneReplayBundle(out.ChangeRequests[i].Replay)
	}
	out.Artifacts = append([]session.ArtifactRef(nil), pkg.Artifacts...)
	return &out
}

func cloneReplayPointerPathForTransmission(in []session.PointerSample) []session.PointerSample {
	if len(in) == 0 {
		return nil
	}
	out := make([]session.PointerSample, len(in))
	copy(out, in)
	return out
}

func (s *Server) buildTransmissionPackage(pkg *session.CanonicalPackage, opts transmissionOptions) (*session.CanonicalPackage, []string, bool, error) {
	if pkg == nil {
		return nil, nil, false, nil
	}
	out := cloneCanonicalPackageForTransmission(pkg)
	if out == nil {
		return nil, nil, false, nil
	}
	warnings := make([]string, 0, len(out.Artifacts))
	requiresDecision := false

	if opts.RedactReplayValues {
		for i := range out.ChangeRequests {
			out.ChangeRequests[i].Replay = redactReplayBundleForTransmission(out.ChangeRequests[i].Summary, out.ChangeRequests[i].Replay)
		}
	}

	for i := range out.Artifacts {
		artifact := &out.Artifacts[i]
		if strings.TrimSpace(artifact.Kind) == "video" && (opts.OmitVideoClips || transmissionVideoOmittedForEvent(opts.OmitVideoEventIDs, artifact.EventID)) {
			artifact.TransmissionStatus = "omitted_by_user"
			artifact.TransmissionNote = "Video clip omitted for this request. Knit will rely on the snapshot and the written change request instead."
			artifact.InlineDataURL = ""
			artifact.Ref = ""
			continue
		}
		switch strings.TrimSpace(artifact.Kind) {
		case "screenshot", "video", "audio":
			if strings.TrimSpace(artifact.Ref) == "" {
				continue
			}
			payload, err := s.artifacts.Load(artifact.Ref)
			if err != nil {
				return nil, nil, false, fmt.Errorf("load %s artifact %s: %w", artifact.Kind, artifact.Ref, err)
			}
			artifact.MIMEType = inferArtifactMIMEType(artifact.Ref, payload)
			artifact.SizeBytes = int64(len(payload))
			limit := transmissionInlineLimit(artifact.Kind)
			if !opts.AllowLargeInline && limit > 0 && artifact.SizeBytes > limit {
				artifact.TransmissionStatus = "omitted_due_to_limit"
				artifact.TransmissionNote = artifactDecisionWarning(artifact.Kind, artifact.EventID, artifact.SizeBytes, limit)
				artifact.Ref = ""
				warnings = append(warnings, artifact.TransmissionNote)
				requiresDecision = true
				continue
			}
			artifact.InlineDataURL = "data:" + artifact.MIMEType + ";base64," + base64.StdEncoding.EncodeToString(payload)
			artifact.TransmissionStatus = "inline"
			artifact.TransmissionNote = ""
		}
	}
	return out, warnings, requiresDecision, nil
}

func transmissionVideoOmittedForEvent(eventIDs map[string]struct{}, eventID string) bool {
	if len(eventIDs) == 0 {
		return false
	}
	_, ok := eventIDs[strings.TrimSpace(eventID)]
	return ok
}

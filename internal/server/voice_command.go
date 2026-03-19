package server

import (
	"fmt"
	"os"
	"strings"

	"knit/internal/agents"
	"knit/internal/audit"
	"knit/internal/session"
)

const (
	voiceCommandStartSession   = "start_session"
	voiceCommandPauseCapture   = "pause_capture"
	voiceCommandCaptureNote    = "capture_note"
	voiceCommandFreezeScreen   = "freeze_screen"
	voiceCommandSubmitFeedback = "submit_feedback"
	voiceCommandDiscardLast    = "discard_last_note"
)

func parseVoiceCommand(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	replacer := strings.NewReplacer(".", "", "!", "", "?", "", ",", "")
	normalized = replacer.Replace(normalized)
	switch {
	case strings.Contains(normalized, "start session"):
		return voiceCommandStartSession
	case strings.Contains(normalized, "pause capture"):
		return voiceCommandPauseCapture
	case strings.Contains(normalized, "capture note"):
		return voiceCommandCaptureNote
	case strings.Contains(normalized, "freeze screen"):
		return voiceCommandFreezeScreen
	case strings.Contains(normalized, "submit feedback"):
		return voiceCommandSubmitFeedback
	case strings.Contains(normalized, "discard last note"):
		return voiceCommandDiscardLast
	default:
		return ""
	}
}

func (s *Server) handleVoiceCommand(curr *session.Session, rawTranscript string) (bool, map[string]any, error) {
	command := parseVoiceCommand(rawTranscript)
	if command == "" || curr == nil {
		return false, nil, nil
	}

	response := map[string]any{
		"voice_command":  command,
		"handled":        true,
		"session_id":     curr.ID,
		"raw_transcript": strings.TrimSpace(rawTranscript),
	}

	switch command {
	case voiceCommandStartSession:
		if err := s.sessions.Resume(); err != nil {
			return true, nil, err
		}
		s.privilegedCapture.Start()
	case voiceCommandPauseCapture:
		if err := s.sessions.Pause(); err != nil {
			return true, nil, err
		}
		s.privilegedCapture.Pause()
	case voiceCommandCaptureNote:
		// Explicit no-op command used to keep user in capture flow.
	case voiceCommandFreezeScreen:
		response["freeze_screen"] = true
	case voiceCommandDiscardLast:
		updated, discarded, err := s.sessions.DiscardLastFeedback()
		if err != nil {
			return true, nil, err
		}
		if updated != nil {
			if err := s.store.UpsertSession(updated); err != nil {
				return true, nil, fmt.Errorf("upsert session after discard: %w", err)
			}
			response["session"] = updated
		}
		if discarded != nil {
			response["discarded_event_id"] = discarded.ID
		}
	case voiceCommandSubmitFeedback:
		pkg, err := s.sessions.Approve("")
		if err != nil {
			return true, nil, fmt.Errorf("approve for voice submit: %w", err)
		}
		provider := strings.TrimSpace(os.Getenv("KNIT_VOICE_COMMAND_PROVIDER"))
		if provider == "" {
			provider = s.resolveProvider("")
		} else {
			provider = canonicalProviderAlias(provider, s.agents.Names())
		}
		transmissionPkg, transmissionWarnings, requiresDecision, err := s.buildTransmissionPackage(pkg, transmissionOptions{})
		if err != nil {
			return true, nil, err
		}
		if requiresDecision {
			return true, nil, fmt.Errorf("%s", strings.Join(transmissionWarnings, " "))
		}
		redactedPkg := redactPackageForTransmission(*transmissionPkg)
		rc := s.currentRuntimeCodex()
		intent := agents.NormalizeDeliveryIntent(agents.DeliveryIntent{})
		providerPayload, err := agents.PreviewProviderPayloadWithConfig(provider, redactedPkg, rc.CodexModel, rc.ClaudeAPIModel, intent)
		if err != nil {
			return true, nil, err
		}
		attempt := s.enqueueSubmitJob(provider, redactedPkg, providerPayload, intent, "voice_command", "local_operator")
		response["provider"] = provider
		response["attempt_id"] = attempt.AttemptID
		response["submit_queue_state"] = s.submitQueueState()
	}

	updated := s.sessions.Current()
	if updated != nil {
		if err := s.store.UpsertSession(updated); err != nil {
			return true, nil, fmt.Errorf("upsert session after voice command: %w", err)
		}
		response["session"] = updated
	}
	_ = s.audit.Write(audit.Event{
		Type:      "voice_command_handled",
		SessionID: curr.ID,
		Details: map[string]any{
			"command": command,
		},
	})
	return true, response, nil
}

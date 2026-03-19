package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"knit/internal/agents"
	"knit/internal/audit"
	"knit/internal/operatorstate"
	"knit/internal/platform"
	"knit/internal/session"
)

type authContext struct {
	Kind         string
	Actor        string
	Capabilities []string
	PairingID    string
	PairingName  string
}

type pendingExtensionPairing struct {
	ID           string
	Code         string
	Name         string
	Browser      string
	Platform     string
	Capabilities []string
	ExpiresAt    time.Time
}

type extensionPairingPublic struct {
	ID           string     `json:"id"`
	Name         string     `json:"name,omitempty"`
	Browser      string     `json:"browser,omitempty"`
	Platform     string     `json:"platform,omitempty"`
	Capabilities []string   `json:"capabilities,omitempty"`
	CreatedAt    time.Time  `json:"created_at,omitempty"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}

type extensionSessionState struct {
	Session         any                      `json:"session"`
	RuntimePlatform any                      `json:"runtime_platform"`
	PlatformProfile any                      `json:"platform_profile"`
	RuntimeCodex    map[string]any           `json:"runtime_codex"`
	SubmitQueue     map[string]any           `json:"submit_queue"`
	SubmitAttempts  []submitAttempt          `json:"submit_attempts"`
	SubmitRecovery  []string                 `json:"submit_recovery_notices,omitempty"`
	SensitiveBadges map[string]string        `json:"sensitive_badges"`
	Extensions      []extensionPairingPublic `json:"extensions,omitempty"`
}

type authContextKey struct{}

func randomSecretToken() string {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("knit-ext-%d", time.Now().UTC().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func (s *Server) authenticateRequest(r *http.Request) (string, authContext, bool) {
	cfg := s.currentConfig()
	controlToken := strings.TrimSpace(cfg.ControlToken)
	var provided string
	if v := strings.TrimSpace(r.Header.Get("X-Knit-Token")); v != "" {
		provided = v
	}
	if provided == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			provided = strings.TrimSpace(auth[7:])
		}
	}
	if provided == "" {
		provided = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if provided == "" {
		return "", authContext{}, false
	}

	if controlToken != "" && subtleConstantTimeEqual(provided, controlToken) {
		return provided, authContext{
			Kind:         "control",
			Actor:        "local_operator",
			Capabilities: append([]string(nil), cfg.ControlCapabilities...),
		}, true
	}

	pairing, ok := s.findExtensionPairingByToken(provided)
	if !ok {
		return "", authContext{}, false
	}
	ctx := authContext{
		Kind:         "extension",
		Actor:        "extension:" + firstNonEmptyString(pairing.Name, pairing.ID),
		Capabilities: append([]string(nil), pairing.Capabilities...),
		PairingID:    pairing.ID,
		PairingName:  pairing.Name,
	}
	s.markExtensionPairingUsed(pairing.ID)
	return provided, ctx, true
}

func subtleConstantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var out byte
	for i := 0; i < len(a); i++ {
		out |= a[i] ^ b[i]
	}
	return out == 0
}

func (s *Server) requestAuthContext(r *http.Request) authContext {
	if r == nil {
		return authContext{}
	}
	if v, ok := r.Context().Value(authContextKey{}).(authContext); ok {
		return v
	}
	return authContext{}
}

func (s *Server) requestSource(r *http.Request) string {
	ctx := s.requestAuthContext(r)
	switch ctx.Kind {
	case "extension":
		return "browser_extension"
	default:
		return "local_ui"
	}
}

func (s *Server) auditWriteRequest(r *http.Request, evt audit.Event) {
	ctx := s.requestAuthContext(r)
	if evt.Actor == "" && ctx.Actor != "" {
		evt.Actor = ctx.Actor
	}
	_ = s.audit.Write(evt)
}

func (s *Server) findExtensionPairingByToken(token string) (operatorstate.ExtensionPairing, bool) {
	hash := hashToken(token)
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	for _, pairing := range s.runtime.Extensions.Pairings {
		if pairing.TokenHash != hash || pairing.ID == "" || pairing.RevokedAt != nil {
			continue
		}
		return pairing, true
	}
	return operatorstate.ExtensionPairing{}, false
}

func (s *Server) markExtensionPairingUsed(pairingID string) {
	if strings.TrimSpace(pairingID) == "" {
		return
	}
	now := time.Now().UTC()
	state := s.updateRuntimeState(func(state *operatorstate.State) {
		for i := range state.Extensions.Pairings {
			if state.Extensions.Pairings[i].ID != pairingID {
				continue
			}
			state.Extensions.Pairings[i].LastUsedAt = &now
			break
		}
	})
	_ = s.persistOperatorState(s.currentConfig())
	_ = state
}

func (s *Server) extensionPairingsPublic() []extensionPairingPublic {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	if len(s.runtime.Extensions.Pairings) == 0 {
		return nil
	}
	out := make([]extensionPairingPublic, 0, len(s.runtime.Extensions.Pairings))
	for _, pairing := range s.runtime.Extensions.Pairings {
		out = append(out, extensionPairingPublic{
			ID:           pairing.ID,
			Name:         pairing.Name,
			Browser:      pairing.Browser,
			Platform:     pairing.Platform,
			Capabilities: append([]string(nil), pairing.Capabilities...),
			CreatedAt:    pairing.CreatedAt,
			LastUsedAt:   pairing.LastUsedAt,
			RevokedAt:    pairing.RevokedAt,
		})
	}
	return out
}

func (s *Server) extensionSessionPayload() extensionSessionState {
	cfg := s.currentConfig()
	runtimeCodex := s.runtimeAgentState()
	return extensionSessionState{
		Session:         s.sessions.Current(),
		RuntimePlatform: platform.CurrentRuntimeGuide(),
		PlatformProfile: platform.CurrentProfile(),
		RuntimeCodex: map[string]any{
			"default_provider":    runtimeCodex["default_provider"],
			"available_providers": runtimeCodex["available_providers"],
		},
		SubmitQueue:    s.submitQueueState(),
		SubmitAttempts: s.submitAttemptsSnapshot(),
		SubmitRecovery: s.submitRecoveryNotesSnapshot(),
		SensitiveBadges: map[string]string{
			"typed_values":      previewBoolBadgeLabel(currentSessionCaptureInputValues(s.sessions.Current()), "Typed values on", "Typed values redacted"),
			"large_media":       previewBoolBadgeLabel(cfg.AllowRemoteSubmission, "Remote send allowed", "Remote send blocked"),
			"video_mode":        strings.TrimSpace(cfg.VideoMode),
			"audio_mode":        strings.TrimSpace(s.currentRuntimeState().Audio.Mode),
			"destination_class": destinationClassLabel(s.resolveProvider(""), s.agents),
		},
		Extensions: s.extensionPairingsPublic(),
	}
}

func currentSessionCaptureInputValues(curr *session.Session) bool {
	if curr == nil {
		return true
	}
	return curr.CaptureInputValues
}

func previewBoolBadgeLabel(v bool, yes, no string) string {
	if v {
		return yes
	}
	return no
}

func destinationClassLabel(provider string, registry *agents.Registry) string {
	if registry == nil {
		return "Destination unknown"
	}
	if registry.IsRemote(provider) {
		return "Sent to remote provider"
	}
	switch canonicalProviderAlias(provider, registry.Names()) {
	case "codex_cli", "claude_cli", "opencode_cli", "cli":
		return "Sent to local CLI on this machine"
	default:
		return "Stays on this machine"
	}
}

func (s *Server) withAuthContext(r *http.Request, ctx authContext) {
	if r == nil {
		return
	}
	*r = *r.WithContext(context.WithValue(r.Context(), authContextKey{}, ctx))
}

package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"knit/internal/audit"
	"knit/internal/operatorstate"
)

const extensionPairingTTL = 5 * time.Minute

type extensionPairStartRequest struct {
	Name     string `json:"name,omitempty"`
	Browser  string `json:"browser,omitempty"`
	Platform string `json:"platform,omitempty"`
}

type extensionPairCompleteRequest struct {
	PairingCode string `json:"pairing_code"`
	Name        string `json:"name,omitempty"`
	Browser     string `json:"browser,omitempty"`
	Platform    string `json:"platform,omitempty"`
}

type extensionPairRevokeRequest struct {
	PairingID string `json:"pairing_id"`
}

func (s *Server) handleExtensionPairStart(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req extensionPairStartRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	now := time.Now().UTC()
	pairing := pendingExtensionPairing{
		ID:           "ext-" + now.Format("20060102150405.000000000"),
		Code:         randomPairingCode(8),
		Name:         truncateCompanionField(strings.TrimSpace(req.Name), 80),
		Browser:      truncateCompanionField(strings.TrimSpace(req.Browser), 40),
		Platform:     truncateCompanionField(strings.TrimSpace(req.Platform), 40),
		Capabilities: []string{"read", "capture", "submit"},
		ExpiresAt:    now.Add(extensionPairingTTL),
	}
	s.pairMu.Lock()
	for id, item := range s.pendingPairs {
		if item.ExpiresAt.Before(now) {
			delete(s.pendingPairs, id)
		}
	}
	s.pendingPairs[pairing.ID] = pairing
	s.pairMu.Unlock()
	s.auditWriteRequest(r, audit.Event{
		Type: "extension_pairing_started",
		Details: map[string]any{
			"pairing_id": pairing.ID,
			"name":       pairing.Name,
			"browser":    pairing.Browser,
			"platform":   pairing.Platform,
			"expires_at": pairing.ExpiresAt,
		},
	})
	writeJSON(w, map[string]any{
		"pairing_id":   pairing.ID,
		"pairing_code": pairing.Code,
		"expires_at":   pairing.ExpiresAt,
		"capabilities": pairing.Capabilities,
	})
}

func (s *Server) handleExtensionPairComplete(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req extensionPairCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.PairingCode))
	if code == "" {
		http.Error(w, "pairing_code is required", http.StatusBadRequest)
		return
	}
	now := time.Now().UTC()
	var pending pendingExtensionPairing
	var found bool
	s.pairMu.Lock()
	for id, item := range s.pendingPairs {
		if item.ExpiresAt.Before(now) {
			delete(s.pendingPairs, id)
			continue
		}
		if item.Code == code {
			pending = item
			delete(s.pendingPairs, id)
			found = true
			break
		}
	}
	s.pairMu.Unlock()
	if !found {
		http.Error(w, "pairing code not found or expired", http.StatusNotFound)
		return
	}
	token := randomSecretToken()
	tokenHash := hashToken(token)
	pairing := operatorstate.ExtensionPairing{
		ID:           pending.ID,
		Name:         firstNonEmptyString(truncateCompanionField(strings.TrimSpace(req.Name), 80), pending.Name),
		Browser:      firstNonEmptyString(truncateCompanionField(strings.TrimSpace(req.Browser), 40), pending.Browser),
		Platform:     firstNonEmptyString(truncateCompanionField(strings.TrimSpace(req.Platform), 40), pending.Platform),
		Capabilities: append([]string(nil), pending.Capabilities...),
		TokenHash:    tokenHash,
		CreatedAt:    now,
	}
	s.updateRuntimeState(func(state *operatorstate.State) {
		state.Extensions.Pairings = append(state.Extensions.Pairings, pairing)
	})
	if err := s.persistOperatorState(s.currentConfig()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.auditWriteRequest(r, audit.Event{
		Type:  "extension_pairing_completed",
		Actor: "extension:" + firstNonEmptyString(pairing.Name, pairing.ID),
		Details: map[string]any{
			"pairing_id": pairing.ID,
			"name":       pairing.Name,
			"browser":    pairing.Browser,
			"platform":   pairing.Platform,
		},
	})
	writeJSON(w, map[string]any{
		"token":      token,
		"pairing":    s.extensionPairingsPublic(),
		"session":    s.extensionSessionPayload(),
		"daemon_url": "http://" + s.currentConfig().HTTPListenAddr,
	})
}

func (s *Server) handleExtensionPairings(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	writeJSON(w, map[string]any{"pairings": s.extensionPairingsPublic()})
}

func (s *Server) handleExtensionPairRevoke(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req extensionPairRevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	pairingID := strings.TrimSpace(req.PairingID)
	if pairingID == "" {
		http.Error(w, "pairing_id is required", http.StatusBadRequest)
		return
	}
	now := time.Now().UTC()
	var revoked bool
	s.updateRuntimeState(func(state *operatorstate.State) {
		for i := range state.Extensions.Pairings {
			if state.Extensions.Pairings[i].ID != pairingID || state.Extensions.Pairings[i].RevokedAt != nil {
				continue
			}
			state.Extensions.Pairings[i].RevokedAt = &now
			revoked = true
			break
		}
	})
	if !revoked {
		http.Error(w, "pairing not found", http.StatusNotFound)
		return
	}
	if err := s.persistOperatorState(s.currentConfig()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.auditWriteRequest(r, audit.Event{
		Type: "extension_pairing_revoked",
		Details: map[string]any{
			"pairing_id": pairingID,
		},
	})
	writeJSON(w, map[string]any{"ok": true, "pairings": s.extensionPairingsPublic()})
}

func (s *Server) handleExtensionSession(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	writeJSON(w, s.extensionSessionPayload())
}

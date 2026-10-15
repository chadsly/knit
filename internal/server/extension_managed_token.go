package server

import (
	"net/http"
	"strings"
	"time"

	"knit/internal/operatorstate"
)

const managedExtensionPairingName = "Knit MCP Browser Composer"

func (s *Server) managedExtensionTokenState() map[string]any {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	for _, pairing := range s.runtime.Extensions.Pairings {
		if pairing.RevokedAt != nil || strings.TrimSpace(pairing.TokenSource) != managedTokenSourceMCP {
			continue
		}
		return map[string]any{
			"available":    strings.TrimSpace(pairing.RawToken) != "",
			"pairing_id":   pairing.ID,
			"name":         pairing.Name,
			"browser":      pairing.Browser,
			"platform":     pairing.Platform,
			"token_source": pairing.TokenSource,
			"created_at":   pairing.CreatedAt,
			"last_used_at": pairing.LastUsedAt,
		}
	}
	return map[string]any{
		"available": false,
	}
}

func (s *Server) ensureManagedExtensionToken() (operatorstate.ExtensionPairing, bool, error) {
	now := time.Now().UTC()
	var (
		result operatorstate.ExtensionPairing
		reused bool
	)
	s.updateRuntimeState(func(state *operatorstate.State) {
		for i := range state.Extensions.Pairings {
			pairing := &state.Extensions.Pairings[i]
			if pairing.RevokedAt != nil || strings.TrimSpace(pairing.TokenSource) != managedTokenSourceMCP {
				continue
			}
			if strings.TrimSpace(pairing.RawToken) != "" && strings.TrimSpace(pairing.TokenHash) != "" {
				result = *pairing
				reused = true
				return
			}
		}
		token := randomSecretToken()
		pairing := operatorstate.ExtensionPairing{
			ID:           "ext-managed-" + now.Format("20060102150405.000000000"),
			Name:         managedExtensionPairingName,
			Browser:      "chromium",
			Platform:     "local",
			Capabilities: []string{"read", "capture", "submit"},
			TokenHash:    hashToken(token),
			RawToken:     token,
			TokenSource:  managedTokenSourceMCP,
			CreatedAt:    now,
		}
		state.Extensions.Pairings = append(state.Extensions.Pairings, pairing)
		result = pairing
	})
	if err := s.persistOperatorState(s.currentConfig()); err != nil {
		return operatorstate.ExtensionPairing{}, false, err
	}
	return result, reused, nil
}

func (s *Server) handleManagedExtensionToken(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	ensure := r.Method == http.MethodPost || strings.TrimSpace(r.URL.Query().Get("ensure")) == "1"
	reveal := strings.TrimSpace(r.URL.Query().Get("reveal")) == "1"
	if reveal && r.Method != http.MethodPost {
		http.Error(w, "managed extension token reveal requires POST", http.StatusMethodNotAllowed)
		return
	}
	var (
		pairing operatorstate.ExtensionPairing
		reused  bool
		err     error
	)
	if ensure {
		pairing, reused, err = s.ensureManagedExtensionToken()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		s.runtimeMu.RLock()
		for _, item := range s.runtime.Extensions.Pairings {
			if item.RevokedAt == nil && strings.TrimSpace(item.TokenSource) == managedTokenSourceMCP {
				pairing = item
				break
			}
		}
		s.runtimeMu.RUnlock()
	}
	resp := map[string]any{
		"available":    strings.TrimSpace(pairing.RawToken) != "",
		"pairing_id":   pairing.ID,
		"name":         pairing.Name,
		"browser":      pairing.Browser,
		"platform":     pairing.Platform,
		"token_source": pairing.TokenSource,
		"created_at":   pairing.CreatedAt,
		"last_used_at": pairing.LastUsedAt,
		"reused":       reused,
	}
	if reveal && strings.TrimSpace(pairing.RawToken) != "" {
		resp["token"] = pairing.RawToken
	}
	writeJSON(w, resp)
}

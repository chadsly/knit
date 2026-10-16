package server

import (
	"errors"
	"knit/internal/operatorstate"
	"os"
	"path/filepath"
	"strings"
)

// Only an explicitly confirmed panel setup may initialize an empty workspace.
// This never replaces another project or modifies agent permissions.
func (s *Server) configureDesktopWorkspace(workspace string) error {
	cfg := s.currentConfig()
	if cfg.ConfigLocked {
		return errors.New("Workspace changes are locked by policy. Ask your administrator to configure this daemon's workspace.")
	}
	workspace = strings.TrimSpace(workspace)
	info, err := os.Stat(workspace)
	if !filepath.IsAbs(workspace) || err != nil || !info.IsDir() {
		return errors.New("Choose an existing workspace folder.")
	}
	current := s.currentRuntimeCodex().CodexWorkdir
	if current != "" && filepath.Clean(current) != filepath.Clean(workspace) {
		return errors.New("This daemon is already configured for another workspace. Existing workspace settings are never replaced automatically.")
	}
	if current != "" {
		return nil
	}
	s.updateRuntimeState(func(state *operatorstate.State) { state.RuntimeCodex.CodexWorkdir = filepath.Clean(workspace) })
	if err := s.persistOperatorState(cfg); err != nil {
		return errors.New("Could not save the workspace. Try connecting again.")
	}
	return nil
}

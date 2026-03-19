package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"knit/internal/config"
)

const defaultReleaseCheckAPIURL = "https://api.github.com/repos/chadsly/knit/releases/latest"

type updateCheckResponse struct {
	CurrentVersion    string `json:"current_version"`
	LatestVersion     string `json:"latest_version,omitempty"`
	UpdateAvailable   bool   `json:"update_available"`
	VersionComparable bool   `json:"version_comparable"`
	Status            string `json:"status"`
	Message           string `json:"message,omitempty"`
	ReleaseURL        string `json:"release_url,omitempty"`
	ReleaseName       string `json:"release_name,omitempty"`
	PublishedAt       string `json:"published_at,omitempty"`
}

type githubLatestRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
}

type semver struct {
	major int
	minor int
	patch int
}

func runtimeVersion(cfg config.Config) string {
	if v := strings.TrimSpace(cfg.BuildID); v != "" {
		return v
	}
	if v := strings.TrimSpace(cfg.VersionPin); v != "" {
		return v
	}
	return "dev"
}

func parseStableSemver(raw string) (semver, bool) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "v"))
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	var out semver
	for i, part := range parts {
		if part == "" {
			return semver{}, false
		}
		value := 0
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return semver{}, false
			}
			value = value*10 + int(ch-'0')
		}
		switch i {
		case 0:
			out.major = value
		case 1:
			out.minor = value
		case 2:
			out.patch = value
		}
	}
	return out, true
}

func compareSemver(a, b semver) int {
	switch {
	case a.major != b.major:
		if a.major < b.major {
			return -1
		}
		return 1
	case a.minor != b.minor:
		if a.minor < b.minor {
			return -1
		}
		return 1
	case a.patch != b.patch:
		if a.patch < b.patch {
			return -1
		}
		return 1
	default:
		return 0
	}
}

func (s *Server) checkForUpdates(ctx context.Context) (updateCheckResponse, error) {
	cfg := s.currentConfig()
	current := runtimeVersion(cfg)
	result := updateCheckResponse{
		CurrentVersion: current,
		Status:         "unknown",
	}
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, s.updateReleaseAPIURL, nil)
	if err != nil {
		return result, fmt.Errorf("create update request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "knit/"+current)

	client := s.updateHTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("query latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return result, fmt.Errorf("latest release endpoint returned %d", resp.StatusCode)
	}

	var payload githubLatestRelease
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return result, fmt.Errorf("decode latest release: %w", err)
	}
	latest := strings.TrimSpace(strings.TrimPrefix(payload.TagName, "v"))
	result.LatestVersion = latest
	result.ReleaseURL = strings.TrimSpace(payload.HTMLURL)
	result.ReleaseName = strings.TrimSpace(payload.Name)
	result.PublishedAt = strings.TrimSpace(payload.PublishedAt)

	currentSemver, currentOK := parseStableSemver(current)
	latestSemver, latestOK := parseStableSemver(latest)
	result.VersionComparable = currentOK && latestOK
	if !latestOK {
		result.Status = "unavailable"
		result.Message = "Latest release version could not be parsed."
		return result, nil
	}
	if !currentOK {
		result.Status = "current_build_unversioned"
		result.Message = "Current build is not a stable semantic version."
		return result, nil
	}
	if compareSemver(currentSemver, latestSemver) < 0 {
		result.UpdateAvailable = true
		result.Status = "update_available"
		result.Message = "A newer Knit release is available."
		return result, nil
	}
	result.Status = "up_to_date"
	result.Message = "Current build matches the latest stable release."
	return result, nil
}

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
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
	result, err := s.checkForUpdates(r.Context())
	if err != nil {
		writeJSON(w, map[string]any{
			"current_version":  runtimeVersion(s.currentConfig()),
			"status":           "unavailable",
			"message":          err.Error(),
			"update_available": false,
		})
		return
	}
	writeJSON(w, result)
}

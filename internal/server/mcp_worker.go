package server

import (
	"os"
	"strings"
	"time"

	"knit/internal/operatorstate"
)

const (
	executionModeDirect       = operatorstate.DefaultExecutionModeDirect
	executionModeWorker       = operatorstate.DefaultExecutionModeWorker
	managedTokenSourceMCP     = "mcp_worker"
	workerHeartbeatTTL        = 30 * time.Second
	workerStaleTTL            = 45 * time.Second
	workerClaimLeaseTTL       = 45 * time.Second
	daemonInstanceModeNormal  = "normal"
	daemonInstanceModeManaged = "mcp_managed"

	queueCodeNoCompatibleWorker    = "no_compatible_worker"
	queueCodeWorkerNotConnected    = "worker_not_connected"
	queueCodeClaimConflict         = "claim_conflict"
	queueCodeClaimLeaseExpired     = "claim_lease_expired"
	queueCodeWorkerDisconnected    = "worker_disconnected"
	queueCodeRuntimeMismatch       = "runtime_mismatch"
	queueCodeDaemonUnavailable     = "daemon_unavailable"
	queueCodeTokenUnavailable      = "token_unavailable"
	queueCodeTokenInvalid          = "token_invalid"
	queueCodeMCPAttachIncompatible = "mcp_attach_incompatible_daemon"

	workerStatusOnline  = "online"
	workerStatusStale   = "stale"
	workerStatusOffline = "offline"
)

type submitWorker struct {
	WorkerID           string     `json:"worker_id"`
	Label              string     `json:"label,omitempty"`
	SupportedProviders []string   `json:"supported_providers,omitempty"`
	Source             string     `json:"source,omitempty"`
	Status             string     `json:"status,omitempty"`
	ConnectedAt        time.Time  `json:"connected_at"`
	LastHeartbeatAt    time.Time  `json:"last_heartbeat_at"`
	DisconnectedAt     *time.Time `json:"disconnected_at,omitempty"`
	ActiveClaims       int        `json:"active_claim_count,omitempty"`
}

type queueStateMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func normalizeExecutionMode(raw string) string {
	return operatorstate.NormalizeExecutionMode(raw)
}

func normalizeDaemonInstanceMode(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case daemonInstanceModeManaged:
		return daemonInstanceModeManaged
	default:
		return daemonInstanceModeNormal
	}
}

func (s *Server) defaultExecutionMode() string {
	mode := strings.TrimSpace(s.currentRuntimeCodex().DefaultExecutionMode)
	if mode == "" {
		mode = strings.TrimSpace(os.Getenv("KNIT_DEFAULT_EXECUTION_MODE"))
	}
	return normalizeExecutionMode(mode)
}

func (s *Server) daemonInstanceMode() string {
	return normalizeDaemonInstanceMode(s.daemonMode)
}

func (s *Server) daemonInstanceID() string {
	if strings.TrimSpace(s.daemonID) == "" {
		return "daemon-unknown"
	}
	return strings.TrimSpace(s.daemonID)
}

func (s *Server) mcpAttachReady() bool {
	return s.daemonInstanceMode() == daemonInstanceModeManaged && s.defaultExecutionMode() == executionModeWorker
}

func (s *Server) daemonInstanceState() map[string]any {
	return map[string]any{
		"daemon_instance_mode":   s.daemonInstanceMode(),
		"daemon_instance_id":     s.daemonInstanceID(),
		"default_execution_mode": s.defaultExecutionMode(),
		"mcp_attach_ready":       s.mcpAttachReady(),
	}
}

func workerStatus(worker submitWorker, now time.Time) string {
	if strings.TrimSpace(worker.WorkerID) == "" {
		return workerStatusOffline
	}
	if worker.DisconnectedAt != nil {
		return workerStatusOffline
	}
	if worker.LastHeartbeatAt.IsZero() {
		return workerStatusOffline
	}
	age := now.Sub(worker.LastHeartbeatAt)
	switch {
	case age <= workerHeartbeatTTL:
		return workerStatusOnline
	case age <= workerStaleTTL:
		return workerStatusStale
	default:
		return workerStatusOffline
	}
}

func workerEligibleForNewWork(worker submitWorker, now time.Time) bool {
	return workerStatus(worker, now) == workerStatusOnline
}

func normalizeWorkerProviders(raw []string, allowed []string) []string {
	allowedSet := map[string]struct{}{}
	for _, provider := range allowed {
		provider = strings.TrimSpace(provider)
		if provider == "" {
			continue
		}
		allowedSet[provider] = struct{}{}
	}
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, provider := range raw {
		canonical := canonicalProviderAlias(provider, allowed)
		if canonical == "" {
			continue
		}
		if len(allowedSet) > 0 {
			if _, ok := allowedSet[canonical]; !ok {
				continue
			}
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	if len(out) > 0 {
		return out
	}
	return append([]string(nil), allowed...)
}

func workerSupportsProvider(worker submitWorker, provider string) bool {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return false
	}
	for _, candidate := range worker.SupportedProviders {
		if strings.TrimSpace(candidate) == provider {
			return true
		}
	}
	return false
}

func queueStateMessageForWorkerAvailability(provider string, onlineWorkers int, compatibleWorkers int) (string, string) {
	provider = strings.TrimSpace(provider)
	switch {
	case onlineWorkers <= 0:
		return queueCodeWorkerNotConnected, "Queued for worker. No worker session is connected right now."
	case compatibleWorkers <= 0:
		if provider == "" {
			return queueCodeNoCompatibleWorker, "Queued for worker. No compatible worker session is connected for this request."
		}
		return queueCodeNoCompatibleWorker, "Queued for worker. No compatible worker session is connected for request provider " + provider + "."
	default:
		return "", ""
	}
}

func queueStateMessageForCode(code string, provider string) string {
	switch strings.TrimSpace(code) {
	case queueCodeNoCompatibleWorker:
		if strings.TrimSpace(provider) == "" {
			return "Queued for worker. No compatible worker session is connected for this request."
		}
		return "Queued for worker. No compatible worker session is connected for request provider " + strings.TrimSpace(provider) + "."
	case queueCodeWorkerNotConnected:
		return "Queued for worker. No worker session is connected right now."
	case queueCodeWorkerDisconnected:
		return "Queued for worker. The owning worker session disconnected, so this request is waiting to be claimed again."
	case queueCodeClaimLeaseExpired:
		return "The claim lease expired before this request could be completed."
	case queueCodeClaimConflict:
		return "Another worker session already claimed this request."
	case queueCodeRuntimeMismatch:
		return "The request provider or execution mode does not match this worker session."
	case queueCodeDaemonUnavailable:
		return "The daemon instance is unavailable."
	case queueCodeTokenUnavailable:
		return "The extension token is unavailable."
	case queueCodeTokenInvalid:
		return "The extension token is invalid."
	case queueCodeMCPAttachIncompatible:
		return "The existing daemon instance is not compatible with MCP-managed attachment."
	default:
		return ""
	}
}

func (s *Server) registerSubmitWorker(workerID string, providers []string, source string, label string) submitWorker {
	now := time.Now().UTC()
	allowed := allowedSubmitProviders(s.agents.Names(), s.currentConfig())
	worker := submitWorker{
		WorkerID:           strings.TrimSpace(workerID),
		Label:              strings.TrimSpace(label),
		SupportedProviders: normalizeWorkerProviders(providers, allowed),
		Source:             strings.TrimSpace(source),
		Status:             workerStatusOnline,
		ConnectedAt:        now,
		LastHeartbeatAt:    now,
	}
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	s.releaseExpiredWorkerClaimsLocked(now)
	if existing, ok := s.submitWorkers[worker.WorkerID]; ok {
		worker.ConnectedAt = existing.ConnectedAt
	}
	s.submitWorkers[worker.WorkerID] = worker
	s.refreshWorkerClaimCountsLocked()
	return s.submitWorkers[worker.WorkerID]
}

func (s *Server) heartbeatSubmitWorker(workerID string) (submitWorker, bool) {
	now := time.Now().UTC()
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	s.releaseExpiredWorkerClaimsLocked(now)
	worker, ok := s.submitWorkers[strings.TrimSpace(workerID)]
	if !ok {
		return submitWorker{}, false
	}
	if worker.DisconnectedAt != nil {
		return submitWorker{}, false
	}
	worker.LastHeartbeatAt = now
	worker.Status = workerStatusOnline
	s.submitWorkers[worker.WorkerID] = worker
	nextLease := now.Add(workerClaimLeaseTTL)
	for attemptID, job := range s.submitClaimed {
		if strings.TrimSpace(job.ClaimedByWorkerID) != worker.WorkerID {
			continue
		}
		job.LeaseExpiresAt = &nextLease
		s.submitClaimed[attemptID] = job
		s.updateSubmitAttemptLocked(attemptID, func(a *submitAttempt) {
			a.LeaseExpiresAt = &nextLease
			a.StateCode = ""
			a.StateMessage = ""
		})
	}
	s.refreshWorkerClaimCountsLocked()
	s.persistSubmitQueueLocked()
	return worker, true
}

func (s *Server) unregisterSubmitWorker(workerID string) {
	now := time.Now().UTC()
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	if worker, ok := s.submitWorkers[strings.TrimSpace(workerID)]; ok {
		worker.DisconnectedAt = &now
		worker.Status = workerStatusOffline
		s.submitWorkers[worker.WorkerID] = worker
	}
	s.releaseExpiredWorkerClaimsLocked(now)
	s.refreshWorkerClaimCountsLocked()
}

func (s *Server) refreshWorkerClaimCountsLocked() {
	counts := map[string]int{}
	for attemptID, job := range s.submitClaimed {
		_ = attemptID
		if strings.TrimSpace(job.ClaimedByWorkerID) != "" {
			counts[job.ClaimedByWorkerID]++
		}
	}
	for id, worker := range s.submitWorkers {
		worker.ActiveClaims = counts[id]
		worker.Status = workerStatus(worker, time.Now().UTC())
		s.submitWorkers[id] = worker
	}
}

func (s *Server) workerConnectivityState() map[string]any {
	now := time.Now().UTC()
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	s.releaseExpiredWorkerClaimsLocked(now)
	return s.workerConnectivityStateLocked(now)
}

func (s *Server) workerConnectivityStateLocked(now time.Time) map[string]any {
	workers := make([]submitWorker, 0, len(s.submitWorkers))
	providerSet := map[string]struct{}{}
	connected := 0
	stale := 0
	offline := 0
	idle := 0
	busy := 0
	for _, worker := range s.submitWorkers {
		worker.Status = workerStatus(worker, now)
		switch worker.Status {
		case workerStatusOnline:
			connected++
			if worker.ActiveClaims > 0 {
				busy++
			} else {
				idle++
			}
		case workerStatusStale:
			stale++
		default:
			offline++
		}
		for _, provider := range worker.SupportedProviders {
			providerSet[provider] = struct{}{}
		}
		workers = append(workers, worker)
	}
	providers := make([]string, 0, len(providerSet))
	for provider := range providerSet {
		providers = append(providers, provider)
	}
	return map[string]any{
		"connected": connected,
		"online":    connected,
		"stale":     stale,
		"offline":   offline,
		"idle":      idle,
		"busy":      busy,
		"providers": providers,
		"workers":   workers,
	}
}

func (s *Server) releaseExpiredWorkerClaimsLocked(now time.Time) {
	for attemptID, job := range s.submitClaimed {
		if job.LeaseExpiresAt != nil && job.LeaseExpiresAt.After(now) {
			continue
		}
		job.ClaimedByWorkerID = ""
		job.ClaimedAt = nil
		job.LeaseExpiresAt = nil
		s.submitPending = append(s.submitPending, job)
		s.submitQ++
		delete(s.submitClaimed, attemptID)
		s.updateSubmitAttemptLocked(attemptID, func(a *submitAttempt) {
			a.ClaimedByWorkerID = ""
			a.ClaimedAt = nil
			a.LeaseExpiresAt = nil
			a.Status = "queued"
			a.Note = "Queued for worker"
			a.StateCode = queueCodeWorkerDisconnected
			a.StateMessage = queueStateMessageForCode(queueCodeWorkerDisconnected, a.RequestProvider)
			a.QueuePos = s.submitQ
		})
		s.appendAttemptTimelineLocked(attemptID, "requeued", queueStateMessageForCode(queueCodeWorkerDisconnected, job.Provider))
	}
}

func (s *Server) refreshWorkerQueueWarningsLocked(now time.Time) {
	for i := range s.submitAttempts {
		attempt := &s.submitAttempts[i]
		if attempt.ExecutionMode != executionModeWorker || attempt.Status != "queued" {
			continue
		}
		online, compatible := 0, 0
		for _, worker := range s.submitWorkers {
			if workerEligibleForNewWork(worker, now) {
				online++
				if workerSupportsProvider(worker, attempt.RequestProvider) {
					compatible++
				}
			}
		}
		attempt.StateCode, attempt.StateMessage = queueStateMessageForWorkerAvailability(attempt.RequestProvider, online, compatible)
	}
}

func (s *Server) hasCompatibleWorkerLocked(provider string, now time.Time) bool {
	for _, worker := range s.submitWorkers {
		if !workerEligibleForNewWork(worker, now) {
			continue
		}
		if workerSupportsProvider(worker, provider) {
			return true
		}
	}
	return false
}

func (s *Server) claimableWorkerJobLocked(worker submitWorker, now time.Time) (submitJob, bool) {
	for _, job := range s.submitPending {
		if normalizeExecutionMode(job.ExecutionMode) != executionModeWorker {
			continue
		}
		if !workerSupportsProvider(worker, job.Provider) {
			continue
		}
		return job, true
	}
	return submitJob{}, false
}

func (s *Server) waitForWorkerJob(workerID string, timeout time.Duration) (submitJob, bool) {
	deadline := time.Now().Add(timeout)
	for {
		now := time.Now().UTC()
		s.submitMu.Lock()
		s.releaseExpiredWorkerClaimsLocked(now)
		worker, ok := s.submitWorkers[strings.TrimSpace(workerID)]
		if ok && workerEligibleForNewWork(worker, now) {
			if job, found := s.claimableWorkerJobLocked(worker, now); found {
				s.submitMu.Unlock()
				return job, true
			}
		}
		s.submitMu.Unlock()
		if timeout <= 0 || time.Now().After(deadline) {
			return submitJob{}, false
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (s *Server) claimWorkerJob(workerID string, attemptID string) (submitAttempt, bool, string, error) {
	now := time.Now().UTC()
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	s.releaseExpiredWorkerClaimsLocked(now)
	worker, ok := s.submitWorkers[strings.TrimSpace(workerID)]
	if !ok || !workerEligibleForNewWork(worker, now) {
		return submitAttempt{}, false, queueCodeWorkerNotConnected, nil
	}
	for _, existing := range s.submitAttempts {
		if existing.AttemptID == strings.TrimSpace(attemptID) && strings.TrimSpace(existing.ClaimedByWorkerID) != "" && strings.TrimSpace(existing.ClaimedByWorkerID) != strings.TrimSpace(workerID) {
			return submitAttempt{}, true, queueCodeClaimConflict, nil
		}
	}
	for _, job := range s.submitPending {
		if job.AttemptID != strings.TrimSpace(attemptID) {
			continue
		}
		if normalizeExecutionMode(job.ExecutionMode) != executionModeWorker {
			return submitAttempt{}, true, queueCodeRuntimeMismatch, nil
		}
		if !workerSupportsProvider(worker, job.Provider) {
			return submitAttempt{}, true, queueCodeRuntimeMismatch, nil
		}
		claimedAt := now
		leaseExpiresAt := now.Add(workerClaimLeaseTTL)
		job.ClaimedByWorkerID = worker.WorkerID
		job.ClaimedAt = &claimedAt
		job.LeaseExpiresAt = &leaseExpiresAt
		s.removePendingJobLocked(job.AttemptID)
		if s.submitQ > 0 {
			s.submitQ--
		}
		s.submitClaimed[job.AttemptID] = job
		s.updateSubmitAttemptLocked(job.AttemptID, func(a *submitAttempt) {
			a.Status = "in_progress"
			a.Note = "Claimed by worker"
			a.QueuePos = 0
			a.ClaimedByWorkerID = worker.WorkerID
			a.ClaimedAt = &claimedAt
			a.LeaseExpiresAt = &leaseExpiresAt
			a.StateCode = ""
			a.StateMessage = ""
			if a.StartedAt == nil {
				a.StartedAt = &claimedAt
			}
		})
		s.appendAttemptTimelineLocked(job.AttemptID, "claimed", "Claimed by worker "+worker.WorkerID)
		s.refreshWorkerClaimCountsLocked()
		s.persistSubmitQueueLocked()
		for _, attempt := range s.submitAttempts {
			if attempt.AttemptID == job.AttemptID {
				return attempt, true, "", nil
			}
		}
		return submitAttempt{}, true, "", nil
	}
	for _, existing := range s.submitAttempts {
		if existing.AttemptID != strings.TrimSpace(attemptID) {
			continue
		}
		if strings.TrimSpace(existing.ClaimedByWorkerID) == strings.TrimSpace(workerID) {
			return existing, true, "", nil
		}
		if strings.TrimSpace(existing.ClaimedByWorkerID) != "" {
			return submitAttempt{}, true, queueCodeClaimConflict, nil
		}
		return submitAttempt{}, true, queueCodeClaimLeaseExpired, nil
	}
	return submitAttempt{}, false, "", nil
}

func (s *Server) workerJobByAttemptID(workerID string, attemptID string) (submitJob, bool, string) {
	now := time.Now().UTC()
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	s.releaseExpiredWorkerClaimsLocked(now)
	worker, ok := s.submitWorkers[strings.TrimSpace(workerID)]
	if !ok || !workerEligibleForNewWork(worker, now) {
		return submitJob{}, false, queueCodeWorkerNotConnected
	}
	job, ok := s.submitClaimed[strings.TrimSpace(attemptID)]
	if !ok || strings.TrimSpace(job.ClaimedByWorkerID) != strings.TrimSpace(workerID) {
		for _, existing := range s.submitAttempts {
			if existing.AttemptID != strings.TrimSpace(attemptID) {
				continue
			}
			if strings.TrimSpace(existing.ClaimedByWorkerID) != strings.TrimSpace(workerID) {
				return submitJob{}, false, queueCodeClaimLeaseExpired
			}
			break
		}
		return submitJob{}, false, queueCodeClaimLeaseExpired
	}
	if job.LeaseExpiresAt != nil && job.LeaseExpiresAt.Before(now) {
		return submitJob{}, false, queueCodeClaimLeaseExpired
	}
	nextLease := now.Add(workerClaimLeaseTTL)
	job.LeaseExpiresAt = &nextLease
	s.submitClaimed[job.AttemptID] = job
	s.updateSubmitAttemptLocked(job.AttemptID, func(a *submitAttempt) {
		a.LeaseExpiresAt = &nextLease
		a.StateCode = ""
		a.StateMessage = ""
	})
	s.refreshWorkerClaimCountsLocked()
	s.persistSubmitQueueLocked()
	return job, true, ""
}

func (s *Server) completeWorkerJob(workerID, attemptID, summary, ref string) (submitAttempt, bool) {
	now := time.Now().UTC()
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	job, ok := s.submitClaimed[strings.TrimSpace(attemptID)]
	if !ok || strings.TrimSpace(job.ClaimedByWorkerID) != strings.TrimSpace(workerID) {
		return submitAttempt{}, false
	}
	delete(s.submitClaimed, job.AttemptID)
	s.updateSubmitAttemptLocked(job.AttemptID, func(a *submitAttempt) {
		a.Status = "submitted"
		a.Note = "Worker completed request"
		a.Ref = strings.TrimSpace(ref)
		a.CompletedAt = &now
		if strings.TrimSpace(summary) != "" {
			a.AgentSummary = strings.TrimSpace(summary)
		}
		a.ClaimedByWorkerID = workerID
		a.LeaseExpiresAt = nil
		a.StateCode = ""
		a.StateMessage = ""
	})
	s.appendAttemptTimelineLocked(job.AttemptID, "submitted", "Worker completed request")
	s.refreshWorkerClaimCountsLocked()
	s.persistSubmitQueueLocked()
	for _, attempt := range s.submitAttempts {
		if attempt.AttemptID == job.AttemptID {
			return attempt, true
		}
	}
	return submitAttempt{}, true
}

func (s *Server) failWorkerJob(workerID, attemptID, code, message string) (submitAttempt, bool) {
	now := time.Now().UTC()
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	job, ok := s.submitClaimed[strings.TrimSpace(attemptID)]
	if !ok || strings.TrimSpace(job.ClaimedByWorkerID) != strings.TrimSpace(workerID) {
		return submitAttempt{}, false
	}
	delete(s.submitClaimed, job.AttemptID)
	s.updateSubmitAttemptLocked(job.AttemptID, func(a *submitAttempt) {
		a.Status = "failed"
		a.Note = "Worker failed request"
		a.Error = strings.TrimSpace(message)
		a.CompletedAt = &now
		a.ClaimedByWorkerID = workerID
		a.LeaseExpiresAt = nil
		a.StateCode = strings.TrimSpace(code)
		a.StateMessage = firstNonEmptyString(strings.TrimSpace(message), queueStateMessageForCode(code, a.RequestProvider))
	})
	s.appendAttemptTimelineLocked(job.AttemptID, "failed", firstNonEmptyString(strings.TrimSpace(message), queueStateMessageForCode(code, job.RequestProvider)))
	s.refreshWorkerClaimCountsLocked()
	s.persistSubmitQueueLocked()
	for _, attempt := range s.submitAttempts {
		if attempt.AttemptID == job.AttemptID {
			return attempt, true
		}
	}
	return submitAttempt{}, true
}

package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"knit/internal/agents"
)

func (s *Server) handleMCPStatus(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, map[string]any{
		"daemon_ok":               true,
		"daemon_instance":         s.daemonInstanceState(),
		"default_execution_mode":  s.defaultExecutionMode(),
		"submit_execution_mode":   s.submitExecutionMode(),
		"worker_connectivity":     s.workerConnectivityState(),
		"submit_queue":            s.submitQueueState(),
		"managed_extension_token": s.managedExtensionTokenState(),
		"available_providers":     allowedSubmitProviders(s.agents.Names(), s.currentConfig()),
	})
}

func (s *Server) handleMCPWorkerConnect(w http.ResponseWriter, r *http.Request) {
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
	var req mcpWorkerConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, queueCodeRuntimeMismatch, "Bad register_worker request.")
		return
	}
	if strings.TrimSpace(req.WorkerID) == "" {
		writeJSONError(w, http.StatusBadRequest, queueCodeWorkerNotConnected, "worker_id is required for register_worker.")
		return
	}
	worker := s.registerSubmitWorker(req.WorkerID, req.Providers, firstNonEmptyString(req.Source, "knit_mcp"), req.Label)
	writeJSON(w, map[string]any{
		"worker":              worker,
		"worker_connectivity": s.workerConnectivityState(),
	})
}

func (s *Server) handleMCPWorkerDisconnect(w http.ResponseWriter, r *http.Request) {
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
	var req mcpWorkerActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, queueCodeWorkerNotConnected, "Bad worker action request.")
		return
	}
	s.unregisterSubmitWorker(req.WorkerID)
	writeJSON(w, map[string]any{
		"ok":                  true,
		"worker_connectivity": s.workerConnectivityState(),
	})
}

func (s *Server) handleMCPWorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
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
	var req mcpWorkerActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, queueCodeWorkerNotConnected, "Bad heartbeat request.")
		return
	}
	worker, ok := s.heartbeatSubmitWorker(req.WorkerID)
	if !ok {
		writeJSONError(w, http.StatusNotFound, queueCodeWorkerNotConnected, "The worker session is not connected.")
		return
	}
	writeJSON(w, map[string]any{"worker": worker})
}

func (s *Server) handleMCPWaitForRequest(w http.ResponseWriter, r *http.Request) {
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
	var req mcpWorkerActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, queueCodeWorkerNotConnected, "Bad wait_for_request request.")
		return
	}
	if strings.TrimSpace(req.WorkerID) == "" {
		writeJSONError(w, http.StatusBadRequest, queueCodeWorkerNotConnected, "register_worker must run before wait_for_request.")
		return
	}
	timeoutSeconds := 30
	if raw := strings.TrimSpace(r.URL.Query().Get("timeout_seconds")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			timeoutSeconds = n
		}
	}
	job, ok := s.waitForWorkerJob(req.WorkerID, time.Duration(timeoutSeconds)*time.Second)
	if !ok {
		now := time.Now().UTC()
		s.submitMu.Lock()
		worker, exists := s.submitWorkers[strings.TrimSpace(req.WorkerID)]
		s.submitMu.Unlock()
		if !exists || !workerEligibleForNewWork(worker, now) {
			writeJSONError(w, http.StatusNotFound, queueCodeWorkerNotConnected, "The worker session is not connected.")
			return
		}
		writeJSON(w, map[string]any{"found": false})
		return
	}
	writeJSON(w, map[string]any{
		"found":      true,
		"attempt_id": job.AttemptID,
		"provider":   job.Provider,
		"summary":    submitRequestPreview(job.Package),
		"request": map[string]any{
			"attempt_id": job.AttemptID,
			"provider":   job.Provider,
			"package":    job.Package,
			"intent":     agents.NormalizeDeliveryIntent(job.Intent),
		},
	})
}

func (s *Server) handleMCPClaimRequest(w http.ResponseWriter, r *http.Request) {
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
	var req mcpWorkerClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, queueCodeClaimConflict, "Bad claim_request request.")
		return
	}
	attempt, exists, code, err := s.claimWorkerJob(req.WorkerID, req.AttemptID)
	if err != nil {
		writeJSONError(w, http.StatusConflict, queueCodeClaimConflict, err.Error())
		return
	}
	if strings.TrimSpace(code) != "" {
		status := http.StatusConflict
		if code == queueCodeWorkerNotConnected {
			status = http.StatusNotFound
		}
		if code == queueCodeRuntimeMismatch {
			status = http.StatusConflict
		}
		writeJSONError(w, status, code, queueStateMessageForCode(code, attempt.RequestProvider))
		return
	}
	if !exists {
		writeJSONError(w, http.StatusNotFound, queueCodeClaimLeaseExpired, "The request is no longer claimable.")
		return
	}
	writeJSON(w, map[string]any{"attempt": attempt})
}

func (s *Server) handleMCPGetRequest(w http.ResponseWriter, r *http.Request) {
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
	workerID := strings.TrimSpace(r.URL.Query().Get("worker_id"))
	attemptID := strings.TrimSpace(r.URL.Query().Get("attempt_id"))
	job, ok, code := s.workerJobByAttemptID(workerID, attemptID)
	if !ok {
		status := http.StatusNotFound
		if code == queueCodeWorkerNotConnected {
			status = http.StatusBadRequest
		}
		writeJSONError(w, status, code, queueStateMessageForCode(code, ""))
		return
	}
	writeJSON(w, map[string]any{
		"attempt_id": job.AttemptID,
		"provider":   job.Provider,
		"package":    job.Package,
		"intent":     agents.NormalizeDeliveryIntent(job.Intent),
	})
}

func (s *Server) handleMCPCompleteRequest(w http.ResponseWriter, r *http.Request) {
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
	var req mcpWorkerCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, queueCodeClaimLeaseExpired, "Bad complete_request request.")
		return
	}
	job, ok, code := s.workerJobByAttemptID(req.WorkerID, req.AttemptID)
	if !ok {
		status := http.StatusNotFound
		if code == queueCodeWorkerNotConnected {
			status = http.StatusBadRequest
		}
		writeJSONError(w, status, code, queueStateMessageForCode(code, ""))
		return
	}
	attempt, ok := s.completeWorkerJob(req.WorkerID, req.AttemptID, req.Summary, req.Ref)
	if !ok {
		http.Error(w, "request not found", http.StatusNotFound)
		return
	}
	_ = s.persistSubmissionResult(job, agents.Result{
		Provider: job.Provider,
		RunID:    req.AttemptID,
		Status:   "completed",
		Ref:      strings.TrimSpace(req.Ref),
	})
	writeJSON(w, map[string]any{"attempt": attempt})
}

func (s *Server) handleMCPFailRequest(w http.ResponseWriter, r *http.Request) {
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
	var req mcpWorkerFailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, queueCodeClaimLeaseExpired, "Bad fail_request request.")
		return
	}
	_, ok, code := s.workerJobByAttemptID(req.WorkerID, req.AttemptID)
	if !ok {
		status := http.StatusNotFound
		if code == queueCodeWorkerNotConnected {
			status = http.StatusBadRequest
		}
		writeJSONError(w, status, code, queueStateMessageForCode(code, ""))
		return
	}
	attempt, ok := s.failWorkerJob(req.WorkerID, req.AttemptID, firstNonEmptyString(req.Code, queueCodeRuntimeMismatch), req.Error)
	if !ok {
		writeJSONError(w, http.StatusNotFound, queueCodeClaimLeaseExpired, queueStateMessageForCode(queueCodeClaimLeaseExpired, ""))
		return
	}
	writeJSON(w, map[string]any{"attempt": attempt})
}

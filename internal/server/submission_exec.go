package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"knit/internal/agents"
	"knit/internal/audit"
	"knit/internal/session"
)

const (
	submitExecutionSeries   = "series"
	submitExecutionParallel = "parallel"
	submitStatusCanceled    = "canceled"
)

const maxSubmitAttemptHistory = 50
const defaultSubmitMaxAttempts = 2
const defaultSubmitRetryBackoffSeconds = 2
const defaultOfflineRetrySeconds = 15
const maxSubmitRequestPreviewLen = 140
const maxSubmitOutcomeLogBytes = 256 << 10
const maxSubmitAgentSummaryLen = 600

const (
	submitOutcomeNoInput        = "no_input"
	submitOutcomeTrustedDir     = "trusted_directory"
	submitOutcomeWrongWorkspace = "wrong_workspace"
	submitOutcomeReadOnly       = "read_only"
)

type submitJob struct {
	AttemptID       string
	Provider        string
	Intent          agents.DeliveryIntent
	Mode            string
	MaxAttempts     int
	Package         session.CanonicalPackage
	ProviderPayload map[string]any
	WorkdirUsed     string
	EnqueuedAt      time.Time
	DeferredUntil   time.Time
	Source          string
	Actor           string
}

type persistedSubmitJob struct {
	AttemptID     string                   `json:"attempt_id"`
	Provider      string                   `json:"provider"`
	Intent        agents.DeliveryIntent    `json:"intent,omitempty"`
	Mode          string                   `json:"mode"`
	MaxAttempts   int                      `json:"max_attempts"`
	Package       session.CanonicalPackage `json:"package"`
	WorkdirUsed   string                   `json:"workdir_used,omitempty"`
	EnqueuedAt    time.Time                `json:"enqueued_at"`
	DeferredUntil time.Time                `json:"deferred_until"`
	Source        string                   `json:"source,omitempty"`
	Actor         string                   `json:"actor,omitempty"`
}

type legacySubmitQueueStatePersist struct {
	Pending []submitJob `json:"pending"`
	Running []submitJob `json:"running"`
}

type submitAttemptEvent struct {
	Time   time.Time `json:"time"`
	Status string    `json:"status"`
	Note   string    `json:"note,omitempty"`
}

type submitAttempt struct {
	AttemptID          string               `json:"attempt_id"`
	SessionID          string               `json:"session_id"`
	Provider           string               `json:"provider"`
	IntentProfile      string               `json:"intent_profile,omitempty"`
	IntentLabel        string               `json:"intent_label,omitempty"`
	InstructionText    string               `json:"instruction_text,omitempty"`
	CustomInstructions string               `json:"custom_instructions,omitempty"`
	WorkdirUsed        string               `json:"workdir_used,omitempty"`
	Mode               string               `json:"mode"`
	RequestPreview     string               `json:"request_preview,omitempty"`
	MaxAttempts        int                  `json:"max_attempts,omitempty"`
	RetryCount         int                  `json:"retry_count,omitempty"`
	Status             string               `json:"status"`
	Note               string               `json:"note,omitempty"`
	Error              string               `json:"error,omitempty"`
	RunID              string               `json:"run_id,omitempty"`
	Ref                string               `json:"ref,omitempty"`
	QueueWaitMS        int64                `json:"queue_wait_ms,omitempty"`
	QueuePos           int                  `json:"queue_position,omitempty"`
	NextRetryAt        *time.Time           `json:"next_retry_at,omitempty"`
	EnqueuedAt         time.Time            `json:"enqueued_at"`
	StartedAt          *time.Time           `json:"started_at,omitempty"`
	CompletedAt        *time.Time           `json:"completed_at,omitempty"`
	PostSubmit         *postSubmitResult    `json:"post_submit,omitempty"`
	ExecutionRef       string               `json:"execution_ref,omitempty"`
	AgentSummary       string               `json:"agent_summary,omitempty"`
	OutcomeCode        string               `json:"outcome_code,omitempty"`
	OutcomeTitle       string               `json:"outcome_title,omitempty"`
	OutcomeMessage     string               `json:"outcome_message,omitempty"`
	Timeline           []submitAttemptEvent `json:"timeline,omitempty"`
	Source             string               `json:"source,omitempty"`
	Actor              string               `json:"actor,omitempty"`
}

func normalizeSubmitExecutionMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case submitExecutionParallel:
		return submitExecutionParallel
	default:
		return submitExecutionSeries
	}
}

func (s *Server) submitExecutionMode() string {
	mode := strings.TrimSpace(s.currentRuntimeCodex().SubmitExecMode)
	if mode == "" {
		mode = os.Getenv("KNIT_SUBMIT_EXECUTION_MODE")
	}
	return normalizeSubmitExecutionMode(mode)
}

func submitMaxAttempts() int {
	raw := strings.TrimSpace(os.Getenv("KNIT_SUBMIT_MAX_ATTEMPTS"))
	if raw == "" {
		return defaultSubmitMaxAttempts
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return defaultSubmitMaxAttempts
	}
	if n > 10 {
		n = 10
	}
	return n
}

func submitRetryBackoffSeconds() int {
	raw := strings.TrimSpace(os.Getenv("KNIT_SUBMIT_RETRY_BACKOFF_SECONDS"))
	if raw == "" {
		return defaultSubmitRetryBackoffSeconds
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return defaultSubmitRetryBackoffSeconds
	}
	if n > 60 {
		n = 60
	}
	return n
}

func offlineRetrySeconds() int {
	raw := strings.TrimSpace(os.Getenv("KNIT_OFFLINE_RETRY_SECONDS"))
	if raw == "" {
		return defaultOfflineRetrySeconds
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return defaultOfflineRetrySeconds
	}
	if n > 3600 {
		n = 3600
	}
	return n
}

func (s *Server) initSubmitWorkers() {
	if s.submitSeriesCh != nil {
		return
	}
	s.submitSeriesCh = make(chan submitJob, 256)
	s.recoverSubmitQueueFromDisk()
	go s.runSeriesSubmitLoop()
}

func (s *Server) runSeriesSubmitLoop() {
	for job := range s.submitSeriesCh {
		s.processSubmitJob(job)
	}
}

func (s *Server) submitQueueState() map[string]any {
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	return map[string]any{
		"queued":                s.submitQ,
		"running":               s.submitRun,
		"mode":                  s.submitExecutionMode(),
		"parallel_pending":      s.parallelPending,
		"post_submit_running":   s.parallelPostRunning,
		"max_attempts_default":  submitMaxAttempts(),
		"retry_backoff_seconds": submitRetryBackoffSeconds(),
		"offline_retry_seconds": offlineRetrySeconds(),
	}
}

func (s *Server) submitAttemptsSnapshot() []submitAttempt {
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	out := make([]submitAttempt, len(s.submitAttempts))
	copy(out, s.submitAttempts)
	return out
}

func (s *Server) submitRecoveryNotesSnapshot() []string {
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	out := make([]string, len(s.submitRecoveryNotes))
	copy(out, s.submitRecoveryNotes)
	return out
}

func (s *Server) appendSubmitRecoveryNoteLocked(note string) {
	note = strings.TrimSpace(note)
	if note == "" {
		return
	}
	s.submitRecoveryNotes = append([]string{note}, s.submitRecoveryNotes...)
	if len(s.submitRecoveryNotes) > 8 {
		s.submitRecoveryNotes = s.submitRecoveryNotes[:8]
	}
}

func (s *Server) submitAttemptByID(id string) (submitAttempt, bool) {
	s.submitMu.Lock()
	defer s.submitMu.Unlock()
	for _, a := range s.submitAttempts {
		if a.AttemptID == id {
			return a, true
		}
	}
	return submitAttempt{}, false
}

func isCanceledSubmitError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return true
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "canceled")
}

func (s *Server) shouldSkipCanceledAttemptLocked(attemptID string) bool {
	_, ok := s.submitCanceled[attemptID]
	return ok
}

func (s *Server) finalizeCanceledPendingAttemptLocked(job submitJob, note string, completedAt time.Time) {
	if s.submitQ > 0 {
		s.submitQ--
	}
	if job.Mode == submitExecutionParallel && s.parallelPending > 0 {
		s.parallelPending--
	}
	s.removePendingJobLocked(job.AttemptID)
	s.updateSubmitAttemptLocked(job.AttemptID, func(a *submitAttempt) {
		a.Status = submitStatusCanceled
		a.Note = strings.TrimSpace(note)
		a.Error = ""
		a.QueuePos = 0
		a.NextRetryAt = nil
		a.CompletedAt = &completedAt
	})
	s.appendAttemptTimelineLocked(job.AttemptID, submitStatusCanceled, note)
}

func (s *Server) maybeStartParallelPostSubmitLocked() bool {
	if s.parallelPending == 0 && s.submitRun == 0 && !s.parallelPostRunning && s.parallelHasSuccess {
		s.parallelHasSuccess = false
		s.parallelPostRunning = true
		return true
	}
	return false
}

func (s *Server) cancelSubmitAttempt(attemptID, source, actor string) (submitAttempt, bool, error) {
	id := strings.TrimSpace(attemptID)
	if id == "" {
		return submitAttempt{}, false, fmt.Errorf("attempt_id is required")
	}
	var (
		snapshot  submitAttempt
		ok        bool
		canceled  bool
		job       submitJob
		startPost bool
	)
	completedAt := time.Now().UTC()
	s.submitMu.Lock()
	for _, attempt := range s.submitAttempts {
		if attempt.AttemptID == id {
			snapshot = attempt
			ok = true
			break
		}
	}
	if !ok {
		s.submitMu.Unlock()
		return submitAttempt{}, false, nil
	}
	switch strings.TrimSpace(snapshot.Status) {
	case "queued", "deferred_offline":
		for _, pending := range s.submitPending {
			if pending.AttemptID == id {
				job = pending
				canceled = true
				break
			}
		}
		if canceled {
			s.submitCanceled[id] = "pending"
			s.finalizeCanceledPendingAttemptLocked(job, "Submission canceled before execution", completedAt)
			startPost = s.maybeStartParallelPostSubmitLocked()
			s.persistSubmitQueueLocked()
		}
	case "retry_wait", "in_progress":
		cancel := s.submitCancel[id]
		if cancel != nil {
			s.submitCanceled[id] = "running"
			canceled = true
			cancel()
		}
	default:
		s.submitMu.Unlock()
		return snapshot, true, fmt.Errorf("submission can only be stopped while queued or running")
	}
	s.submitMu.Unlock()
	if !canceled {
		return snapshot, true, fmt.Errorf("submission can only be stopped while queued or running")
	}
	_ = s.audit.Write(audit.Event{
		Type:      "submission_canceled",
		SessionID: snapshot.SessionID,
		Actor:     strings.TrimSpace(actor),
		Details: map[string]any{
			"attempt_id": id,
			"provider":   snapshot.Provider,
			"status":     snapshot.Status,
			"source":     strings.TrimSpace(source),
		},
	})
	if startPost {
		go s.completeParallelPostSubmit(snapshot.SessionID)
	}
	if latest, found := s.submitAttemptByID(id); found {
		return latest, true, nil
	}
	return snapshot, true, nil
}

func (s *Server) enqueueSubmitJob(provider string, pkg session.CanonicalPackage, providerPayload map[string]any, intent agents.DeliveryIntent, source string, actor string) submitAttempt {
	mode := s.submitExecutionMode()
	mode = normalizeSubmitExecutionMode(mode)
	enqueuedAt := time.Now().UTC()
	requestPreview := submitRequestPreview(pkg)
	intent = agents.NormalizeDeliveryIntent(intent)
	job := submitJob{
		Provider:        provider,
		Intent:          intent,
		Mode:            mode,
		MaxAttempts:     submitMaxAttempts(),
		Package:         pkg,
		ProviderPayload: providerPayload,
		WorkdirUsed:     s.effectiveSubmitWorkspace(provider),
		EnqueuedAt:      enqueuedAt,
		DeferredUntil:   time.Time{},
		Source:          strings.TrimSpace(source),
		Actor:           strings.TrimSpace(actor),
	}
	s.submitMu.Lock()
	s.submitSeq++
	job.AttemptID = fmt.Sprintf("attempt-%d-%d", enqueuedAt.UnixMilli(), s.submitSeq)
	s.submitQ++
	queuePos := s.submitQ
	if mode == submitExecutionParallel {
		s.parallelPending++
	}
	attempt := submitAttempt{
		AttemptID:          job.AttemptID,
		SessionID:          pkg.SessionID,
		Provider:           provider,
		IntentProfile:      intent.Profile,
		IntentLabel:        intent.Label(),
		InstructionText:    intent.InstructionText,
		CustomInstructions: intent.CustomInstructions,
		WorkdirUsed:        job.WorkdirUsed,
		Mode:               mode,
		RequestPreview:     requestPreview,
		MaxAttempts:        job.MaxAttempts,
		Status:             "queued",
		Note:               "Queued for submission",
		QueuePos:           queuePos,
		EnqueuedAt:         enqueuedAt,
		StartedAt:          nil,
		CompletedAt:        nil,
		Source:             job.Source,
		Actor:              job.Actor,
	}
	s.prependSubmitAttemptLocked(attempt)
	s.appendAttemptTimelineLocked(job.AttemptID, "queued", attempt.Note)
	s.submitPending = append(s.submitPending, job)
	s.persistSubmitQueueLocked()
	s.submitMu.Unlock()
	_ = s.audit.Write(audit.Event{
		Type:      "submission_queued",
		SessionID: pkg.SessionID,
		Actor:     job.Actor,
		Details: map[string]any{
			"provider":   provider,
			"attempt_id": job.AttemptID,
			"mode":       mode,
			"queue_pos":  queuePos,
			"source":     job.Source,
		},
	})
	if mode == submitExecutionSeries {
		s.submitSeriesCh <- job
	} else {
		go s.processSubmitJob(job)
	}
	return attempt
}

func (s *Server) dispatchSubmitJob(job submitJob) {
	if job.Mode == submitExecutionSeries {
		s.submitSeriesCh <- job
		return
	}
	go s.processSubmitJob(job)
}

func (s *Server) scheduleDeferredSubmit(job submitJob, at time.Time) {
	go func() {
		delay := time.Until(at)
		if delay > 0 {
			time.Sleep(delay)
		}
		s.dispatchSubmitJob(job)
	}()
}

func (s *Server) prependSubmitAttemptLocked(attempt submitAttempt) {
	s.submitAttempts = append([]submitAttempt{attempt}, s.submitAttempts...)
	if len(s.submitAttempts) > maxSubmitAttemptHistory {
		s.submitAttempts = s.submitAttempts[:maxSubmitAttemptHistory]
	}
}

func (s *Server) updateSubmitAttemptLocked(attemptID string, mutate func(*submitAttempt)) {
	for i := range s.submitAttempts {
		if s.submitAttempts[i].AttemptID == attemptID {
			mutate(&s.submitAttempts[i])
			return
		}
	}
}

func (s *Server) appendAttemptTimelineLocked(attemptID, status, note string) {
	for i := range s.submitAttempts {
		if s.submitAttempts[i].AttemptID != attemptID {
			continue
		}
		s.submitAttempts[i].Timeline = append(s.submitAttempts[i].Timeline, submitAttemptEvent{
			Time:   time.Now().UTC(),
			Status: strings.TrimSpace(status),
			Note:   strings.TrimSpace(note),
		})
		return
	}
}

func (s *Server) processSubmitJob(job submitJob) {
	startedAt := time.Now().UTC()
	waitMS := time.Since(job.EnqueuedAt).Milliseconds()
	submitCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	submitCtx = agents.WithDeliveryIntent(submitCtx, job.Intent)
	if logPath, err := s.allocateSubmitExecutionLogPath(job.AttemptID); err == nil {
		submitCtx = agents.WithCLILogFile(submitCtx, logPath)
		writeSubmitExecutionLog(logPath, "Attempt %s queued for provider %s", job.AttemptID, job.Provider)
		s.submitMu.Lock()
		s.updateSubmitAttemptLocked(job.AttemptID, func(a *submitAttempt) {
			a.ExecutionRef = logPath
		})
		s.submitMu.Unlock()
	}
	s.submitMu.Lock()
	if s.shouldSkipCanceledAttemptLocked(job.AttemptID) {
		s.removePendingJobLocked(job.AttemptID)
		delete(s.submitCanceled, job.AttemptID)
		s.persistSubmitQueueLocked()
		s.submitMu.Unlock()
		writeSubmitExecutionLog(agentsExecutionLogPath(submitCtx), "Attempt canceled before execution started")
		return
	}
	if strings.TrimSpace(job.WorkdirUsed) == "" {
		job.WorkdirUsed = s.effectiveSubmitWorkspace(job.Provider)
	}
	s.removePendingJobLocked(job.AttemptID)
	s.submitRunning[job.AttemptID] = job
	s.submitCancel[job.AttemptID] = cancel
	if s.submitQ > 0 {
		s.submitQ--
	}
	s.submitRun++
	s.updateSubmitAttemptLocked(job.AttemptID, func(a *submitAttempt) {
		a.Status = "in_progress"
		a.WorkdirUsed = job.WorkdirUsed
		a.Note = fmt.Sprintf("Submitting to adapter (attempt 1/%d)", maxInt(1, job.MaxAttempts))
		a.QueueWaitMS = waitMS
		a.QueuePos = 0
		a.NextRetryAt = nil
		a.StartedAt = &startedAt
	})
	s.appendAttemptTimelineLocked(job.AttemptID, "in_progress", "Dequeued for execution")
	s.persistSubmitQueueLocked()
	s.submitMu.Unlock()
	writeSubmitExecutionLog(agentsExecutionLogPath(submitCtx), "Execution started after %dms queue wait", waitMS)

	maxAttempts := maxInt(1, job.MaxAttempts)
	backoffBaseSec := submitRetryBackoffSeconds()
	var (
		res    agents.Result
		runErr error
	)
	if err := s.agents.ValidateSubmission(job.Provider); err != nil {
		runErr = err
		writeSubmitExecutionLog(agentsExecutionLogPath(submitCtx), "Submission blocked before execution: %v", runErr)
		maxAttempts = 0
	}
	for attemptNum := 1; attemptNum <= maxAttempts; attemptNum++ {
		if submitCtx.Err() != nil {
			runErr = submitCtx.Err()
			writeSubmitExecutionLog(agentsExecutionLogPath(submitCtx), "Attempt canceled before adapter execution: %v", runErr)
			break
		}
		s.submitMu.Lock()
		s.updateSubmitAttemptLocked(job.AttemptID, func(a *submitAttempt) {
			a.RetryCount = attemptNum - 1
			a.Note = fmt.Sprintf("Submitting to adapter (attempt %d/%d)", attemptNum, maxAttempts)
			a.NextRetryAt = nil
		})
		s.appendAttemptTimelineLocked(job.AttemptID, "attempt_started", fmt.Sprintf("Attempt %d started", attemptNum))
		s.submitMu.Unlock()
		writeSubmitExecutionLog(agentsExecutionLogPath(submitCtx), "Attempt %d/%d started", attemptNum, maxAttempts)

		res, runErr = s.agents.Submit(submitCtx, job.Provider, job.Package)
		if runErr == nil {
			runErr = s.persistSubmissionResult(job, res)
		}
		if runErr == nil {
			writeSubmitExecutionLog(agentsExecutionLogPath(submitCtx), "Attempt %d/%d completed: run_id=%s status=%s ref=%s", attemptNum, maxAttempts, res.RunID, res.Status, res.Ref)
			break
		}
		if attemptNum >= maxAttempts {
			writeSubmitExecutionLog(agentsExecutionLogPath(submitCtx), "Attempt %d/%d failed: %v", attemptNum, maxAttempts, runErr)
			break
		}
		backoffSeconds := int(math.Pow(2, float64(attemptNum-1))) * backoffBaseSec
		if backoffSeconds > 30 {
			backoffSeconds = 30
		}
		s.submitMu.Lock()
		s.updateSubmitAttemptLocked(job.AttemptID, func(a *submitAttempt) {
			a.Note = fmt.Sprintf("Retrying in %ds (attempt %d/%d failed)", backoffSeconds, attemptNum, maxAttempts)
		})
		s.appendAttemptTimelineLocked(job.AttemptID, "retry_wait", fmt.Sprintf("Attempt %d failed: %v", attemptNum, runErr))
		s.submitMu.Unlock()
		writeSubmitExecutionLog(agentsExecutionLogPath(submitCtx), "Retrying after attempt %d failure in %ds: %v", attemptNum, backoffSeconds, runErr)
		timer := time.NewTimer(time.Duration(backoffSeconds) * time.Second)
		select {
		case <-submitCtx.Done():
			timer.Stop()
			runErr = submitCtx.Err()
			writeSubmitExecutionLog(agentsExecutionLogPath(submitCtx), "Retry wait canceled: %v", runErr)
			attemptNum = maxAttempts
		case <-timer.C:
		}
	}

	if runErr != nil && !isCanceledSubmitError(runErr) && s.shouldDeferOffline(job.Provider, runErr) {
		s.deferOfflineSubmit(job, runErr)
		return
	}

	var postSubmit *postSubmitResult
	if runErr == nil && job.Mode == submitExecutionSeries {
		postSubmit = s.runPostSubmit()
		if postSubmit != nil {
			_ = s.audit.Write(audit.Event{
				Type:      "post_submit_automation_ran",
				SessionID: job.Package.SessionID,
				Details: map[string]any{
					"mode":           job.Mode,
					"rebuild_status": statusOrEmpty(postSubmit.Rebuild),
					"verify_status":  statusOrEmpty(postSubmit.Verify),
					"workdir":        postSubmit.Workdir,
				},
			})
		}
	}

	completedAt := time.Now().UTC()
	logText := readSubmitOutcomeLogText(agentsExecutionLogPath(submitCtx))
	agentSummary := extractSubmitAgentSummary(logText)
	outcomeCode, outcomeTitle, outcomeMessage := classifySubmitAttemptOutcome(job.Package, job.WorkdirUsed, agentsExecutionLogPath(submitCtx))
	shouldRunParallelPost := false
	s.submitMu.Lock()
	delete(s.submitRunning, job.AttemptID)
	delete(s.submitCancel, job.AttemptID)
	if s.submitRun > 0 {
		s.submitRun--
	}
	if job.Mode == submitExecutionParallel {
		if s.parallelPending > 0 {
			s.parallelPending--
		}
		if runErr == nil {
			s.parallelHasSuccess = true
		}
		if s.parallelPending == 0 && !s.parallelPostRunning && s.parallelHasSuccess {
			s.parallelHasSuccess = false
			s.parallelPostRunning = true
			shouldRunParallelPost = true
		}
	}
	s.updateSubmitAttemptLocked(job.AttemptID, func(a *submitAttempt) {
		a.CompletedAt = &completedAt
		a.PostSubmit = postSubmit
		a.NextRetryAt = nil
		a.AgentSummary = agentSummary
		a.OutcomeCode = outcomeCode
		a.OutcomeTitle = outcomeTitle
		a.OutcomeMessage = outcomeMessage
		if isCanceledSubmitError(runErr) {
			a.Status = submitStatusCanceled
			a.Error = ""
			a.Note = "Submission canceled"
			return
		}
		if runErr != nil {
			a.Status = "failed"
			a.Error = runErr.Error()
			a.Note = "Submission failed"
			return
		}
		a.Status = "submitted"
		a.RunID = res.RunID
		a.Ref = res.Ref
		if strings.TrimSpace(a.ExecutionRef) == "" {
			a.ExecutionRef = res.Ref
		}
		a.Note = "Submission completed"
	})
	if isCanceledSubmitError(runErr) {
		s.appendAttemptTimelineLocked(job.AttemptID, submitStatusCanceled, "Submission canceled")
		delete(s.submitCanceled, job.AttemptID)
	} else if runErr != nil {
		s.appendAttemptTimelineLocked(job.AttemptID, "failed", runErr.Error())
	} else {
		s.appendAttemptTimelineLocked(job.AttemptID, "submitted", "Submission completed")
	}
	s.persistSubmitQueueLocked()
	s.submitMu.Unlock()

	if isCanceledSubmitError(runErr) {
		writeSubmitExecutionLog(agentsExecutionLogPath(submitCtx), "Submission canceled")
	} else if runErr != nil {
		writeSubmitExecutionLog(agentsExecutionLogPath(submitCtx), "Submission failed: %v", runErr)
		_ = s.audit.Write(audit.Event{
			Type:      "submission_failed",
			SessionID: job.Package.SessionID,
			Details: map[string]any{
				"provider":   job.Provider,
				"attempt_id": job.AttemptID,
				"error":      runErr.Error(),
			},
		})
	}

	if shouldRunParallelPost {
		go s.completeParallelPostSubmit(job.Package.SessionID)
	}
}

func (s *Server) completeParallelPostSubmit(sessionID string) {
	ps := s.runPostSubmit()
	s.submitMu.Lock()
	s.parallelPostRunning = false
	if ps != nil {
		for i := range s.submitAttempts {
			if s.submitAttempts[i].SessionID != sessionID {
				continue
			}
			if strings.TrimSpace(s.submitAttempts[i].Status) != "submitted" {
				continue
			}
			s.submitAttempts[i].PostSubmit = ps
			s.submitAttempts[i].Timeline = append(s.submitAttempts[i].Timeline, submitAttemptEvent{
				Time:   time.Now().UTC(),
				Status: "post_submit",
				Note:   "Parallel post-submit automation completed",
			})
			break
		}
	}
	s.persistSubmitQueueLocked()
	s.submitMu.Unlock()
	if ps != nil {
		_ = s.audit.Write(audit.Event{
			Type:      "post_submit_automation_ran",
			SessionID: sessionID,
			Details: map[string]any{
				"mode":           submitExecutionParallel,
				"rebuild_status": statusOrEmpty(ps.Rebuild),
				"verify_status":  statusOrEmpty(ps.Verify),
				"workdir":        ps.Workdir,
			},
		})
	}
}

func (s *Server) deferOfflineSubmit(job submitJob, runErr error) {
	retryAfter := time.Duration(offlineRetrySeconds()) * time.Second
	nextRetry := time.Now().UTC().Add(retryAfter)
	job.DeferredUntil = nextRetry

	s.submitMu.Lock()
	delete(s.submitRunning, job.AttemptID)
	delete(s.submitCancel, job.AttemptID)
	if s.submitRun > 0 {
		s.submitRun--
	}
	s.submitQ++
	queuePos := s.submitQ
	s.submitPending = append(s.submitPending, job)
	s.updateSubmitAttemptLocked(job.AttemptID, func(a *submitAttempt) {
		a.Status = "deferred_offline"
		a.Error = runErr.Error()
		a.Note = fmt.Sprintf("Deferred due to network unavailability; retrying at %s", nextRetry.Format(time.RFC3339))
		a.QueuePos = queuePos
		a.NextRetryAt = &nextRetry
		a.CompletedAt = nil
	})
	s.appendAttemptTimelineLocked(job.AttemptID, "deferred_offline", runErr.Error())
	s.persistSubmitQueueLocked()
	s.submitMu.Unlock()

	_ = s.audit.Write(audit.Event{
		Type:      "submission_deferred_offline",
		SessionID: job.Package.SessionID,
		Details: map[string]any{
			"provider":      job.Provider,
			"attempt_id":    job.AttemptID,
			"retry_at":      nextRetry.Format(time.RFC3339),
			"offline_error": runErr.Error(),
		},
	})
	s.scheduleDeferredSubmit(job, nextRetry)
}

func (s *Server) shouldDeferOffline(provider string, err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	// Local adapters should generally fail fast; defer only remote submissions or clear network outages.
	if !s.agents.IsRemote(provider) && !containsAny(msg,
		"no such host",
		"network is unreachable",
		"temporary failure in name resolution",
		"connection refused",
		"connection reset",
		"dial tcp",
		"i/o timeout",
		"tls handshake timeout",
		"failed to connect",
		"error sending request",
		"stream disconnected",
		"lookup address",
	) {
		return false
	}
	return containsAny(msg,
		"no such host",
		"network is unreachable",
		"temporary failure in name resolution",
		"connection refused",
		"connection reset",
		"dial tcp",
		"i/o timeout",
		"tls handshake timeout",
		"failed to connect",
		"error sending request",
		"stream disconnected",
		"lookup address",
		"timeout awaiting response headers",
	)
}

type submitQueueStatePersist struct {
	Pending []persistedSubmitJob `json:"pending"`
	Running []persistedSubmitJob `json:"running"`
}

func parsePersistedSubmitQueue(b []byte) ([]submitJob, error) {
	var persisted submitQueueStatePersist
	if err := json.Unmarshal(b, &persisted); err == nil {
		recovered := make([]submitJob, 0, len(persisted.Running)+len(persisted.Pending))
		for _, job := range persisted.Running {
			recovered = append(recovered, job.toSubmitJob())
		}
		for _, job := range persisted.Pending {
			recovered = append(recovered, job.toSubmitJob())
		}
		if len(recovered) > 0 {
			return recovered, nil
		}
	}

	var legacy legacySubmitQueueStatePersist
	if err := json.Unmarshal(b, &legacy); err != nil {
		return nil, err
	}
	recovered := append([]submitJob(nil), legacy.Running...)
	recovered = append(recovered, legacy.Pending...)
	return recovered, nil
}

func normalizeRecoveredSubmitJob(job submitJob, state string, index int) (submitJob, []string, string) {
	notes := []string{}
	job.Provider = canonicalProviderAlias(job.Provider, nil)
	job.Intent = agents.NormalizeDeliveryIntent(job.Intent)
	if job.Provider == "" {
		return submitJob{}, nil, "provider missing"
	}
	job.Mode = normalizeSubmitExecutionMode(job.Mode)
	if strings.TrimSpace(job.Package.SessionID) == "" {
		return submitJob{}, nil, "session id missing"
	}
	if len(job.Package.ChangeRequests) == 0 {
		return submitJob{}, nil, "no change requests to recover"
	}
	if strings.TrimSpace(job.AttemptID) == "" {
		job.AttemptID = fmt.Sprintf("attempt-recovered-%d-%d", time.Now().UTC().UnixMilli(), index+1)
		notes = append(notes, fmt.Sprintf("Recovered a %s delivery without an attempt id; Knit reassigned it as %s.", state, job.AttemptID))
	}
	if job.MaxAttempts < 1 {
		job.MaxAttempts = submitMaxAttempts()
		notes = append(notes, fmt.Sprintf("Recovered delivery %s was missing retry settings; Knit restored the default retry limit.", job.AttemptID))
	}
	if job.EnqueuedAt.IsZero() {
		job.EnqueuedAt = time.Now().UTC()
		notes = append(notes, fmt.Sprintf("Recovered delivery %s was missing its queue time; Knit treated it as newly recovered.", job.AttemptID))
	}
	if state == "running" {
		notes = append(notes, fmt.Sprintf("Recovered in-progress delivery %s after restart. Knit resumed it automatically, so check Current run or Recent runs for its latest status.", job.AttemptID))
	}
	return job, notes, ""
}

func toPersistedSubmitJob(job submitJob) persistedSubmitJob {
	return persistedSubmitJob{
		AttemptID:     job.AttemptID,
		Provider:      job.Provider,
		Intent:        job.Intent,
		Mode:          job.Mode,
		MaxAttempts:   job.MaxAttempts,
		Package:       job.Package,
		WorkdirUsed:   job.WorkdirUsed,
		EnqueuedAt:    job.EnqueuedAt,
		DeferredUntil: job.DeferredUntil,
		Source:        job.Source,
		Actor:         job.Actor,
	}
}

func (job persistedSubmitJob) toSubmitJob() submitJob {
	return submitJob{
		AttemptID:     job.AttemptID,
		Provider:      job.Provider,
		Intent:        job.Intent,
		Mode:          job.Mode,
		MaxAttempts:   job.MaxAttempts,
		Package:       job.Package,
		WorkdirUsed:   job.WorkdirUsed,
		EnqueuedAt:    job.EnqueuedAt,
		DeferredUntil: job.DeferredUntil,
		Source:        job.Source,
		Actor:         job.Actor,
	}
}

func (s *Server) providerPayloadForJob(job submitJob) map[string]any {
	if len(job.ProviderPayload) > 0 {
		return job.ProviderPayload
	}
	rc := s.currentRuntimeCodex()
	payload, err := agents.PreviewProviderPayloadWithConfig(job.Provider, job.Package, rc.CodexModel, rc.ClaudeAPIModel, job.Intent)
	if err != nil {
		return nil
	}
	return payload
}

func (s *Server) removePendingJobLocked(attemptID string) {
	if strings.TrimSpace(attemptID) == "" || len(s.submitPending) == 0 {
		return
	}
	out := s.submitPending[:0]
	for _, job := range s.submitPending {
		if job.AttemptID == attemptID {
			continue
		}
		out = append(out, job)
	}
	s.submitPending = out
}

func (s *Server) persistSubmitQueueLocked() {
	path := strings.TrimSpace(s.submitQueuePath)
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	running := make([]submitJob, 0, len(s.submitRunning))
	for _, job := range s.submitRunning {
		running = append(running, job)
	}
	pending := make([]persistedSubmitJob, 0, len(s.submitPending))
	for _, job := range s.submitPending {
		pending = append(pending, toPersistedSubmitJob(job))
	}
	persistedRunning := make([]persistedSubmitJob, 0, len(running))
	for _, job := range running {
		persistedRunning = append(persistedRunning, toPersistedSubmitJob(job))
	}
	payload := submitQueueStatePersist{
		Pending: pending,
		Running: persistedRunning,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o600)
}

func (s *Server) recoverSubmitQueueFromDisk() {
	path := strings.TrimSpace(s.submitQueuePath)
	if path == "" {
		return
	}
	b, err := os.ReadFile(path)
	if err != nil || len(strings.TrimSpace(string(b))) == 0 {
		return
	}
	recovered, err := parsePersistedSubmitQueue(b)
	if err != nil {
		return
	}
	if len(recovered) == 0 {
		return
	}

	type recoveredJobState struct {
		job   submitJob
		state string
	}
	jobStates := make([]recoveredJobState, 0, len(recovered))
	var persistedNew submitQueueStatePersist
	if err := json.Unmarshal(b, &persistedNew); err == nil && (len(persistedNew.Running) > 0 || len(persistedNew.Pending) > 0) {
		for _, job := range persistedNew.Running {
			jobStates = append(jobStates, recoveredJobState{job: job.toSubmitJob(), state: "running"})
		}
		for _, job := range persistedNew.Pending {
			jobStates = append(jobStates, recoveredJobState{job: job.toSubmitJob(), state: "pending"})
		}
	} else {
		var persistedLegacy legacySubmitQueueStatePersist
		if err := json.Unmarshal(b, &persistedLegacy); err == nil {
			for _, job := range persistedLegacy.Running {
				jobStates = append(jobStates, recoveredJobState{job: job, state: "running"})
			}
			for _, job := range persistedLegacy.Pending {
				jobStates = append(jobStates, recoveredJobState{job: job, state: "pending"})
			}
		}
	}
	if len(jobStates) == 0 {
		for _, job := range recovered {
			jobStates = append(jobStates, recoveredJobState{job: job, state: "pending"})
		}
	}

	validRecovered := make([]submitJob, 0, len(jobStates))
	s.submitMu.Lock()
	for idx, recoveredJob := range jobStates {
		job, notes, discardReason := normalizeRecoveredSubmitJob(recoveredJob.job, recoveredJob.state, idx)
		if discardReason != "" {
			s.appendSubmitRecoveryNoteLocked(fmt.Sprintf("Discarded a stale recovered delivery because %s.", discardReason))
			continue
		}
		for _, note := range notes {
			s.appendSubmitRecoveryNoteLocked(note)
		}
		validRecovered = append(validRecovered, job)
		s.submitQ++
		if job.Mode == submitExecutionParallel {
			s.parallelPending++
		}
		s.submitPending = append(s.submitPending, job)
		attempt := submitAttempt{
			AttemptID:          job.AttemptID,
			SessionID:          job.Package.SessionID,
			Provider:           job.Provider,
			IntentProfile:      agents.NormalizeDeliveryIntent(job.Intent).Profile,
			IntentLabel:        agents.NormalizeDeliveryIntent(job.Intent).Label(),
			InstructionText:    agents.NormalizeDeliveryIntent(job.Intent).InstructionText,
			CustomInstructions: agents.NormalizeDeliveryIntent(job.Intent).CustomInstructions,
			WorkdirUsed:        job.WorkdirUsed,
			Mode:               job.Mode,
			RequestPreview:     submitRequestPreview(job.Package),
			MaxAttempts:        job.MaxAttempts,
			Status:             "queued",
			Note:               "Recovered after daemon restart",
			QueuePos:           s.submitQ,
			EnqueuedAt:         job.EnqueuedAt,
		}
		if !job.DeferredUntil.IsZero() && job.DeferredUntil.After(time.Now().UTC()) {
			attempt.Status = "deferred_offline"
			attempt.Note = fmt.Sprintf("Recovered deferred submission; retrying at %s", job.DeferredUntil.Format(time.RFC3339))
			attempt.NextRetryAt = &job.DeferredUntil
		}
		s.prependSubmitAttemptLocked(attempt)
		s.appendAttemptTimelineLocked(job.AttemptID, "recovered", "Recovered queued submission after daemon restart")
	}
	s.persistSubmitQueueLocked()
	s.submitMu.Unlock()

	for _, job := range validRecovered {
		if !job.DeferredUntil.IsZero() && job.DeferredUntil.After(time.Now().UTC()) {
			s.scheduleDeferredSubmit(job, job.DeferredUntil)
			continue
		}
		s.dispatchSubmitJob(job)
	}
}

func (s *Server) persistSubmissionResult(job submitJob, res agents.Result) error {
	eventIDs := make([]string, 0, len(job.Package.ChangeRequests))
	for _, cr := range job.Package.ChangeRequests {
		if strings.TrimSpace(cr.EventID) != "" {
			eventIDs = append(eventIDs, cr.EventID)
		}
	}
	curr := s.sessions.Current()
	if curr != nil && strings.TrimSpace(curr.ID) == strings.TrimSpace(job.Package.SessionID) {
		if err := s.sessions.MarkSubmittedFor(res.Ref, eventIDs); err != nil {
			return fmt.Errorf("mark submitted: %w", err)
		}
		curr = s.sessions.Current()
	}
	submissionPayload := map[string]any{
		"provider_payload": s.providerPayloadForJob(job),
		"delivery_intent":  agents.NormalizeDeliveryIntent(job.Intent),
		"result":           res,
		"transmitted": map[string]any{
			"provider_endpoint": s.agents.Endpoint(job.Provider),
			"provider_remote":   s.agents.IsRemote(job.Provider),
			"session_id":        job.Package.SessionID,
			"change_requests":   len(job.Package.ChangeRequests),
			"artifact_count":    len(job.Package.Artifacts),
			"source":            job.Source,
			"workdir_used":      job.WorkdirUsed,
		},
	}
	if err := s.store.SaveSubmission(job.Package.SessionID, job.Provider, res.RunID, res.Status, res.Ref, submissionPayload); err != nil {
		return fmt.Errorf("save submission: %w", err)
	}
	if curr != nil {
		if err := s.store.UpsertSession(curr); err != nil {
			return fmt.Errorf("upsert session: %w", err)
		}
	}
	_ = s.audit.Write(audit.Event{
		Type:      "submission_sent",
		SessionID: job.Package.SessionID,
		Actor:     job.Actor,
		Details: map[string]any{
			"provider":   job.Provider,
			"run_id":     res.RunID,
			"ref":        res.Ref,
			"attempt_id": job.AttemptID,
			"source":     job.Source,
		},
	})
	return nil
}

func (s *Server) runPostSubmit() *postSubmitResult {
	if s.postSubmitRunner == nil {
		return nil
	}
	return s.postSubmitRunner()
}

func (s *Server) allocateSubmitExecutionLogPath(attemptID string) (string, error) {
	outputDir := strings.TrimSpace(s.currentRuntimeCodex().CodexOutputDir)
	if outputDir == "" {
		outputDir = strings.TrimSpace(os.Getenv("KNIT_CODEX_OUTPUT_DIR"))
	}
	if outputDir == "" {
		outputDir = strings.TrimSpace(os.Getenv("TMPDIR"))
	}
	if outputDir == "" {
		outputDir = os.TempDir()
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	safeID := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-").Replace(strings.TrimSpace(attemptID))
	if safeID == "" {
		safeID = "attempt"
	}
	f, err := os.CreateTemp(outputDir, "knit-codex-"+safeID+"-*.log")
	if err != nil {
		return "", err
	}
	_ = f.Close()
	return filepath.Clean(f.Name()), nil
}

func agentsExecutionLogPath(ctx context.Context) string {
	return strings.TrimSpace(agents.ExecutionLogPathFromContext(ctx))
}

func writeSubmitExecutionLog(path string, format string, args ...any) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	msg := strings.TrimRight(fmt.Sprintf(format, args...), "\n")
	if msg == "" {
		return
	}
	_, _ = fmt.Fprintf(f, "[%s] %s\n", time.Now().UTC().Format(time.RFC3339), msg)
}

func classifySubmitAttemptOutcome(pkg session.CanonicalPackage, workdir, logPath string) (string, string, string) {
	if len(pkg.ChangeRequests) == 0 && len(pkg.Artifacts) == 0 {
		return submitOutcomeNoInput, "No input", "Knit submitted this run without any captured change requests or artifacts, so the coding agent had nothing to change."
	}
	logText := strings.ToLower(readSubmitOutcomeLogText(logPath))
	if containsAny(logText,
		"not inside a trusted directory",
		"trusted directory",
		"git repo check",
		"--skip-git-repo-check was not specified",
	) {
		message := "Go back to Capture, Review, and Send, open Settings, then check Workspace first. If the wrong repository is selected, choose the correct workspace for this project and rerun. If the workspace is already correct, open Settings > Agent and switch Sandbox to danger-full-access before rerunning."
		if strings.TrimSpace(workdir) != "" {
			message += " Workspace used: " + strings.TrimSpace(workdir) + "."
		}
		return submitOutcomeTrustedDir, "Trusted directory required", message
	}
	if containsAny(logText,
		"not a git repository",
		"wrong workspace",
		"workspace does not match",
		"workspace didn't match",
		"workspace did not match",
	) {
		message := "Go back to Capture, Review, and Send, open Settings > Workspace, and choose the repository that matches this request before rerunning."
		if strings.TrimSpace(workdir) != "" {
			message += " Workspace used: " + strings.TrimSpace(workdir) + "."
		}
		return submitOutcomeWrongWorkspace, "Wrong workspace", message
	}
	if containsAny(logText,
		"sandbox: read-only",
		"workspace is read-only",
		"read-only file system",
		"operation not permitted",
		"apply_patch tool call failed",
		"failed with `operation not permitted`",
	) {
		return submitOutcomeReadOnly, "Read-only", "Go back to Capture, Review, and Send, open Settings > Agent, and switch Sandbox to danger-full-access before rerunning."
	}
	return "", "", ""
}

func readSubmitOutcomeLogText(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxSubmitOutcomeLogBytes))
	if err != nil {
		return ""
	}
	return string(b)
}

func extractSubmitAgentSummary(logText string) string {
	commentary := extractSubmitAgentCommentary(logText)
	if commentary == "" {
		return ""
	}
	if summary := extractExplicitSubmitSummary(commentary); summary != "" {
		return summary
	}
	blocks := strings.Split(strings.ReplaceAll(commentary, "\r\n", "\n"), "\n\n")
	for i := len(blocks) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(blocks[i])
		if candidate == "" {
			continue
		}
		if submitSummaryLooksActionable(candidate) {
			return truncateSubmitAgentSummary(candidate)
		}
	}
	lines := nonEmptySubmitLines(commentary)
	for i := len(lines) - 1; i >= 0; i-- {
		if submitSummaryLooksActionable(lines[i]) {
			return truncateSubmitAgentSummary(lines[i])
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return truncateSubmitAgentSummary(lines[len(lines)-1])
}

func extractSubmitAgentCommentary(logText string) string {
	if strings.TrimSpace(logText) == "" {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(logText, "\r\n", "\n"), "\n")
	commentary := make([]string, 0, len(lines))
	mode := "work"
	omittingPrompt := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "user" {
			omittingPrompt = true
			mode = "work"
			continue
		}
		if omittingPrompt {
			if trimmed == "codex" || isSubmitLiveOutputWorkMarker(trimmed) {
				omittingPrompt = false
			} else {
				continue
			}
		}
		if trimmed == "codex" {
			mode = "commentary"
			continue
		}
		if isSubmitLikelyPayloadLine(line) {
			continue
		}
		if isSubmitLiveOutputWorkMarker(trimmed) {
			mode = "work"
			continue
		}
		if mode == "commentary" || isSubmitLikelyCommentaryLine(trimmed) {
			commentary = append(commentary, line)
		}
	}
	return strings.TrimSpace(strings.Join(commentary, "\n"))
}

func extractExplicitSubmitSummary(commentary string) string {
	lines := nonEmptySubmitLines(commentary)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		lower := strings.ToLower(line)
		for _, prefix := range []string{"summary:", "short summary:", "final summary:", "change summary:"} {
			if strings.HasPrefix(lower, prefix) {
				return truncateSubmitAgentSummary(strings.TrimSpace(line[len(prefix):]))
			}
		}
	}
	return ""
}

func nonEmptySubmitLines(text string) []string {
	raw := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func submitSummaryLooksActionable(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	if strings.HasPrefix(lower, "i'm ") || strings.HasPrefix(lower, "i am ") || strings.HasPrefix(lower, "i found") || strings.HasPrefix(lower, "next i") || strings.HasPrefix(lower, "i can") || strings.HasPrefix(lower, "the current ") {
		return false
	}
	return containsAny(lower,
		"changed",
		"updated",
		"added",
		"fixed",
		"implemented",
		"removed",
		"wired",
		"rewrote",
		"reworked",
		"refactored",
		"documented",
		"made no changes",
		"no files changed",
		"tests passed",
		"now",
	)
}

func truncateSubmitAgentSummary(text string) string {
	value := strings.TrimSpace(text)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxSubmitAgentSummaryLen {
		return value
	}
	return strings.TrimSpace(string(runes[:maxSubmitAgentSummaryLen-1])) + "…"
}

func isSubmitLiveOutputWorkMarker(trimmed string) bool {
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "T") {
		return true
	}
	if trimmed == "exec" || strings.HasPrefix(trimmed, "exec ") {
		return true
	}
	if strings.HasPrefix(trimmed, "mcp:") {
		return true
	}
	if trimmed == "apply_patch" || strings.HasPrefix(trimmed, "apply_patch ") {
		return true
	}
	if trimmed == "--------" || strings.HasPrefix(trimmed, "OpenAI Codex v") {
		return true
	}
	if hasAnyPrefixFold(trimmed, "workdir:", "model:", "provider:", "approval:", "sandbox:", "reasoning effort:", "reasoning summaries:", "session id:") {
		return true
	}
	if hasAnyPrefixFold(trimmed, "/bin/", "bash ", "sh ", "git ", "rg ", "sed ", "cat ", "go ", "npm ", "pnpm ") {
		return true
	}
	if len(trimmed) > 4 && trimmed[3] == ':' && trimmed[0] >= '0' && trimmed[0] <= '9' {
		return true
	}
	return strings.Contains(trimmed, " succeeded in ") || strings.Contains(trimmed, " failed in ")
}

func isSubmitLikelyPayloadLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, `{"created":`) || strings.Contains(trimmed, `"inline_data_url"`) {
		return true
	}
	if strings.Contains(trimmed, "data:image/") || strings.Contains(trimmed, "data:video/") || strings.Contains(trimmed, "data:audio/") {
		return true
	}
	return len(line) > 4000
}

func isSubmitLikelyCommentaryLine(trimmed string) bool {
	lower := strings.ToLower(strings.TrimSpace(trimmed))
	return strings.HasPrefix(lower, "i'm") ||
		strings.HasPrefix(lower, "i am") ||
		strings.HasPrefix(lower, "i'll") ||
		strings.HasPrefix(lower, "i will") ||
		strings.HasPrefix(lower, "i have") ||
		strings.HasPrefix(lower, "i found") ||
		strings.HasPrefix(lower, "i can") ||
		strings.HasPrefix(lower, "next i") ||
		strings.HasPrefix(lower, "the current ") ||
		strings.HasPrefix(lower, "this is ") ||
		strings.HasPrefix(lower, "that means")
}

func hasAnyPrefixFold(s string, prefixes ...string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, strings.ToLower(strings.TrimSpace(prefix))) {
			return true
		}
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func containsAny(s string, needles ...string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	for _, n := range needles {
		if strings.Contains(s, strings.ToLower(strings.TrimSpace(n))) {
			return true
		}
	}
	return false
}

func submitRequestPreview(pkg session.CanonicalPackage) string {
	preview := strings.TrimSpace(pkg.Summary)
	if preview == "" {
		for _, req := range pkg.ChangeRequests {
			preview = strings.TrimSpace(req.Summary)
			if preview != "" {
				break
			}
		}
	}
	if preview == "" {
		return ""
	}
	return truncatePreview(preview, maxSubmitRequestPreviewLen)
}

func (s *Server) effectiveSubmitWorkspace(provider string) string {
	switch canonicalProviderAlias(provider, nil) {
	case "codex_cli", "claude_cli", "opencode_cli":
		rc := s.currentRuntimeCodex()
		if workdir := strings.TrimSpace(rc.CodexWorkdir); workdir != "" {
			return workdir
		}
		if workdir := strings.TrimSpace(os.Getenv("KNIT_CODEX_WORKDIR")); workdir != "" {
			return workdir
		}
		if cwd, err := os.Getwd(); err == nil {
			return strings.TrimSpace(cwd)
		}
	}
	return ""
}

func truncatePreview(value string, maxLen int) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if trimmed == "" || maxLen < 2 {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= maxLen {
		return trimmed
	}
	return strings.TrimSpace(string(runes[:maxLen-1])) + "…"
}

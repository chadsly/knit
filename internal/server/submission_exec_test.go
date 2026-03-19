package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"knit/internal/agents"
	"knit/internal/config"
	"knit/internal/session"
)

func TestNormalizeSubmitExecutionMode(t *testing.T) {
	if got := normalizeSubmitExecutionMode(""); got != submitExecutionSeries {
		t.Fatalf("expected default series, got %q", got)
	}
	if got := normalizeSubmitExecutionMode("parallel"); got != submitExecutionParallel {
		t.Fatalf("expected parallel, got %q", got)
	}
	if got := normalizeSubmitExecutionMode("PARALLEL"); got != submitExecutionParallel {
		t.Fatalf("expected case-insensitive parallel, got %q", got)
	}
	if got := normalizeSubmitExecutionMode("weird"); got != submitExecutionSeries {
		t.Fatalf("expected unknown mode to normalize to series, got %q", got)
	}
}

func TestSubmitRequestPreviewUsesSummaryAndTruncates(t *testing.T) {
	longSummary := strings.Repeat("queue detail ", 20)
	pkg := session.CanonicalPackage{
		Summary: longSummary,
		ChangeRequests: []session.ChangeReq{
			{Summary: "first request"},
		},
	}

	got := submitRequestPreview(pkg)
	if got == "" {
		t.Fatal("expected request preview")
	}
	if !strings.HasPrefix(got, "queue detail") {
		t.Fatalf("expected preview to come from package summary, got %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected preview to be truncated with ellipsis, got %q", got)
	}
	if len([]rune(got)) > maxSubmitRequestPreviewLen {
		t.Fatalf("expected preview length <= %d, got %d", maxSubmitRequestPreviewLen, len([]rune(got)))
	}
}

func TestSubmitRequestPreviewFallsBackToFirstChangeRequest(t *testing.T) {
	pkg := session.CanonicalPackage{
		ChangeRequests: []session.ChangeReq{
			{Summary: "Show the currently running request in the queue"},
			{Summary: "Second request"},
		},
	}

	if got := submitRequestPreview(pkg); got != "Show the currently running request in the queue" {
		t.Fatalf("expected first change request summary, got %q", got)
	}
}

func TestAllocateSubmitExecutionLogPathUsesLogSuffix(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	logDir := t.TempDir()
	t.Setenv("KNIT_CODEX_OUTPUT_DIR", logDir)

	logPath, err := srv.allocateSubmitExecutionLogPath("attempt:1 / name")
	if err != nil {
		t.Fatalf("allocate submit execution log path: %v", err)
	}
	if filepath.Dir(logPath) != logDir {
		t.Fatalf("expected log dir %q, got %q", logDir, filepath.Dir(logPath))
	}
	if !strings.HasSuffix(logPath, ".log") {
		t.Fatalf("expected .log suffix, got %q", logPath)
	}
	if strings.Contains(logPath, "XXXX") {
		t.Fatalf("expected sanitized temp name without literal XXXX, got %q", logPath)
	}
}

func TestSubmissionQueueSeriesProcessesInOrder(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionSeries)
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	adapter := &fakeQueueAdapter{name: "fake_queue", delay: 120 * time.Millisecond}
	srv.agents = agents.NewRegistry(adapter)

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "first", NormalizedText: "first"}); err != nil {
		t.Fatalf("add first feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve first: %v", err)
	}
	pkg1, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve first package: %v", err)
	}
	srv.enqueueSubmitJob("fake_queue", *pkg1, map[string]any{"id": "one"}, agents.DeliveryIntent{}, "test", "test")

	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "second", NormalizedText: "second"}); err != nil {
		t.Fatalf("add second feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve second: %v", err)
	}
	pkg2, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve second package: %v", err)
	}
	srv.enqueueSubmitJob("fake_queue", *pkg2, map[string]any{"id": "two"}, agents.DeliveryIntent{}, "test", "test")

	waitForSubmitDrain(t, srv, 3*time.Second)
	attempts := srv.submitAttemptsSnapshot()
	if len(attempts) < 2 {
		t.Fatalf("expected at least 2 attempts, got %d", len(attempts))
	}
	for i := 0; i < 2; i++ {
		if attempts[i].Status != "submitted" {
			t.Fatalf("expected attempt %s submitted, got %s", attempts[i].AttemptID, attempts[i].Status)
		}
	}
	if got := adapter.maxConcurrency(); got != 1 {
		t.Fatalf("expected series mode max concurrency 1, got %d", got)
	}
	lens := adapter.requestLens()
	if len(lens) < 2 {
		t.Fatalf("expected 2 adapter calls, got %d", len(lens))
	}
	if lens[0] != 1 || lens[1] != 1 {
		t.Fatalf("expected each queued submission to contain only its newly approved request, got %v", lens[:2])
	}
}

func TestSubmissionQueueParallelDefersPostSubmitUntilDrain(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionParallel)
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	adapter := &fakeQueueAdapter{name: "fake_queue", delay: 150 * time.Millisecond}
	srv.agents = agents.NewRegistry(adapter)

	var postSubmitRuns int
	var seenRunning int
	var seenQueued int
	srv.postSubmitRunner = func() *postSubmitResult {
		postSubmitRuns++
		q := srv.submitQueueState()
		seenRunning, _ = q["running"].(int)
		seenQueued, _ = q["queued"].(int)
		return &postSubmitResult{
			Enabled: true,
			Rebuild: &postSubmitStepResult{
				Command: "echo rebuild",
				Status:  "success",
			},
		}
	}

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "first", NormalizedText: "first"}); err != nil {
		t.Fatalf("add first feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve first: %v", err)
	}
	pkg1, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve first package: %v", err)
	}
	srv.enqueueSubmitJob("fake_queue", *pkg1, map[string]any{"id": "one"}, agents.DeliveryIntent{}, "test", "test")

	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "second", NormalizedText: "second"}); err != nil {
		t.Fatalf("add second feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve second: %v", err)
	}
	pkg2, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve second package: %v", err)
	}
	srv.enqueueSubmitJob("fake_queue", *pkg2, map[string]any{"id": "two"}, agents.DeliveryIntent{}, "test", "test")

	waitForSubmitDrain(t, srv, 3*time.Second)
	if got := adapter.maxConcurrency(); got < 2 {
		t.Fatalf("expected parallel mode to run concurrently, max concurrency=%d", got)
	}
	if postSubmitRuns != 1 {
		t.Fatalf("expected one post-submit run after parallel drain, got %d", postSubmitRuns)
	}
	if seenRunning != 0 || seenQueued != 0 {
		t.Fatalf("expected post-submit to run after queue drained, saw running=%d queued=%d", seenRunning, seenQueued)
	}
}

func TestSubmissionRetryBackoffSucceedsOnSecondAttempt(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionSeries)
	t.Setenv("KNIT_SUBMIT_MAX_ATTEMPTS", "3")
	t.Setenv("KNIT_SUBMIT_RETRY_BACKOFF_SECONDS", "1")
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	adapter := &fakeQueueAdapter{name: "fake_queue", delay: 10 * time.Millisecond, failUntilCall: 1}
	srv.agents = agents.NewRegistry(adapter)

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "retry me", NormalizedText: "retry me"}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	pkg, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve package: %v", err)
	}
	attempt := srv.enqueueSubmitJob("fake_queue", *pkg, map[string]any{"id": "retry"}, agents.DeliveryIntent{}, "test", "test")
	waitForSubmitDrain(t, srv, 4*time.Second)

	snapshot, ok := srv.submitAttemptByID(attempt.AttemptID)
	if !ok {
		t.Fatalf("attempt not found")
	}
	if snapshot.Status != "submitted" {
		t.Fatalf("expected submitted after retry, got %s", snapshot.Status)
	}
	if snapshot.RetryCount != 1 {
		t.Fatalf("expected retry_count=1, got %d", snapshot.RetryCount)
	}
	if adapter.calls() != 2 {
		t.Fatalf("expected two submit calls, got %d", adapter.calls())
	}
	foundRetryEvent := false
	for _, e := range snapshot.Timeline {
		if e.Status == "retry_wait" {
			foundRetryEvent = true
			break
		}
	}
	if !foundRetryEvent {
		t.Fatalf("expected retry_wait timeline event")
	}
}

func TestSubmissionOfflineDeferredRetriesAndEventuallySubmits(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionSeries)
	t.Setenv("KNIT_SUBMIT_MAX_ATTEMPTS", "1")
	t.Setenv("KNIT_OFFLINE_RETRY_SECONDS", "1")
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	adapter := &fakeQueueAdapter{
		name:          "fake_remote",
		delay:         10 * time.Millisecond,
		remote:        true,
		failUntilCall: 1,
		failError:     "dial tcp: lookup api.openai.com: no such host",
	}
	srv.agents = agents.NewRegistry(adapter)

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "offline retry", NormalizedText: "offline retry"}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	pkg, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve package: %v", err)
	}
	attempt := srv.enqueueSubmitJob("fake_remote", *pkg, map[string]any{"id": "offline"}, agents.DeliveryIntent{}, "test", "test")
	waitForAttemptStatusDirect(t, srv, attempt.AttemptID, "submitted", 6*time.Second)

	snapshot, ok := srv.submitAttemptByID(attempt.AttemptID)
	if !ok {
		t.Fatalf("attempt not found")
	}
	if adapter.calls() < 2 {
		t.Fatalf("expected at least 2 calls after deferred retry, got %d", adapter.calls())
	}
	foundDeferred := false
	for _, e := range snapshot.Timeline {
		if e.Status == "deferred_offline" {
			foundDeferred = true
			break
		}
	}
	if !foundDeferred {
		t.Fatalf("expected deferred_offline timeline event")
	}
}

func TestCancelQueuedSubmitAttemptMarksAttemptCanceled(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionSeries)
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	adapter := &fakeQueueAdapter{name: "fake_queue", delay: 300 * time.Millisecond}
	srv.agents = agents.NewRegistry(adapter)

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "first", NormalizedText: "first"}); err != nil {
		t.Fatalf("add first feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve first: %v", err)
	}
	pkg1, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve first package: %v", err)
	}
	first := srv.enqueueSubmitJob("fake_queue", *pkg1, map[string]any{"id": "one"}, agents.DeliveryIntent{}, "test", "test")

	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "second", NormalizedText: "second"}); err != nil {
		t.Fatalf("add second feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve second: %v", err)
	}
	pkg2, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve second package: %v", err)
	}
	second := srv.enqueueSubmitJob("fake_queue", *pkg2, map[string]any{"id": "two"}, agents.DeliveryIntent{}, "test", "test")

	waitForAttemptStatusDirect(t, srv, first.AttemptID, "in_progress", 2*time.Second)
	if _, found, err := srv.cancelSubmitAttempt(second.AttemptID, "test", "test"); err != nil || !found {
		t.Fatalf("cancel queued attempt err=%v found=%v", err, found)
	}

	waitForAttemptStatusDirect(t, srv, first.AttemptID, "submitted", 3*time.Second)
	waitForAttemptStatusDirect(t, srv, second.AttemptID, submitStatusCanceled, 2*time.Second)

	snapshot, ok := srv.submitAttemptByID(second.AttemptID)
	if !ok {
		t.Fatalf("queued attempt not found")
	}
	if snapshot.Status != submitStatusCanceled {
		t.Fatalf("expected canceled queued attempt, got %s", snapshot.Status)
	}
	if adapter.calls() != 1 {
		t.Fatalf("expected only running attempt to reach adapter, got %d calls", adapter.calls())
	}
}

func TestCancelRunningSubmitAttemptStopsExecution(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionSeries)
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	adapter := &fakeQueueAdapter{name: "fake_queue", delay: 2 * time.Second}
	srv.agents = agents.NewRegistry(adapter)

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "cancel running", NormalizedText: "cancel running"}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	pkg, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve package: %v", err)
	}
	attempt := srv.enqueueSubmitJob("fake_queue", *pkg, map[string]any{"id": "cancel"}, agents.DeliveryIntent{}, "test", "test")

	waitForAttemptStatusDirect(t, srv, attempt.AttemptID, "in_progress", 2*time.Second)
	if _, found, err := srv.cancelSubmitAttempt(attempt.AttemptID, "test", "test"); err != nil || !found {
		t.Fatalf("cancel running attempt err=%v found=%v", err, found)
	}
	waitForAttemptStatusDirect(t, srv, attempt.AttemptID, submitStatusCanceled, 3*time.Second)

	snapshot, ok := srv.submitAttemptByID(attempt.AttemptID)
	if !ok {
		t.Fatalf("running attempt not found")
	}
	if snapshot.Status != submitStatusCanceled {
		t.Fatalf("expected canceled running attempt, got %s", snapshot.Status)
	}
	if strings.TrimSpace(snapshot.Error) != "" {
		t.Fatalf("expected canceled attempt to avoid error text, got %q", snapshot.Error)
	}
}

func TestNamedCLIProvidersCaptureExecutionLog(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)

	command := `sh -lc 'printf "claude live output\n" >> "$KNIT_CLI_LOG_FILE"; echo "{\"run_id\":\"claude-run\",\"status\":\"accepted\",\"ref\":\"claude-ref\"}"'`
	if runtime.GOOS == "windows" {
		command = `powershell -Command "$p=$env:KNIT_CLI_LOG_FILE; Add-Content -Path $p -Value 'claude live output'; Write-Output '{\"run_id\":\"claude-run\",\"status\":\"accepted\",\"ref\":\"claude-ref\"}'"`
	}
	t.Setenv("KNIT_CLAUDE_CLI_ADAPTER_CMD", command)
	srv.agents = agents.NewRegistry(agents.NewCodexAPIAdapterFromEnv(), agents.NewCLIAdapterFromEnv(), agents.NewClaudeCLIAdapterFromEnv(), agents.NewOpenCodeCLIAdapterFromEnv())

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "stream claude output", NormalizedText: "stream claude output"}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	pkg, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve package: %v", err)
	}
	attempt := srv.enqueueSubmitJob("claude_cli", *pkg, map[string]any{"id": "claude-log"}, agents.DeliveryIntent{}, "test", "test")
	waitForSubmitDrain(t, srv, 3*time.Second)

	snapshot, ok := srv.submitAttemptByID(attempt.AttemptID)
	if !ok {
		t.Fatalf("attempt not found")
	}
	if snapshot.Status != "submitted" {
		t.Fatalf("expected submitted attempt, got %s", snapshot.Status)
	}
	if strings.TrimSpace(snapshot.ExecutionRef) == "" {
		t.Fatal("expected execution log path for named cli provider")
	}
	b, err := os.ReadFile(snapshot.ExecutionRef)
	if err != nil {
		t.Fatalf("read execution log: %v", err)
	}
	if !strings.Contains(string(b), "claude live output") {
		t.Fatalf("expected named cli output in execution log, got %q", string(b))
	}
}

func TestSubmitQueueRecoveryLoadsQueuedJobsAfterRestart(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionSeries)
	t.Setenv("KNIT_SUBMIT_MAX_ATTEMPTS", "1")
	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"recover-run","status":"accepted","ref":"/tmp/recovered.log"}'`)
	dataDir := t.TempDir()
	queuePath := filepath.Join(dataDir, "submit_queue.json")

	persisted := submitQueueStatePersist{
		Pending: []persistedSubmitJob{
			{
				AttemptID:   "attempt-recovered-1",
				Provider:    "cli",
				Mode:        submitExecutionSeries,
				MaxAttempts: 1,
				Package: session.CanonicalPackage{
					SessionID:      "sess-recovered",
					ChangeRequests: []session.ChangeReq{{EventID: "evt-1", Summary: "Recovered update"}},
				},
				EnqueuedAt: time.Now().UTC(),
			},
		},
	}
	b, err := json.Marshal(persisted)
	if err != nil {
		t.Fatalf("marshal persisted queue: %v", err)
	}
	if err := os.WriteFile(queuePath, b, 0o600); err != nil {
		t.Fatalf("write persisted queue file: %v", err)
	}

	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.DataDir = dataDir
	cfg.SQLitePath = filepath.Join(dataDir, "recover.db")
	srv := newTestServer(t, cfg)
	waitForSubmitDrain(t, srv, 3*time.Second)

	a, ok := srv.submitAttemptByID("attempt-recovered-1")
	if !ok {
		t.Fatalf("expected recovered attempt in history")
	}
	if a.RequestPreview != "Recovered update" {
		t.Fatalf("expected recovered request preview, got %q", a.RequestPreview)
	}
	if a.Status != "submitted" {
		t.Fatalf("expected recovered attempt to submit, got %s", a.Status)
	}
	foundRecoveredEvent := false
	for _, evt := range a.Timeline {
		if evt.Status == "recovered" {
			foundRecoveredEvent = true
			break
		}
	}
	if !foundRecoveredEvent {
		t.Fatalf("expected recovered timeline event")
	}
}

func TestSubmitQueueRecoveryHonorsDeferredUntil(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionSeries)
	t.Setenv("KNIT_SUBMIT_MAX_ATTEMPTS", "1")
	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"recover-deferred-run","status":"accepted","ref":"/tmp/recovered-deferred.log"}'`)
	dataDir := t.TempDir()
	queuePath := filepath.Join(dataDir, "submit_queue.json")
	deferredUntil := time.Now().UTC().Add(450 * time.Millisecond)

	persisted := submitQueueStatePersist{
		Pending: []persistedSubmitJob{
			{
				AttemptID:   "attempt-recovered-deferred-1",
				Provider:    "cli",
				Mode:        submitExecutionSeries,
				MaxAttempts: 1,
				Package: session.CanonicalPackage{
					SessionID:      "sess-recovered-deferred",
					ChangeRequests: []session.ChangeReq{{EventID: "evt-1", Summary: "Recovered deferred update"}},
				},
				EnqueuedAt:    time.Now().UTC(),
				DeferredUntil: deferredUntil,
			},
		},
	}
	b, err := json.Marshal(persisted)
	if err != nil {
		t.Fatalf("marshal persisted queue: %v", err)
	}
	if err := os.WriteFile(queuePath, b, 0o600); err != nil {
		t.Fatalf("write persisted queue file: %v", err)
	}

	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.DataDir = dataDir
	cfg.SQLitePath = filepath.Join(dataDir, "recover-deferred.db")
	srv := newTestServer(t, cfg)

	time.Sleep(120 * time.Millisecond)
	a, ok := srv.submitAttemptByID("attempt-recovered-deferred-1")
	if !ok {
		t.Fatalf("expected recovered attempt in history")
	}
	if a.Status != "deferred_offline" {
		t.Fatalf("expected recovered deferred attempt status deferred_offline, got %s", a.Status)
	}
	if a.RequestPreview != "Recovered deferred update" {
		t.Fatalf("expected deferred recovered request preview, got %q", a.RequestPreview)
	}
	if a.NextRetryAt == nil {
		t.Fatalf("expected next_retry_at to be set for deferred recovery")
	}

	waitForAttemptStatusDirect(t, srv, "attempt-recovered-deferred-1", "submitted", 4*time.Second)
}

func TestPersistSubmitQueueOmitsProviderPayload(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionSeries)
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.DataDir = t.TempDir()
	cfg.SQLitePath = filepath.Join(cfg.DataDir, "queue.db")
	srv := newTestServer(t, cfg)

	srv.sessions.Start("Browser Preview", "https://example.com")
	if _, err := srv.sessions.AddFeedback(session.FeedbackEvt{RawTranscript: "queued", NormalizedText: "queued"}); err != nil {
		t.Fatalf("add feedback: %v", err)
	}
	if _, err := srv.sessions.Approve(""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	pkg, err := srv.sessions.ReserveApprovedPackage()
	if err != nil {
		t.Fatalf("reserve package: %v", err)
	}
	srv.submitMu.Lock()
	srv.submitPending = append(srv.submitPending, submitJob{
		AttemptID:       "attempt-omit-payload",
		Provider:        "codex_cli",
		Mode:            submitExecutionSeries,
		MaxAttempts:     1,
		Package:         *pkg,
		ProviderPayload: map[string]any{"package": map[string]any{"artifacts": []map[string]any{{"inline_data_url": "data:video/webm;base64,AAAA"}}}},
		EnqueuedAt:      time.Now().UTC(),
	})
	srv.persistSubmitQueueLocked()
	srv.submitMu.Unlock()

	b, err := os.ReadFile(filepath.Join(cfg.DataDir, "submit_queue.json"))
	if err != nil {
		t.Fatalf("read submit queue: %v", err)
	}
	body := string(b)
	if strings.Contains(body, "provider_payload") {
		t.Fatalf("expected persisted queue to omit provider_payload")
	}
	if strings.Contains(body, "inline_data_url") {
		t.Fatalf("expected persisted queue to omit inline data from provider payload")
	}
}

func TestEffectiveSubmitWorkspaceUsesConfiguredCodexWorkdir(t *testing.T) {
	t.Setenv("KNIT_CODEX_WORKDIR", "/tmp/intent-manager")
	cfg := config.Default()
	cfg.ControlToken = "test-token"
	srv := newTestServer(t, cfg)
	if got := srv.effectiveSubmitWorkspace("codex_cli"); got != "/tmp/intent-manager" {
		t.Fatalf("expected effective workspace to be recorded, got %q", got)
	}
}

func TestSubmitQueueRecoveryRepairsMalformedRecoveredJob(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionSeries)
	t.Setenv("KNIT_SUBMIT_MAX_ATTEMPTS", "1")
	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"recover-run","status":"accepted","ref":"/tmp/recovered.log"}'`)
	dataDir := t.TempDir()
	queuePath := filepath.Join(dataDir, "submit_queue.json")

	persisted := submitQueueStatePersist{
		Running: []persistedSubmitJob{{
			Provider: "cli",
			Mode:     submitExecutionSeries,
			Package: session.CanonicalPackage{
				SessionID:      "sess-recovered-fixed",
				ChangeRequests: []session.ChangeReq{{EventID: "evt-1", Summary: "Recovered fixed update"}},
			},
		}},
	}
	b, err := json.Marshal(persisted)
	if err != nil {
		t.Fatalf("marshal persisted queue: %v", err)
	}
	if err := os.WriteFile(queuePath, b, 0o600); err != nil {
		t.Fatalf("write persisted queue file: %v", err)
	}

	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.DataDir = dataDir
	cfg.SQLitePath = filepath.Join(dataDir, "recover-fixed.db")
	srv := newTestServer(t, cfg)
	waitForSubmitDrain(t, srv, 3*time.Second)

	attempts := srv.submitAttemptsSnapshot()
	if len(attempts) == 0 {
		t.Fatalf("expected recovered attempt history")
	}
	if attempts[0].AttemptID == "" {
		t.Fatalf("expected repaired attempt id")
	}
	notes := srv.submitRecoveryNotesSnapshot()
	if len(notes) == 0 {
		t.Fatalf("expected submit recovery notice")
	}
	if !strings.Contains(strings.Join(notes, "\n"), "reassigned") {
		t.Fatalf("expected repair notice, got %v", notes)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "Current run or Recent runs") {
		t.Fatalf("expected resumed-status guidance in recovery notice, got %v", notes)
	}
}

func TestSubmitQueueRecoveryDiscardsMalformedRecoveredJob(t *testing.T) {
	dataDir := t.TempDir()
	queuePath := filepath.Join(dataDir, "submit_queue.json")

	persisted := submitQueueStatePersist{
		Running: []persistedSubmitJob{{
			Provider: "codex_cli",
			Mode:     submitExecutionSeries,
			Package:  session.CanonicalPackage{},
		}},
	}
	b, err := json.Marshal(persisted)
	if err != nil {
		t.Fatalf("marshal persisted queue: %v", err)
	}
	if err := os.WriteFile(queuePath, b, 0o600); err != nil {
		t.Fatalf("write persisted queue file: %v", err)
	}

	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.DataDir = dataDir
	cfg.SQLitePath = filepath.Join(dataDir, "recover-discard.db")
	srv := newTestServer(t, cfg)

	if got := srv.submitAttemptsSnapshot(); len(got) != 0 {
		t.Fatalf("expected malformed recovered job to be discarded, got %d attempts", len(got))
	}
	notes := srv.submitRecoveryNotesSnapshot()
	if len(notes) == 0 || !strings.Contains(strings.Join(notes, "\n"), "Discarded a stale recovered delivery") {
		t.Fatalf("expected discard notice, got %v", notes)
	}
}

func TestSubmitQueueRecoveryFailsWhenCLIProviderCommandIsInvalid(t *testing.T) {
	t.Setenv("KNIT_SUBMIT_EXECUTION_MODE", submitExecutionSeries)
	t.Setenv("KNIT_SUBMIT_MAX_ATTEMPTS", "1")
	t.Setenv("KNIT_CLI_ADAPTER_CMD", "knit-codex-missing-command")
	dataDir := t.TempDir()
	queuePath := filepath.Join(dataDir, "submit_queue.json")

	persisted := submitQueueStatePersist{
		Pending: []persistedSubmitJob{
			{
				AttemptID:   "attempt-recovered-missing-cli",
				Provider:    "codex_cli",
				Mode:        submitExecutionSeries,
				MaxAttempts: 1,
				Package: session.CanonicalPackage{
					SessionID:      "sess-recovered-missing-cli",
					ChangeRequests: []session.ChangeReq{{EventID: "evt-1", Summary: "Recovered update should fail clearly"}},
				},
				EnqueuedAt: time.Now().UTC(),
			},
		},
	}
	b, err := json.Marshal(persisted)
	if err != nil {
		t.Fatalf("marshal persisted queue: %v", err)
	}
	if err := os.WriteFile(queuePath, b, 0o600); err != nil {
		t.Fatalf("write persisted queue file: %v", err)
	}

	cfg := config.Default()
	cfg.ControlToken = "test-token"
	cfg.DataDir = dataDir
	cfg.SQLitePath = filepath.Join(dataDir, "recover-missing-cli.db")
	srv := newTestServer(t, cfg)
	waitForAttemptStatusDirect(t, srv, "attempt-recovered-missing-cli", "failed", 3*time.Second)

	a, ok := srv.submitAttemptByID("attempt-recovered-missing-cli")
	if !ok {
		t.Fatalf("expected recovered attempt in history")
	}
	if !strings.Contains(strings.ToLower(a.Error), "knit-codex-missing-command") {
		t.Fatalf("expected invalid cli command error, got %q", a.Error)
	}
	if strings.HasPrefix(a.RunID, "stub-") {
		t.Fatalf("expected no stub run id, got %q", a.RunID)
	}
}

func waitForSubmitDrain(t *testing.T, srv *Server, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		q := srv.submitQueueState()
		running, _ := q["running"].(int)
		queued, _ := q["queued"].(int)
		postRunning, _ := q["post_submit_running"].(bool)
		if running == 0 && queued == 0 && !postRunning {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for submit drain; state=%v", srv.submitQueueState())
}

func waitForAttemptStatusDirect(t *testing.T, srv *Server, attemptID, want string, timeout time.Duration) submitAttempt {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		a, ok := srv.submitAttemptByID(attemptID)
		if ok && a.Status == want {
			return a
		}
		time.Sleep(20 * time.Millisecond)
	}
	a, _ := srv.submitAttemptByID(attemptID)
	t.Fatalf("timeout waiting for attempt %s status=%s; got=%s timeline=%v", attemptID, want, a.Status, a.Timeline)
	return submitAttempt{}
}

type fakeQueueAdapter struct {
	name          string
	delay         time.Duration
	failUntilCall int
	remote        bool
	failError     string

	mu      sync.Mutex
	active  int
	max     int
	callSeq int
	reqLens []int
}

func (a *fakeQueueAdapter) Name() string { return a.name }
func (a *fakeQueueAdapter) IsRemote() bool {
	return a.remote
}
func (a *fakeQueueAdapter) Endpoint() string {
	if a.remote {
		return "https://api.example.com"
	}
	return "local-process"
}

func (a *fakeQueueAdapter) Submit(ctx context.Context, pkg session.CanonicalPackage) (agents.Result, error) {
	a.mu.Lock()
	a.callSeq++
	call := a.callSeq
	a.active++
	if a.active > a.max {
		a.max = a.active
	}
	a.reqLens = append(a.reqLens, len(pkg.ChangeRequests))
	a.mu.Unlock()

	timer := time.NewTimer(a.delay)
	select {
	case <-ctx.Done():
		timer.Stop()
		a.mu.Lock()
		a.active--
		a.mu.Unlock()
		return agents.Result{}, ctx.Err()
	case <-timer.C:
	}

	if a.failUntilCall > 0 && call <= a.failUntilCall {
		a.mu.Lock()
		a.active--
		a.mu.Unlock()
		if strings.TrimSpace(a.failError) != "" {
			return agents.Result{}, fmt.Errorf("%s", a.failError)
		}
		return agents.Result{}, fmt.Errorf("forced adapter failure on call %d", call)
	}

	a.mu.Lock()
	a.active--
	a.mu.Unlock()

	return agents.Result{
		Provider: a.name,
		RunID:    fmt.Sprintf("run-%d", call),
		Status:   "accepted",
		Ref:      fmt.Sprintf("ref-%d", call),
	}, nil
}

func (a *fakeQueueAdapter) maxConcurrency() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.max
}

func (a *fakeQueueAdapter) requestLens() []int {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]int, len(a.reqLens))
	copy(out, a.reqLens)
	return out
}

func (a *fakeQueueAdapter) calls() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.callSeq
}

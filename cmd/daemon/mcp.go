package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"knit/internal/app"
	"knit/internal/config"
)

type daemonClient struct {
	baseURL string
	token   string
	http    *http.Client
}

const (
	daemonInstanceModeManaged      = "mcp_managed"
	executionModeWorker            = "mcp_worker"
	queueCodeWorkerNotConnected    = "worker_not_connected"
	queueCodeMCPAttachIncompatible = "mcp_attach_incompatible_daemon"
)

type managedDaemon struct {
	cmd  *exec.Cmd
	owns bool
}

type mcpRegisterWorkerInput struct {
	WorkerID           string   `json:"worker_id,omitempty" jsonschema:"Optional worker session id; generated when omitted"`
	Label              string   `json:"label,omitempty" jsonschema:"Optional worker session label"`
	Source             string   `json:"source,omitempty" jsonschema:"Optional worker session source metadata"`
	SupportedProviders []string `json:"supported_providers,omitempty" jsonschema:"Request providers this worker session can process"`
}

type mcpHeartbeatWorkerInput struct {
	WorkerID string `json:"worker_id,omitempty" jsonschema:"Optional worker session id; defaults to the active worker session"`
}

type mcpUnregisterWorkerInput struct {
	WorkerID string `json:"worker_id,omitempty" jsonschema:"Optional worker session id; defaults to the active worker session"`
}

type mcpWaitInput struct {
	WorkerID       string `json:"worker_id,omitempty" jsonschema:"Optional worker session id; defaults to the active worker session"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"Maximum seconds to wait for a queued request"`
}

type mcpClaimInput struct {
	WorkerID  string `json:"worker_id,omitempty" jsonschema:"Optional worker session id; defaults to the active worker session"`
	AttemptID string `json:"attempt_id" jsonschema:"Queued Knit attempt id to claim"`
}

type mcpGetInput struct {
	WorkerID  string `json:"worker_id,omitempty" jsonschema:"Optional worker session id; defaults to the active worker session"`
	AttemptID string `json:"attempt_id" jsonschema:"Claimed Knit attempt id to fetch"`
}

type mcpCompleteInput struct {
	WorkerID  string `json:"worker_id,omitempty" jsonschema:"Optional worker session id; defaults to the active worker session"`
	AttemptID string `json:"attempt_id" jsonschema:"Claimed Knit attempt id to complete"`
	Summary   string `json:"summary,omitempty" jsonschema:"Short completion summary"`
	Ref       string `json:"ref,omitempty" jsonschema:"Optional run or log reference"`
}

type mcpFailInput struct {
	WorkerID  string `json:"worker_id,omitempty" jsonschema:"Optional worker session id; defaults to the active worker session"`
	AttemptID string `json:"attempt_id" jsonschema:"Claimed Knit attempt id to fail"`
	Code      string `json:"code,omitempty" jsonschema:"Stable daemon-authored failure code"`
	Error     string `json:"error,omitempty" jsonschema:"Failure message"`
}

type bridgeWorkerSession struct {
	mu                 sync.Mutex
	workerID           string
	label              string
	source             string
	supportedProviders []string
	heartbeatCancel    context.CancelFunc
}

func (s *bridgeWorkerSession) snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]any{
		"worker_id":           strings.TrimSpace(s.workerID),
		"label":               strings.TrimSpace(s.label),
		"source":              strings.TrimSpace(s.source),
		"supported_providers": append([]string(nil), s.supportedProviders...),
		"registered":          strings.TrimSpace(s.workerID) != "",
	}
}

func (s *bridgeWorkerSession) activeWorkerID(fallback string) string {
	workerID := strings.TrimSpace(fallback)
	if workerID != "" {
		return workerID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimSpace(s.workerID)
}

func (s *bridgeWorkerSession) setActive(client *daemonClient, workerID, label, source string, providers []string) {
	s.stop(client)
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.workerID = strings.TrimSpace(workerID)
	s.label = strings.TrimSpace(label)
	s.source = strings.TrimSpace(source)
	s.supportedProviders = append([]string(nil), providers...)
	s.heartbeatCancel = cancel
	s.mu.Unlock()
	go func(worker string) {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = client.postJSON("/api/mcp/worker/heartbeat", map[string]any{"worker_id": worker})
			}
		}
	}(strings.TrimSpace(workerID))
}

func (s *bridgeWorkerSession) stop(client *daemonClient) {
	s.mu.Lock()
	cancel := s.heartbeatCancel
	workerID := strings.TrimSpace(s.workerID)
	s.heartbeatCancel = nil
	s.workerID = ""
	s.label = ""
	s.source = ""
	s.supportedProviders = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if client != nil && workerID != "" {
		_, _ = client.postJSON("/api/mcp/worker/disconnect", map[string]any{"worker_id": workerID})
	}
}

func runMCPCommand(args []string) error {
	if len(args) > 0 && args[0] == "--desktop" {
		if len(args) != 1 {
			return fmt.Errorf("usage: knit mcp --desktop")
		}
		if err := configureDesktopEnvironment(); err != nil {
			return fmt.Errorf("desktop configuration: %w", err)
		}
	}
	if len(args) > 0 && strings.TrimSpace(args[0]) == "token" {
		return runMCPTokenCommand()
	}
	cfg := app.DefaultConfig()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	managed, err := ensureDaemonForMCP(cfg)
	if err != nil {
		return err
	}
	defer shutdownManagedDaemon(managed)

	client := newDaemonClient(cfg)
	tokenInfo, err := client.managedExtensionToken(true)
	if err != nil {
		return err
	}
	printMCPStartupBanner(cfg, managed.owns, tokenInfo)

	workerSession := &bridgeWorkerSession{}
	defer workerSession.stop(client)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "knit",
		Title:   "Knit MCP",
		Version: startupVersion(cfg),
	}, &mcp.ServerOptions{
		Logger:       slog.New(slog.NewTextHandler(ioDiscard{}, nil)),
		Capabilities: &mcp.ServerCapabilities{},
	})
	addDesktopTools(server, client)

	mcp.AddTool(server, &mcp.Tool{Name: "knit_status", Description: "Get Knit daemon, queue, worker, and extension setup diagnostics."},
		func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, map[string]any, error) {
			data, err := client.status()
			if err != nil {
				return nil, nil, err
			}
			data["active_worker_session"] = workerSession.snapshot()
			return nil, anyMap(data), nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "register_worker", Description: "Register a worker session with its supported request providers."},
		func(ctx context.Context, req *mcp.CallToolRequest, input mcpRegisterWorkerInput) (*mcp.CallToolResult, map[string]any, error) {
			workerID := strings.TrimSpace(input.WorkerID)
			if workerID == "" {
				workerID = "knit-worker-" + uuid.NewString()
			}
			label := firstNonEmptyString(strings.TrimSpace(input.Label), "Knit MCP Worker")
			source := firstNonEmptyString(strings.TrimSpace(input.Source), "knit_mcp")
			providers := append([]string(nil), input.SupportedProviders...)
			if len(providers) == 0 {
				status, err := client.status()
				if err != nil {
					return nil, nil, err
				}
				providers = stringSlice(anyMap(status)["available_providers"])
			}
			if _, err := client.postJSON("/api/mcp/worker/connect", map[string]any{
				"worker_id": workerID,
				"label":     label,
				"source":    source,
				"providers": providers,
			}); err != nil {
				return nil, nil, err
			}
			workerSession.setActive(client, workerID, label, source, providers)
			status, err := client.status()
			if err != nil {
				return nil, nil, err
			}
			return nil, map[string]any{
				"worker_id":             workerID,
				"label":                 label,
				"source":                source,
				"supported_providers":   providers,
				"heartbeat_started":     true,
				"worker_connectivity":   anyMap(status)["worker_connectivity"],
				"active_worker_session": workerSession.snapshot(),
			}, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "heartbeat_worker", Description: "Refresh worker-session liveness for the active or specified worker session."},
		func(ctx context.Context, req *mcp.CallToolRequest, input mcpHeartbeatWorkerInput) (*mcp.CallToolResult, map[string]any, error) {
			workerID := workerSession.activeWorkerID(input.WorkerID)
			if workerID == "" {
				return nil, nil, fmt.Errorf("%s: register_worker must run before heartbeat_worker", queueCodeWorkerNotConnected)
			}
			data, err := client.postJSON("/api/mcp/worker/heartbeat", map[string]any{"worker_id": workerID})
			if err != nil {
				return nil, nil, err
			}
			return nil, anyMap(data), nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "unregister_worker", Description: "Disconnect the active or specified worker session."},
		func(ctx context.Context, req *mcp.CallToolRequest, input mcpUnregisterWorkerInput) (*mcp.CallToolResult, map[string]any, error) {
			workerID := workerSession.activeWorkerID(input.WorkerID)
			if workerID == "" {
				return nil, nil, fmt.Errorf("%s: register_worker must run before unregister_worker", queueCodeWorkerNotConnected)
			}
			data, err := client.postJSON("/api/mcp/worker/disconnect", map[string]any{"worker_id": workerID})
			if err != nil {
				return nil, nil, err
			}
			if strings.TrimSpace(input.WorkerID) == "" || strings.TrimSpace(input.WorkerID) == workerID {
				workerSession.stop(nil)
			}
			return nil, anyMap(data), nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "list_requests", Description: "List recent Knit requests and attempts from the daemon."},
		func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, map[string]any, error) {
			data, err := client.state()
			if err != nil {
				return nil, nil, err
			}
			attempts, _ := anyMap(data)["submit_attempts"].([]any)
			active := workerSession.snapshot()
			workerID := strings.TrimSpace(anyString(active["worker_id"]))
			providers := stringSlice(active["supported_providers"])
			for _, item := range attempts {
				attempt, ok := item.(map[string]any)
				if !ok {
					continue
				}
				claimable := false
				if workerID != "" &&
					strings.TrimSpace(anyString(attempt["execution_mode"])) == executionModeWorker &&
					strings.TrimSpace(anyString(attempt["claimed_by_worker_id"])) == "" {
					provider := strings.TrimSpace(anyString(firstNonEmptyAny(attempt["request_provider"], attempt["provider"])))
					for _, supported := range providers {
						if strings.TrimSpace(supported) == provider {
							claimable = true
							break
						}
					}
				}
				attempt["claimable_for_active_worker"] = claimable
			}
			return nil, map[string]any{
				"submit_attempts":       anyMap(data)["submit_attempts"],
				"submit_queue":          anyMap(data)["submit_queue"],
				"active_worker_session": active,
			}, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "wait_for_request", Description: "Wait for a queued Knit request compatible with this worker."},
		func(ctx context.Context, req *mcp.CallToolRequest, input mcpWaitInput) (*mcp.CallToolResult, map[string]any, error) {
			workerID := workerSession.activeWorkerID(input.WorkerID)
			if workerID == "" {
				return nil, nil, fmt.Errorf("%s: register_worker must run before wait_for_request", queueCodeWorkerNotConnected)
			}
			timeout := input.TimeoutSeconds
			if timeout <= 0 {
				timeout = 30
			}
			data, err := client.postJSON("/api/mcp/request/wait?timeout_seconds="+fmt.Sprintf("%d", timeout), map[string]any{"worker_id": workerID})
			if err != nil {
				return nil, nil, err
			}
			return nil, anyMap(data), nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "claim_request", Description: "Claim a queued Knit request for this worker."},
		func(ctx context.Context, req *mcp.CallToolRequest, input mcpClaimInput) (*mcp.CallToolResult, map[string]any, error) {
			workerID := workerSession.activeWorkerID(input.WorkerID)
			if workerID == "" {
				return nil, nil, fmt.Errorf("%s: register_worker must run before claim_request", queueCodeWorkerNotConnected)
			}
			data, err := client.postJSON("/api/mcp/request/claim", map[string]any{"worker_id": workerID, "attempt_id": strings.TrimSpace(input.AttemptID)})
			if err != nil {
				return nil, nil, err
			}
			return nil, anyMap(data), nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "get_request", Description: "Fetch the claimed Knit request payload for implementation."},
		func(ctx context.Context, req *mcp.CallToolRequest, input mcpGetInput) (*mcp.CallToolResult, map[string]any, error) {
			workerID := workerSession.activeWorkerID(input.WorkerID)
			if workerID == "" {
				return nil, nil, fmt.Errorf("%s: register_worker must run before get_request", queueCodeWorkerNotConnected)
			}
			data, err := client.getJSON("/api/mcp/request/get?worker_id=" + workerID + "&attempt_id=" + strings.TrimSpace(input.AttemptID))
			if err != nil {
				return nil, nil, err
			}
			return nil, anyMap(data), nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "complete_request", Description: "Mark the claimed Knit request as completed."},
		func(ctx context.Context, req *mcp.CallToolRequest, input mcpCompleteInput) (*mcp.CallToolResult, map[string]any, error) {
			workerID := workerSession.activeWorkerID(input.WorkerID)
			if workerID == "" {
				return nil, nil, fmt.Errorf("%s: register_worker must run before complete_request", queueCodeWorkerNotConnected)
			}
			data, err := client.postJSON("/api/mcp/request/complete", map[string]any{
				"worker_id":  workerID,
				"attempt_id": strings.TrimSpace(input.AttemptID),
				"summary":    strings.TrimSpace(input.Summary),
				"ref":        strings.TrimSpace(input.Ref),
			})
			if err != nil {
				return nil, nil, err
			}
			return nil, anyMap(data), nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "fail_request", Description: "Mark the claimed Knit request as failed."},
		func(ctx context.Context, req *mcp.CallToolRequest, input mcpFailInput) (*mcp.CallToolResult, map[string]any, error) {
			workerID := workerSession.activeWorkerID(input.WorkerID)
			if workerID == "" {
				return nil, nil, fmt.Errorf("%s: register_worker must run before fail_request", queueCodeWorkerNotConnected)
			}
			data, err := client.postJSON("/api/mcp/request/fail", map[string]any{
				"worker_id":  workerID,
				"attempt_id": strings.TrimSpace(input.AttemptID),
				"code":       strings.TrimSpace(input.Code),
				"error":      strings.TrimSpace(input.Error),
			})
			if err != nil {
				return nil, nil, err
			}
			return nil, anyMap(data), nil
		})

	return server.Run(ctx, &mcp.StdioTransport{})
}

func runMCPTokenCommand() error {
	cfg := app.DefaultConfig()
	managed, err := ensureDaemonForMCP(cfg)
	if err != nil {
		return err
	}
	defer shutdownManagedDaemon(managed)
	client := newDaemonClient(cfg)
	tokenInfo, err := client.managedExtensionToken(true)
	if err != nil {
		return err
	}
	token := strings.TrimSpace(anyString(tokenInfo["token"]))
	if token == "" {
		return fmt.Errorf("managed extension token unavailable")
	}
	_, err = fmt.Fprintln(os.Stdout, token)
	return err
}

func ensureDaemonForMCP(cfg config.Config) (managedDaemon, error) {
	if app.ProbeDaemon(cfg.HTTPListenAddr, 2*time.Second) {
		client := newDaemonClient(cfg)
		health, err := client.health()
		if err != nil {
			return managedDaemon{}, fmt.Errorf("%s: could not read daemon instance health: %w", queueCodeMCPAttachIncompatible, err)
		}
		if ok, reason := compatibleDaemonForMCP(health); !ok {
			return managedDaemon{}, fmt.Errorf("%s: %s", queueCodeMCPAttachIncompatible, reason)
		}
		if _, err := client.status(); err != nil {
			return managedDaemon{}, fmt.Errorf("%s: daemon instance is running but MCP status/auth validation failed: %w", queueCodeMCPAttachIncompatible, err)
		}
		return managedDaemon{owns: false}, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return managedDaemon{}, fmt.Errorf("resolve executable: %w", err)
	}
	instanceID := "daemon-" + uuid.NewString()
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(),
		"KNIT_CONTROL_TOKEN="+cfg.ControlToken,
		"KNIT_DAEMON_INSTANCE_MODE="+daemonInstanceModeManaged,
		"KNIT_DAEMON_INSTANCE_ID="+instanceID,
		"KNIT_DEFAULT_EXECUTION_MODE=mcp_worker",
	)
	if err := cmd.Start(); err != nil {
		return managedDaemon{}, fmt.Errorf("start daemon: %w", err)
	}
	if !app.WaitForDaemon(cfg.HTTPListenAddr, 12*time.Second) {
		_ = cmd.Process.Kill()
		return managedDaemon{}, fmt.Errorf("daemon did not become healthy at http://%s/healthz", cfg.HTTPListenAddr)
	}
	return managedDaemon{cmd: cmd, owns: true}, nil
}

func compatibleDaemonForMCP(health map[string]any) (bool, string) {
	mode := strings.TrimSpace(anyString(health["daemon_instance_mode"]))
	defaultExecutionMode := strings.TrimSpace(anyString(health["default_execution_mode"]))
	attachReady, _ := health["mcp_attach_ready"].(bool)
	switch {
	case mode != daemonInstanceModeManaged:
		return false, "the existing daemon instance is running in normal mode"
	case defaultExecutionMode != executionModeWorker:
		return false, "the existing daemon instance is not using mcp_worker as its default execution mode"
	case !attachReady:
		return false, "the existing daemon instance is not marked as MCP attach-ready"
	default:
		return true, ""
	}
}

func shutdownManagedDaemon(managed managedDaemon) {
	if !managed.owns || managed.cmd == nil || managed.cmd.Process == nil {
		return
	}
	_ = managed.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		_, _ = managed.cmd.Process.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = managed.cmd.Process.Kill()
	}
}

func printMCPStartupBanner(cfg config.Config, ownsDaemon bool, tokenInfo map[string]any) {
	lines := []string{
		"Knit MCP connected.",
		"Daemon URL: " + startupUIURL(cfg),
	}
	if ownsDaemon {
		lines = append(lines, "Daemon mode: managed (started by knit mcp)")
	} else {
		lines = append(lines, "Daemon mode: attached to existing daemon")
	}
	if token := strings.TrimSpace(anyString(tokenInfo["token"])); token != "" {
		lines = append(lines, "Chrome extension token: "+token)
	}
	for _, line := range lines {
		_, _ = fmt.Fprintln(os.Stderr, line)
	}
}

func newDaemonClient(cfg config.Config) *daemonClient {
	return &daemonClient{
		baseURL: startupUIURL(cfg),
		token:   cfg.ControlToken,
		http:    &http.Client{Timeout: 35 * time.Second},
	}
}

func (c *daemonClient) getJSON(path string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Knit-Token", c.token)
	return c.do(req)
}

func (c *daemonClient) health() (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/healthz", nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *daemonClient) postJSON(path string, payload map[string]any) (map[string]any, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Knit-Token", c.token)
	req.Header.Set("X-Knit-Nonce", fmt.Sprintf("%d-%d", time.Now().UTC().UnixNano(), rand.Int()))
	req.Header.Set("X-Knit-Timestamp", fmt.Sprintf("%d", time.Now().UTC().UnixMilli()))
	return c.do(req)
}

func (c *daemonClient) do(req *http.Request) (map[string]any, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil && resp.ContentLength != 0 {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		code := strings.TrimSpace(anyString(data["code"]))
		msg := strings.TrimSpace(anyString(firstNonEmptyAny(data["message"], data["error"])))
		if msg == "" {
			msg = "request failed"
		}
		if code != "" {
			return nil, fmt.Errorf("%s: %s", code, msg)
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return data, nil
}

func (c *daemonClient) status() (map[string]any, error) {
	return c.getJSON("/api/mcp/status")
}

func (c *daemonClient) state() (map[string]any, error) {
	return c.getJSON("/api/state")
}

func (c *daemonClient) managedExtensionToken(reveal bool) (map[string]any, error) {
	path := "/api/extension/managed-token?ensure=1"
	if reveal {
		path += "&reveal=1"
	}
	return c.postJSON(path, map[string]any{})
}

func anyMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok && m != nil {
		return m
	}
	return map[string]any{}
}

func anyString(v any) string {
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
}

func firstNonEmptyAny(values ...any) any {
	for _, value := range values {
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text != "" && text != "<nil>" {
			return value
		}
	}
	return nil
}

func stringSlice(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(fmt.Sprintf("%v", item))
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

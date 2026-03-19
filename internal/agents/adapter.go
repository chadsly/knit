package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"knit/internal/security"
	"knit/internal/session"
)

type Result struct {
	Provider string `json:"provider"`
	RunID    string `json:"run_id"`
	Status   string `json:"status"`
	Ref      string `json:"ref"`
}

type Adapter interface {
	Name() string
	Submit(ctx context.Context, pkg session.CanonicalPackage) (Result, error)
	IsRemote() bool
	Endpoint() string
}

type Registry struct {
	adapters map[string]Adapter
}

type submissionValidator interface {
	ValidateSubmission() error
}

type cliLogFileContextKey struct{}
type deliveryIntentContextKey struct{}

const maxCLIPayloadChars = 700000

func NewRegistry(adapters ...Adapter) *Registry {
	m := make(map[string]Adapter, len(adapters))
	for _, a := range adapters {
		m[a.Name()] = a
	}
	return &Registry{adapters: m}
}

func WithCLILogFile(ctx context.Context, path string) context.Context {
	if strings.TrimSpace(path) == "" {
		return ctx
	}
	return context.WithValue(ctx, cliLogFileContextKey{}, strings.TrimSpace(path))
}

func cliLogFileFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(cliLogFileContextKey{}).(string)
	return strings.TrimSpace(v)
}

func ExecutionLogPathFromContext(ctx context.Context) string {
	return cliLogFileFromContext(ctx)
}

func WithDeliveryIntent(ctx context.Context, intent DeliveryIntent) context.Context {
	intent = NormalizeDeliveryIntent(intent)
	return context.WithValue(ctx, deliveryIntentContextKey{}, intent)
}

func deliveryIntentFromContext(ctx context.Context) DeliveryIntent {
	if ctx == nil {
		return NormalizeDeliveryIntent(DeliveryIntent{})
	}
	intent, _ := ctx.Value(deliveryIntentContextKey{}).(DeliveryIntent)
	return NormalizeDeliveryIntent(intent)
}

func appendExecutionLog(ctx context.Context, format string, args ...any) {
	path := cliLogFileFromContext(ctx)
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
	line := strings.TrimRight(fmt.Sprintf(format, args...), "\n")
	if line == "" {
		return
	}
	_, _ = fmt.Fprintf(f, "[%s] %s\n", time.Now().UTC().Format(time.RFC3339), line)
}

func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.adapters))
	for k := range r.adapters {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func normalizeProviderAlias(provider string) string {
	switch strings.TrimSpace(provider) {
	case "cli":
		return "codex_cli"
	default:
		return strings.TrimSpace(provider)
	}
}

func (r *Registry) adapter(provider string) (Adapter, bool) {
	trimmed := strings.TrimSpace(provider)
	if trimmed == "" {
		return nil, false
	}
	if a, ok := r.adapters[trimmed]; ok {
		return a, true
	}
	if alias := normalizeProviderAlias(trimmed); alias != "" && alias != trimmed {
		if a, ok := r.adapters[alias]; ok {
			return a, true
		}
	}
	return nil, false
}

func (r *Registry) Submit(ctx context.Context, provider string, pkg session.CanonicalPackage) (Result, error) {
	a, ok := r.adapter(provider)
	if !ok {
		return Result{}, fmt.Errorf("adapter not found: %s", provider)
	}
	return a.Submit(ctx, pkg)
}

func (r *Registry) ValidateSubmission(provider string) error {
	a, ok := r.adapter(provider)
	if !ok {
		return fmt.Errorf("adapter not found: %s", provider)
	}
	if validator, ok := a.(submissionValidator); ok {
		if err := validator.ValidateSubmission(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) IsRemote(provider string) bool {
	a, ok := r.adapter(provider)
	if !ok {
		return false
	}
	return a.IsRemote()
}

func (r *Registry) Endpoint(provider string) string {
	a, ok := r.adapter(provider)
	if !ok {
		return ""
	}
	return a.Endpoint()
}

func PreviewProviderPayload(provider string, pkg session.CanonicalPackage, intent DeliveryIntent) (map[string]any, error) {
	return PreviewProviderPayloadWithConfig(
		provider,
		pkg,
		strings.TrimSpace(os.Getenv("CODEX_MODEL")),
		strings.TrimSpace(os.Getenv("KNIT_CLAUDE_API_MODEL")),
		intent,
	)
}

func PreviewProviderPayloadWithConfig(provider string, pkg session.CanonicalPackage, codexModel string, claudeModel string, intent DeliveryIntent) (map[string]any, error) {
	provider = normalizeProviderAlias(provider)
	intent = NormalizeDeliveryIntent(intent)
	switch provider {
	case "codex_api":
		model := fallback(strings.TrimSpace(codexModel), "gpt-5-codex")
		return BuildCodexPayload(pkg, model, intent)
	case "claude_api":
		model := fallback(strings.TrimSpace(claudeModel), "claude-3-7-sonnet-latest")
		return BuildClaudePayload(pkg, model, intent)
	case "codex_cli", "claude_cli", "opencode_cli":
		return BuildCLIPayload(pkg, intent), nil
	default:
		return nil, fmt.Errorf("unknown provider for payload preview: %s", provider)
	}
}

func BuildCodexPayload(pkg session.CanonicalPackage, model string, intent DeliveryIntent) (map[string]any, error) {
	inputBytes, err := json.Marshal(pkg)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical package: %w", err)
	}
	payload := map[string]any{
		"model": model,
		"input": []map[string]any{{
			"role": "user",
			"content": []map[string]any{{
				"type": "input_text",
				"text": RenderInstructionText(intent) + "\n\nCanonical Knit feedback package JSON:\n" + string(inputBytes),
			}},
		}},
	}
	return payload, nil
}

func BuildClaudePayload(pkg session.CanonicalPackage, model string, intent DeliveryIntent) (map[string]any, error) {
	inputBytes, err := json.Marshal(pkg)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical package: %w", err)
	}
	payload := map[string]any{
		"model":      model,
		"max_tokens": 4096,
		"messages": []map[string]any{{
			"role":    "user",
			"content": RenderInstructionText(intent) + "\n\nCanonical Knit feedback package JSON:\n" + string(inputBytes),
		}},
	}
	return payload, nil
}

func BuildCLIPayload(pkg session.CanonicalPackage, intent DeliveryIntent) map[string]any {
	intent = NormalizeDeliveryIntent(intent)
	pkg = trimCLIPackageForPrompt(pkg, maxCLIPayloadChars)
	return map[string]any{
		"schema":              "knit.cli.v2",
		"created":             time.Now().UTC().Format(time.RFC3339Nano),
		"intent_profile":      intent.Profile,
		"custom_instructions": intent.CustomInstructions,
		"instruction_text":    RenderInstructionText(intent),
		"package":             pkg,
	}
}

func trimCLIPackageForPrompt(pkg session.CanonicalPackage, limit int) session.CanonicalPackage {
	if limit <= 0 {
		return pkg
	}
	if n := cliPayloadCharCount(pkg); n > 0 && n <= limit {
		return pkg
	}
	trimmed, ok := cloneCanonicalPackageForCLIPayload(pkg)
	if !ok || len(trimmed.Artifacts) == 0 {
		return pkg
	}
	type candidate struct {
		idx  int
		size int
	}
	candidates := make([]candidate, 0, len(trimmed.Artifacts))
	for i := range trimmed.Artifacts {
		inline := strings.TrimSpace(trimmed.Artifacts[i].InlineDataURL)
		if inline == "" {
			continue
		}
		candidates = append(candidates, candidate{idx: i, size: len(inline)})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].size == candidates[j].size {
			return candidates[i].idx < candidates[j].idx
		}
		return candidates[i].size > candidates[j].size
	})
	for _, item := range candidates {
		artifact := &trimmed.Artifacts[item.idx]
		if strings.TrimSpace(artifact.InlineDataURL) == "" {
			continue
		}
		artifact.InlineDataURL = ""
		if status := strings.TrimSpace(artifact.TransmissionStatus); status == "" || status == "inline" {
			artifact.TransmissionStatus = "omitted_for_cli_size_limit"
		}
		const note = "Inline media omitted from the local CLI payload to fit the prompt size limit."
		switch existing := strings.TrimSpace(artifact.TransmissionNote); {
		case existing == "":
			artifact.TransmissionNote = note
		case !strings.Contains(existing, note):
			artifact.TransmissionNote = existing + " " + note
		}
		if n := cliPayloadCharCount(*trimmed); n > 0 && n <= limit {
			return *trimmed
		}
	}
	return *trimmed
}

func cliPayloadCharCount(pkg session.CanonicalPackage) int {
	intent := NormalizeDeliveryIntent(DeliveryIntent{})
	payload := map[string]any{
		"schema":              "knit.cli.v2",
		"created":             time.Now().UTC().Format(time.RFC3339Nano),
		"intent_profile":      intent.Profile,
		"custom_instructions": intent.CustomInstructions,
		"instruction_text":    RenderInstructionText(intent),
		"package":             pkg,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return 0
	}
	return len(b)
}

func cloneCanonicalPackageForCLIPayload(pkg session.CanonicalPackage) (*session.CanonicalPackage, bool) {
	b, err := json.Marshal(pkg)
	if err != nil {
		return nil, false
	}
	var out session.CanonicalPackage
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, false
	}
	return &out, true
}

type CodexAPIAdapter struct {
	APIKey      string
	Model       string
	BaseURL     string
	Timeout     time.Duration
	OrgID       string
	ProjectID   string
	HTTPClient  *http.Client
	AppName     string
	AppVersion  string
	APISubroute string
}

type ClaudeAPIAdapter struct {
	APIKey      string
	Model       string
	BaseURL     string
	Timeout     time.Duration
	HTTPClient  *http.Client
	AppName     string
	AppVersion  string
	APISubroute string
}

func NewCodexAPIAdapter(apiKey, model, baseURL string, timeout time.Duration, orgID, projectID string) CodexAPIAdapter {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	httpClient := &http.Client{Timeout: timeout}
	if client, err := security.NewHTTPClientFromEnv(timeout); err == nil && client != nil {
		httpClient = client
	}
	return CodexAPIAdapter{
		APIKey:      strings.TrimSpace(apiKey),
		Model:       fallback(strings.TrimSpace(model), "gpt-5-codex"),
		BaseURL:     strings.TrimRight(baseURL, "/"),
		Timeout:     timeout,
		OrgID:       strings.TrimSpace(orgID),
		ProjectID:   strings.TrimSpace(projectID),
		HTTPClient:  httpClient,
		AppName:     "knit",
		AppVersion:  "v1-scaffold",
		APISubroute: "/v1/responses",
	}
}

func NewCodexAPIAdapterFromEnv() CodexAPIAdapter {
	timeout := 60 * time.Second
	if v := os.Getenv("CODEX_TIMEOUT_SECONDS"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			timeout = time.Duration(sec) * time.Second
		}
	}
	baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return NewCodexAPIAdapter(
		strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		strings.TrimSpace(os.Getenv("CODEX_MODEL")),
		baseURL,
		timeout,
		firstNonEmpty(os.Getenv("OPENAI_ORG_ID"), os.Getenv("OPENAI_ORGANIZATION")),
		os.Getenv("OPENAI_PROJECT_ID"),
	)
}

func NewClaudeAPIAdapter(apiKey, model, baseURL string, timeout time.Duration) ClaudeAPIAdapter {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	httpClient := &http.Client{Timeout: timeout}
	if client, err := security.NewHTTPClientFromEnv(timeout); err == nil && client != nil {
		httpClient = client
	}
	return ClaudeAPIAdapter{
		APIKey:      strings.TrimSpace(apiKey),
		Model:       fallback(strings.TrimSpace(model), "claude-3-7-sonnet-latest"),
		BaseURL:     strings.TrimRight(baseURL, "/"),
		Timeout:     timeout,
		HTTPClient:  httpClient,
		AppName:     "knit",
		AppVersion:  "v1-scaffold",
		APISubroute: "/v1/messages",
	}
}

func NewClaudeAPIAdapterFromEnv() ClaudeAPIAdapter {
	timeout := 60 * time.Second
	if v := os.Getenv("KNIT_CLAUDE_API_TIMEOUT_SECONDS"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			timeout = time.Duration(sec) * time.Second
		}
	}
	baseURL := strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return NewClaudeAPIAdapter(
		strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		strings.TrimSpace(os.Getenv("KNIT_CLAUDE_API_MODEL")),
		baseURL,
		timeout,
	)
}

func (a CodexAPIAdapter) Name() string { return "codex_api" }
func (a CodexAPIAdapter) IsRemote() bool {
	return true
}
func (a CodexAPIAdapter) Endpoint() string {
	return a.BaseURL
}

func (a CodexAPIAdapter) Submit(ctx context.Context, pkg session.CanonicalPackage) (Result, error) {
	if a.APIKey == "" {
		return Result{}, fmt.Errorf("OPENAI_API_KEY is required for codex_api adapter")
	}
	intent := deliveryIntentFromContext(ctx)
	body, err := BuildCodexPayload(pkg, a.Model, intent)
	if err != nil {
		return Result{}, err
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Result{}, fmt.Errorf("marshal request body: %w", err)
	}

	url := a.BaseURL + a.APISubroute
	appendExecutionLog(ctx, "Submitting to codex_api endpoint %s with model %s", url, a.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		appendExecutionLog(ctx, "Failed to build codex_api request: %v", err)
		return Result{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", a.AppName+"/"+a.AppVersion)
	if a.OrgID != "" {
		req.Header.Set("OpenAI-Organization", a.OrgID)
	}
	if a.ProjectID != "" {
		req.Header.Set("OpenAI-Project", a.ProjectID)
	}

	client := a.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: a.Timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		appendExecutionLog(ctx, "codex_api request failed: %v", err)
		return Result{}, fmt.Errorf("submit to codex api: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		appendExecutionLog(ctx, "codex_api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		return Result{}, fmt.Errorf("codex api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var decoded map[string]any
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		appendExecutionLog(ctx, "Failed to decode codex_api response: %v", err)
		return Result{}, fmt.Errorf("decode codex api response: %w", err)
	}
	runID, _ := decoded["id"].(string)
	if runID == "" {
		runID = "unknown"
	}
	status, _ := decoded["status"].(string)
	if status == "" {
		status = "accepted"
	}
	appendExecutionLog(ctx, "codex_api accepted request id=%s status=%s", runID, status)
	return Result{
		Provider: a.Name(),
		RunID:    runID,
		Status:   status,
		Ref:      fmt.Sprintf("response:%s", runID),
	}, nil
}

func (a ClaudeAPIAdapter) Name() string { return "claude_api" }
func (a ClaudeAPIAdapter) IsRemote() bool {
	return true
}
func (a ClaudeAPIAdapter) Endpoint() string {
	return a.BaseURL
}

func (a ClaudeAPIAdapter) Submit(ctx context.Context, pkg session.CanonicalPackage) (Result, error) {
	if a.APIKey == "" {
		return Result{}, fmt.Errorf("ANTHROPIC_API_KEY is required for claude_api adapter")
	}
	intent := deliveryIntentFromContext(ctx)
	body, err := BuildClaudePayload(pkg, a.Model, intent)
	if err != nil {
		return Result{}, err
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Result{}, fmt.Errorf("marshal request body: %w", err)
	}

	url := a.BaseURL + a.APISubroute
	appendExecutionLog(ctx, "Submitting to claude_api endpoint %s with model %s", url, a.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		appendExecutionLog(ctx, "Failed to build claude_api request: %v", err)
		return Result{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", a.AppName+"/"+a.AppVersion)

	client := a.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: a.Timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		appendExecutionLog(ctx, "claude_api request failed: %v", err)
		return Result{}, fmt.Errorf("submit to claude api: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		appendExecutionLog(ctx, "claude_api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		return Result{}, fmt.Errorf("claude api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var decoded map[string]any
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		appendExecutionLog(ctx, "Failed to decode claude_api response: %v", err)
		return Result{}, fmt.Errorf("decode claude api response: %w", err)
	}
	runID, _ := decoded["id"].(string)
	if runID == "" {
		runID = "unknown"
	}
	status := "completed"
	if stopReason, _ := decoded["stop_reason"].(string); strings.TrimSpace(stopReason) != "" {
		status = stopReason
	}
	appendExecutionLog(ctx, "claude_api accepted request id=%s status=%s", runID, status)
	return Result{
		Provider: a.Name(),
		RunID:    runID,
		Status:   status,
		Ref:      fmt.Sprintf("message:%s", runID),
	}, nil
}

type CLIAdapter struct {
	NameValue  string
	CommandEnv string
	TimeoutEnv string
	Command    string
	Timeout    time.Duration
}

const defaultCLITimeout = 10 * time.Minute
const defaultCLIExecutionLogMaxBytes int64 = 8 << 20
const cliCapturedOutputMaxBytes = 64 << 10

func NewCLIAdapterFromEnv() CLIAdapter {
	return newNamedCLIAdapter("codex_cli", "KNIT_CLI_ADAPTER_CMD", "KNIT_CLI_TIMEOUT_SECONDS")
}

func NewCLIAdapter(name, command string, timeout time.Duration) CLIAdapter {
	return CLIAdapter{
		NameValue: strings.TrimSpace(name),
		Command:   strings.TrimSpace(command),
		Timeout:   timeout,
	}
}

func NewClaudeCLIAdapterFromEnv() CLIAdapter {
	return newNamedCLIAdapter("claude_cli", "KNIT_CLAUDE_CLI_ADAPTER_CMD", "KNIT_CLAUDE_CLI_TIMEOUT_SECONDS")
}

func NewOpenCodeCLIAdapterFromEnv() CLIAdapter {
	return newNamedCLIAdapter("opencode_cli", "KNIT_OPENCODE_CLI_ADAPTER_CMD", "KNIT_OPENCODE_CLI_TIMEOUT_SECONDS")
}

func newNamedCLIAdapter(name, commandEnv, timeoutEnv string) CLIAdapter {
	timeout := defaultCLITimeout
	if v := strings.TrimSpace(os.Getenv(timeoutEnv)); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			timeout = time.Duration(sec) * time.Second
		}
	}
	return CLIAdapter{
		NameValue:  strings.TrimSpace(name),
		CommandEnv: strings.TrimSpace(commandEnv),
		TimeoutEnv: strings.TrimSpace(timeoutEnv),
		Command:    strings.TrimSpace(os.Getenv(commandEnv)),
		Timeout:    timeout,
	}
}

func (a CLIAdapter) Name() string {
	if strings.TrimSpace(a.NameValue) == "" {
		return "codex_cli"
	}
	return a.NameValue
}
func (a CLIAdapter) IsRemote() bool {
	return false
}
func (a CLIAdapter) Endpoint() string {
	return "local-process"
}

func (a CLIAdapter) ValidateSubmission() error {
	commandEnv := strings.TrimSpace(a.CommandEnv)
	if commandEnv == "" {
		commandEnv = "KNIT_CLI_ADAPTER_CMD"
	}
	command := strings.TrimSpace(a.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv(commandEnv))
	}
	if command == "" {
		return fmt.Errorf("%s is not configured: set %s before submitting", a.Name(), commandEnv)
	}
	return nil
}

func (a CLIAdapter) Submit(ctx context.Context, pkg session.CanonicalPackage) (Result, error) {
	commandEnv := strings.TrimSpace(a.CommandEnv)
	if commandEnv == "" {
		commandEnv = "KNIT_CLI_ADAPTER_CMD"
	}
	timeoutEnv := strings.TrimSpace(a.TimeoutEnv)
	if timeoutEnv == "" {
		timeoutEnv = "KNIT_CLI_TIMEOUT_SECONDS"
	}
	command := strings.TrimSpace(a.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv(commandEnv))
	}
	timeout := a.Timeout
	if timeout <= 0 {
		timeout = defaultCLITimeout
	}
	if timeout == a.Timeout {
		if v := strings.TrimSpace(os.Getenv(timeoutEnv)); v != "" {
			if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
				timeout = time.Duration(sec) * time.Second
			}
		}
	}

	if err := a.ValidateSubmission(); err != nil {
		appendExecutionLog(ctx, "%s command not configured: %v", a.Name(), err)
		return Result{}, err
	}
	payload := BuildCLIPayload(pkg, deliveryIntentFromContext(ctx))
	body, err := json.Marshal(payload)
	if err != nil {
		return Result{}, fmt.Errorf("marshal cli payload: %w", err)
	}

	runCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := shellCommand(runCtx, command)
	cmd.Env = os.Environ()
	var logFile *os.File
	var logWriter io.Writer
	if logPath := cliLogFileFromContext(runCtx); logPath != "" {
		cmd.Env = append(cmd.Env, "KNIT_CLI_LOG_FILE="+logPath)
		appendExecutionLog(runCtx, "Starting %s command", a.Name())
		if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err == nil {
			if f, openErr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); openErr == nil {
				logFile = f
				defer logFile.Close()
				logWriter = newExecutionLogWriter(f, cliExecutionLogMaxBytes())
			}
		}
	}
	cmd.Stdin = bytes.NewReader(body)
	var stdout tailBuffer
	var stderr tailBuffer
	stdout.maxBytes = cliCapturedOutputMaxBytes
	stderr.maxBytes = cliCapturedOutputMaxBytes
	if logWriter != nil {
		cmd.Stdout = io.MultiWriter(&stdout, logWriter)
		cmd.Stderr = io.MultiWriter(&stderr, logWriter)
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}
	if err := cmd.Run(); err != nil {
		if runErr := runCtx.Err(); runErr != nil {
			if errors.Is(runErr, context.DeadlineExceeded) {
				return Result{}, fmt.Errorf("cli adapter command timed out after %s", timeout)
			}
			if errors.Is(runErr, context.Canceled) {
				return Result{}, fmt.Errorf("cli adapter command canceled before completion")
			}
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		if strings.Contains(strings.ToLower(msg), "signal: killed") {
			msg = msg + " (process terminated; if this is a long-running codex run, increase KNIT_CLI_TIMEOUT_SECONDS)"
		}
		appendExecutionLog(runCtx, "%s command failed: %s", a.Name(), msg)
		return Result{}, fmt.Errorf("cli adapter command failed: %s", msg)
	}

	output := strings.TrimSpace(stdout.String())
	result := Result{
		Provider: a.Name(),
		RunID:    fmt.Sprintf("cli-%d", time.Now().UTC().UnixNano()),
		Status:   "accepted",
		Ref:      fmt.Sprintf("session:%s", pkg.SessionID),
	}
	if output != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(output), &parsed); err == nil {
			if v, ok := parsed["run_id"].(string); ok && v != "" {
				result.RunID = v
			}
			if v, ok := parsed["status"].(string); ok && v != "" {
				result.Status = v
			}
			if v, ok := parsed["ref"].(string); ok && v != "" {
				result.Ref = v
			}
		}
	}
	appendExecutionLog(runCtx, "%s command completed: run_id=%s status=%s ref=%s", a.Name(), result.RunID, result.Status, result.Ref)
	return result, nil
}

func cliExecutionLogMaxBytes() int64 {
	raw := strings.TrimSpace(os.Getenv("KNIT_CLI_LOG_MAX_BYTES"))
	if raw == "" {
		return defaultCLIExecutionLogMaxBytes
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return defaultCLIExecutionLogMaxBytes
	}
	return n
}

type executionLogWriter struct {
	mu        sync.Mutex
	file      *os.File
	remaining int64
	limit     int64
	truncated bool
}

func newExecutionLogWriter(file *os.File, limit int64) *executionLogWriter {
	used := int64(0)
	if file != nil {
		if info, err := file.Stat(); err == nil {
			used = info.Size()
		}
	}
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}
	return &executionLogWriter{
		file:      file,
		remaining: remaining,
		limit:     limit,
	}
}

func (w *executionLogWriter) Write(p []byte) (int, error) {
	if w == nil || w.file == nil || len(p) == 0 {
		return len(p), nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	overflowed := false
	if w.remaining > 0 {
		n := len(p)
		if int64(n) > w.remaining {
			n = int(w.remaining)
			overflowed = true
		}
		if n > 0 {
			if _, err := w.file.Write(p[:n]); err != nil {
				return 0, err
			}
			w.remaining -= int64(n)
		}
	} else {
		overflowed = true
	}
	if overflowed && !w.truncated {
		w.truncated = true
		_, _ = fmt.Fprintf(
			w.file,
			"\n[%s] execution log truncated after %d bytes; additional adapter output omitted\n",
			time.Now().UTC().Format(time.RFC3339),
			w.limit,
		)
	}
	return len(p), nil
}

type tailBuffer struct {
	maxBytes int
	buf      []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	if b == nil || len(p) == 0 {
		return len(p), nil
	}
	if b.maxBytes <= 0 {
		return len(p), nil
	}
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.maxBytes {
		overflow := len(b.buf) - b.maxBytes
		copy(b.buf, b.buf[overflow:])
		b.buf = b.buf[:b.maxBytes]
	}
	return len(p), nil
}

func (b *tailBuffer) String() string {
	if b == nil {
		return ""
	}
	return string(b.buf)
}

func shellCommand(ctx context.Context, raw string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", raw)
	}
	return exec.CommandContext(ctx, "sh", "-lc", raw)
}

func fallback(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

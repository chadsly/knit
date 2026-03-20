package agents

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"knit/internal/session"
)

func TestRegistryRemoteAndEndpointMetadata(t *testing.T) {
	reg := NewRegistry(NewCodexAPIAdapterFromEnv(), NewClaudeAPIAdapterFromEnv(), NewCLIAdapterFromEnv(), NewClaudeCLIAdapterFromEnv(), NewOpenCodeCLIAdapterFromEnv())
	if !reg.IsRemote("codex_api") {
		t.Fatalf("expected codex_api to be remote")
	}
	if !reg.IsRemote("claude_api") {
		t.Fatalf("expected claude_api to be remote")
	}
	if reg.IsRemote("cli") {
		t.Fatalf("expected cli adapter to be local")
	}
	if reg.Endpoint("cli") == "" {
		t.Fatalf("expected cli endpoint metadata")
	}
	if reg.Endpoint("claude_cli") == "" || reg.Endpoint("opencode_cli") == "" || reg.Endpoint("claude_api") == "" {
		t.Fatalf("expected provider-compatible cli endpoint metadata")
	}
}

func TestPreviewProviderPayload(t *testing.T) {
	pkg := session.CanonicalPackage{
		SessionID: "sess-1",
		ChangeRequests: []session.ChangeReq{
			{EventID: "evt-1", Summary: "Increase button size", Category: "layout", Priority: "medium"},
		},
	}
	if _, err := PreviewProviderPayload("codex_api", pkg, DeliveryIntent{}); err != nil {
		t.Fatalf("preview codex payload: %v", err)
	}
	if _, err := PreviewProviderPayload("claude_api", pkg, DeliveryIntent{}); err != nil {
		t.Fatalf("preview claude payload: %v", err)
	}
	if _, err := PreviewProviderPayload("cli", pkg, DeliveryIntent{}); err != nil {
		t.Fatalf("preview cli payload: %v", err)
	}
	if _, err := PreviewProviderPayload("claude_cli", pkg, DeliveryIntent{}); err != nil {
		t.Fatalf("preview claude_cli payload: %v", err)
	}
	if _, err := PreviewProviderPayload("opencode_cli", pkg, DeliveryIntent{}); err != nil {
		t.Fatalf("preview opencode_cli payload: %v", err)
	}
}

func TestCLIAdapterUsesRuntimeCommandEnv(t *testing.T) {
	t.Setenv("KNIT_CLI_ADAPTER_CMD", `echo '{"run_id":"cli-runtime","status":"accepted","ref":"runtime"}'`)
	adapter := CLIAdapter{Command: "", Timeout: 0}
	res, err := adapter.Submit(context.Background(), session.CanonicalPackage{SessionID: "sess-1"})
	if err != nil {
		t.Fatalf("submit with runtime command env: %v", err)
	}
	if res.RunID != "cli-runtime" || res.Ref != "runtime" {
		t.Fatalf("unexpected runtime cli result: %#v", res)
	}
}

func TestCLIAdapterRequiresConfiguredCommand(t *testing.T) {
	t.Setenv("KNIT_CLI_ADAPTER_CMD", "")
	adapter := CLIAdapter{Command: "", Timeout: 0}
	if err := adapter.ValidateSubmission(); err == nil {
		t.Fatal("expected ValidateSubmission to fail without configured command")
	}
	_, err := adapter.Submit(context.Background(), session.CanonicalPackage{SessionID: "sess-1"})
	if err == nil {
		t.Fatal("expected Submit to fail without configured command")
	}
	if !strings.Contains(err.Error(), "KNIT_CLI_ADAPTER_CMD") {
		t.Fatalf("expected missing command env in error, got %v", err)
	}
}

func TestCLIAdapterTimeoutReturnsClearError(t *testing.T) {
	command := "sleep 2"
	if runtime.GOOS == "windows" {
		command = `powershell -Command "Start-Sleep -Seconds 2"`
	}
	adapter := CLIAdapter{Command: command, Timeout: 1 * time.Second}
	_, err := adapter.Submit(context.Background(), session.CanonicalPackage{SessionID: "sess-timeout"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestCLIAdapterCanceledContextReturnsClearError(t *testing.T) {
	command := "sleep 2"
	if runtime.GOOS == "windows" {
		command = `powershell -Command "Start-Sleep -Seconds 2"`
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	adapter := CLIAdapter{Command: command, Timeout: 5 * time.Second}
	_, err := adapter.Submit(ctx, session.CanonicalPackage{SessionID: "sess-cancel"})
	if err == nil {
		t.Fatal("expected canceled error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "canceled") {
		t.Fatalf("expected canceled error, got: %v", err)
	}
}

func TestCLIAdapterPassesConfiguredLogFileToCommand(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "knit-codex-adapter-test.log")
	command := `echo '{"run_id":"cli-log","status":"accepted","ref":"log"}'`
	if runtime.GOOS == "windows" {
		command = `powershell -Command "$p=$env:KNIT_CLI_LOG_FILE; Add-Content -Path $p -Value 'stream line'; Write-Output '{\"run_id\":\"cli-log\",\"status\":\"accepted\",\"ref\":\"log\"}'"`
	} else {
		command = `sh -lc 'printf "stream line\n" >> "$KNIT_CLI_LOG_FILE"; echo "{\"run_id\":\"cli-log\",\"status\":\"accepted\",\"ref\":\"log\"}"'`
	}
	adapter := CLIAdapter{Command: command, Timeout: 15 * time.Second}
	ctx := WithCLILogFile(context.Background(), logPath)
	if _, err := adapter.Submit(ctx, session.CanonicalPackage{SessionID: "sess-log"}); err != nil {
		t.Fatalf("submit with log path context: %v", err)
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log path file: %v", err)
	}
	if !strings.Contains(string(b), "stream line") {
		t.Fatalf("expected streamed content in log file, got %q", string(b))
	}
}

func TestCLIAdapterCapsExecutionLogAndKeepsFinalJSON(t *testing.T) {
	t.Setenv("KNIT_CLI_LOG_MAX_BYTES", "1024")
	logPath := filepath.Join(t.TempDir(), "knit-codex-capped.log")
	command := `sh -lc 'printf "%2048s\n" "" | tr " " x >&2; echo "{\"run_id\":\"cli-trunc\",\"status\":\"accepted\",\"ref\":\"log\"}"'`
	if runtime.GOOS == "windows" {
		command = `powershell -Command "$chunk = 'x' * 2048; [Console]::Error.WriteLine($chunk); Write-Output '{\"run_id\":\"cli-trunc\",\"status\":\"accepted\",\"ref\":\"log\"}'"`
	}
	adapter := CLIAdapter{Command: command, Timeout: 15 * time.Second}
	ctx := WithCLILogFile(context.Background(), logPath)
	res, err := adapter.Submit(ctx, session.CanonicalPackage{SessionID: "sess-log-cap"})
	if err != nil {
		t.Fatalf("submit with capped log path: %v", err)
	}
	if res.RunID != "cli-trunc" || res.Ref != "log" {
		t.Fatalf("unexpected result after capped log run: %#v", res)
	}
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("stat log path: %v", err)
	}
	if info.Size() > 2048 {
		t.Fatalf("expected capped log file size, got %d bytes", info.Size())
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read capped log file: %v", err)
	}
	logText := string(b)
	if !strings.Contains(logText, "execution log truncated after 1024 bytes") {
		t.Fatalf("expected truncation notice in execution log, got %q", logText)
	}
	if !strings.Contains(logText, "Starting codex_cli command") {
		t.Fatalf("expected adapter lifecycle entry in execution log, got %q", logText)
	}
}

func TestNamedCLIAdaptersUseDedicatedCommandEnv(t *testing.T) {
	t.Setenv("KNIT_CLAUDE_CLI_ADAPTER_CMD", `echo '{"run_id":"claude-run","status":"accepted","ref":"claude-ref"}'`)
	t.Setenv("KNIT_OPENCODE_CLI_ADAPTER_CMD", `echo '{"run_id":"opencode-run","status":"accepted","ref":"opencode-ref"}'`)

	claude := NewClaudeCLIAdapterFromEnv()
	opencode := NewOpenCodeCLIAdapterFromEnv()

	claudeRes, err := claude.Submit(context.Background(), session.CanonicalPackage{SessionID: "sess-claude"})
	if err != nil {
		t.Fatalf("claude adapter submit failed: %v", err)
	}
	if claudeRes.Provider != "claude_cli" || claudeRes.RunID != "claude-run" {
		t.Fatalf("unexpected claude result: %#v", claudeRes)
	}

	opencodeRes, err := opencode.Submit(context.Background(), session.CanonicalPackage{SessionID: "sess-opencode"})
	if err != nil {
		t.Fatalf("opencode adapter submit failed: %v", err)
	}
	if opencodeRes.Provider != "opencode_cli" || opencodeRes.RunID != "opencode-run" {
		t.Fatalf("unexpected opencode result: %#v", opencodeRes)
	}
}

func TestCodexAPIAdapterWritesExecutionLog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		_, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp-123",
			"status": "completed",
		})
	}))
	defer server.Close()

	logPath := filepath.Join(t.TempDir(), "knit-codex-api.log")
	adapter := CodexAPIAdapter{
		APIKey:      "token",
		Model:       "gpt-5-codex",
		BaseURL:     server.URL,
		Timeout:     15 * time.Second,
		HTTPClient:  server.Client(),
		AppName:     "test",
		AppVersion:  "v1",
		APISubroute: "/v1/responses",
	}
	ctx := WithCLILogFile(context.Background(), logPath)
	res, err := adapter.Submit(ctx, session.CanonicalPackage{SessionID: "sess-api"})
	if err != nil {
		t.Fatalf("codex api submit failed: %v", err)
	}
	if res.RunID != "resp-123" {
		t.Fatalf("expected run id resp-123, got %#v", res)
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	logText := string(b)
	if !strings.Contains(logText, "Submitting to codex_api endpoint") || !strings.Contains(logText, "codex_api accepted request id=resp-123 status=completed") {
		t.Fatalf("expected codex api execution log content, got %q", logText)
	}
}

func TestCodexAPIAdapterSubmitUsesProxyOverride(t *testing.T) {
	var seenRequest bool
	var seenAuth string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenRequest = true
		seenAuth = r.Header.Get("Authorization")
		if got := r.URL.String(); got != "http://api.proxy-target.test/v1/responses" {
			t.Fatalf("expected proxied absolute URL, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp-proxy",
			"status": "completed",
		})
	}))
	defer proxy.Close()

	t.Setenv("KNIT_HTTP_PROXY", proxy.URL)
	t.Setenv("KNIT_NO_PROXY", "")

	adapter := NewCodexAPIAdapter("token", "gpt-5-codex", "http://api.proxy-target.test", 5*time.Second, "", "")
	res, err := adapter.Submit(context.Background(), session.CanonicalPackage{SessionID: "sess-proxy"})
	if err != nil {
		t.Fatalf("submit via proxy failed: %v", err)
	}
	if !seenRequest {
		t.Fatalf("expected proxy to receive request")
	}
	if seenAuth != "Bearer token" {
		t.Fatalf("expected auth header through proxy, got %q", seenAuth)
	}
	if res.RunID != "resp-proxy" || res.Status != "completed" {
		t.Fatalf("unexpected proxy submit result: %#v", res)
	}
}

func TestCodexAPIAdapterSubmitSupportsCustomCATrustAndPinnedCert(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp-tls",
			"status": "completed",
		})
	}))
	defer server.Close()

	leaf := mustLeafCertFromTLSServer(t, server)
	sum := sha256.Sum256(leaf.Raw)
	t.Setenv("KNIT_HTTP_PROXY", "")
	t.Setenv("KNIT_NO_PROXY", "")
	t.Setenv("KNIT_TLS_CA_FILE", writeCertPEMForAdapterTest(t, leaf))
	t.Setenv("KNIT_TLS_PINNED_CERT_SHA256", hex.EncodeToString(sum[:]))

	adapter := NewCodexAPIAdapter("token", "gpt-5-codex", server.URL, 5*time.Second, "", "")
	res, err := adapter.Submit(context.Background(), session.CanonicalPackage{SessionID: "sess-tls"})
	if err != nil {
		t.Fatalf("submit with custom CA and pin failed: %v", err)
	}
	if res.RunID != "resp-tls" {
		t.Fatalf("expected TLS-backed submit result, got %#v", res)
	}
}

func TestCodexAPIAdapterSubmitRejectsPinnedCertMismatch(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"resp-mismatch","status":"completed"}`))
	}))
	defer server.Close()

	leaf := mustLeafCertFromTLSServer(t, server)
	t.Setenv("KNIT_HTTP_PROXY", "")
	t.Setenv("KNIT_NO_PROXY", "")
	t.Setenv("KNIT_TLS_CA_FILE", writeCertPEMForAdapterTest(t, leaf))
	t.Setenv("KNIT_TLS_PINNED_CERT_SHA256", strings.Repeat("aa", sha256.Size))

	adapter := NewCodexAPIAdapter("token", "gpt-5-codex", server.URL, 5*time.Second, "", "")
	_, err := adapter.Submit(context.Background(), session.CanonicalPackage{SessionID: "sess-mismatch"})
	if err == nil {
		t.Fatal("expected pin mismatch to fail submit")
	}
	if !strings.Contains(err.Error(), "tls pin mismatch") {
		t.Fatalf("expected tls pin mismatch error, got %v", err)
	}
}

func TestRegistrySubmitResolvesCLIProviderAlias(t *testing.T) {
	reg := NewRegistry(stubAdapter{name: "codex_cli", result: Result{Provider: "codex_cli", RunID: "alias-run"}})
	res, err := reg.Submit(context.Background(), "cli", session.CanonicalPackage{SessionID: "sess-alias"})
	if err != nil {
		t.Fatalf("submit with cli alias: %v", err)
	}
	if res.Provider != "codex_cli" || res.RunID != "alias-run" {
		t.Fatalf("unexpected alias submit result: %#v", res)
	}
}

func TestPreviewProviderPayloadWithConfigProducesProviderSpecificContracts(t *testing.T) {
	pkg := session.CanonicalPackage{
		SessionID: "sess-payload",
		Summary:   "Tighten the settings layout and improve copy clarity.",
		ChangeRequests: []session.ChangeReq{{
			EventID:         "evt-1",
			Summary:         "Align the save action with the content column",
			Category:        "layout",
			Priority:        "medium",
			VisualTargetRef: "button#save",
		}},
	}

	intent := DeliveryIntent{Profile: IntentCreateJira, CustomInstructions: "Focus on rollout risk."}
	codexPayload, err := PreviewProviderPayloadWithConfig("codex_api", pkg, "gpt-5-codex-mini", "", intent)
	if err != nil {
		t.Fatalf("preview codex payload: %v", err)
	}
	if codexPayload["model"] != "gpt-5-codex-mini" {
		t.Fatalf("expected codex model override, got %#v", codexPayload["model"])
	}
	input, _ := codexPayload["input"].([]map[string]any)
	if len(input) != 1 {
		t.Fatalf("expected codex input envelope, got %#v", codexPayload["input"])
	}
	content, _ := input[0]["content"].([]map[string]any)
	if len(content) != 1 {
		t.Fatalf("expected codex content payload, got %#v", input[0]["content"])
	}
	text, _ := content[0]["text"].(string)
	if !strings.Contains(text, "\"session_id\":\"sess-payload\"") || !strings.Contains(text, "\"visual_target_ref\":\"button#save\"") {
		t.Fatalf("expected canonical package to be embedded in codex payload, got %q", text)
	}
	if !strings.Contains(text, "Jira-ready implementation tickets") || !strings.Contains(text, "Focus on rollout risk.") {
		t.Fatalf("expected codex payload to include intent-specific instructions, got %q", text)
	}

	claudePayload, err := PreviewProviderPayloadWithConfig("claude_api", pkg, "", "claude-test-model", intent)
	if err != nil {
		t.Fatalf("preview claude payload: %v", err)
	}
	if claudePayload["model"] != "claude-test-model" {
		t.Fatalf("expected claude model override, got %#v", claudePayload["model"])
	}
	if claudePayload["max_tokens"] != 4096 {
		t.Fatalf("expected claude max_tokens, got %#v", claudePayload["max_tokens"])
	}
	messages, _ := claudePayload["messages"].([]map[string]any)
	if len(messages) != 1 {
		t.Fatalf("expected claude message envelope, got %#v", claudePayload["messages"])
	}
	messageText, _ := messages[0]["content"].(string)
	if !strings.Contains(messageText, "\"session_id\":\"sess-payload\"") || !strings.Contains(messageText, "\"visual_target_ref\":\"button#save\"") {
		t.Fatalf("expected canonical package to be embedded in claude payload, got %q", messageText)
	}
	if !strings.Contains(messageText, "Jira-ready implementation tickets") || !strings.Contains(messageText, "Focus on rollout risk.") {
		t.Fatalf("expected claude payload to include intent-specific instructions, got %q", messageText)
	}

	for _, provider := range []string{"cli", "claude_cli", "opencode_cli"} {
		payload, err := PreviewProviderPayloadWithConfig(provider, pkg, "", "", intent)
		if err != nil {
			t.Fatalf("preview %s payload: %v", provider, err)
		}
		if payload["schema"] != "knit.cli.v2" {
			t.Fatalf("expected CLI-compatible schema for %s, got %#v", provider, payload["schema"])
		}
		encodedPkg, _ := payload["package"].(session.CanonicalPackage)
		if encodedPkg.SessionID != "sess-payload" || len(encodedPkg.ChangeRequests) != 1 {
			t.Fatalf("expected canonical package passthrough for %s, got %#v", provider, payload["package"])
		}
		if payload["intent_profile"] != IntentCreateJira {
			t.Fatalf("expected intent profile for %s, got %#v", provider, payload["intent_profile"])
		}
		if !strings.Contains(payload["instruction_text"].(string), "Jira-ready implementation tickets") {
			t.Fatalf("expected CLI instruction text for %s, got %#v", provider, payload["instruction_text"])
		}
	}
}

func TestRenderInstructionTextHonorsIntentProfiles(t *testing.T) {
	if text := RenderInstructionText(DeliveryIntent{}); !strings.Contains(text, "Implement the requested software changes") {
		t.Fatalf("expected default implement intent, got %q", text)
	}
	if text := RenderInstructionText(DeliveryIntent{Profile: "draft_plan"}); !strings.Contains(text, "Implement the requested software changes") {
		t.Fatalf("expected legacy draft-plan profile to fall back to implement guidance, got %q", text)
	}
	if text := RenderInstructionText(DeliveryIntent{Profile: IntentCreateJira, CustomInstructions: "Group by team."}); !strings.Contains(text, "Jira-ready implementation tickets") || !strings.Contains(text, "Group by team.") {
		t.Fatalf("expected jira guidance plus custom instructions, got %q", text)
	}
}

func TestClaudeAPIAdapterWritesExecutionLog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if got := r.Header.Get("x-api-key"); got != "anthropic-token" {
			t.Fatalf("expected anthropic key header, got %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Fatalf("expected anthropic-version header, got %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["model"] != "claude-test-model" {
			t.Fatalf("expected model in request body, got %#v", body["model"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "msg-123",
			"type":        "message",
			"stop_reason": "end_turn",
		})
	}))
	defer server.Close()

	logPath := filepath.Join(t.TempDir(), "knit-claude-api.log")
	adapter := ClaudeAPIAdapter{
		APIKey:      "anthropic-token",
		Model:       "claude-test-model",
		BaseURL:     server.URL,
		Timeout:     15 * time.Second,
		HTTPClient:  server.Client(),
		AppName:     "test",
		AppVersion:  "v1",
		APISubroute: "/v1/messages",
	}
	ctx := WithCLILogFile(context.Background(), logPath)
	res, err := adapter.Submit(ctx, session.CanonicalPackage{SessionID: "sess-api"})
	if err != nil {
		t.Fatalf("claude api submit failed: %v", err)
	}
	if res.RunID != "msg-123" {
		t.Fatalf("expected run id msg-123, got %#v", res)
	}
	if res.Provider != "claude_api" || res.Status != "end_turn" {
		t.Fatalf("unexpected claude api result: %#v", res)
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	logText := string(b)
	if !strings.Contains(logText, "Submitting to claude_api endpoint") || !strings.Contains(logText, "claude_api accepted request id=msg-123 status=end_turn") {
		t.Fatalf("expected claude api execution log content, got %q", logText)
	}
}

func TestBuildCLIPayloadTrimsInlineMediaToFitPromptBudget(t *testing.T) {
	largeData := "data:image/png;base64," + strings.Repeat("A", 900000)
	pkg := session.CanonicalPackage{
		SessionID: "sess-oversized-cli",
		Summary:   "Use the screenshot to adjust the layout.",
		ChangeRequests: []session.ChangeReq{{
			EventID: "evt-1",
			Summary: "Move the primary card higher.",
		}},
		Artifacts: []session.ArtifactRef{
			{
				Kind:               "screenshot",
				EventID:            "evt-1",
				InlineDataURL:      largeData,
				TransmissionStatus: "inline",
			},
			{
				Kind:               "audio",
				EventID:            "evt-1",
				InlineDataURL:      "data:audio/webm;base64," + strings.Repeat("B", 5000),
				TransmissionStatus: "inline",
			},
		},
	}

	payload := BuildCLIPayload(pkg, DeliveryIntent{})
	encodedPkg, _ := payload["package"].(session.CanonicalPackage)
	if len(encodedPkg.Artifacts) != 2 {
		t.Fatalf("expected artifacts to remain present, got %#v", encodedPkg.Artifacts)
	}
	if strings.TrimSpace(encodedPkg.Artifacts[0].InlineDataURL) != "" {
		t.Fatalf("expected oversized inline screenshot to be removed from cli payload")
	}
	if encodedPkg.Artifacts[0].TransmissionStatus != "omitted_for_cli_size_limit" {
		t.Fatalf("expected trimmed screenshot status, got %#v", encodedPkg.Artifacts[0].TransmissionStatus)
	}
	if !strings.Contains(encodedPkg.Artifacts[0].TransmissionNote, "fit the prompt size limit") {
		t.Fatalf("expected cli trim note, got %#v", encodedPkg.Artifacts[0].TransmissionNote)
	}
	if strings.TrimSpace(encodedPkg.Artifacts[1].InlineDataURL) == "" {
		t.Fatalf("expected smaller inline artifact to remain when payload fits")
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal trimmed cli payload: %v", err)
	}
	if len(b) > maxCLIPayloadChars {
		t.Fatalf("expected trimmed cli payload to fit prompt budget, got %d bytes", len(b))
	}
}

type stubAdapter struct {
	name   string
	remote bool
	result Result
}

func (a stubAdapter) Name() string { return a.name }

func (a stubAdapter) Submit(context.Context, session.CanonicalPackage) (Result, error) {
	return a.result, nil
}

func (a stubAdapter) IsRemote() bool { return a.remote }

func (a stubAdapter) Endpoint() string { return "" }

func mustLeafCertFromTLSServer(t *testing.T, srv *httptest.Server) *x509.Certificate {
	t.Helper()
	if len(srv.TLS.Certificates) == 0 || len(srv.TLS.Certificates[0].Certificate) == 0 {
		t.Fatal("missing TLS certificate chain")
	}
	leaf, err := x509.ParseCertificate(srv.TLS.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatalf("parse leaf cert: %v", err)
	}
	return leaf
}

func writeCertPEMForAdapterTest(t *testing.T, cert *x509.Certificate) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ca.pem")
	block := &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write cert pem: %v", err)
	}
	return path
}

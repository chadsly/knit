package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"knit/internal/security"
)

const defaultLMStudioSTTModel = "whisper-large-v3-turbo"

var fasterWhisperAllowedModels = map[string]struct{}{
	"tiny.en":           {},
	"tiny":              {},
	"base.en":           {},
	"base":              {},
	"small.en":          {},
	"small":             {},
	"medium.en":         {},
	"medium":            {},
	"large-v1":          {},
	"large-v2":          {},
	"large-v3":          {},
	"large":             {},
	"distil-large-v2":   {},
	"distil-medium.en":  {},
	"distil-small.en":   {},
	"distil-large-v3":   {},
	"distil-large-v3.5": {},
	"large-v3-turbo":    {},
	"turbo":             {},
}

func DefaultFasterWhisperModel() string {
	return defaultManagedFasterWhisperModel
}

func IsValidFasterWhisperModel(model string) bool {
	_, ok := fasterWhisperAllowedModels[strings.TrimSpace(strings.ToLower(model))]
	return ok
}

func NormalizeFasterWhisperModel(model string) string {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return defaultManagedFasterWhisperModel
	}
	if IsValidFasterWhisperModel(trimmed) {
		return trimmed
	}
	return defaultManagedFasterWhisperModel
}

type Provider interface {
	Name() string
	Transcribe(ctx context.Context, audioRef string) (string, error)
	Mode() string
	Endpoint() string
}

type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

type OpenAISpeechToTextProvider struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

func NewOpenAISpeechToTextProvider(apiKey, baseURL, model string, timeout time.Duration) OpenAISpeechToTextProvider {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	httpClient := &http.Client{Timeout: timeout}
	if client, err := security.NewHTTPClientFromEnv(timeout); err == nil && client != nil {
		httpClient = client
	}
	return OpenAISpeechToTextProvider{
		APIKey:     strings.TrimSpace(apiKey),
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Model:      defaultString(model, "gpt-4o-mini-transcribe"),
		HTTPClient: httpClient,
	}
}

func NewOpenAISpeechToTextProviderFromEnv() OpenAISpeechToTextProvider {
	baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	model := strings.TrimSpace(os.Getenv("OPENAI_STT_MODEL"))
	if model == "" {
		model = "gpt-4o-mini-transcribe"
	}
	return NewOpenAISpeechToTextProvider(strings.TrimSpace(os.Getenv("OPENAI_API_KEY")), baseURL, model, 90*time.Second)
}

func (p OpenAISpeechToTextProvider) Name() string { return "openai_speech_to_text" }
func (p OpenAISpeechToTextProvider) Mode() string { return "remote" }
func (p OpenAISpeechToTextProvider) Endpoint() string {
	return p.BaseURL
}

func (p OpenAISpeechToTextProvider) HealthCheck(ctx context.Context) error {
	if strings.TrimSpace(p.APIKey) == "" {
		return fmt.Errorf("OPENAI_API_KEY is required for OpenAI transcription")
	}
	return probeModelsEndpoint(ctx, p.HTTPClient, p.BaseURL, p.APIKey)
}

func (p OpenAISpeechToTextProvider) Transcribe(ctx context.Context, audioRef string) (string, error) {
	if p.APIKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is required for OpenAI transcription")
	}
	return transcribeMultipartRequest(ctx, p.HTTPClient, p.BaseURL, p.APIKey, p.Model, audioRef)
}

type LMStudioSpeechToTextProvider struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

func NewLMStudioSpeechToTextProvider(apiKey, baseURL, model string, timeout time.Duration) LMStudioSpeechToTextProvider {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "http://127.0.0.1:1234"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultLMStudioSTTModel
	}
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	httpClient := &http.Client{Timeout: timeout}
	if client, err := security.NewHTTPClientFromEnv(timeout); err == nil && client != nil {
		httpClient = client
	}
	return LMStudioSpeechToTextProvider{
		APIKey:     strings.TrimSpace(apiKey),
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Model:      model,
		HTTPClient: httpClient,
	}
}

func NewLMStudioSpeechToTextProviderFromEnv() LMStudioSpeechToTextProvider {
	baseURL := strings.TrimSpace(os.Getenv("KNIT_LMSTUDIO_BASE_URL"))
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("LMSTUDIO_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = "http://127.0.0.1:1234"
	}
	model := strings.TrimSpace(os.Getenv("KNIT_LMSTUDIO_STT_MODEL"))
	if model == "" {
		model = strings.TrimSpace(os.Getenv("LMSTUDIO_STT_MODEL"))
	}
	if model == "" {
		model = strings.TrimSpace(os.Getenv("OPENAI_STT_MODEL"))
	}
	if model == "" {
		model = defaultLMStudioSTTModel
	}
	apiKey := strings.TrimSpace(os.Getenv("KNIT_LMSTUDIO_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("LMSTUDIO_API_KEY"))
	}
	timeout := 90 * time.Second
	if v := strings.TrimSpace(os.Getenv("KNIT_LMSTUDIO_STT_TIMEOUT_SECONDS")); v != "" {
		if sec, err := time.ParseDuration(v + "s"); err == nil && sec > 0 {
			timeout = sec
		}
	}
	return NewLMStudioSpeechToTextProvider(apiKey, baseURL, model, timeout)
}

func (p LMStudioSpeechToTextProvider) Name() string { return "lmstudio_speech_to_text" }
func (p LMStudioSpeechToTextProvider) Mode() string { return "lmstudio" }
func (p LMStudioSpeechToTextProvider) Endpoint() string {
	return p.BaseURL
}

func (p LMStudioSpeechToTextProvider) HealthCheck(ctx context.Context) error {
	return probeModelsEndpoint(ctx, p.HTTPClient, p.BaseURL, p.APIKey)
}

func (p LMStudioSpeechToTextProvider) Transcribe(ctx context.Context, audioRef string) (string, error) {
	return transcribeMultipartRequest(ctx, p.HTTPClient, p.BaseURL, p.APIKey, p.Model, audioRef)
}

func transcribeMultipartRequest(ctx context.Context, httpClient *http.Client, baseURL, apiKey, model, audioRef string) (string, error) {
	if strings.TrimSpace(model) == "" {
		model = "whisper-1"
	}
	f, err := os.Open(audioRef)
	if err != nil {
		return "", fmt.Errorf("open audio file: %w", err)
	}
	defer f.Close()

	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	defer security.ZeroBytes(buf.Bytes())
	if err := writer.WriteField("model", model); err != nil {
		return "", fmt.Errorf("set model field: %w", err)
	}
	fileWriter, err := writer.CreateFormFile("file", filepath.Base(audioRef))
	if err != nil {
		return "", fmt.Errorf("create file form field: %w", err)
	}
	if _, err := io.Copy(fileWriter, f); err != nil {
		return "", fmt.Errorf("copy audio content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	url := strings.TrimRight(baseURL, "/") + "/v1/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buf)
	if err != nil {
		return "", fmt.Errorf("build transcription request: %w", err)
	}
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("transcription request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	defer security.ZeroBytes(respBody)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("transcription api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode transcription response: %w", err)
	}
	if strings.TrimSpace(parsed.Text) == "" {
		return "", fmt.Errorf("transcription response did not include text")
	}
	return parsed.Text, nil
}

type LocalCLIProvider struct {
	Command string
	Timeout time.Duration
}

func NewLocalCLIProviderFromEnv() LocalCLIProvider {
	timeout := 90 * time.Second
	if v := strings.TrimSpace(os.Getenv("KNIT_LOCAL_STT_TIMEOUT_SECONDS")); v != "" {
		if sec, err := time.ParseDuration(v + "s"); err == nil && sec > 0 {
			timeout = sec
		}
	}
	return NewLocalCLIProvider(strings.TrimSpace(os.Getenv("KNIT_LOCAL_STT_CMD")), timeout)
}

func NewLocalCLIProvider(command string, timeout time.Duration) LocalCLIProvider {
	return LocalCLIProvider{
		Command: strings.TrimSpace(command),
		Timeout: timeout,
	}
}

func (p LocalCLIProvider) Name() string     { return "local_cli_stt" }
func (p LocalCLIProvider) Mode() string     { return "local" }
func (p LocalCLIProvider) Endpoint() string { return "local-process" }

func (p LocalCLIProvider) HealthCheck(ctx context.Context) error {
	if strings.TrimSpace(p.Command) == "" {
		return fmt.Errorf("local transcription command is not configured (KNIT_LOCAL_STT_CMD)")
	}
	_ = ctx
	return nil
}

func (p LocalCLIProvider) Transcribe(ctx context.Context, audioRef string) (string, error) {
	if strings.TrimSpace(p.Command) == "" {
		return "", fmt.Errorf("local transcription command is not configured (KNIT_LOCAL_STT_CMD)")
	}
	runCtx := ctx
	if p.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, p.Timeout)
		defer cancel()
	}
	cmd := shellCommand(runCtx, p.Command)
	cmd.Env = append(os.Environ(), "KNIT_STT_AUDIO_PATH="+audioRef)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("local transcription command failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	transcript := strings.TrimSpace(string(output))
	if transcript == "" {
		return "", fmt.Errorf("local transcription command returned empty transcript")
	}
	return transcript, nil
}

type DisabledProvider struct {
	mode string
}

func (p DisabledProvider) Name() string     { return "disabled" }
func (p DisabledProvider) Mode() string     { return p.mode }
func (p DisabledProvider) Endpoint() string { return "" }
func (p DisabledProvider) HealthCheck(ctx context.Context) error {
	return fmt.Errorf("transcription provider disabled")
}
func (p DisabledProvider) Transcribe(ctx context.Context, audioRef string) (string, error) {
	return "", fmt.Errorf("transcription provider disabled")
}

func NewProviderFromEnv(mode string) Provider {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "local":
		return NewLocalCLIProviderFromEnv()
	case "faster_whisper":
		return NewManagedFasterWhisperProviderFromEnv()
	case "lmstudio":
		return NewLMStudioSpeechToTextProviderFromEnv()
	case "remote":
		return NewOpenAISpeechToTextProviderFromEnv()
	default:
		return DisabledProvider{mode: "local"}
	}
}

func shellCommand(ctx context.Context, raw string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", raw)
	}
	return exec.CommandContext(ctx, "sh", "-lc", raw)
}

func probeModelsEndpoint(ctx context.Context, client *http.Client, baseURL, apiKey string) error {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if baseURL == "" {
		return fmt.Errorf("transcription endpoint is empty")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return fmt.Errorf("build models probe request: %w", err)
	}
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("models probe failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("models probe returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func HealthCheckProvider(ctx context.Context, p Provider) error {
	if p == nil {
		return fmt.Errorf("transcription provider is not configured")
	}
	if hc, ok := p.(HealthChecker); ok {
		return hc.HealthCheck(ctx)
	}
	return nil
}

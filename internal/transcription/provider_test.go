package transcription

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewProviderFromEnvLocalMode(t *testing.T) {
	t.Setenv("KNIT_LOCAL_STT_CMD", "")
	p := NewProviderFromEnv("local")
	if p.Mode() != "local" {
		t.Fatalf("expected local mode, got %s", p.Mode())
	}
	if p.Endpoint() != "local-process" {
		t.Fatalf("expected local endpoint marker, got %s", p.Endpoint())
	}
	if _, err := p.Transcribe(context.Background(), "sample.wav"); err == nil {
		t.Fatalf("expected local provider without command to return an error")
	}
}

func TestNewProviderFromEnvRemoteMode(t *testing.T) {
	p := NewProviderFromEnv("remote")
	if p.Mode() != "remote" {
		t.Fatalf("expected remote mode, got %s", p.Mode())
	}
	if p.Endpoint() == "" {
		t.Fatalf("expected remote endpoint")
	}
}

func TestNewProviderFromEnvLMStudioMode(t *testing.T) {
	t.Setenv("KNIT_LMSTUDIO_BASE_URL", "http://127.0.0.1:1234")
	p := NewProviderFromEnv("lmstudio")
	if p.Mode() != "lmstudio" {
		t.Fatalf("expected lmstudio mode for lmstudio provider, got %s", p.Mode())
	}
	if !strings.Contains(p.Endpoint(), "127.0.0.1") {
		t.Fatalf("expected lmstudio endpoint, got %s", p.Endpoint())
	}
}

func TestNewProviderFromEnvManagedFasterWhisperMode(t *testing.T) {
	t.Setenv("KNIT_DATA_DIR", t.TempDir())
	p := NewProviderFromEnv("faster_whisper")
	if p.Mode() != "faster_whisper" {
		t.Fatalf("expected faster_whisper mode for faster_whisper provider, got %s", p.Mode())
	}
	if p.Endpoint() != "local-managed:faster-whisper" {
		t.Fatalf("expected managed faster-whisper endpoint marker, got %s", p.Endpoint())
	}
}

func TestNewProviderFromEnvManagedFasterWhisperNormalizesInvalidModel(t *testing.T) {
	t.Setenv("KNIT_DATA_DIR", t.TempDir())
	t.Setenv("KNIT_FASTER_WHISPER_MODEL", "gpt-4o-mini-transcribe")
	p := NewProviderFromEnv("faster_whisper")
	provider, ok := p.(*ManagedFasterWhisperProvider)
	if !ok {
		t.Fatalf("expected managed faster-whisper provider, got %T", p)
	}
	if provider.Model != DefaultFasterWhisperModel() {
		t.Fatalf("expected normalized faster-whisper model %q, got %q", DefaultFasterWhisperModel(), provider.Model)
	}
}

func TestLMStudioProviderTranscribeSuccess(t *testing.T) {
	var authHeader string
	var modelField string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			http.NotFound(w, r)
			return
		}
		authHeader = r.Header.Get("Authorization")
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			http.Error(w, "bad multipart", http.StatusBadRequest)
			return
		}
		modelField = strings.TrimSpace(r.FormValue("model"))
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		if _, err := io.ReadAll(file); err != nil {
			http.Error(w, "read file", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"text":"hello from lm studio"}`))
	}))
	defer server.Close()

	audioPath := filepath.Join(t.TempDir(), "note.webm")
	if err := os.WriteFile(audioPath, []byte("fake-audio"), 0o600); err != nil {
		t.Fatalf("write audio fixture: %v", err)
	}
	p := LMStudioSpeechToTextProvider{
		APIKey:  "lm-token",
		BaseURL: server.URL,
		Model:   defaultLMStudioSTTModel,
	}
	got, err := p.Transcribe(context.Background(), audioPath)
	if err != nil {
		t.Fatalf("transcribe failed: %v", err)
	}
	if got != "hello from lm studio" {
		t.Fatalf("unexpected transcript: %q", got)
	}
	if authHeader != "Bearer lm-token" {
		t.Fatalf("expected auth header to be forwarded")
	}
	if modelField != defaultLMStudioSTTModel {
		t.Fatalf("expected model %s, got %q", defaultLMStudioSTTModel, modelField)
	}
}

func TestOpenAIProviderTranscribeSupportsCustomCATrustAndPinnedCert(t *testing.T) {
	var authHeader string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			http.NotFound(w, r)
			return
		}
		authHeader = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"text":"secure transcript"}`))
	}))
	defer server.Close()

	leaf := mustLeafCertForProviderTest(t, server)
	sum := sha256.Sum256(leaf.Raw)
	t.Setenv("KNIT_HTTP_PROXY", "")
	t.Setenv("KNIT_NO_PROXY", "")
	t.Setenv("KNIT_TLS_CA_FILE", writeCertPEMForProviderTest(t, leaf))
	t.Setenv("KNIT_TLS_PINNED_CERT_SHA256", hex.EncodeToString(sum[:]))

	audioPath := filepath.Join(t.TempDir(), "note.webm")
	if err := os.WriteFile(audioPath, []byte("audio"), 0o600); err != nil {
		t.Fatalf("write audio fixture: %v", err)
	}
	provider := NewOpenAISpeechToTextProvider("secure-token", server.URL, "gpt-4o-mini-transcribe", 5*time.Second)
	got, err := provider.Transcribe(context.Background(), audioPath)
	if err != nil {
		t.Fatalf("secure transcribe failed: %v", err)
	}
	if got != "secure transcript" {
		t.Fatalf("unexpected transcript: %q", got)
	}
	if authHeader != "Bearer secure-token" {
		t.Fatalf("expected auth header to be forwarded, got %q", authHeader)
	}
}

func TestOpenAIProviderTranscribeRejectsPinnedCertMismatch(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"text":"should not succeed"}`))
	}))
	defer server.Close()

	leaf := mustLeafCertForProviderTest(t, server)
	t.Setenv("KNIT_HTTP_PROXY", "")
	t.Setenv("KNIT_NO_PROXY", "")
	t.Setenv("KNIT_TLS_CA_FILE", writeCertPEMForProviderTest(t, leaf))
	t.Setenv("KNIT_TLS_PINNED_CERT_SHA256", strings.Repeat("aa", sha256.Size))

	audioPath := filepath.Join(t.TempDir(), "note.webm")
	if err := os.WriteFile(audioPath, []byte("audio"), 0o600); err != nil {
		t.Fatalf("write audio fixture: %v", err)
	}
	provider := NewOpenAISpeechToTextProvider("secure-token", server.URL, "gpt-4o-mini-transcribe", 5*time.Second)
	_, err := provider.Transcribe(context.Background(), audioPath)
	if err == nil {
		t.Fatal("expected tls pin mismatch")
	}
	if !strings.Contains(err.Error(), "tls pin mismatch") {
		t.Fatalf("expected tls pin mismatch error, got %v", err)
	}
}

func TestTranscribeMultipartRequestReadsTextField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
			http.Error(w, "bad content type", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"text":"ok"}`))
	}))
	defer server.Close()

	audioPath := filepath.Join(t.TempDir(), "note.wav")
	if err := os.WriteFile(audioPath, []byte("pcm"), 0o600); err != nil {
		t.Fatalf("write audio fixture: %v", err)
	}
	got, err := transcribeMultipartRequest(context.Background(), nil, server.URL, "", "model-a", audioPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ok" {
		t.Fatalf("expected ok transcript, got %q", got)
	}
}

func TestTranscribeMultipartRequestRejectsMissingText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":"missing-text"}`))
	}))
	defer server.Close()

	audioPath := filepath.Join(t.TempDir(), "note.wav")
	if err := os.WriteFile(audioPath, []byte("pcm"), 0o600); err != nil {
		t.Fatalf("write audio fixture: %v", err)
	}
	_, err := transcribeMultipartRequest(context.Background(), &http.Client{}, server.URL, "", "model-a", audioPath)
	if err == nil {
		t.Fatalf("expected missing text error")
	}
}

func TestTranscribeMultipartRequestIncludesModelField(t *testing.T) {
	var model string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reader, err := r.MultipartReader()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if part.FormName() == "model" {
				b, _ := io.ReadAll(part)
				model = string(b)
			}
		}
		_, _ = w.Write([]byte(`{"text":"ok"}`))
	}))
	defer server.Close()

	audioPath := filepath.Join(t.TempDir(), "note.webm")
	if err := os.WriteFile(audioPath, []byte("audio"), 0o600); err != nil {
		t.Fatalf("write audio fixture: %v", err)
	}
	_, err := transcribeMultipartRequest(context.Background(), nil, server.URL, "", "test-model", audioPath)
	if err != nil {
		t.Fatalf("transcribe failed: %v", err)
	}
	if strings.TrimSpace(model) != "test-model" {
		t.Fatalf("expected model field, got %q", model)
	}
}

func TestTranscribeMultipartRequestDefaultModelFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			http.Error(w, "bad multipart", http.StatusBadRequest)
			return
		}
		if r.FormValue("model") != "whisper-1" {
			http.Error(w, "unexpected model", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"text":"ok"}`))
	}))
	defer server.Close()

	audioPath := filepath.Join(t.TempDir(), "note.webm")
	if err := os.WriteFile(audioPath, []byte("audio"), 0o600); err != nil {
		t.Fatalf("write audio fixture: %v", err)
	}
	if _, err := transcribeMultipartRequest(context.Background(), nil, server.URL, "", "", audioPath); err != nil {
		t.Fatalf("expected default model fallback, got %v", err)
	}
}

func TestLMStudioProviderEnvConfig(t *testing.T) {
	t.Setenv("KNIT_LMSTUDIO_BASE_URL", "http://127.0.0.1:15432")
	t.Setenv("KNIT_LMSTUDIO_STT_MODEL", "my-local-stt")
	t.Setenv("KNIT_LMSTUDIO_API_KEY", "local-token")
	t.Setenv("KNIT_LMSTUDIO_STT_TIMEOUT_SECONDS", "12")
	p := NewLMStudioSpeechToTextProviderFromEnv()
	if p.BaseURL != "http://127.0.0.1:15432" {
		t.Fatalf("unexpected base url: %s", p.BaseURL)
	}
	if p.Model != "my-local-stt" {
		t.Fatalf("unexpected model: %s", p.Model)
	}
	if p.APIKey != "local-token" {
		t.Fatalf("unexpected api key")
	}
	if p.HTTPClient == nil || p.HTTPClient.Timeout.Seconds() != 12 {
		t.Fatalf("expected timeout 12 seconds")
	}
}

func TestLMStudioProviderUsesDefaultModelWhenUnset(t *testing.T) {
	t.Setenv("KNIT_LMSTUDIO_STT_MODEL", "")
	t.Setenv("LMSTUDIO_STT_MODEL", "")
	t.Setenv("OPENAI_STT_MODEL", "")
	p := NewLMStudioSpeechToTextProviderFromEnv()
	if p.Model != defaultLMStudioSTTModel {
		t.Fatalf("expected default lmstudio model %q, got %q", defaultLMStudioSTTModel, p.Model)
	}
}

func TestHealthCheckProviderLocalCommand(t *testing.T) {
	p := LocalCLIProvider{Command: "printf ok"}
	if err := HealthCheckProvider(context.Background(), p); err != nil {
		t.Fatalf("expected local command health check to pass, got %v", err)
	}
}

func mustLeafCertForProviderTest(t *testing.T, srv *httptest.Server) *x509.Certificate {
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

func writeCertPEMForProviderTest(t *testing.T, cert *x509.Certificate) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ca.pem")
	block := &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write cert pem: %v", err)
	}
	return path
}

func TestHealthCheckProviderDisabledFails(t *testing.T) {
	p := DisabledProvider{mode: "local"}
	if err := HealthCheckProvider(context.Background(), p); err == nil {
		t.Fatalf("expected disabled provider health check to fail")
	}
}

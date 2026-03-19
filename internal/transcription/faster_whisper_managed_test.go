package transcription

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManagedFasterWhisperProviderTranscribeUsesManagedRuntime(t *testing.T) {
	p := NewManagedFasterWhisperProviderFromEnv()
	p.Model = "small"
	p.Device = "cpu"
	p.ComputeType = "int8"
	p.Timeout = 5 * time.Second

	p.ensureFn = func(_ context.Context, provider *ManagedFasterWhisperProvider) error {
		provider.venvPython = "/tmp/fake-python"
		provider.scriptPath = "/tmp/fake-script.py"
		return nil
	}

	var gotName string
	var gotArgs []string
	var gotEnv []string
	p.execFn = func(_ context.Context, name string, args []string, env []string, _ string) ([]byte, error) {
		gotName = name
		gotArgs = append([]string(nil), args...)
		gotEnv = append([]string(nil), env...)
		return []byte("turn left at settings panel"), nil
	}

	out, err := p.Transcribe(context.Background(), "/tmp/sample.webm")
	if err != nil {
		t.Fatalf("transcribe failed: %v", err)
	}
	if out != "turn left at settings panel" {
		t.Fatalf("unexpected transcript: %q", out)
	}
	if gotName != "/tmp/fake-python" {
		t.Fatalf("expected managed python executable, got %q", gotName)
	}
	if len(gotArgs) != 1 || gotArgs[0] != "/tmp/fake-script.py" {
		t.Fatalf("unexpected managed script args: %#v", gotArgs)
	}
	joined := strings.Join(gotEnv, "\n")
	if !strings.Contains(joined, "KNIT_STT_AUDIO_PATH=/tmp/sample.webm") {
		t.Fatalf("expected audio path env in managed exec")
	}
	if !strings.Contains(joined, "KNIT_FASTER_WHISPER_MODEL=small") {
		t.Fatalf("expected model env in managed exec")
	}
}

func TestManagedFasterWhisperProviderTranscribeExecFailure(t *testing.T) {
	p := NewManagedFasterWhisperProviderFromEnv()
	p.ensureFn = func(_ context.Context, provider *ManagedFasterWhisperProvider) error {
		provider.venvPython = "/tmp/fake-python"
		provider.scriptPath = "/tmp/fake-script.py"
		return nil
	}
	p.execFn = func(_ context.Context, _ string, _ []string, _ []string, _ string) ([]byte, error) {
		return nil, errors.New("run failed")
	}

	_, err := p.Transcribe(context.Background(), "/tmp/sample.webm")
	if err == nil {
		t.Fatalf("expected managed faster-whisper exec failure")
	}
}

func TestManagedFasterWhisperProviderRequiresAudioPath(t *testing.T) {
	p := NewManagedFasterWhisperProviderFromEnv()
	_, err := p.Transcribe(context.Background(), "")
	if err == nil {
		t.Fatalf("expected missing audio path error")
	}
}

func TestManagedFasterWhisperProviderHealthCheckCallsEnsure(t *testing.T) {
	p := NewManagedFasterWhisperProviderFromEnv()
	called := 0
	p.ensureFn = func(_ context.Context, provider *ManagedFasterWhisperProvider) error {
		called++
		provider.venvPython = "/tmp/fake-python"
		provider.scriptPath = "/tmp/fake-script.py"
		return nil
	}
	if err := p.HealthCheck(context.Background()); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	if called != 1 {
		t.Fatalf("expected ensureFn to be called once, got %d", called)
	}
}

func TestEnsureManagedFasterWhisperRuntimeBootstrapsVenvWhenMissing(t *testing.T) {
	runtimeDir := t.TempDir()
	p := NewManagedFasterWhisperProviderFromEnv()
	p.RuntimeDir = runtimeDir
	p.BasePython = "python3"
	p.AutoInstall = true

	var calls []string
	p.execFn = func(_ context.Context, name string, args []string, _ []string, _ string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if strings.Contains(strings.Join(args, " "), "-m venv") {
			venvPython := filepath.Join(runtimeDir, "venv", "bin", "python")
			if err := os.MkdirAll(filepath.Dir(venvPython), 0o755); err != nil {
				t.Fatalf("mkdir venv python dir: %v", err)
			}
			if err := os.WriteFile(venvPython, []byte("#!/bin/sh\necho fake"), 0o755); err != nil {
				t.Fatalf("write venv python: %v", err)
			}
		}
		return []byte("ok"), nil
	}

	if err := ensureManagedFasterWhisperRuntime(context.Background(), p); err != nil {
		t.Fatalf("ensure runtime failed: %v", err)
	}
	if len(calls) < 4 {
		t.Fatalf("expected bootstrap command sequence, got %v", calls)
	}
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, "-m venv") {
		t.Fatalf("expected venv creation command in calls: %v", calls)
	}
	if !strings.Contains(joined, "pip install faster-whisper") {
		t.Fatalf("expected faster-whisper install command in calls: %v", calls)
	}
}

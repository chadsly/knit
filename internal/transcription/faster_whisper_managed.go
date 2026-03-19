package transcription

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultManagedFasterWhisperModel       = "small"
	defaultManagedFasterWhisperDevice      = "cpu"
	defaultManagedFasterWhisperComputeType = "int8"
)

//go:embed assets/faster_whisper_stt.py
var fasterWhisperScript string

var execCommandContext = exec.CommandContext

type managedExecFn func(ctx context.Context, name string, args []string, env []string, dir string) ([]byte, error)

type ManagedFasterWhisperProvider struct {
	BasePython              string
	RuntimeDir              string
	Model                   string
	Device                  string
	ComputeType             string
	Language                string
	BeamSize                int
	VADFilter               bool
	ConditionOnPreviousText bool
	AutoInstall             bool
	Timeout                 time.Duration

	mu         sync.Mutex
	ready      bool
	venvPython string
	scriptPath string

	ensureFn func(context.Context, *ManagedFasterWhisperProvider) error
	execFn   managedExecFn
}

func NewManagedFasterWhisperProvider(runtimeDir, basePython, model, device, computeType, language string, timeout time.Duration) *ManagedFasterWhisperProvider {
	if strings.TrimSpace(runtimeDir) == "" {
		runtimeDir = filepath.Join(".knit", "runtime", "faster-whisper")
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &ManagedFasterWhisperProvider{
		BasePython:              strings.TrimSpace(defaultString(basePython, "python3")),
		RuntimeDir:              runtimeDir,
		Model:                   NormalizeFasterWhisperModel(defaultString(model, defaultManagedFasterWhisperModel)),
		Device:                  strings.TrimSpace(defaultString(device, defaultManagedFasterWhisperDevice)),
		ComputeType:             strings.TrimSpace(defaultString(computeType, defaultManagedFasterWhisperComputeType)),
		Language:                strings.TrimSpace(language),
		BeamSize:                1,
		VADFilter:               true,
		ConditionOnPreviousText: false,
		AutoInstall:             true,
		Timeout:                 timeout,
		ensureFn:                ensureManagedFasterWhisperRuntime,
		execFn:                  runManagedCommand,
	}
}

func NewManagedFasterWhisperProviderFromEnv() *ManagedFasterWhisperProvider {
	runtimeDir := strings.TrimSpace(os.Getenv("KNIT_FASTER_WHISPER_RUNTIME_DIR"))
	if runtimeDir == "" {
		dataDir := strings.TrimSpace(os.Getenv("KNIT_DATA_DIR"))
		if dataDir == "" {
			dataDir = ".knit"
		}
		runtimeDir = filepath.Join(dataDir, "runtime", "faster-whisper")
	}
	timeout := 120 * time.Second
	if v := strings.TrimSpace(os.Getenv("KNIT_FASTER_WHISPER_TIMEOUT_SECONDS")); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			timeout = time.Duration(sec) * time.Second
		}
	} else if v := strings.TrimSpace(os.Getenv("KNIT_LOCAL_STT_TIMEOUT_SECONDS")); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			timeout = time.Duration(sec) * time.Second
		}
	}
	beamSize := 1
	if v := strings.TrimSpace(os.Getenv("KNIT_FASTER_WHISPER_BEAM_SIZE")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			beamSize = parsed
		}
	}
	provider := NewManagedFasterWhisperProvider(
		runtimeDir,
		os.Getenv("KNIT_FASTER_WHISPER_PYTHON"),
		os.Getenv("KNIT_FASTER_WHISPER_MODEL"),
		os.Getenv("KNIT_FASTER_WHISPER_DEVICE"),
		os.Getenv("KNIT_FASTER_WHISPER_COMPUTE_TYPE"),
		os.Getenv("KNIT_FASTER_WHISPER_LANGUAGE"),
		timeout,
	)
	provider.BeamSize = beamSize
	provider.VADFilter = parseBoolEnvWithDefault("KNIT_FASTER_WHISPER_VAD_FILTER", true)
	provider.ConditionOnPreviousText = parseBoolEnvWithDefault("KNIT_FASTER_WHISPER_CONDITION_ON_PREV_TEXT", false)
	provider.AutoInstall = parseBoolEnvWithDefault("KNIT_FASTER_WHISPER_AUTO_INSTALL", true)
	return provider
}

func (p *ManagedFasterWhisperProvider) Name() string { return "managed_faster_whisper_stt" }
func (p *ManagedFasterWhisperProvider) Mode() string { return "faster_whisper" }
func (p *ManagedFasterWhisperProvider) Endpoint() string {
	return "local-managed:faster-whisper"
}

func (p *ManagedFasterWhisperProvider) HealthCheck(ctx context.Context) error {
	runCtx := ctx
	if p.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, p.Timeout)
		defer cancel()
	}
	if err := p.ensureFn(runCtx, p); err != nil {
		return fmt.Errorf("managed faster-whisper runtime unavailable: %w", err)
	}
	return nil
}

func (p *ManagedFasterWhisperProvider) Transcribe(ctx context.Context, audioRef string) (string, error) {
	if strings.TrimSpace(audioRef) == "" {
		return "", fmt.Errorf("audio reference is required")
	}
	runCtx := ctx
	if p.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, p.Timeout)
		defer cancel()
	}
	if err := p.ensureFn(runCtx, p); err != nil {
		return "", err
	}
	env := append(os.Environ(),
		"KNIT_STT_AUDIO_PATH="+audioRef,
		"KNIT_FASTER_WHISPER_MODEL="+p.Model,
		"KNIT_FASTER_WHISPER_DEVICE="+p.Device,
		"KNIT_FASTER_WHISPER_COMPUTE_TYPE="+p.ComputeType,
		"KNIT_FASTER_WHISPER_BEAM_SIZE="+strconv.Itoa(max(1, p.BeamSize)),
		"KNIT_FASTER_WHISPER_VAD_FILTER="+strconv.FormatBool(p.VADFilter),
		"KNIT_FASTER_WHISPER_CONDITION_ON_PREV_TEXT="+strconv.FormatBool(p.ConditionOnPreviousText),
	)
	if p.Language != "" {
		env = append(env, "KNIT_FASTER_WHISPER_LANGUAGE="+p.Language)
	}
	out, err := p.execFn(runCtx, p.venvPython, []string{p.scriptPath}, env, p.RuntimeDir)
	if err != nil {
		return "", fmt.Errorf("managed faster-whisper transcription failed: %w", err)
	}
	transcript := strings.TrimSpace(string(out))
	if transcript == "" {
		return "", fmt.Errorf("managed faster-whisper returned empty transcript")
	}
	return transcript, nil
}

func ensureManagedFasterWhisperRuntime(ctx context.Context, p *ManagedFasterWhisperProvider) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ready && p.venvPython != "" && p.scriptPath != "" {
		return nil
	}

	runtimeDir := strings.TrimSpace(p.RuntimeDir)
	if runtimeDir == "" {
		return fmt.Errorf("managed faster-whisper runtime dir is empty")
	}
	absRuntime, err := filepath.Abs(runtimeDir)
	if err != nil {
		return fmt.Errorf("resolve runtime dir: %w", err)
	}
	if err := os.MkdirAll(absRuntime, 0o700); err != nil {
		return fmt.Errorf("create runtime dir: %w", err)
	}

	scriptPath := filepath.Join(absRuntime, "faster_whisper_stt.py")
	if err := os.WriteFile(scriptPath, []byte(fasterWhisperScript), 0o700); err != nil {
		return fmt.Errorf("write managed faster-whisper script: %w", err)
	}

	venvDir := filepath.Join(absRuntime, "venv")
	venvPython := filepath.Join(venvDir, "bin", "python")
	if runtime.GOOS == "windows" {
		venvPython = filepath.Join(venvDir, "Scripts", "python.exe")
	}

	if _, statErr := os.Stat(venvPython); statErr != nil {
		if !p.AutoInstall {
			return fmt.Errorf("managed faster-whisper runtime is not installed and auto-install is disabled")
		}
		if _, err := p.execFn(ctx, p.BasePython, []string{"-m", "venv", venvDir}, os.Environ(), absRuntime); err != nil {
			return fmt.Errorf("create managed faster-whisper venv: %w", err)
		}
		if _, err := p.execFn(ctx, venvPython, []string{"-m", "pip", "install", "--upgrade", "pip"}, os.Environ(), absRuntime); err != nil {
			return fmt.Errorf("upgrade pip for managed faster-whisper runtime: %w", err)
		}
		if _, err := p.execFn(ctx, venvPython, []string{"-m", "pip", "install", "faster-whisper"}, os.Environ(), absRuntime); err != nil {
			return fmt.Errorf("install faster-whisper runtime dependency: %w", err)
		}
	}

	if _, err := p.execFn(ctx, venvPython, []string{"-c", "import faster_whisper"}, os.Environ(), absRuntime); err != nil {
		if !p.AutoInstall {
			return fmt.Errorf("managed faster-whisper dependency check failed and auto-install is disabled: %w", err)
		}
		if _, installErr := p.execFn(ctx, venvPython, []string{"-m", "pip", "install", "faster-whisper"}, os.Environ(), absRuntime); installErr != nil {
			return fmt.Errorf("repair/install faster-whisper dependency: %w", installErr)
		}
		if _, verifyErr := p.execFn(ctx, venvPython, []string{"-c", "import faster_whisper"}, os.Environ(), absRuntime); verifyErr != nil {
			return fmt.Errorf("verify faster-whisper dependency: %w", verifyErr)
		}
	}

	p.RuntimeDir = absRuntime
	p.scriptPath = scriptPath
	p.venvPython = venvPython
	p.ready = true
	return nil
}

func runManagedCommand(ctx context.Context, name string, args []string, env []string, dir string) ([]byte, error) {
	cmd := execCommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = env
	}
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s: %s", name, msg)
	}
	return out, nil
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func parseBoolEnvWithDefault(name string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

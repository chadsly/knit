package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"knit/internal/agents"
	"knit/internal/audio"
	"knit/internal/audit"
	"knit/internal/capture"
	"knit/internal/companion"
	"knit/internal/config"
	"knit/internal/operatorstate"
	"knit/internal/platform"
	"knit/internal/privileged"
	"knit/internal/retention"
	"knit/internal/security"
	"knit/internal/server"
	"knit/internal/session"
	"knit/internal/storage"
	"knit/internal/transcription"
	"knit/internal/userconfig"
)

type App struct {
	server    *server.Server
	store     storage.Store
	retention *retention.Worker
}

func DefaultConfig() config.Config {
	cfg := config.Default()
	if v := os.Getenv("KNIT_CONTROL_TOKEN"); strings.TrimSpace(v) != "" {
		cfg.ControlToken = strings.TrimSpace(v)
	} else {
		cfg.ControlToken = generateControlToken()
	}
	if v := os.Getenv("KNIT_CONTROL_CAPABILITIES"); strings.TrimSpace(v) != "" {
		cfg.ControlCapabilities = splitCSV(v)
	}
	if v := os.Getenv("KNIT_CONFIG_LOCKED"); strings.EqualFold(strings.TrimSpace(v), "true") || strings.TrimSpace(v) == "1" {
		cfg.ConfigLocked = true
	}
	if v := os.Getenv("KNIT_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("KNIT_ADDR"); v != "" {
		cfg.HTTPListenAddr = v
	}
	if v := os.Getenv("KNIT_SQLITE_PATH"); v != "" {
		cfg.SQLitePath = v
	}
	if v := os.Getenv("KNIT_CONFIG_PATH"); strings.TrimSpace(v) != "" {
		cfg.UserConfigPath = strings.TrimSpace(v)
	}
	if v := os.Getenv("KNIT_PROFILE"); strings.TrimSpace(v) != "" {
		cfg.LocalProfile = strings.TrimSpace(v)
	}
	if v := os.Getenv("KNIT_ENVIRONMENT"); strings.TrimSpace(v) != "" {
		cfg.EnvironmentName = strings.TrimSpace(v)
	}
	if v := os.Getenv("KNIT_BUILD_ID"); strings.TrimSpace(v) != "" {
		cfg.BuildID = strings.TrimSpace(v)
	}
	if v := os.Getenv("KNIT_VERSION_PIN"); strings.TrimSpace(v) != "" {
		cfg.VersionPin = strings.TrimSpace(v)
	}
	if v := os.Getenv("KNIT_MANAGED_DEPLOYMENT_ID"); strings.TrimSpace(v) != "" {
		cfg.ManagedDeploymentID = strings.TrimSpace(v)
	}
	if v := os.Getenv("KNIT_POINTER_SAMPLE_HZ"); strings.TrimSpace(v) != "" {
		if hz, err := parseInt(v); err == nil && hz > 0 {
			cfg.PointerSampleHz = hz
		}
	}
	if v := os.Getenv("KNIT_CAPTURE_SETTINGS_LOCKED"); strings.TrimSpace(v) != "" {
		cfg.CaptureSettingsLocked = parseBool(v, cfg.CaptureSettingsLocked)
	}
	if v := os.Getenv("KNIT_AUTO_START"); strings.TrimSpace(v) != "" {
		cfg.AutoStartEnabled = parseBool(v, cfg.AutoStartEnabled)
	}
	if v := os.Getenv("KNIT_TRANSCRIPTION_MODE"); strings.TrimSpace(v) != "" {
		cfg.TranscriptionMode = strings.TrimSpace(v)
	}
	if v := os.Getenv("KNIT_ALLOW_REMOTE_STT"); strings.TrimSpace(v) != "" {
		cfg.AllowRemoteSTT = parseBool(v, cfg.AllowRemoteSTT)
	}
	if v := os.Getenv("KNIT_ALLOW_REMOTE_SUBMISSION"); strings.TrimSpace(v) != "" {
		cfg.AllowRemoteSubmission = parseBool(v, cfg.AllowRemoteSubmission)
	}
	if v := os.Getenv("OPENAI_BASE_URL"); strings.TrimSpace(v) != "" {
		// Presence of OpenAI base URL implies remote provider mode in v1.
		cfg.TranscriptionMode = "remote"
	}
	if v := os.Getenv("KNIT_OUTBOUND_ALLOWLIST"); strings.TrimSpace(v) != "" {
		cfg.OutboundAllowlist = splitCSV(v)
	}
	if v := os.Getenv("KNIT_OUTBOUND_BLOCKLIST"); strings.TrimSpace(v) != "" {
		cfg.BlockedTargets = splitCSV(v)
	}
	if v := os.Getenv("KNIT_ALLOWED_SUBMIT_PROVIDERS"); strings.TrimSpace(v) != "" {
		cfg.AllowedSubmitProviders = splitCSV(v)
	}
	if v := os.Getenv("KNIT_SIEM_JSONL_PATH"); strings.TrimSpace(v) != "" {
		cfg.SIEMLogPath = strings.TrimSpace(v)
	}
	if v := os.Getenv("KNIT_AUDIO_RETENTION"); strings.TrimSpace(v) != "" {
		cfg.AudioRetention = parseDuration(v, cfg.AudioRetention)
	}
	if v := os.Getenv("KNIT_SCREENSHOT_RETENTION"); strings.TrimSpace(v) != "" {
		cfg.ScreenshotRetention = parseDuration(v, cfg.ScreenshotRetention)
	}
	if v := os.Getenv("KNIT_VIDEO_RETENTION"); strings.TrimSpace(v) != "" {
		cfg.VideoRetention = parseDuration(v, cfg.VideoRetention)
	}
	if v := os.Getenv("KNIT_TRANSCRIPT_RETENTION"); strings.TrimSpace(v) != "" {
		cfg.TranscriptRetention = parseDuration(v, cfg.TranscriptRetention)
	}
	if v := os.Getenv("KNIT_STRUCTURED_RETENTION"); strings.TrimSpace(v) != "" {
		cfg.StructuredRetention = parseDuration(v, cfg.StructuredRetention)
	}
	if v := os.Getenv("KNIT_PURGE_INTERVAL"); strings.TrimSpace(v) != "" {
		cfg.PurgeInterval = parseDuration(v, cfg.PurgeInterval)
	}
	if v := os.Getenv("KNIT_PURGE_SCHEDULE"); strings.EqualFold(strings.TrimSpace(v), "false") || strings.TrimSpace(v) == "0" {
		cfg.PurgeScheduleEnabled = false
	}
	if v := os.Getenv("KNIT_ARTIFACT_MAX_FILES"); strings.TrimSpace(v) != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			cfg.ArtifactMaxFiles = n
		}
	}
	return cfg
}

func New(cfg config.Config) (*App, error) {
	if err := security.VerifyStartupIntegrityFromEnv(); err != nil {
		return nil, fmt.Errorf("startup integrity verification failed: %w", err)
	}
	if err := config.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	if strings.TrimSpace(cfg.VersionPin) != "" && strings.TrimSpace(cfg.BuildID) != strings.TrimSpace(cfg.VersionPin) {
		return nil, fmt.Errorf("managed version pin mismatch: build %q does not match required %q", cfg.BuildID, cfg.VersionPin)
	}
	dataDir, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	sqlitePath := cfg.SQLitePath
	if !filepath.IsAbs(sqlitePath) {
		sqlitePath = filepath.Join(dataDir, sqlitePath)
	}
	cfg.SQLitePath = sqlitePath
	cfg.UserConfigPath = userconfig.ResolvePath(cfg)
	key, err := security.ResolveKey()
	if err != nil {
		return nil, fmt.Errorf("resolve encryption key: %w", err)
	}
	encryptor, err := security.NewEncryptor(key)
	if err != nil {
		return nil, fmt.Errorf("new encryptor: %w", err)
	}
	security.ZeroBytes(key)
	store, err := storage.NewSQLiteStore(sqlitePath, encryptor)
	if err != nil {
		return nil, fmt.Errorf("sqlite store: %w", err)
	}
	artifactStore, err := storage.NewArtifactStore(filepath.Join(dataDir, "artifacts"), encryptor)
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("artifact store: %w", err)
	}

	auditLogger, err := audit.NewLogger(dataDir, encryptor, cfg.SIEMLogPath)
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("audit logger: %w", err)
	}

	audioController := audio.NewController()
	loadedUserConfig, err := userconfig.Load(cfg, audioController.State())
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("load user config: %w", err)
	}
	cfg = loadedUserConfig.Config
	persisted, err := store.LoadOperatorState()
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("load persisted operator state: %w", err)
	}
	if persisted == nil {
		persisted = &operatorstate.State{Version: 1}
	}
	persisted.System = loadedUserConfig.State.System
	persisted.RuntimeCodex = loadedUserConfig.State.RuntimeCodex
	persisted.RuntimeTranscription = loadedUserConfig.State.RuntimeTranscription
	persisted.Audio = loadedUserConfig.State.Audio
	cfg = operatorstate.Apply(cfg, audioController, persisted)
	if err := config.Validate(cfg); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("invalid persisted config: %w", err)
	}
	if _, err := userconfig.Save(cfg, *persisted); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("write user config: %w", err)
	}

	sessions := session.NewService()
	history, err := store.ListSessions()
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("load persisted sessions: %w", err)
	}
	if len(history) > 0 {
		var approvedPkg *session.CanonicalPackage
		latest := history[0]
		if latest != nil && latest.Approved {
			approvedPkg, err = store.LoadLatestCanonicalPackage(latest.ID)
			if err != nil {
				_ = store.Close()
				return nil, fmt.Errorf("load latest canonical package: %w", err)
			}
		}
		sessions.Bootstrap(history, approvedPkg)
	}
	captureManager := capture.NewManager()
	pointerTracker := companion.NewTracker(360)
	privilegedCapture := privileged.NewCaptureBroker(captureManager, pointerTracker, audioController)
	transcriptionProvider := transcription.NewProviderFromEnv(cfg.TranscriptionMode)
	agentRegistry := agents.NewRegistry(
		agents.NewCodexAPIAdapterFromEnv(),
		agents.NewCLIAdapterFromEnv(),
		agents.NewClaudeCLIAdapterFromEnv(),
		agents.NewOpenCodeCLIAdapterFromEnv(),
	)
	execPath, err := os.Executable()
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	autoStart := platform.NewAutoStartManager("Knit", execPath, nil)
	if _, err := autoStart.Ensure(cfg.AutoStartEnabled); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("configure auto-start: %w", err)
	}
	httpServer := server.New(
		cfg,
		sessions,
		privilegedCapture,
		auditLogger,
		agentRegistry,
		store,
		artifactStore,
		autoStart,
		transcriptionProvider,
	)
	retentionWorker := retention.NewWorker(cfg, store, artifactStore, auditLogger)

	return &App{server: httpServer, store: store, retention: retentionWorker}, nil
}

func (a *App) Run(ctx context.Context) error {
	if a.retention != nil {
		go a.retention.Run(ctx)
	}
	defer func() { _ = a.store.Close() }()
	return a.server.Run(ctx)
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func generateControlToken() string {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("knit-%d", time.Now().UTC().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil || d < 0 {
		return fallback
	}
	return d
}

func parseInt(raw string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(raw))
}

func parseBool(raw string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(raw))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

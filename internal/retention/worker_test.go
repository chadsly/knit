package retention

import (
	"path/filepath"
	"testing"
	"time"

	"knit/internal/audit"
	"knit/internal/config"
	"knit/internal/security"
	"knit/internal/session"
	"knit/internal/storage"
)

func TestWorkerRunOncePurgesExpiredData(t *testing.T) {
	dir := t.TempDir()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	encryptor, err := security.NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	store, err := storage.NewSQLiteStore(filepath.Join(dir, "knit.db"), encryptor)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	artifactStore, err := storage.NewArtifactStore(filepath.Join(dir, "artifacts"), encryptor)
	if err != nil {
		t.Fatalf("new artifact store: %v", err)
	}
	auditLogger, err := audit.NewLogger(dir, encryptor, "")
	if err != nil {
		t.Fatalf("new audit logger: %v", err)
	}

	now := time.Now().UTC().Add(-2 * time.Second)
	sess := &session.Session{
		ID:               "sess-retention",
		TargetWindow:     "Browser",
		Status:           session.StatusStopped,
		CreatedAt:        now,
		UpdatedAt:        now,
		ApprovalRequired: true,
		Approved:         false,
	}
	if err := store.UpsertSession(sess); err != nil {
		t.Fatalf("upsert session: %v", err)
	}
	if err := store.SaveCanonicalPackage(&session.CanonicalPackage{SessionID: sess.ID, GeneratedAt: now}); err != nil {
		t.Fatalf("save package: %v", err)
	}
	if err := store.SaveSubmission(sess.ID, "cli", "run-1", "accepted", "ref-1", map[string]any{"ok": true}); err != nil {
		t.Fatalf("save submission: %v", err)
	}
	if _, err := artifactStore.Save("screenshot", sess.ID, []byte("a"), "png"); err != nil {
		t.Fatalf("save screenshot: %v", err)
	}
	if _, err := artifactStore.Save("clip", sess.ID, []byte("b"), "webm"); err != nil {
		t.Fatalf("save clip: %v", err)
	}
	if _, err := artifactStore.Save("audio", sess.ID, []byte("c"), "webm"); err != nil {
		t.Fatalf("save audio: %v", err)
	}

	cfg := config.Default()
	cfg.StructuredRetention = time.Nanosecond
	cfg.AudioRetention = time.Nanosecond
	cfg.ScreenshotRetention = time.Nanosecond
	cfg.VideoRetention = time.Nanosecond
	cfg.PurgeScheduleEnabled = true
	cfg.ArtifactMaxFiles = 10

	time.Sleep(5 * time.Millisecond)
	worker := NewWorker(cfg, store, artifactStore, auditLogger)
	worker.RunOnce()

	list, err := store.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected sessions to be purged, got %d", len(list))
	}
	entries, err := filepath.Glob(filepath.Join(dir, "artifacts", "*"))
	if err != nil {
		t.Fatalf("glob artifacts: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected artifacts to be purged, got %d", len(entries))
	}
}

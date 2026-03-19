package storage

import (
	"path/filepath"
	"testing"
	"time"

	"knit/internal/operatorstate"
	"knit/internal/security"
	"knit/internal/session"
)

func TestSQLiteStoreSessionPersistence(t *testing.T) {
	dir := t.TempDir()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	encryptor, err := security.NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	store, err := NewSQLiteStore(filepath.Join(dir, "test.db"), encryptor)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	sess := &session.Session{
		ID:               "sess-1",
		TargetWindow:     "Browser Preview",
		TargetURL:        "https://localhost:3000",
		Status:           session.StatusActive,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		ApprovalRequired: true,
		Approved:         false,
		Feedback:         []session.FeedbackEvt{{ID: "evt-1", RawTranscript: "make button bigger", NormalizedText: "Increase button size"}},
	}

	if err := store.UpsertSession(sess); err != nil {
		t.Fatalf("upsert session: %v", err)
	}
	if err := store.SaveCanonicalPackage(&session.CanonicalPackage{SessionID: sess.ID, GeneratedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("save package: %v", err)
	}
	if err := store.SaveSubmission(sess.ID, "codex_api", "run-1", "accepted", "ref-1", map[string]any{"ok": true}); err != nil {
		t.Fatalf("save submission: %v", err)
	}

	list, err := store.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}
	if list[0].ID != sess.ID {
		t.Fatalf("session mismatch: got %s want %s", list[0].ID, sess.ID)
	}

	if err := store.DeleteSessionByID(sess.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	list, err = store.ListSessions()
	if err != nil {
		t.Fatalf("list sessions after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no sessions after delete, got %d", len(list))
	}
}

func TestSQLiteStoreOperatorStateAndLatestPackagePersistence(t *testing.T) {
	dir := t.TempDir()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	encryptor, err := security.NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	store, err := NewSQLiteStore(filepath.Join(dir, "test.db"), encryptor)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	state := &operatorstate.State{
		Version: 1,
		RuntimeCodex: operatorstate.RuntimeCodex{
			DefaultProvider:    "codex_cli",
			CodexWorkdir:       "/tmp/repo",
			SubmitExecMode:     "series",
			CodexSkipRepoCheck: true,
		},
		RuntimeTranscription: operatorstate.RuntimeTranscription{
			Mode:        "faster_whisper",
			Model:       "small",
			Device:      "cpu",
			ComputeType: "int8",
		},
		Audio: operatorstate.Audio{
			Mode:          "always_on",
			InputDeviceID: "default",
			Muted:         false,
			Paused:        false,
			LevelMin:      0.02,
			LevelMax:      0.95,
		},
	}
	if err := store.SaveOperatorState(state); err != nil {
		t.Fatalf("save operator state: %v", err)
	}
	loadedState, err := store.LoadOperatorState()
	if err != nil {
		t.Fatalf("load operator state: %v", err)
	}
	if loadedState == nil || loadedState.RuntimeCodex.CodexWorkdir != "/tmp/repo" || loadedState.RuntimeTranscription.Model != "small" {
		t.Fatalf("unexpected operator state: %#v", loadedState)
	}

	first := &session.CanonicalPackage{SessionID: "sess-1", Summary: "first", GeneratedAt: time.Now().UTC().Add(-time.Minute)}
	second := &session.CanonicalPackage{SessionID: "sess-1", Summary: "second", GeneratedAt: time.Now().UTC()}
	if err := store.SaveCanonicalPackage(first); err != nil {
		t.Fatalf("save first package: %v", err)
	}
	if err := store.SaveCanonicalPackage(second); err != nil {
		t.Fatalf("save second package: %v", err)
	}
	latest, err := store.LoadLatestCanonicalPackage("sess-1")
	if err != nil {
		t.Fatalf("load latest canonical package: %v", err)
	}
	if latest == nil || latest.Summary != "second" {
		t.Fatalf("expected latest package summary second, got %#v", latest)
	}
}

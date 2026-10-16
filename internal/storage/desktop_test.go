package storage

import (
	"bytes"
	"knit/internal/desktop"
	"knit/internal/security"
	"knit/internal/session"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopQueueAndConsentSurviveSQLiteRestart(t *testing.T) {
	encryptor, err := security.NewEncryptor(bytes.Repeat([]byte{29}, 32))
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "restart.db")
	store, err := NewSQLiteStore(file, encryptor)
	if err != nil {
		t.Fatal(err)
	}
	svc := desktop.New(store)
	binding, err := svc.Connect("Saved conversation", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetAutomaticDelivery(binding.ID, true); err != nil {
		t.Fatal(err)
	}
	pkg := session.CanonicalPackage{SessionID: "review", Summary: "Fix button", ChangeRequests: []session.ChangeReq{{Summary: "Fix button"}}}
	delivery, err := svc.Queue(binding.ID, "stable-request", binding.Workspace, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = NewSQLiteStore(file, encryptor)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc = desktop.New(store)
	state, err := svc.Heartbeat(binding.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Bindings) != 1 || !state.Bindings[0].AutomaticDelivery || len(state.Deliveries) != 1 || state.Deliveries[0].Status != "queued" {
		t.Fatal("restart lost saved consent or queued input")
	}
	again, err := svc.Queue(binding.ID, "stable-request", binding.Workspace, pkg)
	if err != nil || again.ID != delivery.ID {
		t.Fatal("restart duplicated request", err)
	}
	if _, err := svc.Prepare(binding.ID, delivery.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.New(store).Prepare(binding.ID, delivery.ID); err == nil {
		t.Fatal("uncertain send replayed after reload")
	}
	if err := svc.Disconnect(binding.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.New(store).Heartbeat(binding.ID); err == nil {
		t.Fatal("revoked connection restored")
	}
}

func TestDesktopStateEncryptedDurableAndPurged(t *testing.T) {
	encryptor, err := security.NewEncryptor(bytes.Repeat([]byte{17}, 32))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "desktop.db")
	store, err := NewSQLiteStore(path, encryptor)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	raw := []byte(`{"deliveries":[{"summary":"private-feedback-marker"}]}`)
	if err := store.SaveDesktopState(raw); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := store.db.QueryRow(`SELECT state_json FROM operator_state WHERE id='desktop'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, "private-feedback-marker") {
		t.Fatal("feedback stored as plaintext")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = NewSQLiteStore(path, encryptor)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadDesktopState()
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("lost durable state: %s %v", got, err)
	}
	if _, err := store.db.Exec(`UPDATE operator_state SET state_json='plaintext' WHERE id='desktop'`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadDesktopState(); err == nil {
		t.Fatal("corrupt ciphertext accepted")
	}
	if err := store.PurgeAll(); err != nil {
		t.Fatal(err)
	}
	got, err = store.LoadDesktopState()
	if err != nil || len(got) != 0 {
		t.Fatal("purge retained desktop feedback")
	}
}

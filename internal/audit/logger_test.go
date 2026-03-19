package audit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"knit/internal/security"
)

func TestLoggerProducesTamperEvidentHashChain(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	encryptor, err := security.NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	dir := t.TempDir()
	logger, err := NewLogger(dir, encryptor, "")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	if err := logger.Write(Event{Timestamp: time.Now().UTC(), Type: "session_started", SessionID: "sess-1"}); err != nil {
		t.Fatalf("write first event: %v", err)
	}
	if err := logger.Write(Event{Timestamp: time.Now().UTC(), Type: "submission_sent", SessionID: "sess-1", Details: map[string]any{"provider": "cli"}}); err != nil {
		t.Fatalf("write second event: %v", err)
	}

	events := decodeAuditEvents(t, filepath.Join(dir, "audit.log.enc.jsonl"), encryptor)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].EventHash == "" || events[1].EventHash == "" {
		t.Fatalf("expected event hashes to be populated")
	}
	if events[1].PrevHash != events[0].EventHash {
		t.Fatalf("expected prev hash chain link, got prev=%q first=%q", events[1].PrevHash, events[0].EventHash)
	}

	if !verifyEventHash(events[0]) || !verifyEventHash(events[1]) {
		t.Fatalf("event hash verification failed")
	}
}

func TestLoggerExportsEventsAndMirrorsSIEMJSONL(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	encryptor, err := security.NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	dir := t.TempDir()
	logger, err := NewLogger(dir, encryptor, "siem/audit.jsonl")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	if err := logger.Write(Event{Timestamp: time.Now().UTC(), Type: "config_imported", Actor: "admin"}); err != nil {
		t.Fatalf("write event: %v", err)
	}
	if err := logger.Write(Event{Timestamp: time.Now().UTC(), Type: "session_started", SessionID: "sess-2"}); err != nil {
		t.Fatalf("write event: %v", err)
	}

	exported, err := logger.Export(1)
	if err != nil {
		t.Fatalf("export events: %v", err)
	}
	if len(exported) != 1 || exported[0].Type != "session_started" {
		t.Fatalf("expected last exported event, got %#v", exported)
	}

	siemPath := filepath.Join(dir, "siem", "audit.jsonl")
	raw, err := os.ReadFile(siemPath)
	if err != nil {
		t.Fatalf("read siem log: %v", err)
	}
	if !strings.Contains(string(raw), `"type":"config_imported"`) || !strings.Contains(string(raw), `"type":"session_started"`) {
		t.Fatalf("expected SIEM mirror to contain written events, got %s", string(raw))
	}
}

func decodeAuditEvents(t *testing.T, path string, encryptor *security.Encryptor) []Event {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open audit log: %v", err)
	}
	defer f.Close()

	out := []Event{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		plain, err := encryptor.Decrypt(line)
		if err != nil {
			t.Fatalf("decrypt line: %v", err)
		}
		var evt Event
		if err := json.Unmarshal(plain, &evt); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		out = append(out, evt)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan log: %v", err)
	}
	return out
}

func verifyEventHash(evt Event) bool {
	hashPayload := struct {
		Timestamp time.Time      `json:"timestamp"`
		Type      string         `json:"type"`
		SessionID string         `json:"session_id,omitempty"`
		Actor     string         `json:"actor,omitempty"`
		Details   map[string]any `json:"details,omitempty"`
		PrevHash  string         `json:"prev_hash,omitempty"`
	}{
		Timestamp: evt.Timestamp,
		Type:      evt.Type,
		SessionID: evt.SessionID,
		Actor:     evt.Actor,
		Details:   evt.Details,
		PrevHash:  evt.PrevHash,
	}
	b, err := json.Marshal(hashPayload)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(append([]byte(evt.PrevHash), b...))
	return evt.EventHash == hex.EncodeToString(sum[:])
}

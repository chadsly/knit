package audit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"knit/internal/security"
)

type Logger struct {
	mu        sync.Mutex
	path      string
	siemPath  string
	encryptor *security.Encryptor
	lastHash  string
}

type Event struct {
	Timestamp time.Time      `json:"timestamp"`
	Type      string         `json:"type"`
	SessionID string         `json:"session_id,omitempty"`
	Actor     string         `json:"actor,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
	PrevHash  string         `json:"prev_hash,omitempty"`
	EventHash string         `json:"event_hash,omitempty"`
}

func NewLogger(dataDir string, encryptor *security.Encryptor, siemPath string) (*Logger, error) {
	if encryptor == nil {
		return nil, os.ErrInvalid
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dataDir, "audit.log.enc.jsonl")
	if siemPath != "" {
		if !filepath.IsAbs(siemPath) {
			siemPath = filepath.Join(dataDir, siemPath)
		}
		if err := os.MkdirAll(filepath.Dir(siemPath), 0o700); err != nil {
			return nil, err
		}
	}
	lastHash, err := loadLastHash(path, encryptor)
	if err != nil {
		return nil, err
	}
	return &Logger{path: path, siemPath: siemPath, encryptor: encryptor, lastHash: lastHash}, nil
}

func (l *Logger) Write(evt Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	evt.PrevHash = l.lastHash
	evt.EventHash = ""
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
	hb, err := json.Marshal(hashPayload)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(append([]byte(evt.PrevHash), hb...))
	evt.EventHash = hex.EncodeToString(sum[:])
	b, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	ciphertext, err := l.encryptor.Encrypt(b)
	if err != nil {
		return err
	}
	if _, err := f.Write(append([]byte(ciphertext), '\n')); err != nil {
		return err
	}
	if l.siemPath != "" {
		if err := appendSIEMEvent(l.siemPath, evt); err != nil {
			return err
		}
	}
	l.lastHash = evt.EventHash
	return nil
}

func (l *Logger) Export(limit int) ([]Event, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return loadEvents(l.path, l.encryptor, limit)
}

func appendSIEMEvent(path string, evt Event) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

func loadEvents(path string, encryptor *security.Encryptor, limit int) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	capHint := 16
	if limit > capHint {
		capHint = limit
	}
	out := make([]Event, 0, capHint)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := stringTrimLine(scanner.Text())
		if line == "" {
			continue
		}
		plain, decErr := encryptor.Decrypt(line)
		if decErr != nil {
			continue
		}
		var evt Event
		if err := json.Unmarshal(plain, &evt); err != nil {
			continue
		}
		out = append(out, evt)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if limit > 0 && len(out) > limit {
		return append([]Event(nil), out[len(out)-limit:]...), nil
	}
	return out, nil
}

func loadLastHash(path string, encryptor *security.Encryptor) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()

	var last string
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		line = stringTrimLine(line)
		if line != "" {
			plain, decErr := encryptor.Decrypt(line)
			if decErr == nil {
				var evt Event
				if jsonErr := json.Unmarshal(plain, &evt); jsonErr == nil && evt.EventHash != "" {
					last = evt.EventHash
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
	}
	return last, nil
}

func stringTrimLine(v string) string {
	for len(v) > 0 && (v[len(v)-1] == '\n' || v[len(v)-1] == '\r') {
		v = v[:len(v)-1]
	}
	return v
}

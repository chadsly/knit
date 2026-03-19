package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"knit/internal/security"
)

type ArtifactStore struct {
	baseDir   string
	encryptor *security.Encryptor
}

func NewArtifactStore(baseDir string, encryptor *security.Encryptor) (*ArtifactStore, error) {
	if encryptor == nil {
		return nil, fmt.Errorf("encryptor is required")
	}
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("create artifact dir: %w", err)
	}
	return &ArtifactStore{baseDir: baseDir, encryptor: encryptor}, nil
}

func (s *ArtifactStore) Save(kind, sessionID string, payload []byte, ext string) (string, error) {
	if len(payload) == 0 {
		return "", fmt.Errorf("artifact payload is empty")
	}
	if sessionID == "" {
		sessionID = "unknown-session"
	}
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "bin"
	}
	ciphertext, err := s.encryptor.Encrypt(payload)
	if err != nil {
		return "", fmt.Errorf("encrypt artifact: %w", err)
	}
	filename := fmt.Sprintf("%s_%s_%d.%s.enc", kind, sessionID, time.Now().UTC().UnixNano(), ext)
	path := filepath.Join(s.baseDir, filename)
	if err := os.WriteFile(path, []byte(ciphertext), 0o600); err != nil {
		return "", fmt.Errorf("write artifact: %w", err)
	}
	return path, nil
}

func (s *ArtifactStore) Load(ref string) ([]byte, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("artifact ref is required")
	}
	absBase, err := filepath.Abs(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve artifact base dir: %w", err)
	}
	absRef, err := filepath.Abs(filepath.Clean(ref))
	if err != nil {
		return nil, fmt.Errorf("resolve artifact path: %w", err)
	}
	rel, err := filepath.Rel(absBase, absRef)
	if err != nil {
		return nil, fmt.Errorf("validate artifact path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("artifact path outside store")
	}
	ciphertext, err := os.ReadFile(absRef)
	if err != nil {
		return nil, fmt.Errorf("read artifact: %w", err)
	}
	plain, err := s.encryptor.Decrypt(string(ciphertext))
	if err != nil {
		return nil, fmt.Errorf("decrypt artifact: %w", err)
	}
	return plain, nil
}

func (s *ArtifactStore) PurgeOlderThan(kind string, cutoff time.Time) (int64, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return 0, fmt.Errorf("read artifact dir: %w", err)
	}
	var deleted int64
	prefix := ""
	if kind != "" {
		prefix = kind + "_"
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(s.baseDir, name)); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

func (s *ArtifactStore) PruneToLimit(maxFiles int) (int64, error) {
	if maxFiles <= 0 {
		return 0, nil
	}
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return 0, fmt.Errorf("read artifact dir: %w", err)
	}
	type fileInfo struct {
		name string
		mod  time.Time
	}
	files := make([]fileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: e.Name(), mod: info.ModTime()})
	}
	if len(files) <= maxFiles {
		return 0, nil
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })
	toDelete := files[:len(files)-maxFiles]
	var deleted int64
	for _, f := range toDelete {
		if err := os.Remove(filepath.Join(s.baseDir, f.name)); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

func (s *ArtifactStore) RemoveBySession(sessionID string) (int64, error) {
	if strings.TrimSpace(sessionID) == "" {
		return 0, nil
	}
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return 0, fmt.Errorf("read artifact dir: %w", err)
	}
	var deleted int64
	needle := "_" + sessionID + "_"
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.Contains(e.Name(), needle) {
			continue
		}
		if err := os.Remove(filepath.Join(s.baseDir, e.Name())); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

func (s *ArtifactStore) PurgeAll() (int64, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return 0, fmt.Errorf("read artifact dir: %w", err)
	}
	var deleted int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(s.baseDir, e.Name())); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

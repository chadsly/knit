package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// Desktop state uses its own encrypted record, separate from operator config.
func (s *SQLiteStore) SaveDesktopState(raw []byte) error {
	ciphertext, err := s.encryptor.Encrypt(raw)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO operator_state(id,updated_at,state_json) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET updated_at=excluded.updated_at,state_json=excluded.state_json`, "desktop", time.Now().UTC().Format(time.RFC3339Nano), ciphertext)
	return err
}

func (s *SQLiteStore) LoadDesktopState() ([]byte, error) {
	var ciphertext string
	err := s.db.QueryRow(`SELECT state_json FROM operator_state WHERE id=?`, "desktop").Scan(&ciphertext)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	raw, err := s.encryptor.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt desktop state: %w", err)
	}
	return raw, nil
}

package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"knit/internal/operatorstate"
	"knit/internal/security"
	"knit/internal/session"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db        *sql.DB
	encryptor *security.Encryptor
}

func NewSQLiteStore(path string, encryptor *security.Encryptor) (*SQLiteStore, error) {
	if encryptor == nil {
		return nil, fmt.Errorf("encryptor is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set wal mode: %w", err)
	}
	store := &SQLiteStore{db: db, encryptor: encryptor}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  approved INTEGER NOT NULL,
  approval_required INTEGER NOT NULL,
  target_window TEXT NOT NULL,
  target_url TEXT,
  version_reference TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  session_json TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS canonical_packages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  package_json TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS submissions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  run_id TEXT,
  status TEXT,
  ref TEXT,
  created_at TEXT NOT NULL,
  payload_json TEXT
);
CREATE TABLE IF NOT EXISTS operator_state (
  id TEXT PRIMARY KEY,
  updated_at TEXT NOT NULL,
  state_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_updated_at ON sessions(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_packages_session_id ON canonical_packages(session_id);
CREATE INDEX IF NOT EXISTS idx_submissions_session_id ON submissions(session_id);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate sqlite schema: %w", err)
	}
	return nil
}

func (s *SQLiteStore) UpsertSession(sess *session.Session) error {
	if sess == nil {
		return nil
	}
	b, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	ciphertext, err := s.encryptor.Encrypt(b)
	if err != nil {
		return fmt.Errorf("encrypt session payload: %w", err)
	}
	_, err = s.db.Exec(`
INSERT INTO sessions (id, status, approved, approval_required, target_window, target_url, version_reference, created_at, updated_at, session_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  status=excluded.status,
  approved=excluded.approved,
  approval_required=excluded.approval_required,
  target_window=excluded.target_window,
  target_url=excluded.target_url,
  version_reference=excluded.version_reference,
  updated_at=excluded.updated_at,
  session_json=excluded.session_json;
`,
		sess.ID,
		sess.Status,
		boolToInt(sess.Approved),
		boolToInt(sess.ApprovalRequired),
		sess.TargetWindow,
		sess.TargetURL,
		sess.VersionReference,
		sess.CreatedAt.Format(time.RFC3339Nano),
		sess.UpdatedAt.Format(time.RFC3339Nano),
		ciphertext,
	)
	if err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}
	return nil
}

func (s *SQLiteStore) SaveCanonicalPackage(pkg *session.CanonicalPackage) error {
	if pkg == nil {
		return nil
	}
	b, err := json.Marshal(pkg)
	if err != nil {
		return fmt.Errorf("marshal canonical package: %w", err)
	}
	ciphertext, err := s.encryptor.Encrypt(b)
	if err != nil {
		return fmt.Errorf("encrypt canonical package payload: %w", err)
	}
	_, err = s.db.Exec(`
INSERT INTO canonical_packages (session_id, created_at, package_json)
VALUES (?, ?, ?)
`, pkg.SessionID, time.Now().UTC().Format(time.RFC3339Nano), ciphertext)
	if err != nil {
		return fmt.Errorf("save canonical package: %w", err)
	}
	return nil
}

func (s *SQLiteStore) LoadLatestCanonicalPackage(sessionID string) (*session.CanonicalPackage, error) {
	if sessionID == "" {
		return nil, nil
	}
	row := s.db.QueryRow(`SELECT package_json FROM canonical_packages WHERE session_id = ? ORDER BY created_at DESC, id DESC LIMIT 1`, sessionID)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("load canonical package: %w", err)
	}
	var pkg session.CanonicalPackage
	decrypted, err := s.encryptor.Decrypt(payload)
	if err == nil {
		if err := json.Unmarshal(decrypted, &pkg); err != nil {
			return nil, fmt.Errorf("unmarshal canonical package: %w", err)
		}
		return &pkg, nil
	}
	if err := json.Unmarshal([]byte(payload), &pkg); err != nil {
		return nil, fmt.Errorf("decrypt/unmarshal canonical package payload: %w", err)
	}
	return &pkg, nil
}

func (s *SQLiteStore) SaveSubmission(sessionID, provider, runID, status, ref string, payload any) error {
	var payloadJSON string
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal submission payload: %w", err)
		}
		ciphertext, err := s.encryptor.Encrypt(b)
		if err != nil {
			return fmt.Errorf("encrypt submission payload: %w", err)
		}
		payloadJSON = ciphertext
	}
	_, err := s.db.Exec(`
INSERT INTO submissions (session_id, provider, run_id, status, ref, created_at, payload_json)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, sessionID, provider, runID, status, ref, time.Now().UTC().Format(time.RFC3339Nano), payloadJSON)
	if err != nil {
		return fmt.Errorf("save submission: %w", err)
	}
	return nil
}

func (s *SQLiteStore) SaveOperatorState(state *operatorstate.State) error {
	if state == nil {
		return nil
	}
	b, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal operator state: %w", err)
	}
	ciphertext, err := s.encryptor.Encrypt(b)
	if err != nil {
		return fmt.Errorf("encrypt operator state: %w", err)
	}
	_, err = s.db.Exec(`
INSERT INTO operator_state (id, updated_at, state_json)
VALUES (?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  updated_at=excluded.updated_at,
  state_json=excluded.state_json;
`, "shared", time.Now().UTC().Format(time.RFC3339Nano), ciphertext)
	if err != nil {
		return fmt.Errorf("save operator state: %w", err)
	}
	return nil
}

func (s *SQLiteStore) LoadOperatorState() (*operatorstate.State, error) {
	row := s.db.QueryRow(`SELECT state_json FROM operator_state WHERE id = ?`, "shared")
	var payload string
	if err := row.Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("load operator state: %w", err)
	}
	var state operatorstate.State
	decrypted, err := s.encryptor.Decrypt(payload)
	if err == nil {
		if err := json.Unmarshal(decrypted, &state); err != nil {
			return nil, fmt.Errorf("unmarshal operator state: %w", err)
		}
		return &state, nil
	}
	if err := json.Unmarshal([]byte(payload), &state); err != nil {
		return nil, fmt.Errorf("decrypt/unmarshal operator state payload: %w", err)
	}
	return &state, nil
}

func (s *SQLiteStore) ListSessions() ([]*session.Session, error) {
	rows, err := s.db.Query(`SELECT session_json FROM sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	out := []*session.Session{}
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		var sess session.Session
		decrypted, err := s.encryptor.Decrypt(payload)
		if err == nil {
			if err := json.Unmarshal(decrypted, &sess); err != nil {
				return nil, fmt.Errorf("unmarshal session: %w", err)
			}
		} else {
			// Backward compatibility for pre-encryption dev data.
			if err := json.Unmarshal([]byte(payload), &sess); err != nil {
				return nil, fmt.Errorf("decrypt/unmarshal session payload: %w", err)
			}
		}
		out = append(out, &sess)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) DeleteSessionByID(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	if _, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM canonical_packages WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete canonical packages by session: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM submissions WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete submissions by session: %w", err)
	}
	return nil
}

func (s *SQLiteStore) PurgeSessionsOlderThan(cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM sessions WHERE updated_at < ?`, cutoff.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("purge sessions: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge sessions rows affected: %w", err)
	}
	return n, nil
}

func (s *SQLiteStore) PurgeCanonicalPackagesOlderThan(cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM canonical_packages WHERE created_at < ?`, cutoff.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("purge canonical packages: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge canonical packages rows affected: %w", err)
	}
	return n, nil
}

func (s *SQLiteStore) PurgeSubmissionsOlderThan(cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM submissions WHERE created_at < ?`, cutoff.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("purge submissions: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge submissions rows affected: %w", err)
	}
	return n, nil
}

func (s *SQLiteStore) PurgeAll() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin purge all transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range []string{
		`DELETE FROM sessions`,
		`DELETE FROM canonical_packages`,
		`DELETE FROM submissions`,
		`DELETE FROM operator_state`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("purge all statement failed: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit purge all transaction: %w", err)
	}
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

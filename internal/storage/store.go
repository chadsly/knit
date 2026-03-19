package storage

import (
	"time"

	"knit/internal/operatorstate"
	"knit/internal/session"
)

type Store interface {
	UpsertSession(sess *session.Session) error
	LoadLatestCanonicalPackage(sessionID string) (*session.CanonicalPackage, error)
	SaveCanonicalPackage(pkg *session.CanonicalPackage) error
	SaveSubmission(sessionID, provider, runID, status, ref string, payload any) error
	SaveOperatorState(state *operatorstate.State) error
	LoadOperatorState() (*operatorstate.State, error)
	ListSessions() ([]*session.Session, error)
	DeleteSessionByID(sessionID string) error
	PurgeSessionsOlderThan(cutoff time.Time) (int64, error)
	PurgeCanonicalPackagesOlderThan(cutoff time.Time) (int64, error)
	PurgeSubmissionsOlderThan(cutoff time.Time) (int64, error)
	PurgeAll() error
	Close() error
}

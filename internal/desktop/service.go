// Package desktop owns durable delivery to explicitly connected desktop panels.
package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"knit/internal/session"
)

type Store interface {
	LoadDesktopState() ([]byte, error)
	SaveDesktopState([]byte) error
}

type Binding struct {
	ID                string    `json:"id"`
	Label             string    `json:"label"`
	Workspace         string    `json:"workspace"`
	CreatedAt         time.Time `json:"created_at"`
	LastSeenAt        time.Time `json:"last_seen_at"`
	Revoked           bool      `json:"revoked"`
	AutomaticDelivery bool      `json:"automatic_delivery"`
}

type Delivery struct {
	ID              string                   `json:"id"`
	BindingID       string                   `json:"binding_id"`
	Workspace       string                   `json:"workspace"`
	SessionID       string                   `json:"session_id"`
	ClientRequestID string                   `json:"client_request_id"`
	Status          string                   `json:"status"`
	Summary         string                   `json:"summary"`
	Result          string                   `json:"result,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
	Package         session.CanonicalPackage `json:"package"`
}

type Snapshot struct {
	Bindings   []Binding  `json:"bindings"`
	Deliveries []Delivery `json:"deliveries"`
}

type Service struct {
	mu    sync.Mutex
	store Store
}

func New(store Store) *Service { return &Service{store: store} }

func (s *Service) SetAutomaticDelivery(id string, enabled bool) (Binding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load()
	if err != nil {
		return Binding{}, err
	}
	b, err := findBinding(&state, id)
	if err != nil {
		return Binding{}, err
	}
	b.AutomaticDelivery = enabled
	return *b, s.save(state)
}

// CaptureReference authorizes a single capture from an already delivered request.
// Caller-supplied filesystem paths are never accepted.
func (s *Service) CaptureReference(bindingID, id string, index int) (session.ArtifactRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load()
	if err != nil {
		return session.ArtifactRef{}, err
	}
	if _, err = findBinding(&state, bindingID); err != nil {
		return session.ArtifactRef{}, err
	}
	for _, d := range state.Deliveries {
		if d.ID != id || d.BindingID != bindingID {
			continue
		}
		if d.Status != "delivered" && d.Status != "working" {
			return session.ArtifactRef{}, errors.New("Captures can only be read after delivery and before the run finishes.")
		}
		if index < 0 || index >= len(d.Package.Artifacts) {
			return session.ArtifactRef{}, errors.New("Capture not found in this request.")
		}
		return d.Package.Artifacts[index], nil
	}
	return session.ArtifactRef{}, errors.New("Request not found for this connection.")
}

// PurgeBefore applies the same structured-data retention window to feedback.
func (s *Service) PurgeBefore(cutoff time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load()
	if err != nil {
		return err
	}
	kept := state.Deliveries[:0]
	for _, d := range state.Deliveries {
		if !d.UpdatedAt.Before(cutoff) {
			kept = append(kept, d)
		}
	}
	if len(kept) == len(state.Deliveries) {
		return nil
	}
	state.Deliveries = kept
	return s.save(state)
}

func (s *Service) load() (Snapshot, error) {
	state := Snapshot{Bindings: []Binding{}, Deliveries: []Delivery{}}
	raw, err := s.store.LoadDesktopState()
	if err != nil {
		return state, err
	}
	if len(raw) > 0 {
		err = json.Unmarshal(raw, &state)
	}
	return state, err
}

func (s *Service) save(state Snapshot) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return s.store.SaveDesktopState(raw)
}

func (s *Service) Snapshot() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Service) Connect(label, workspace string) (Binding, error) {
	label = strings.TrimSpace(label)
	workspace = strings.TrimSpace(workspace)
	if label == "" || len(label) > 120 || strings.ContainsAny(label, "\r\n") {
		return Binding{}, errors.New("Choose a short conversation name.")
	}
	if !filepath.IsAbs(workspace) {
		return Binding{}, errors.New("Choose an absolute workspace directory.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load()
	if err != nil {
		return Binding{}, err
	}
	active := 0
	for _, b := range state.Bindings {
		if !b.Revoked {
			active++
		}
	}
	if active >= 20 {
		return Binding{}, errors.New("Disconnect an old conversation before connecting another.")
	}
	now := time.Now().UTC()
	b := Binding{ID: uuid.NewString(), Label: label, Workspace: filepath.Clean(workspace), CreatedAt: now, LastSeenAt: now}
	state.Bindings = append(state.Bindings, b)
	return b, s.save(state)
}

func findBinding(state *Snapshot, id string) (*Binding, error) {
	for i := range state.Bindings {
		if state.Bindings[i].ID == id && !state.Bindings[i].Revoked {
			return &state.Bindings[i], nil
		}
	}
	return nil, errors.New("This conversation is disconnected. Connect it again before sending.")
}

func (s *Service) Heartbeat(id string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load()
	if err != nil {
		return state, err
	}
	binding, err := findBinding(&state, id)
	if err != nil {
		return Snapshot{}, err
	}
	binding.LastSeenAt = time.Now().UTC()
	if err = s.save(state); err != nil {
		return Snapshot{}, err
	}
	filtered := Snapshot{Bindings: []Binding{*binding}, Deliveries: []Delivery{}}
	for _, d := range state.Deliveries {
		if d.BindingID == id {
			filtered.Deliveries = append(filtered.Deliveries, d)
		}
	}
	return filtered, nil
}

func (s *Service) Queue(bindingID, clientID, workspace string, pkg session.CanonicalPackage) (Delivery, error) {
	if strings.TrimSpace(clientID) == "" || len(clientID) > 100 {
		return Delivery{}, errors.New("A request identifier is required.")
	}
	if len(pkg.ChangeRequests) == 0 && len(pkg.Artifacts) == 0 {
		return Delivery{}, errors.New("Capture and approve at least one note first.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load()
	if err != nil {
		return Delivery{}, err
	}
	binding, err := findBinding(&state, bindingID)
	if err != nil {
		return Delivery{}, err
	}
	if !filepath.IsAbs(workspace) || filepath.Clean(workspace) != binding.Workspace {
		return Delivery{}, errors.New("Workspace mismatch. Select the workspace connected to this Codex panel.")
	}
	for _, d := range state.Deliveries {
		if d.ClientRequestID == clientID {
			if d.BindingID != bindingID {
				return Delivery{}, errors.New("Request identifier already belongs to another destination.")
			}
			return d, nil
		}
	}
	if len(state.Deliveries) >= 500 {
		return Delivery{}, errors.New("Desktop delivery history is full. Clear old test data before continuing.")
	}
	now := time.Now().UTC()
	d := Delivery{ID: uuid.NewString(), BindingID: bindingID, Workspace: binding.Workspace, SessionID: pkg.SessionID, ClientRequestID: clientID, Status: "queued", Summary: pkg.Summary, CreatedAt: now, UpdatedAt: now, Package: pkg}
	if d.Summary == "" && len(pkg.ChangeRequests) > 0 {
		d.Summary = pkg.ChangeRequests[0].Summary
	}
	state.Deliveries = append(state.Deliveries, d)
	return d, s.save(state)
}

// Prepare persists the uncertain sending state before exposing the payload.
// It never silently retries a delivery after a lost host acknowledgement.
func (s *Service) Prepare(bindingID, id string) (Delivery, error) {
	return s.transition(bindingID, id, "sending", "")
}

func (s *Service) Report(bindingID, id, status, result string) (Delivery, error) {
	if status != "delivered" && status != "working" && status != "completed" && status != "failed" && status != "needs_attention" {
		return Delivery{}, errors.New("Unsupported delivery status.")
	}
	if len(result) > 4000 {
		return Delivery{}, errors.New("Keep the result summary under 4000 characters.")
	}
	return s.transition(bindingID, id, status, result)
}

func (s *Service) transition(bindingID, id, next, result string) (Delivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load()
	if err != nil {
		return Delivery{}, err
	}
	if _, err = findBinding(&state, bindingID); err != nil {
		return Delivery{}, err
	}
	for i := range state.Deliveries {
		d := &state.Deliveries[i]
		if d.ID != id || d.BindingID != bindingID {
			continue
		}
		if d.Status == next && next != "sending" {
			return *d, nil
		}
		// The model can report progress before the host acknowledges the send.
		// A late acknowledgement must never downgrade that progress.
		if next == "delivered" && (d.Status == "working" || d.Status == "completed" || d.Status == "failed") {
			return *d, nil
		}
		valid := false
		switch next {
		case "sending":
			valid = d.Status == "queued"
		case "delivered":
			valid = d.Status == "sending"
		case "working":
			valid = d.Status == "delivered" || d.Status == "sending"
		case "completed":
			valid = (d.Status == "working" || d.Status == "delivered") && strings.TrimSpace(result) != ""
		case "failed":
			valid = d.Status == "delivered" || d.Status == "working"
		case "needs_attention":
			valid = d.Status == "sending"
		}
		if !valid {
			return Delivery{}, fmt.Errorf("Cannot change %s to %s. Uncertain deliveries need review; they are never resent automatically.", d.Status, next)
		}
		d.Status = next
		d.Result = strings.TrimSpace(result)
		d.UpdatedAt = time.Now().UTC()
		return *d, s.save(state)
	}
	return Delivery{}, errors.New("Delivery not found for this conversation.")
}

func (s *Service) Disconnect(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load()
	if err != nil {
		return err
	}
	b, err := findBinding(&state, id)
	if err != nil {
		return err
	}
	b.Revoked = true
	for i := range state.Deliveries {
		d := &state.Deliveries[i]
		if d.BindingID == id && (d.Status == "queued" || d.Status == "sending") {
			d.Status = "needs_attention"
			d.Result = "Conversation disconnected. Feedback has not been rerouted."
		}
	}
	return s.save(state)
}

func (s *Service) PurgeSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load()
	if err != nil {
		return err
	}
	kept := state.Deliveries[:0]
	for _, d := range state.Deliveries {
		if d.SessionID != id {
			kept = append(kept, d)
		}
	}
	state.Deliveries = kept
	return s.save(state)
}

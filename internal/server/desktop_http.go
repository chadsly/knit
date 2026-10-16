package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"knit/internal/audit"
	"knit/internal/desktop"
	"knit/internal/session"
)

// DesktopService shares the queue lock with the daemon retention worker.
func (s *Server) DesktopService() *desktop.Service { return s.desktop }

// Public desktop state contains routing/status only, not captured payloads.
func (s *Server) desktopPublicState() map[string]any {
	state, err := s.desktop.Snapshot()
	if err != nil {
		return map[string]any{"error": "Desktop connection state is unavailable."}
	}
	bindings := []desktop.Binding{}
	for _, b := range state.Bindings {
		if !b.Revoked {
			bindings = append(bindings, b)
		}
	}
	deliveries := []map[string]any{}
	for _, d := range state.Deliveries {
		deliveries = append(deliveries, map[string]any{"id": d.ID, "binding_id": d.BindingID, "status": d.Status, "summary": d.Summary, "result": d.Result})
	}
	return map[string]any{"bindings": bindings, "deliveries": deliveries, "delivery_mode": "panel_open_required", "supported_capture_kinds": []string{"screenshot", "audio", "video", "replay_json", "playwright_script"}}
}

func (s *Server) handleDesktopState(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(204)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(405)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	writeJSON(w, s.desktopPublicState())
}

func (s *Server) handleDesktopPanel(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(204)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req struct {
		Action             string `json:"action"`
		BindingID          string `json:"binding_id"`
		DeliveryID         string `json:"delivery_id"`
		Label              string `json:"label"`
		Workspace          string `json:"workspace"`
		Status             string `json:"status"`
		Result             string `json:"result"`
		ConfigureWorkspace bool   `json:"configure_workspace"`
		AutomaticDelivery  bool   `json:"automatic_delivery"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&req) != nil {
		writeJSONError(w, 400, "invalid_desktop_request", "Invalid desktop request.")
		return
	}
	var result any
	var err error
	switch req.Action {
	case "connect":
		if label := strings.TrimSpace(req.Label); label == "" || len(label) > 120 || strings.ContainsAny(label, "\r\n") {
			writeJSONError(w, 400, "desktop_label_required", "Choose a short conversation name.")
			return
		}
		if req.ConfigureWorkspace {
			if err := s.configureDesktopWorkspace(req.Workspace); err != nil {
				writeJSONError(w, 409, "desktop_workspace_setup_blocked", err.Error())
				return
			}
		}
		result, err = s.desktop.Connect(req.Label, req.Workspace)
	case "heartbeat":
		result, err = s.desktop.Heartbeat(req.BindingID)
	case "delivery_settings":
		result, err = s.desktop.SetAutomaticDelivery(req.BindingID, req.AutomaticDelivery)
	case "prepare":
		result, err = s.desktop.Prepare(req.BindingID, req.DeliveryID)
	case "report":
		result, err = s.desktop.Report(req.BindingID, req.DeliveryID, req.Status, req.Result)
	case "disconnect":
		err = s.desktop.Disconnect(req.BindingID)
		result = map[string]bool{"disconnected": err == nil}
	default:
		writeJSONError(w, 400, "invalid_desktop_action", "Unsupported desktop action.")
		return
	}
	if err != nil {
		writeJSONError(w, 409, "desktop_action_blocked", err.Error())
		return
	}
	if req.Action != "heartbeat" {
		s.auditWriteRequest(r, audit.Event{Type: "desktop_" + req.Action, Details: map[string]any{"binding_id": req.BindingID, "delivery_id": req.DeliveryID, "status": req.Status}})
	}
	writeJSON(w, map[string]any{"result": result})
}

func (s *Server) handleDesktopSubmit(w http.ResponseWriter, r *http.Request) {
	allowCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(204)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	if !s.requireAuth(w, r) {
		return
	}
	var req struct {
		BindingID string `json:"binding_id"`
		RequestID string `json:"request_id"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req) != nil || strings.TrimSpace(req.BindingID) == "" {
		writeJSONError(w, 400, "destination_required", "Choose a connected Codex panel.")
		return
	}
	s.desktopSubmitMu.Lock()
	defer s.desktopSubmitMu.Unlock()
	state, err := s.desktop.Snapshot()
	if err != nil {
		writeJSONError(w, 503, "desktop_unavailable", "Desktop queue is unavailable.")
		return
	}
	for _, d := range state.Deliveries {
		if d.ClientRequestID == req.RequestID && d.BindingID == req.BindingID {
			writeJSON(w, map[string]any{"delivery_id": d.ID, "status": d.Status, "execution_mode": "connected_session"})
			return
		}
	}
	pkg, err := s.sessions.ApprovedPackage()
	if err != nil || !canonicalPackageHasInput(pkg) {
		writeJSONError(w, 412, "approval_required", "Capture and approve feedback before sending to Codex.")
		return
	}
	var delivery desktop.Delivery
	_, err = s.sessions.ReserveApprovedPackageWith(func(current *session.CanonicalPackage) error {
		if !canonicalPackageHasInput(current) {
			return fmt.Errorf("Feedback changed. Review and approve it again.")
		}
		if err := s.validateDesktopCaptures(current.Artifacts); err != nil {
			return err
		}
		var queueErr error
		delivery, queueErr = s.desktop.Queue(req.BindingID, req.RequestID, s.currentRuntimeCodex().CodexWorkdir, redactPackageForTransmission(*current))
		return queueErr
	})
	if err != nil {
		writeJSONError(w, 409, "desktop_submission_blocked", err.Error())
		return
	}
	if curr := s.sessions.Current(); curr != nil {
		if err = s.store.UpsertSession(curr); err != nil {
			writeJSONError(w, 500, "desktop_session_save_failed", "Feedback is queued; refresh before sending more feedback.")
			return
		}
	}
	s.auditWriteRequest(r, audit.Event{Type: "desktop_feedback_queued", SessionID: pkg.SessionID, Details: map[string]any{"binding_id": delivery.BindingID, "delivery_id": delivery.ID}})
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]any{"delivery_id": delivery.ID, "status": delivery.Status, "execution_mode": "connected_session", "message": "Queued for your Codex panel. Open the connected panel to deliver."})
}

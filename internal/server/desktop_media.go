package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"knit/internal/audit"
	"knit/internal/capture"
	"knit/internal/session"
	"net/http"
	"strings"
	"unicode/utf8"
)

const desktopCaptureLimit = 10 * 1024 * 1024
const desktopRequestCaptureLimit = 24 * 1024 * 1024

func (s *Server) loadDesktopCapture(ref session.ArtifactRef) ([]byte, string, error) {
	if ref.Kind != "screenshot" && ref.Kind != "audio" && ref.Kind != "video" && ref.Kind != "replay_json" && ref.Kind != "playwright_script" {
		return nil, "", errors.New("This capture type is not supported.")
	}
	payload, err := s.artifacts.Load(ref.Ref)
	if err != nil {
		return nil, "", errors.New("A captured file is no longer available. Capture it again before submitting.")
	}
	if len(payload) > desktopCaptureLimit {
		return nil, "", errors.New("A capture exceeds the 10 MB desktop delivery limit. Remove it or record a shorter clip before submitting.")
	}
	mime := inferArtifactMIMEType(ref.Ref, payload)
	if ref.Kind == "replay_json" {
		if !json.Valid(payload) {
			return nil, "", errors.New("The replay context is invalid. Capture the note again.")
		}
		mime = "application/json"
	}
	if ref.Kind == "playwright_script" {
		if !utf8.Valid(payload) {
			return nil, "", errors.New("The replay script is not valid text. Capture the note again.")
		}
		mime = "text/plain"
	}
	if ref.Kind == "screenshot" {
		mime = http.DetectContentType(payload)
	}
	expected := map[string]string{"screenshot": "image/", "audio": "audio/", "video": "video/", "replay_json": "application/json", "playwright_script": "text/plain"}[ref.Kind]
	if !strings.HasPrefix(mime, expected) {
		return nil, "", errors.New("A capture's format does not match its type.")
	}
	// Do not expose active image formats through a model image response.
	if ref.Kind == "screenshot" && mime != "image/png" && mime != "image/jpeg" && mime != "image/webp" {
		return nil, "", errors.New("Use a PNG, JPEG, or WebP snapshot.")
	}
	return payload, mime, nil
}

func (s *Server) validateDesktopCaptures(refs []session.ArtifactRef) error {
	if len(refs) > 8 {
		return errors.New("Send at most eight captures in one review request.")
	}
	total := 0
	for _, ref := range refs {
		payload, _, err := s.loadDesktopCapture(ref)
		if err != nil {
			return err
		}
		total += len(payload)
		if total > desktopRequestCaptureLimit {
			return errors.New("Captures exceed the 24 MB request limit. Send fewer captures together.")
		}
	}
	return nil
}

func (s *Server) handleDesktopCapture(w http.ResponseWriter, r *http.Request) {
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
		BindingID   string `json:"binding_id"`
		DeliveryID  string `json:"delivery_id"`
		Index       int    `json:"capture_index"`
		VideoReview bool   `json:"video_review"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 2048)).Decode(&req) != nil {
		writeJSONError(w, 400, "desktop_capture_invalid", "Invalid capture request.")
		return
	}
	ref, err := s.desktop.CaptureReference(req.BindingID, req.DeliveryID, req.Index)
	if err != nil {
		writeJSONError(w, 409, "desktop_capture_blocked", err.Error())
		return
	}
	payload, mime, err := s.loadDesktopCapture(ref)
	if err != nil {
		writeJSONError(w, 409, "desktop_capture_unavailable", err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if ref.Kind == "video" && req.VideoReview {
		frames, err := capture.ReviewVideo(r.Context(), payload)
		if err != nil {
			writeJSONError(w, 409, "video_review_unavailable", err.Error())
			return
		}
		s.auditWriteRequest(r, audit.Event{Type: "desktop_video_review_read", Details: map[string]any{"binding_id": req.BindingID, "delivery_id": req.DeliveryID, "capture_index": req.Index, "frame_count": len(frames)}})
		writeJSON(w, map[string]any{"kind": "video_review", "frames": frames, "message": "Sampled video frames at approximate offsets, up to 25 seconds. Use the approved request transcript for narration. Original recording retained; unsampled motion and audio have not been inspected."})
		return
	}
	s.auditWriteRequest(r, audit.Event{Type: "desktop_capture_read", Details: map[string]any{"binding_id": req.BindingID, "delivery_id": req.DeliveryID, "capture_index": req.Index, "kind": ref.Kind}})
	writeJSON(w, map[string]any{"mime_type": mime, "kind": ref.Kind, "data": base64.StdEncoding.EncodeToString(payload)})
}

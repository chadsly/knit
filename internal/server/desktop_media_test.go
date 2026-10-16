package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"knit/internal/config"
	"knit/internal/session"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
)

func TestDesktopCapturesPersistAndRemainScoped(t *testing.T) {
	cfg := config.Default()
	cfg.ControlToken = "media-test"
	srv := newTestServer(t, cfg)
	workspace := t.TempDir()
	srv.runtime.RuntimeCodex.CodexWorkdir = workspace
	b, _ := srv.desktop.Connect("Shop", workspace)
	if kinds, ok := srv.desktopPublicState()["supported_capture_kinds"].([]string); !ok || len(kinds) != 5 {
		t.Fatal("desktop diagnostics must advertise replay capture support")
	}
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Wl6LsAAAAAASUVORK5CYII=")
	for _, kind := range []string{"screenshot", "audio", "video"} {
		t.Run(kind, func(t *testing.T) {
			payload := png
			ext := "png"
			if kind != "screenshot" {
				payload = []byte{0x1a, 0x45, 0xdf, 0xa3, 0x81, 0}
				ext = "webm"
			}
			videoDecoderAvailable := false
			if kind == "video" {
				if program, err := exec.LookPath("ffmpeg"); err == nil {
					payload, err = exec.Command(program, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:size=64x64:rate=5", "-t", "2", "-c:v", "libvpx", "-threads", "1", "-f", "webm", "pipe:1").Output()
					if err != nil {
						t.Fatal(err)
					}
					videoDecoderAvailable = true
				}
			}
			ref, err := srv.artifacts.Save(kind, "capture-session", payload, ext)
			if err != nil {
				t.Fatal(err)
			}
			srv.sessions.Start("Shop", "https://example.com")
			event := session.FeedbackEvt{ID: "capture-note", RawTranscript: "Review this capture", NormalizedText: "Review this capture"}
			event.Replay = &session.ReplayBundle{URL: "https://example.com", Steps: []session.ReplayStep{{Type: "click", TargetID: "primary-action"}}}
			switch kind {
			case "screenshot":
				event.ScreenshotRef = ref
			case "audio":
				event.AudioRef = ref
			case "video":
				event.VideoClipRef = ref
			}
			if _, err := srv.sessions.AddFeedback(event); err != nil {
				t.Fatal(err)
			}
			call := func(path string, body any, nonce string, want int) map[string]any {
				t.Helper()
				raw, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
				addAuth(req, cfg.ControlToken, true, kind+nonce)
				rec := httptest.NewRecorder()
				srv.httpSrv.Handler.ServeHTTP(rec, req)
				if rec.Code != want {
					t.Fatalf("%s %d: %s", path, rec.Code, rec.Body.String())
				}
				var data map[string]any
				_ = json.Unmarshal(rec.Body.Bytes(), &data)
				return data
			}
			call("/api/session/approve", map[string]any{"summary": "Review capture"}, "approve", 200)
			approved, err := srv.sessions.ApprovedPackage()
			if err != nil || len(approved.Artifacts) != 3 {
				t.Fatalf("expected media and two generated replay exports: %+v %v", approved, err)
			}
			queued := call("/api/desktop/submit", map[string]any{"binding_id": b.ID, "request_id": "capture-" + kind}, "submit", 202)
			id := queued["delivery_id"].(string)
			input := map[string]any{"binding_id": b.ID, "delivery_id": id, "capture_index": 0}
			call("/api/desktop/capture", input, "before", 409)
			if _, err := srv.desktop.Prepare(b.ID, id); err != nil {
				t.Fatal(err)
			}
			if _, err := srv.desktop.Report(b.ID, id, "working", ""); err != nil {
				t.Fatal(err)
			}
			data := call("/api/desktop/capture", input, "read", 200)
			decoded, err := base64.StdEncoding.DecodeString(data["data"].(string))
			if err != nil || !bytes.Equal(decoded, payload) {
				t.Fatal("capture lost in transport")
			}
			if kind == "video" {
				input["video_review"] = true
				if videoDecoderAvailable {
					review := call("/api/desktop/capture", input, "review", 200)
					frames := review["frames"].([]any)
					if review["kind"] != "video_review" || len(frames) == 0 || frames[0].(map[string]any)["mime_type"] != "image/jpeg" {
						t.Fatal("video review images missing")
					}
				} else {
					call("/api/desktop/capture", input, "review", 409)
				}
				delete(input, "video_review")
			}
			for index, mime := range []string{"application/json", "text/plain"} {
				input["capture_index"] = index + 1
				context := call("/api/desktop/capture", input, fmt.Sprintf("context-%d", index), 200)
				if context["mime_type"] != mime {
					t.Fatal("generated replay context not preserved")
				}
			}
			input["capture_index"] = 99
			call("/api/desktop/capture", input, "invalid", 409)
			if _, err := srv.desktop.Report(b.ID, id, "completed", "Reviewed capture transport."); err != nil {
				t.Fatal(err)
			}
		})
	}
	oversized, err := srv.artifacts.Save("video", "large", bytes.Repeat([]byte{1}, desktopCaptureLimit+1), "webm")
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.validateDesktopCaptures([]session.ArtifactRef{{Kind: "video", Ref: oversized}}); err == nil {
		t.Fatal("oversized capture accepted")
	}
	large, err := srv.artifacts.Save("video", "combined-limit", bytes.Repeat([]byte{1}, 9*1024*1024), "webm")
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.validateDesktopCaptures([]session.ArtifactRef{{Kind: "video", Ref: large}, {Kind: "video", Ref: large}, {Kind: "video", Ref: large}}); err == nil {
		t.Fatal("combined request size limit ignored")
	}
	refs := make([]session.ArtifactRef, 9)
	if err := srv.validateDesktopCaptures(refs); err == nil {
		t.Fatal("capture-count limit ignored")
	}
	req := httptest.NewRequest(http.MethodPost, "/api/desktop/capture", bytes.NewBufferString(fmt.Sprintf(`{"binding_id":%q}`, b.ID)))
	rec := httptest.NewRecorder()
	srv.httpSrv.Handler.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatal("unauthenticated capture access allowed")
	}
}

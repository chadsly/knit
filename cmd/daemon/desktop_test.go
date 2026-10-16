package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDesktopCaptureMCPContent(t *testing.T) {
	for _, mime := range []string{"image/png", "audio/webm", "video/webm", "application/json", "text/plain"} {
		t.Run(mime, func(t *testing.T) {
			payload := []byte("isolated-capture-fixture")
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/desktop/capture" || r.Header.Get("X-Knit-Token") != "test-token" {
					t.Error("wrong authenticated route")
					w.WriteHeader(401)
					return
				}
				var input desktopCaptureInput
				if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
					t.Error(err)
				}
				if input.BindingID != "binding" || input.DeliveryID != "delivery" || input.CaptureIndex != 2 {
					t.Error("capture identifiers lost")
				}
				if mime == "video/webm" {
					_ = json.NewEncoder(w).Encode(map[string]any{"kind": "video_review", "message": "Sampled frames; use request transcript", "frames": []map[string]any{{"timestamp_ms": 5000, "mime_type": "image/jpeg", "data": payload}}})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"mime_type": mime, "data": base64.StdEncoding.EncodeToString(payload)})
			}))
			defer backend.Close()
			implementation := &mcp.Implementation{Name: "capture-test", Version: "1"}
			server := mcp.NewServer(implementation, nil)
			addDesktopTools(server, &daemonClient{baseURL: backend.URL, token: "test-token", http: backend.Client()})
			st, ct := mcp.NewInMemoryTransports()
			ctx := context.Background()
			ss, err := server.Connect(ctx, st, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer ss.Close()
			cs, err := mcp.NewClient(implementation, nil).Connect(ctx, ct, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer cs.Close()
			resource, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "ui://knit/desktop-feedback"})
			if err != nil || len(resource.Contents) != 1 {
				t.Fatalf("desktop resource: %v", err)
			}
			metadata, _ := json.Marshal(resource.Contents[0].Meta)
			if !bytes.Contains(metadata, []byte(`"availableDisplayModes":["inline","pip","fullscreen"]`)) {
				t.Fatal("desktop resource must advertise floating presentation")
			}
			blocked, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "desktop_panel_action", Arguments: map[string]any{"action": "delivery_settings", "binding_id": "binding"}})
			if err != nil || !blocked.IsError {
				t.Fatal("model-facing action must not enable automatic delivery")
			}
			listing, err := cs.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			privateFound := false
			for _, tool := range listing.Tools {
				if tool.Name == "desktop_delivery_settings" {
					encoded, _ := json.Marshal(tool.Meta)
					if !bytes.Contains(encoded, []byte(`"visibility":["app"]`)) {
						t.Fatal("consent tool is not app-only")
					}
					privateFound = true
				}
			}
			if !privateFound {
				t.Fatal("missing app-only consent tool")
			}
			result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "desktop_read_capture", Arguments: map[string]any{"binding_id": "binding", "delivery_id": "delivery", "capture_index": 2}})
			if err != nil || result.IsError {
				t.Fatalf("invalid MCP result: %+v %v", result, err)
			}
			if mime == "video/webm" {
				if len(result.Content) != 3 {
					t.Fatal("video must return caveat, timestamp and native image")
				}
				image, ok := result.Content[2].(*mcp.ImageContent)
				if !ok || image.MIMEType != "image/jpeg" || !bytes.Equal(image.Data, payload) {
					t.Fatal("missing video frame image")
				}
				if result.Content[1].(*mcp.TextContent).Text != "Video frame near 5.0 seconds" {
					t.Fatal("missing frame timestamp")
				}
				return
			}
			if len(result.Content) != 1 {
				t.Fatal("unexpected capture content")
			}
			if mime == "image/png" {
				image, ok := result.Content[0].(*mcp.ImageContent)
				if !ok || image.MIMEType != mime || !bytes.Equal(image.Data, payload) {
					t.Fatal("image not returned as native MCP image")
				}
			} else if mime == "application/json" || mime == "text/plain" {
				resource, ok := result.Content[0].(*mcp.EmbeddedResource)
				if !ok || resource.Resource.Text != string(payload) || resource.Resource.MIMEType != mime {
					t.Fatal("replay context not returned as text resource")
				}
			} else {
				resource, ok := result.Content[0].(*mcp.EmbeddedResource)
				if !ok || resource.Resource.MIMEType != mime || !bytes.Equal(resource.Resource.Blob, payload) {
					t.Fatal("media not returned as embedded resource")
				}
			}
		})
	}
}

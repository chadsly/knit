package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"knit/internal/desktopui"
	"strings"
)

type desktopActionInput struct {
	Action             string `json:"action"`
	BindingID          string `json:"binding_id,omitempty"`
	DeliveryID         string `json:"delivery_id,omitempty"`
	Label              string `json:"label,omitempty"`
	Workspace          string `json:"workspace,omitempty"`
	Status             string `json:"status,omitempty"`
	Result             string `json:"result,omitempty"`
	ConfigureWorkspace bool   `json:"configure_workspace,omitempty"`
}

type desktopCaptureInput struct {
	BindingID    string `json:"binding_id"`
	DeliveryID   string `json:"delivery_id"`
	CaptureIndex int    `json:"capture_index"`
}

func addDesktopTools(server *mcp.Server, client *daemonClient) {
	mcp.AddTool(server, &mcp.Tool{Name: "desktop_delivery_settings", Description: "Save the user's automatic-delivery choice from the Knit connection UI.", Meta: mcp.Meta{"ui": map[string]any{"visibility": []string{"app"}}, "openai/visibility": "private", "openai/widgetAccessible": true}}, func(_ context.Context, _ *mcp.CallToolRequest, input struct {
		BindingID         string `json:"binding_id"`
		AutomaticDelivery bool   `json:"automatic_delivery"`
	}) (*mcp.CallToolResult, map[string]any, error) {
		data, err := client.postJSON("/api/desktop/panel", map[string]any{"action": "delivery_settings", "binding_id": input.BindingID, "automatic_delivery": input.AutomaticDelivery})
		return nil, data, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "desktop_read_capture", Description: "Read one approved capture for the delivered Knit request. Screenshots are images; audio and video are embedded media resources. If this host cannot interpret a resource, report that limitation; never claim to have reviewed it. Capture indexes are zero-based."}, func(_ context.Context, _ *mcp.CallToolRequest, input desktopCaptureInput) (*mcp.CallToolResult, map[string]any, error) {
		data, err := client.postJSON("/api/desktop/capture", map[string]any{"binding_id": input.BindingID, "delivery_id": input.DeliveryID, "capture_index": input.CaptureIndex, "video_review": true})
		if err != nil {
			return nil, nil, err
		}
		if anyString(data["kind"]) == "video_review" {
			var review struct {
				Frames []struct {
					TimestampMS int    `json:"timestamp_ms"`
					MIMEType    string `json:"mime_type"`
					Data        []byte `json:"data"`
				} `json:"frames"`
				Message string `json:"message"`
			}
			encoded, _ := json.Marshal(data)
			if err := json.Unmarshal(encoded, &review); err != nil || len(review.Frames) == 0 {
				return nil, nil, fmt.Errorf("invalid video review response")
			}
			contents := []mcp.Content{&mcp.TextContent{Text: review.Message}}
			for _, frame := range review.Frames {
				contents = append(contents, &mcp.TextContent{Text: fmt.Sprintf("Video frame near %.1f seconds", float64(frame.TimestampMS)/1000)}, &mcp.ImageContent{MIMEType: frame.MIMEType, Data: frame.Data})
			}
			return &mcp.CallToolResult{Content: contents}, nil, nil
		}
		payload, err := base64.StdEncoding.DecodeString(anyString(data["data"]))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid capture response")
		}
		mime := anyString(data["mime_type"])
		var content mcp.Content
		if strings.HasPrefix(mime, "image/") {
			content = &mcp.ImageContent{MIMEType: mime, Data: payload}
		} else if mime == "application/json" || mime == "text/plain" {
			content = &mcp.EmbeddedResource{Resource: &mcp.ResourceContents{URI: fmt.Sprintf("knit-capture://%s/%d", input.DeliveryID, input.CaptureIndex), MIMEType: mime, Text: string(payload)}}
		} else {
			content = &mcp.EmbeddedResource{Resource: &mcp.ResourceContents{URI: fmt.Sprintf("knit-capture://%s/%d", input.DeliveryID, input.CaptureIndex), MIMEType: mime, Blob: payload}}
		}
		return &mcp.CallToolResult{Content: []mcp.Content{content}}, nil, nil
	})
	const uri = "ui://knit/desktop-feedback"
	server.AddResource(&mcp.Resource{URI: uri, Name: "Knit browser feedback", MIMEType: "text/html;profile=mcp-app"}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: "text/html;profile=mcp-app", Text: desktopui.Panel, Meta: mcp.Meta{"openai/ui": map[string]any{"availableDisplayModes": []string{"inline", "pip", "fullscreen"}}}}}}, nil
	})
	mcp.AddTool(server, &mcp.Tool{Name: "open_desktop_feedback", Description: "Open Knit browser feedback in this conversation. User connects the panel explicitly; queued feedback never launches a CLI agent.", Meta: mcp.Meta{"ui": map[string]any{"resourceUri": uri}, "openai/outputTemplate": uri, "openai/ui": map[string]any{"entrypoints": []map[string]string{{"type": "thread"}}}}}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, map[string]any, error) {
		state, err := client.state()
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"workspace": anyMap(state["runtime_codex"])["codex_workdir"], "message": "Connect your browser to this conversation. Enable automatic delivery in the connection UI to forward approved feedback while the connection remains open. Captures are retrieved through desktop_read_capture; audio/video interpretation requires host support."}, nil
	})
	mcp.AddTool(server, &mcp.Tool{Name: "desktop_panel_action", Description: "Manage an explicitly connected Knit panel. Report working/completed/failed only for the binding_id and delivery_id supplied in the approved feedback. Include a factual result summary on completion. Never claim completion based on message delivery alone.", Meta: mcp.Meta{"openai/widgetAccessible": true}}, func(_ context.Context, _ *mcp.CallToolRequest, input desktopActionInput) (*mcp.CallToolResult, map[string]any, error) {
		if input.Action == "delivery_settings" {
			return nil, nil, fmt.Errorf("Automatic delivery requires the user's explicit choice in the connection UI")
		}
		if input.Action == "pair_browser" {
			data, err := client.postJSON("/api/extension/pair/start", map[string]any{"name": "Knit desktop browser", "browser": "chromium"})
			if err != nil {
				return nil, nil, err
			}
			return nil, map[string]any{"pairing_code": data["pairing_code"], "expires_at": data["expires_at"], "daemon_url": client.baseURL}, nil
		}
		data, err := client.postJSON("/api/desktop/panel", map[string]any{"action": input.Action, "binding_id": input.BindingID, "delivery_id": input.DeliveryID, "label": input.Label, "workspace": input.Workspace, "status": input.Status, "result": input.Result, "configure_workspace": input.ConfigureWorkspace})
		return nil, data, err
	})
}

package main

import (
	"context"
	_ "embed"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"io"
	"log/slog"
	"os"
)

//go:embed panel.html
var panel string

const panelURI = "ui://knit/desktop-connection-probe"

func newProbeServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "knit-desktop-probe", Version: "0.0.1"}, &mcp.ServerOptions{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	s.AddResource(&mcp.Resource{URI: panelURI, Name: "Knit connection check", MIMEType: "text/html;profile=mcp-app"}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: panelURI, MIMEType: "text/html;profile=mcp-app", Text: panel}}}, nil
	})
	mcp.AddTool(s, &mcp.Tool{Name: "open_connection_check", Description: "Open a Knit desktop connection check panel. Sends nothing until the user clicks the test button.", Meta: mcp.Meta{"ui": map[string]any{"resourceUri": panelURI}, "openai/outputTemplate": panelURI, "openai/ui": map[string]any{"entrypoints": []map[string]string{{"type": "thread"}}}}}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, map[string]any, error) {
		return nil, map[string]any{"status": "awaiting_user_test", "message": "Open the connection panel and select Send test message. A host acknowledgement does not prove agent completion."}, nil
	})
	return s
}

func main() {
	if err := newProbeServer().Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		os.Exit(1)
	}
}

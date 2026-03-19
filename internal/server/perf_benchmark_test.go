package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"knit/internal/config"
)

func BenchmarkServerPointerIngest(b *testing.B) {
	cfg := config.Default()
	cfg.ControlToken = "bench-token"
	srv := newTestServer(b, cfg)
	curr := srv.sessions.Start("Browser Preview", "https://example.com/app")
	sessionID := ""
	if curr != nil {
		sessionID = curr.ID
	}
	if sessionID == "" {
		b.Fatalf("expected benchmark session id")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := []byte(fmt.Sprintf(`{"session_id":"%s","x":%d,"y":%d,"event_type":"move","window":"Browser Preview","url":"https://example.com/app","route":"/app","timestamp":"2026-03-09T00:00:00Z"}`, sessionID, i%1024, (i*3)%768))
		req := httptest.NewRequest(http.MethodPost, "/api/companion/pointer", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuth(req, cfg.ControlToken, true, fmt.Sprintf("bench-pointer-%d", i))
		rec := httptest.NewRecorder()
		srv.httpSrv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("pointer ingest failed: %d %s", rec.Code, rec.Body.String())
		}
	}
}

func BenchmarkServerFeedbackCapture(b *testing.B) {
	cfg := config.Default()
	cfg.ControlToken = "bench-token"
	srv := newTestServer(b, cfg)
	_ = srv.sessions.Start("Browser Preview", "https://example.com/app")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := []byte(fmt.Sprintf(`{"raw_transcript":"feedback %d","normalized":"feedback %d","pointer_x":%d,"pointer_y":%d,"window":"Browser Preview"}`, i, i, i%640, i%480))
		req := httptest.NewRequest(http.MethodPost, "/api/session/feedback", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuth(req, cfg.ControlToken, true, fmt.Sprintf("bench-feedback-%d", i))
		rec := httptest.NewRecorder()
		srv.httpSrv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("feedback capture failed: %d %s", rec.Code, rec.Body.String())
		}
	}
}

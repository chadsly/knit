package server

import (
	"testing"

	"knit/internal/config"
	"knit/internal/session"
)

func TestEndpointPolicyRequiresSecureTransport(t *testing.T) {
	cfg := config.Default()
	if !endpointAllowedByPolicy("https://api.openai.com", cfg) {
		t.Fatalf("expected secure endpoint to be allowed with default policy")
	}
	if endpointAllowedByPolicy("http://example.com", cfg) {
		t.Fatalf("expected non-loopback http endpoint to be blocked")
	}
	if !endpointAllowedByPolicy("http://127.0.0.1:9999", cfg) {
		t.Fatalf("expected loopback http endpoint to be allowed")
	}
}

func TestRedactPackageForTransmission(t *testing.T) {
	pkg := session.CanonicalPackage{
		Summary: "token=abc123",
		ChangeRequests: []session.ChangeReq{
			{Summary: "password=mysecret"},
		},
	}
	out := redactPackageForTransmission(pkg)
	if out.Summary == pkg.Summary || out.ChangeRequests[0].Summary == pkg.ChangeRequests[0].Summary {
		t.Fatalf("expected redacted transmission package")
	}
}

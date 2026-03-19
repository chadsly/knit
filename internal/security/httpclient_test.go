package security

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParsePinnedSHA256SupportsHexAndBase64(t *testing.T) {
	sum := sha256.Sum256([]byte("abc"))
	hexRaw := hex.EncodeToString(sum[:])
	if got, err := parsePinnedSHA256(hexRaw); err != nil || !equalBytes(got, sum[:]) {
		t.Fatalf("hex parse failed: %v", err)
	}
	if got, err := parsePinnedSHA256("sha256:" + hexRaw); err != nil || !equalBytes(got, sum[:]) {
		t.Fatalf("sha256:hex parse failed: %v", err)
	}
}

func TestNewHTTPClientFromEnvPinnedCertMatch(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	leaf := mustLeafCert(t, srv)
	sum := sha256.Sum256(leaf.Raw)

	caFile := writeCertPEM(t, leaf)
	t.Setenv("KNIT_TLS_CA_FILE", caFile)
	t.Setenv("KNIT_TLS_PINNED_CERT_SHA256", hex.EncodeToString(sum[:]))

	client, err := NewHTTPClientFromEnv(5 * time.Second)
	if err != nil {
		t.Fatalf("build client: %v", err)
	}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("expected pinned cert request success, got %v", err)
	}
	_ = resp.Body.Close()
}

func TestNewHTTPClientFromEnvPinnedCertMismatchFails(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	leaf := mustLeafCert(t, srv)
	caFile := writeCertPEM(t, leaf)
	t.Setenv("KNIT_TLS_CA_FILE", caFile)
	t.Setenv("KNIT_TLS_PINNED_CERT_SHA256", strings.Repeat("aa", sha256.Size))

	client, err := NewHTTPClientFromEnv(5 * time.Second)
	if err != nil {
		t.Fatalf("build client: %v", err)
	}
	_, err = client.Get(srv.URL)
	if err == nil {
		t.Fatalf("expected tls pin mismatch error")
	}
}

func mustLeafCert(t *testing.T, srv *httptest.Server) *x509.Certificate {
	t.Helper()
	if len(srv.TLS.Certificates) == 0 || len(srv.TLS.Certificates[0].Certificate) == 0 {
		t.Fatalf("missing test tls cert chain")
	}
	leaf, err := x509.ParseCertificate(srv.TLS.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatalf("parse leaf cert: %v", err)
	}
	return leaf
}

func writeCertPEM(t *testing.T, cert *x509.Certificate) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ca.pem")
	block := &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}
	if err := os.WriteFile(p, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write ca pem: %v", err)
	}
	return p
}

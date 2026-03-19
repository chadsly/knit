package security

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// NewHTTPClientFromEnv builds an HTTP client with optional enterprise controls:
// - proxy (KNIT_HTTP_PROXY; falls back to environment proxy behavior)
// - custom CA trust bundle (KNIT_TLS_CA_FILE)
// - leaf cert pinning (KNIT_TLS_PINNED_CERT_SHA256, hex or base64)
func NewHTTPClientFromEnv(timeout time.Duration) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment

	if proxyURL := strings.TrimSpace(os.Getenv("KNIT_HTTP_PROXY")); proxyURL != "" {
		if err := os.Setenv("HTTPS_PROXY", proxyURL); err != nil {
			return nil, fmt.Errorf("set HTTPS_PROXY from KNIT_HTTP_PROXY: %w", err)
		}
		if err := os.Setenv("HTTP_PROXY", proxyURL); err != nil {
			return nil, fmt.Errorf("set HTTP_PROXY from KNIT_HTTP_PROXY: %w", err)
		}
	}
	if noProxy := strings.TrimSpace(os.Getenv("KNIT_NO_PROXY")); noProxy != "" {
		if err := os.Setenv("NO_PROXY", noProxy); err != nil {
			return nil, fmt.Errorf("set NO_PROXY from KNIT_NO_PROXY: %w", err)
		}
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}

	if caFile := strings.TrimSpace(os.Getenv("KNIT_TLS_CA_FILE")); caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read KNIT_TLS_CA_FILE: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("append custom CA certs failed")
		}
		tlsConfig.RootCAs = pool
	}

	if pinRaw := strings.TrimSpace(os.Getenv("KNIT_TLS_PINNED_CERT_SHA256")); pinRaw != "" {
		pin, err := parsePinnedSHA256(pinRaw)
		if err != nil {
			return nil, err
		}
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no peer certificate provided")
			}
			sum := sha256.Sum256(rawCerts[0])
			if !equalBytes(sum[:], pin) {
				return fmt.Errorf("tls pin mismatch")
			}
			return nil
		}
	}

	transport.TLSClientConfig = tlsConfig
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
	return client, nil
}

func parsePinnedSHA256(raw string) ([]byte, error) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "sha256:"))
	if raw == "" {
		return nil, fmt.Errorf("empty pinned cert hash")
	}
	if decoded, err := hex.DecodeString(strings.ToLower(raw)); err == nil && len(decoded) == sha256.Size {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) == sha256.Size {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(raw); err == nil && len(decoded) == sha256.Size {
		return decoded, nil
	}
	return nil, fmt.Errorf("invalid KNIT_TLS_PINNED_CERT_SHA256 format")
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var out byte
	for i := range a {
		out |= a[i] ^ b[i]
	}
	return out == 0
}

package server

import (
	"encoding/base64"
	"testing"
)

func TestCompressScreenshotReturnsCompressedImage(t *testing.T) {
	src, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO6L8WQAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode source png: %v", err)
	}
	out, ext := compressScreenshot(src)
	if len(out) == 0 {
		t.Fatalf("expected screenshot bytes")
	}
	if ext == "" {
		t.Fatalf("expected screenshot extension")
	}
}

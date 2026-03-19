package server

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func BenchmarkCompressScreenshot(b *testing.B) {
	payload := benchmarkScreenshotPayload(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if out, format := compressScreenshot(payload); len(out) == 0 || format == "" {
			b.Fatalf("expected compressed screenshot output")
		}
	}
}

func benchmarkScreenshotPayload(b *testing.B) []byte {
	b.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1280, 720))
	for y := 0; y < 720; y++ {
		for x := 0; x < 1280; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8((x * 17) % 255),
				G: uint8((y * 29) % 255),
				B: uint8((x + y) % 255),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		b.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

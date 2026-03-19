package server

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png"
)

func compressScreenshot(payload []byte) ([]byte, string) {
	if len(payload) == 0 {
		return payload, "png"
	}
	img, _, err := image.Decode(bytes.NewReader(payload))
	if err != nil {
		return payload, "png"
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 82}); err != nil {
		return payload, "png"
	}
	if out.Len() == 0 || out.Len() > len(payload) {
		return payload, "png"
	}
	return out.Bytes(), "jpg"
}

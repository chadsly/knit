package capture

import (
	"bytes"
	"context"
	"image/jpeg"
	"os/exec"
	"testing"
)

func TestVideoReviewRequiresFFmpeg(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := ReviewVideo(context.Background(), []byte("video")); err == nil {
		t.Fatal("missing decoder accepted")
	}
}

func TestVideoReviewRealWebM(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("FFmpeg not installed")
	}
	video, err := exec.Command(ffmpeg, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc=size=320x180:rate=5", "-t", "7", "-c:v", "libvpx", "-threads", "1", "-f", "webm", "pipe:1").Output()
	if err != nil {
		t.Fatal(err)
	}
	frames, err := ReviewVideo(context.Background(), video)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("expected samples near 0 and 5 seconds, got %d", len(frames))
	}
	for i, frame := range frames {
		if frame.TimestampMS != i*5000 || frame.MIMEType != "image/jpeg" {
			t.Fatal("lost timestamp or image type")
		}
		if _, err := jpeg.Decode(bytes.NewReader(frame.Data)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ReviewVideo(context.Background(), []byte("invalid")); err == nil {
		t.Fatal("corrupt video accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ReviewVideo(ctx, video); err == nil {
		t.Fatal("cancelled extraction accepted")
	}
}

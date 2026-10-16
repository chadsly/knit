package capture

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os/exec"
	"time"
)

type VideoFrame struct {
	TimestampMS int    `json:"timestamp_ms"`
	MIMEType    string `json:"mime_type"`
	Data        []byte `json:"data"`
}

type boundedImage struct{ bytes.Buffer }

func (b *boundedImage) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 12*1024*1024 {
		return 0, fmt.Errorf("frame exceeds image size limit")
	}
	return b.Buffer.Write(p)
}

// ReviewVideo samples only approved bytes, without plaintext files or network input.
// These are approximate offsets, not a claim to have inspected every frame.
func ReviewVideo(ctx context.Context, video []byte) ([]VideoFrame, error) {
	program, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("Video review requires FFmpeg on the Knit daemon host. Install FFmpeg or send a snapshot with your note")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var frames []VideoFrame
	var output boundedImage
		cmd := exec.CommandContext(ctx, program, "-hide_banner", "-loglevel", "error", "-nostdin", "-protocol_whitelist", "pipe", "-threads", "1", "-i", "pipe:0", "-frames:v", "6", "-an", "-vf", "fps=1/5:start_time=0:round=up,scale=960:540:force_original_aspect_ratio=decrease", "-threads", "1", "-f", "mpjpeg", "-boundary_tag", "knitframe", "-vcodec", "mjpeg", "pipe:1")
	cmd.Stdin = bytes.NewReader(video)
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Could not prepare video review frames. Send a shorter recording or a snapshot with your note: %w", err)
	}
	reader := multipart.NewReader(bytes.NewReader(output.Bytes()), "knitframe")
	for len(frames) < 6 {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Invalid video frame stream")
		}
		data, err := io.ReadAll(io.LimitReader(part, 2*1024*1024+1))
		if err != nil || len(data) > 2*1024*1024 {
			return nil, fmt.Errorf("Video frame exceeds the review size limit")
		}
		frames = append(frames, VideoFrame{TimestampMS: len(frames) * 5000, MIMEType: "image/jpeg", Data: data})
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("No video frames could be decoded. Send a snapshot with your note")
	}
	return frames, nil
}

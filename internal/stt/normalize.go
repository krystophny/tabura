package stt

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const ffmpegNormalizeTimeout = 25 * time.Second

// NormalizeForWhisper converts any incoming audio payload to a deterministic
// whisper-friendly format: mono 16k WAV.
func NormalizeForWhisper(mimeType string, data []byte) (string, []byte, error) {
	_ = NormalizeMimeType(mimeType)
	wav, err := transcodeToMono16kWAV(data)
	if err != nil {
		return "", nil, err
	}
	return "audio/wav", wav, nil
}

func transcodeToMono16kWAV(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("audio payload is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), ffmpegNormalizeTimeout)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-i", "pipe:0",
		"-ac", "1",
		"-ar", "16000",
		"-f", "wav",
		"pipe:1",
	)
	cmd.Stdin = bytes.NewReader(data)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(err.Error())
		}
		if msg == "" {
			msg = "unknown ffmpeg failure"
		}
		return nil, fmt.Errorf("ffmpeg normalize failed: %s", msg)
	}
	out := stdout.Bytes()
	if len(out) == 0 {
		return nil, fmt.Errorf("ffmpeg produced empty WAV output")
	}
	return out, nil
}

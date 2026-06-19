package aianalysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// videoProbe is the technical metadata ffprobe extracts from a video file.
type videoProbe struct {
	Container       string
	Codec           string
	Resolution      string
	DurationSeconds int
}

// ffmpegAvailable reports whether the ffmpeg binary is on PATH.
func ffmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func ffprobeAvailable() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}

// probeVideo runs ffprobe to extract container/codec/resolution/duration. It
// returns an error (degrading the analyzer to a partial result) when ffprobe is
// unavailable or the file cannot be read.
func probeVideo(ctx context.Context, path string) (videoProbe, error) {
	if path == "" {
		return videoProbe{}, fmt.Errorf("aianalysis: empty video path")
	}
	if !ffprobeAvailable() {
		return videoProbe{}, fmt.Errorf("aianalysis: ffprobe not available")
	}
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return videoProbe{}, fmt.Errorf("aianalysis: ffprobe: %w", err)
	}
	var doc struct {
		Format struct {
			FormatName string `json:"format_name"`
			Duration   string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		return videoProbe{}, fmt.Errorf("aianalysis: decode ffprobe: %w", err)
	}
	probe := videoProbe{Container: firstFormatName(doc.Format.FormatName)}
	if secs, err := strconv.ParseFloat(strings.TrimSpace(doc.Format.Duration), 64); err == nil {
		probe.DurationSeconds = int(secs)
	}
	for _, s := range doc.Streams {
		if s.CodecType == "video" {
			probe.Codec = s.CodecName
			if s.Width > 0 && s.Height > 0 {
				probe.Resolution = fmt.Sprintf("%dx%d", s.Width, s.Height)
			}
			break
		}
	}
	return probe, nil
}

func firstFormatName(name string) string {
	if name == "" {
		return ""
	}
	if idx := strings.IndexByte(name, ','); idx >= 0 {
		return name[:idx]
	}
	return name
}

// extractFrame grabs a single JPEG keyframe a few seconds into the video for
// vision captioning. Returns an error when ffmpeg is unavailable.
func extractFrame(ctx context.Context, path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("aianalysis: empty video path")
	}
	if !ffmpegAvailable() {
		return nil, fmt.Errorf("aianalysis: ffmpeg not available")
	}
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-ss", "3",
		"-i", path,
		"-frames:v", "1",
		"-vf", "scale=640:-1",
		"-f", "image2",
		"-vcodec", "mjpeg",
		"pipe:1",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("aianalysis: ffmpeg frame: %w", err)
	}
	return out.Bytes(), nil
}

// extractAudio rips the first minute of mono 16kHz WAV audio for transcription.
func extractAudio(ctx context.Context, path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("aianalysis: empty video path")
	}
	if !ffmpegAvailable() {
		return nil, fmt.Errorf("aianalysis: ffmpeg not available")
	}
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", path,
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		"-t", "60",
		"-f", "wav",
		"pipe:1",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("aianalysis: ffmpeg audio: %w", err)
	}
	return out.Bytes(), nil
}

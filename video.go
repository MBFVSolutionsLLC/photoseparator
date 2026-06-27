package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func videoThumbprint(path, tempDir string) (string, int, int, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", 0, 0, fmt.Errorf("ffmpeg is required for video thumbprints")
	}

	args := []string{"-hide_banner", "-loglevel", "error", "-y"}
	if duration, err := videoDuration(path); err == nil && duration > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", duration.Seconds()/2))
	}

	frame := filepath.Join(tempDir, fmt.Sprintf("photoseparator-frame-%d.jpg", time.Now().UnixNano()))
	defer os.Remove(frame)

	args = append(args, "-i", path, "-frames:v", "1", frame)
	cmd := exec.Command("ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", 0, 0, fmt.Errorf("extract middle frame: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return imageThumbprint(frame)
}

func externalImageThumbprint(path string, decodeErr error) (string, int, int, error) {
	frame := filepath.Join(os.TempDir(), fmt.Sprintf("photoseparator-image-%d.jpg", time.Now().UnixNano()))
	defer os.Remove(frame)

	if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-i", path, "-frames:v", "1", frame)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			return imageThumbprint(frame)
		}
	}

	if _, err := exec.LookPath("magick"); err == nil {
		cmd := exec.Command("magick", path, frame)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			return imageThumbprint(frame)
		}
	}

	return "", 0, 0, fmt.Errorf("decode image: %w; install ffmpeg or ImageMagick for this format", decodeErr)
}

func videoDuration(path string) (time.Duration, error) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return 0, err
	}
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

func mediaMetadataDate(path string) (time.Time, error) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return time.Time{}, err
	}
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "format_tags=creation_time:stream_tags=creation_time",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}
	for _, line := range strings.Fields(string(out)) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000000Z", "2006-01-02 15:04:05"} {
			if date, err := time.Parse(layout, line); err == nil {
				return date.Local(), nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("metadata date not found")
}

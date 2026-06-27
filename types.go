package main

import (
	"path/filepath"
	"strings"
	"time"
)

type MediaKind string

const (
	KindPhoto MediaKind = "photo"
	KindVideo MediaKind = "video"
)

type Options struct {
	RootDir      string
	PhotoOutDir  string
	VideoOutDir  string
	Workers      int
	Copy         bool
	DryRun       bool
	TempFrameDir string
}

type MediaFile struct {
	Path string
	Kind MediaKind
}

type ProcessedFile struct {
	MediaFile
	Date         time.Time
	Thumbprint   string
	Size         int64
	Width        int
	Height       int
	OriginalName string
	Err          error
}

func classify(path string) (MediaKind, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := photoExts[ext]; ok {
		return KindPhoto, true
	}
	if _, ok := videoExts[ext]; ok {
		return KindVideo, true
	}
	return "", false
}

var photoExts = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".bmp": {}, ".tif": {}, ".tiff": {},
	".webp": {}, ".heic": {}, ".heif": {},
}

var videoExts = map[string]struct{}{
	".mp4": {}, ".mov": {}, ".m4v": {}, ".avi": {}, ".mkv": {}, ".webm": {}, ".wmv": {},
	".3gp": {}, ".mts": {}, ".m2ts": {},
}

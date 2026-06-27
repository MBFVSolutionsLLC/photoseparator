package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildPlanSendsSmallerDuplicateToDuplicateFolder(t *testing.T) {
	date := time.Date(2024, 5, 17, 21, 30, 45, 0, time.Local)
	files := []ProcessedFile{
		{
			MediaFile:  MediaFile{Path: filepath.Join("root", "small.jpg"), Kind: KindPhoto},
			Date:       date,
			Thumbprint: "abc123",
			Size:       10,
		},
		{
			MediaFile:  MediaFile{Path: filepath.Join("root", "large.jpg"), Kind: KindPhoto},
			Date:       date,
			Thumbprint: "abc123",
			Size:       20,
		},
	}
	opts := Options{PhotoOutDir: filepath.Join("out", "photos"), VideoOutDir: filepath.Join("out", "videos")}

	plan := buildPlan(files, opts)
	if len(plan) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(plan))
	}

	actions := map[string]fileAction{}
	for _, action := range plan {
		actions[filepath.Base(action.Source)] = action
	}

	if actions["large.jpg"].Duplicate {
		t.Fatal("larger file should be kept as the primary copy")
	}
	if !actions["small.jpg"].Duplicate {
		t.Fatal("smaller file should be marked duplicate")
	}
	if !strings.Contains(actions["small.jpg"].Destination, string(filepath.Separator)+"duplicates"+string(filepath.Separator)) {
		t.Fatalf("duplicate destination missing duplicates folder: %s", actions["small.jpg"].Destination)
	}
}

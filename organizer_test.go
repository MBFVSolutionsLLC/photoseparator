package main

import (
	"os"
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

func TestRunReturnsErrorWhenFilesAreSkipped(t *testing.T) {
	root := t.TempDir()
	photoOut := filepath.Join(t.TempDir(), "photos")
	videoOut := filepath.Join(t.TempDir(), "videos")
	if err := os.WriteFile(filepath.Join(root, "broken.jpg"), []byte("not a jpeg"), 0600); err != nil {
		t.Fatal(err)
	}

	err := Run(Options{
		RootDir:      root,
		PhotoOutDir:  photoOut,
		VideoOutDir:  videoOut,
		Workers:      1,
		TempFrameDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected an error when a media file is skipped")
	}
}

func TestRunDoesNotRescanOutputDirectoriesInsideRoot(t *testing.T) {
	root := t.TempDir()
	photoOut := filepath.Join(root, "organized", "photos")
	videoOut := filepath.Join(root, "organized", "videos")
	for _, path := range []string{
		filepath.Join(root, "source.jpg"),
		filepath.Join(photoOut, "previous.jpg"),
		filepath.Join(videoOut, "previous.mp4"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("media"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	files, err := scanMedia(root, photoOut, videoOut)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || filepath.Base(files[0].Path) != "source.jpg" {
		t.Fatalf("expected only the source file to be scanned, got %+v", files)
	}
}

func TestScanDoesNotExcludeRootWhenOutputIsItsParent(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "source")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "photo.jpg")
	if err := os.WriteFile(source, []byte("media"), 0600); err != nil {
		t.Fatal(err)
	}

	files, err := scanMedia(root, parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != source {
		t.Fatalf("expected source file to be scanned, got %+v", files)
	}
}

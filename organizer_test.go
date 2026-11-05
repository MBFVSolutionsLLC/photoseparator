package main

import (
	"image"
	"image/png"
	"io"
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
	brokenFile := filepath.Join(root, "broken.jpg")
	if err := os.WriteFile(brokenFile, []byte("not a jpeg"), 0600); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() {
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
	})
	if !strings.Contains(output, "Done. Processed 0 files, skipped or failed 1 files.") {
		t.Fatalf("summary missing from output:\n%s", output)
	}
	if !strings.Contains(output, "Skipped or failed files:\n  "+brokenFile+": ") {
		t.Fatalf("failed file path and reason missing from output:\n%s", output)
	}
}

func TestRunWaitsForConfirmationBeforeMovingFiles(t *testing.T) {
	root := t.TempDir()
	photoOut := filepath.Join(t.TempDir(), "photos")
	videoOut := filepath.Join(t.TempDir(), "videos")
	source := filepath.Join(root, "IMG_20240517_213045.png")
	file, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	confirmed := false
	output := captureStdout(t, func() {
		err := Run(Options{
			RootDir:      root,
			PhotoOutDir:  photoOut,
			VideoOutDir:  videoOut,
			Workers:      1,
			TempFrameDir: t.TempDir(),
			Confirm: func() error {
				confirmed = true
				if _, err := os.Stat(source); err != nil {
					t.Fatalf("source should still exist before confirmation: %v", err)
				}
				if _, err := os.Stat(photoOut); !os.IsNotExist(err) {
					t.Fatalf("photo output should not exist before confirmation, got %v", err)
				}
				return nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	if !confirmed {
		t.Fatal("expected confirmation callback to be called")
	}
	if !strings.Contains(output, "Planned actions:\n  MOVE PHOTO: "+source+" -> "+photoOut) {
		t.Fatalf("planned source and destination missing from output:\n%s", output)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source should be moved after confirmation, got %v", err)
	}
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previousStdout := os.Stdout
	os.Stdout = writer
	defer func() {
		os.Stdout = previousStdout
	}()

	run()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	return string(output)
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

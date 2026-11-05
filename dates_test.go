package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDateFromWhatsAppStyleFilename(t *testing.T) {
	date, ok := dateFromString("IMG-20240517-WA0042.jpg")
	if !ok {
		t.Fatal("expected date from WhatsApp filename")
	}
	if got := date.Format("20060102_150405"); got != "20240517_000000" {
		t.Fatalf("unexpected date %s", got)
	}
}

func TestDateFromCameraFilename(t *testing.T) {
	date, ok := dateFromString("IMG_20240517_213045.jpg")
	if !ok {
		t.Fatal("expected date from camera filename")
	}
	if got := date.Format("20060102_150405"); got != "20240517_213045" {
		t.Fatalf("unexpected date %s", got)
	}
}

func TestInferDatePrefersEarlierFileTimeForPhotosAndVideos(t *testing.T) {
	root := t.TempDir()
	modified := time.Date(2020, time.November, 30, 22, 15, 44, 0, time.Local)
	for _, test := range []struct {
		name string
		kind MediaKind
	}{
		{name: "video", kind: KindVideo},
		{name: "photo", kind: KindPhoto},
	} {
		t.Run(test.name, func(t *testing.T) {
			extension := ".mov"
			if test.kind == KindPhoto {
				extension = ".jpg"
			}
			path := filepath.Join(root, "2023", "07", "04", "20230704_132155"+extension)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("media"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(path, modified, modified); err != nil {
				t.Fatal(err)
			}

			date := inferDate(path, test.kind)
			if date.Year() != 2020 || date.Month() != time.November || date.Day() != 30 {
				t.Fatalf("expected earlier file date 2020-11-30, got %s", date.Format("2006-01-02"))
			}
		})
	}
}

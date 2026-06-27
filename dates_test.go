package main

import "testing"

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

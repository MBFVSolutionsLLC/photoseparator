package main

import (
	"image"
	"image/color"
	"testing"
)

func TestPerceptualHashMatchesDifferentResolutions(t *testing.T) {
	large := blockImage(400, 300)
	small := blockImage(80, 60)

	largeHash := perceptualHash(large)
	smallHash := perceptualHash(small)

	if largeHash != smallHash {
		t.Fatalf("expected matching hashes, got %s and %s", largeHash, smallHash)
	}
}

func blockImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			shade := uint8(30)
			switch {
			case x > width/2 && y <= height/2:
				shade = 90
			case x <= width/2 && y > height/2:
				shade = 160
			case x > width/2 && y > height/2:
				shade = 230
			}
			img.SetRGBA(x, y, color.RGBA{
				R: shade,
				G: shade,
				B: shade,
				A: 255,
			})
		}
	}
	return img
}

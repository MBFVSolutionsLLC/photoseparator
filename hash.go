package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"strings"

	"image/gif"
	"image/jpeg"
	"image/png"
)

const hashSize = 8
const sampleSize = 32

func init() {
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("png", "\x89PNG\r\n\x1a\n", png.Decode, png.DecodeConfig)
	image.RegisterFormat("gif", "GIF8?a", gif.Decode, gif.DecodeConfig)
}

func imageThumbprint(path string) (string, int, int, error) {
	if strings.EqualFold(filepath.Ext(path), ".bmp") {
		img, err := decodeBMP(path)
		if err != nil {
			return "", 0, 0, err
		}
		bounds := img.Bounds()
		return perceptualHash(img), bounds.Dx(), bounds.Dy(), nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", 0, 0, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return externalImageThumbprint(path, err)
	}
	bounds := img.Bounds()
	return perceptualHash(img), bounds.Dx(), bounds.Dy(), nil
}

func perceptualHash(img image.Image) string {
	gray := resizeGray(img, sampleSize, sampleSize)
	dct := dct2D(gray)

	values := make([]float64, 0, hashSize*hashSize-1)
	for y := 0; y < hashSize; y++ {
		for x := 0; x < hashSize; x++ {
			if x == 0 && y == 0 {
				continue
			}
			values = append(values, dct[y][x])
		}
	}
	median := medianFloat(values)

	var bits uint64
	var bit uint
	for y := 0; y < hashSize; y++ {
		for x := 0; x < hashSize; x++ {
			if x == 0 && y == 0 {
				continue
			}
			if dct[y][x] > median {
				bits |= 1 << bit
			}
			bit++
		}
	}

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, bits)
	return fmt.Sprintf("%016x", binary.BigEndian.Uint64(buf))
}

func resizeGray(img image.Image, width, height int) [][]float64 {
	src := img.Bounds()
	out := make([][]float64, height)
	for y := range out {
		out[y] = make([]float64, width)
		srcY := src.Min.Y + y*src.Dy()/height
		for x := 0; x < width; x++ {
			srcX := src.Min.X + x*src.Dx()/width
			out[y][x] = luminance(img.At(srcX, srcY))
		}
	}
	return out
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
}

func dct2D(input [][]float64) [][]float64 {
	out := make([][]float64, sampleSize)
	for u := 0; u < sampleSize; u++ {
		out[u] = make([]float64, sampleSize)
		for v := 0; v < sampleSize; v++ {
			sum := 0.0
			for y := 0; y < sampleSize; y++ {
				for x := 0; x < sampleSize; x++ {
					sum += input[y][x] *
						math.Cos((float64(2*x+1)*float64(v)*math.Pi)/(2*sampleSize)) *
						math.Cos((float64(2*y+1)*float64(u)*math.Pi)/(2*sampleSize))
				}
			}
			out[u][v] = alpha(u) * alpha(v) * sum
		}
	}
	return out
}

func alpha(n int) float64 {
	if n == 0 {
		return 1 / math.Sqrt(float64(sampleSize))
	}
	return math.Sqrt(2 / float64(sampleSize))
}

func medianFloat(values []float64) float64 {
	sorted := append([]float64(nil), values...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	return sorted[len(sorted)/2]
}

func normalizedExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jpeg" {
		return ".jpg"
	}
	return ext
}

func decodeBMP(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 54 || string(data[:2]) != "BM" {
		return nil, fmt.Errorf("decode bmp: invalid bitmap")
	}

	pixelOffset := int(binary.LittleEndian.Uint32(data[10:14]))
	headerSize := int(binary.LittleEndian.Uint32(data[14:18]))
	if headerSize < 40 || len(data) < 14+headerSize {
		return nil, fmt.Errorf("decode bmp: unsupported DIB header")
	}

	width := int(int32(binary.LittleEndian.Uint32(data[18:22])))
	rawHeight := int(int32(binary.LittleEndian.Uint32(data[22:26])))
	planes := binary.LittleEndian.Uint16(data[26:28])
	bitsPerPixel := binary.LittleEndian.Uint16(data[28:30])
	compression := binary.LittleEndian.Uint32(data[30:34])
	if width <= 0 || rawHeight == 0 || planes != 1 || compression != 0 {
		return nil, fmt.Errorf("decode bmp: only uncompressed BMP files are supported")
	}
	if bitsPerPixel != 24 && bitsPerPixel != 32 {
		return nil, fmt.Errorf("decode bmp: only 24-bit and 32-bit BMP files are supported")
	}
	if pixelOffset < 0 || pixelOffset >= len(data) {
		return nil, fmt.Errorf("decode bmp: bad pixel offset")
	}

	height := rawHeight
	topDown := false
	if height < 0 {
		height = -height
		topDown = true
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.Black, image.Point{}, draw.Src)

	bytesPerPixel := int(bitsPerPixel / 8)
	rowStride := ((width*bytesPerPixel + 3) / 4) * 4
	for y := 0; y < height; y++ {
		sourceY := y
		if !topDown {
			sourceY = height - 1 - y
		}
		row := pixelOffset + sourceY*rowStride
		if row+width*bytesPerPixel > len(data) {
			return nil, fmt.Errorf("decode bmp: truncated pixel data")
		}
		for x := 0; x < width; x++ {
			p := row + x*bytesPerPixel
			b, g, r := data[p], data[p+1], data[p+2]
			a := byte(255)
			if bytesPerPixel == 4 && data[p+3] != 0 {
				a = data[p+3]
			}
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
	return img, nil
}

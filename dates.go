package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var filenameDatePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:IMG|VID|PXL|WA|Screenshot|Scan|Photo|Image)[-_ ]?(\d{4})(\d{2})(\d{2})[-_ ]?(\d{2})(\d{2})(\d{2})`),
	regexp.MustCompile(`(?i)(\d{4})(\d{2})(\d{2})[-_ ]?(\d{2})(\d{2})(\d{2})`),
	regexp.MustCompile(`(?i)(\d{4})[-_](\d{2})[-_](\d{2})[-_ ](\d{2})[-_](\d{2})[-_](\d{2})`),
	regexp.MustCompile(`(?i)(\d{4})[-_](\d{2})[-_](\d{2})`),
	regexp.MustCompile(`(?i)(\d{4})(\d{2})(\d{2})`),
}

func inferDate(path string, kind MediaKind) time.Time {
	candidates := make([]time.Time, 0, 4)

	if kind == KindPhoto {
		if exifDate, err := jpegExifDate(path); err == nil && !exifDate.IsZero() {
			candidates = append(candidates, exifDate)
		}
	}
	if metadataDate, err := mediaMetadataDate(path); err == nil && !metadataDate.IsZero() {
		candidates = append(candidates, metadataDate)
	}
	if nameDate, ok := dateFromString(filepath.Base(path)); ok {
		candidates = append(candidates, nameDate)
	}
	if folderDate, ok := dateFromString(filepath.Dir(path)); ok {
		candidates = append(candidates, folderDate)
	}

	if info, err := os.Stat(path); err == nil {
		candidates = append(candidates, olderFileTime(info))
	}

	if len(candidates) > 0 {
		return chooseBestDate(candidates)
	}
	return time.Now()
}

// olderFileTime returns the earlier of the created and modified times.
func olderFileTime(info os.FileInfo) time.Time {
	date := info.ModTime()
	if created, ok := fileCreationTime(info); ok && created.Year() >= 1850 && created.Before(date) {
		return created
	}
	return date
}

func chooseBestDate(candidates []time.Time) time.Time {
	now := time.Now().Add(24 * time.Hour)
	var best time.Time
	for _, candidate := range candidates {
		if candidate.Year() >= 1850 && candidate.Before(now) && (best.IsZero() || candidate.Before(best)) {
			best = candidate
		}
	}
	if !best.IsZero() {
		for _, candidate := range candidates {
			if candidate.Year() < 1850 || !candidate.Before(now) ||
				candidate.Year() != best.Year() || candidate.YearDay() != best.YearDay() {
				continue
			}
			candidateHasTime := candidate.Hour() != 0 || candidate.Minute() != 0 ||
				candidate.Second() != 0 || candidate.Nanosecond() != 0
			bestHasTime := best.Hour() != 0 || best.Minute() != 0 ||
				best.Second() != 0 || best.Nanosecond() != 0
			if candidateHasTime && (!bestHasTime || candidate.Before(best)) {
				best = candidate
			}
		}
		return best
	}
	return candidates[0]
}

func dateFromString(value string) (time.Time, bool) {
	for _, pattern := range filenameDatePatterns {
		match := pattern.FindStringSubmatch(value)
		if len(match) == 0 {
			continue
		}
		year, _ := strconv.Atoi(match[1])
		month, _ := strconv.Atoi(match[2])
		day, _ := strconv.Atoi(match[3])
		hour, minute, second := 0, 0, 0
		if len(match) >= 7 {
			hour, _ = strconv.Atoi(match[4])
			minute, _ = strconv.Atoi(match[5])
			second, _ = strconv.Atoi(match[6])
		}
		date := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
		if date.Year() == year && int(date.Month()) == month && date.Day() == day {
			return date, true
		}
	}
	return time.Time{}, false
}

func jpegExifDate(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return time.Time{}, fmt.Errorf("not jpeg")
	}

	for offset := 2; offset+4 < len(data); {
		if data[offset] != 0xff {
			break
		}
		marker := data[offset+1]
		offset += 2
		if marker == 0xda || marker == 0xd9 {
			break
		}
		if offset+2 > len(data) {
			break
		}
		size := int(binary.BigEndian.Uint16(data[offset : offset+2]))
		if size < 2 || offset+size > len(data) {
			break
		}
		segment := data[offset+2 : offset+size]
		if marker == 0xe1 && bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			return parseExifTIFF(segment[6:])
		}
		offset += size
	}
	return time.Time{}, fmt.Errorf("exif not found")
}

func parseExifTIFF(tiff []byte) (time.Time, error) {
	if len(tiff) < 8 {
		return time.Time{}, fmt.Errorf("short tiff")
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return time.Time{}, fmt.Errorf("unknown byte order")
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return time.Time{}, fmt.Errorf("bad tiff magic")
	}
	firstIFD := int(order.Uint32(tiff[4:8]))
	tags, err := readIFD(tiff, firstIFD, order)
	if err != nil {
		return time.Time{}, err
	}

	for _, tag := range []uint16{0x9003, 0x9004, 0x0132} {
		if raw, ok := tags[tag]; ok {
			if date, ok := parseExifDateString(raw); ok {
				return date, nil
			}
		}
	}

	for _, pointerTag := range []uint16{0x8769, 0x8825} {
		raw, ok := tags[pointerTag]
		if !ok || len(raw) < 4 {
			continue
		}
		offset := int(order.Uint32(raw[:4]))
		nested, err := readIFD(tiff, offset, order)
		if err != nil {
			continue
		}
		for _, tag := range []uint16{0x9003, 0x9004, 0x0132} {
			if raw, ok := nested[tag]; ok {
				if date, ok := parseExifDateString(raw); ok {
					return date, nil
				}
			}
		}
	}

	return time.Time{}, fmt.Errorf("date tags not found")
}

func readIFD(tiff []byte, offset int, order binary.ByteOrder) (map[uint16][]byte, error) {
	if offset < 0 || offset+2 > len(tiff) {
		return nil, fmt.Errorf("bad ifd offset")
	}
	count := int(order.Uint16(tiff[offset : offset+2]))
	pos := offset + 2
	tags := make(map[uint16][]byte, count)
	for i := 0; i < count; i++ {
		if pos+12 > len(tiff) {
			return tags, nil
		}
		entry := tiff[pos : pos+12]
		tag := order.Uint16(entry[0:2])
		fieldType := order.Uint16(entry[2:4])
		fieldCount := int(order.Uint32(entry[4:8]))
		valueLen := fieldCount * exifTypeSize(fieldType)
		if valueLen <= 4 {
			tags[tag] = append([]byte(nil), entry[8:8+max(0, valueLen)]...)
		} else {
			valueOffset := int(order.Uint32(entry[8:12]))
			if valueOffset >= 0 && valueOffset+valueLen <= len(tiff) {
				tags[tag] = append([]byte(nil), tiff[valueOffset:valueOffset+valueLen]...)
			}
		}
		pos += 12
	}
	return tags, nil
}

func exifTypeSize(fieldType uint16) int {
	switch fieldType {
	case 1, 2, 7:
		return 1
	case 3:
		return 2
	case 4, 9:
		return 4
	case 5, 10:
		return 8
	default:
		return 1
	}
}

func parseExifDateString(raw []byte) (time.Time, bool) {
	text := strings.TrimRight(string(raw), "\x00 ")
	for _, layout := range []string{"2006:01:02 15:04:05", "2006-01-02 15:04:05"} {
		if date, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return date, true
		}
	}
	return time.Time{}, false
}

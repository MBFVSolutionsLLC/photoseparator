package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func scanMedia(root string, excludedDirs ...string) ([]MediaFile, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root directory: %w", err)
	}
	exclusions := make([]string, 0, len(excludedDirs))
	for _, dir := range excludedDirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("resolve excluded directory: %w", err)
		}
		absDir = filepath.Clean(absDir)
		if samePath(absDir, root) {
			return nil, fmt.Errorf("output directory cannot be the scan root")
		}
		if isPathWithin(absDir, root) {
			exclusions = append(exclusions, absDir)
		}
	}

	var files []MediaFile
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			for _, excludedDir := range exclusions {
				if isPathWithin(path, excludedDir) {
					return fs.SkipDir
				}
			}
			return nil
		}
		kind, ok := classify(path)
		if !ok {
			return nil
		}
		files = append(files, MediaFile{Path: path, Kind: kind})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan media: %w", err)
	}
	return files, nil
}

func isPathWithin(path, parent string) bool {
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
		parent = strings.ToLower(parent)
	}
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func samePath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

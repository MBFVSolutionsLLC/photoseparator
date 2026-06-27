package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

func scanMedia(root string) ([]MediaFile, error) {
	var files []MediaFile
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
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

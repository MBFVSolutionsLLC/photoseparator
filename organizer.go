package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

func Run(opts Options) error {
	if err := validateOptions(opts); err != nil {
		return err
	}

	files, err := scanMedia(opts.RootDir, opts.PhotoOutDir, opts.VideoOutDir)
	if err != nil {
		return err
	}
	fmt.Printf("Found %d media files\n", len(files))

	processed := processFiles(files, opts)
	sort.Slice(processed, func(i, j int) bool {
		return processed[i].Path < processed[j].Path
	})

	successes := make([]ProcessedFile, 0, len(processed))
	failures := 0
	for _, item := range processed {
		if item.Err != nil {
			failures++
			fmt.Printf("SKIP %s: %v\n", item.Path, item.Err)
			continue
		}
		successes = append(successes, item)
	}

	plan := buildPlan(successes, opts)
	for _, action := range plan {
		if err := applyAction(action, opts); err != nil {
			fmt.Printf("ERROR %s: %v\n", action.Source, err)
			failures++
		}
	}

	fmt.Printf("Done. Processed %d files, skipped or failed %d files.\n", len(plan), failures)
	if failures > 0 {
		return fmt.Errorf("%d files skipped or failed", failures)
	}
	return nil
}

func validateOptions(opts Options) error {
	for label, path := range map[string]string{
		"root directory":         opts.RootDir,
		"photo output directory": opts.PhotoOutDir,
		"video output directory": opts.VideoOutDir,
	} {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("%s is required", label)
		}
	}
	info, err := os.Stat(opts.RootDir)
	if err != nil {
		return fmt.Errorf("root directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("root path is not a directory")
	}
	return nil
}

func processFiles(files []MediaFile, opts Options) []ProcessedFile {
	jobs := make(chan MediaFile)
	results := make(chan ProcessedFile)
	var done atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				results <- processOne(file, opts)
				count := done.Add(1)
				if count%25 == 0 {
					fmt.Printf("Processed %d/%d\n", count, len(files))
				}
			}
		}()
	}

	go func() {
		for _, file := range files {
			jobs <- file
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var processed []ProcessedFile
	for result := range results {
		processed = append(processed, result)
	}
	return processed
}

func processOne(file MediaFile, opts Options) ProcessedFile {
	info, err := os.Stat(file.Path)
	result := ProcessedFile{
		MediaFile:    file,
		OriginalName: filepath.Base(file.Path),
	}
	if err != nil {
		result.Err = err
		return result
	}
	result.Size = info.Size()
	result.Date = inferDate(file.Path, file.Kind)

	if file.Kind == KindPhoto {
		result.Thumbprint, result.Width, result.Height, result.Err = imageThumbprint(file.Path)
		return result
	}
	result.Thumbprint, result.Width, result.Height, result.Err = videoThumbprint(file.Path, opts.TempFrameDir)
	return result
}

type fileAction struct {
	Source      string
	Destination string
	Duplicate   bool
	Move        bool
}

func buildPlan(files []ProcessedFile, opts Options) []fileAction {
	groups := make(map[string][]ProcessedFile)
	for _, file := range files {
		key := file.KindKey() + "|" + file.Date.Format("20060102_150405") + "|" + file.Thumbprint
		groups[key] = append(groups[key], file)
	}

	var actions []fileAction
	usedDestinations := make(map[string]struct{})
	for _, group := range groups {
		sort.Slice(group, func(i, j int) bool {
			if group[i].Size == group[j].Size {
				return group[i].Path < group[j].Path
			}
			return group[i].Size > group[j].Size
		})
		for i, file := range group {
			duplicate := i > 0
			dest := destinationFor(file, duplicate, opts, usedDestinations)
			usedDestinations[dest] = struct{}{}
			actions = append(actions, fileAction{
				Source:      file.Path,
				Destination: dest,
				Duplicate:   duplicate,
				Move:        !opts.Copy,
			})
		}
	}

	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Source < actions[j].Source
	})
	return actions
}

func (p ProcessedFile) KindKey() string {
	return string(p.Kind)
}

func destinationFor(file ProcessedFile, duplicate bool, opts Options, used map[string]struct{}) string {
	base := opts.PhotoOutDir
	if file.Kind == KindVideo {
		base = opts.VideoOutDir
	}
	parts := []string{
		base,
		file.Date.Format("2006"),
		file.Date.Format("01"),
		file.Date.Format("02"),
	}
	if duplicate {
		parts = append(parts, "duplicates")
	}
	dir := filepath.Join(parts...)
	name := fmt.Sprintf("%s.%s%s", file.Date.Format("20060102_150405"), file.Thumbprint, normalizedExt(file.Path))
	return uniquePath(filepath.Join(dir, name), used)
}

func uniquePath(path string, used map[string]struct{}) string {
	if _, ok := used[path]; !ok {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
	}

	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s_%02d%s", stem, i, ext)
		if _, ok := used[candidate]; ok {
			continue
		}
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func applyAction(action fileAction, opts Options) error {
	label := "MOVE"
	if !action.Move {
		label = "COPY"
	}
	if action.Duplicate {
		label += " DUPLICATE"
	}
	fmt.Printf("%s %s -> %s\n", label, action.Source, action.Destination)
	if opts.DryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(action.Destination), 0755); err != nil {
		return err
	}
	if action.Move {
		if err := os.Rename(action.Source, action.Destination); err == nil {
			return nil
		}
		if err := copyFile(action.Source, action.Destination); err != nil {
			return err
		}
		return os.Remove(action.Source)
	}
	return copyFile(action.Source, action.Destination)
}

func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

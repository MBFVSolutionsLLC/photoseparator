package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
)

const appVersion = "1.0.0"

func main() {
	fmt.Printf("Photo Separator v%s\n", appVersion)

	workers := flag.Int("workers", runtime.NumCPU(), "number of concurrent workers")
	copyMode := flag.Bool("copy", false, "copy files instead of moving them")
	dryRun := flag.Bool("dry-run", false, "show planned actions without changing files")
	rootFlag := flag.String("root", "", "root directory to scan (prompted if empty)")
	photoFlag := flag.String("photo-out", "", "photo output directory (prompted if empty)")
	videoFlag := flag.String("video-out", "", "video output directory (prompted if empty)")
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)
	root := pathOrPrompt(reader, *rootFlag, "Root directory to scan")
	photoOut := pathOrPrompt(reader, *photoFlag, "Photo output directory")
	videoOut := pathOrPrompt(reader, *videoFlag, "Video output directory")

	if !confirm(reader, root, photoOut, videoOut) {
		fmt.Println("Cancelled.")
		return
	}

	opts := Options{
		RootDir:      root,
		PhotoOutDir:  photoOut,
		VideoOutDir:  videoOut,
		Workers:      max(1, *workers),
		Copy:         *copyMode,
		DryRun:       *dryRun,
		TempFrameDir: os.TempDir(),
		Confirm:      waitForKey,
	}

	if err := Run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func confirm(reader *bufio.Reader, root, photoOut, videoOut string) bool {
	fmt.Printf("\nPhoto Separator version %s\n", appVersion)
	fmt.Printf("  Source folder:      %s\n", root)
	fmt.Printf("  Photo destination:  %s\n", photoOut)
	fmt.Printf("  Video destination:  %s\n", videoOut)
	fmt.Print("Start processing? [y/N]: ")
	value, _ := reader.ReadString('\n')
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "y" || value == "yes"
}

func pathOrPrompt(reader *bufio.Reader, value, label string) string {
	if value = strings.Trim(strings.TrimSpace(value), `"`); value != "" {
		return value
	}
	return promptPath(reader, label)
}

func promptPath(reader *bufio.Reader, label string) string {
	for {
		fmt.Printf("%s: ", label)
		value, _ := reader.ReadString('\n')
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if value != "" {
			return value
		}
	}
}

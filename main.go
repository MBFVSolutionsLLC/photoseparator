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
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)
	root := promptPath(reader, "Root directory to scan")
	photoOut := promptPath(reader, "Photo output directory")
	videoOut := promptPath(reader, "Video output directory")

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

//go:build !windows

package main

import (
	"os"
	"time"
)

func fileCreationTime(info os.FileInfo) (time.Time, bool) {
	return time.Time{}, false
}

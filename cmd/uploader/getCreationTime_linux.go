//go:build linux

package main

import (
	"io/fs"
	"time"
)

// getCreationTime extracts the file creation time (birthtime) from os.FileInfo on Linux.
// Linux doesn't easily provide birth time, so we fall back to modification time.
func getCreationTime(info fs.FileInfo) time.Time {
	// Linux doesn't have Birthtimespec in syscall.Stat_t
	// Fall back to modification time
	return info.ModTime()
}


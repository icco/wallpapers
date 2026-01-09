//go:build !darwin && !linux

package main

import (
	"io/fs"
	"time"
)

// getCreationTime extracts the file creation time (birthtime) from os.FileInfo.
// Default implementation for platforms that don't support birth time.
func getCreationTime(info fs.FileInfo) time.Time {
	// Fall back to modification time for unsupported platforms
	return info.ModTime()
}


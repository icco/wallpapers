//go:build darwin

package main

import (
	"io/fs"
	"syscall"
	"time"
)

// getCreationTime extracts the file creation time (birthtime) from os.FileInfo on macOS.
func getCreationTime(info fs.FileInfo) time.Time {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		// On macOS, Birthtimespec contains the creation time
		return time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
	}
	// Fallback to modification time if birth time not available
	return info.ModTime()
}


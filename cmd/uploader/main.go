package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/icco/wallpapers"
)

const DropboxPath = "/Photos/Wallpapers/DesktopWallpapers"

var (
	knownLocalFiles map[string]bool
)

func main() {
	ctx := context.Background()
	knownRemoteFiles, err := wallpapers.GetAll(ctx)
	if err != nil {
		log.Printf("error walking: %+v", err)
		os.Exit(1)
	}
	knownLocalFiles = map[string]bool{}

	u, err := user.Lookup("nat")
	if err != nil {
		log.Printf("error getting nat: %+v", err)
		os.Exit(1)
	}
	localFiles := filepath.Join(u.HomeDir, "Dropbox", DropboxPath)

	if err := filepath.Walk(localFiles, walkFn); err != nil {
		log.Printf("error walking: %+v", err)
		os.Exit(1)
	}

	for _, file := range knownRemoteFiles {
		filename := file.Name
		if !knownLocalFiles[filename] {
			if err := wallpapers.DeleteFile(ctx, filename); err != nil {
				log.Printf("could not delete %q: %+v", filename, err)
				os.Exit(1)
			}
			log.Printf("deleted %q", filename)
		}
	}
}

func walkFn(path string, info fs.FileInfo, err error) error {
	if err != nil {
		return fmt.Errorf("prevent panic by handling failure accessing a path %q: %w", path, err)
	}

	if info.IsDir() {
		log.Printf("found a dir: %q", info.Name())
		return nil
	}

	// Skip hidden files
	if strings.HasPrefix(info.Name(), ".") {
		return nil
	}

	ctx := context.Background()

	// Rename
	folder := filepath.Dir(path)
	oldName := info.Name()

	newName := wallpapers.FormatName(info.Name())
	newPath := filepath.Join(folder, newName)
	if newName != info.Name() {
		if err := os.Rename(path, newPath); err != nil {
			return fmt.Errorf("could not rename: %w", err)
		}
		log.Printf("renamed %q => %q", oldName, newName)
	}

	// log existence
	knownLocalFiles[newName] = true

	// Upload
	//gosec:disable G304 We are uploading a file, so we need to read it
	dat, err := os.ReadFile(newPath)
	if err != nil {
		return fmt.Errorf("could not read file: %w", err)
	}

	gc, err := wallpapers.GetGoogleCRC(ctx, newName)
	if err != nil {
		return fmt.Errorf("could not get crc: %w", err)
	}
	lc := wallpapers.GetFileCRC(dat)
	if gc == lc {
		log.Printf("%q unchanged, skipping", newName)
		return nil
	}

	if err := wallpapers.UploadFile(ctx, newName, dat); err != nil {
		return fmt.Errorf("cloud not upload file: %w", err)
	}

	log.Printf("uploaded file: %q", newName)
	return nil
}

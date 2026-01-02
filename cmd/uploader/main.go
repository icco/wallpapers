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
	"time"

	"github.com/icco/wallpapers"
	"github.com/icco/wallpapers/analysis"
	"github.com/icco/wallpapers/db"
)

const DropboxPath = "/Photos/Wallpapers/DesktopWallpapers"

var (
	knownLocalFiles map[string]bool
	database        *db.DB
)

func main() {
	ctx := context.Background()

	// Open database
	var err error
	database, err = db.Open(db.DefaultDBPath())
	if err != nil {
		log.Printf("error opening database: %+v", err)
		os.Exit(1)
	}
	defer func() {
		if cerr := database.Close(); cerr != nil {
			log.Printf("error closing database: %+v", cerr)
		}
	}()

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
			// Also remove from database
			if err := database.Delete(filename); err != nil {
				log.Printf("could not delete from db %q: %+v", filename, err)
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
		log.Printf("%q unchanged, skipping upload", newName)
	} else {
		if err := wallpapers.UploadFile(ctx, newName, dat); err != nil {
			return fmt.Errorf("cloud not upload file: %w", err)
		}
		log.Printf("uploaded file: %q", newName)
	}

	// Check if image needs analysis
	processed, err := database.IsProcessed(newName)
	if err != nil {
		return fmt.Errorf("could not check processing status: %w", err)
	}

	if !processed {
		log.Printf("analyzing %q...", newName)
		if err := analyzeAndStore(ctx, newPath, newName, dat, info.ModTime()); err != nil {
			// Log but don't fail - we can retry later
			log.Printf("warning: failed to analyze %q: %v", newName, err)
		}
	} else {
		log.Printf("%q already processed, skipping analysis", newName)
	}

	return nil
}

// analyzeAndStore analyzes an image and stores the metadata in the database.
func analyzeAndStore(ctx context.Context, filePath, filename string, data []byte, modTime time.Time) error {
	info, err := analysis.AnalyzeImage(ctx, filePath, data)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	now := time.Now()
	img := &db.Image{
		Filename:     filename,
		DateAdded:    now,
		LastModified: modTime,
		Width:        info.Width,
		Height:       info.Height,
		PixelDensity: info.PixelDensity,
		FileFormat:   info.FileFormat,
		Words:        info.Words,
		ProcessedAt:  &now,
	}
	img.SetColors(info.Colors)

	if err := database.UpsertImage(img); err != nil {
		return fmt.Errorf("failed to store image: %w", err)
	}

	log.Printf("stored metadata for %q: %dx%d, %d colors, %d words",
		filename, info.Width, info.Height, len(info.Colors), len(info.Words))
	return nil
}

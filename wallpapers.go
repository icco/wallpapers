package wallpapers

import (
	"context"
	"fmt"
	"hash/crc32"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
)

const (
	Bucket    = "iccowalls"
	NameRegex = regexp.MustCompile("[^a-z0-9]")
)

// FormatName formats a filename to match our requirements.
func FormatName(in string) string {
	ext := strings.ToLower(filepath.Ext(in))
	if ext == ".jpeg" {
		ext = ".jpg"
	}

	name, _ := strings.CutSuffix(in, filepath.Ext(in))
	name = strings.ToLower(filepath.Base(name))
	name = NameRegex.ReplaceAllString(name, "")

	return name + ext
}

// UploadFile takes a file name and content and uploads it to GoogleCloud.
func UploadFile(ctx context.Context, filename string, content []byte) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	wc := client.Bucket(Bucket).Object(filename).NewWriter(ctx)
	wc.CRC32C = crc32.Checksum(content, crc32.MakeTable(crc32.Castagnoli))
	wc.SendCRC32C = true

	if _, err := wc.Write(content); err != nil {
		return fmt.Errorf("failed write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("failed close: %w", err)
	}
	log.Printf("updated object: %+v", wc.Attrs())

	return nil
}

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

// FileUrl returns the URL to the raw file on GCS.
func FileURL(key string) string {
	return fmt.Sprintf("https://%s.storage.googleapis.com/%s", Bucket, key)
}

// FullRezUrl returns the URL a cropped version hosted by imgix.
func FullRezUrl(key string) string {
	return fmt.Sprintf("https://icco-walls.imgix.net/%s?auto=compress&w=2560&h=1440&crop=entropy&fm=png", key)
}

// ThumbUrl returns the URL a small cropped version hosted by imgix.
func ThumpURL(key string) string {
	return fmt.Sprintf("https://icco-walls.imgix.net/%s?w=600&h=400&fit=crop&auto=compress&fm=png", key)
}

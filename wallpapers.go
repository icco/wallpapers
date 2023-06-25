package wallpapers

import (
	"context"
	"fmt"
	"hash/crc32"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

const (
	Bucket = "iccowalls"
)

var (
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

// FullRezURL returns the URL a cropped version hosted by imgix.
func FullRezURL(key string) string {
	return fmt.Sprintf("https://icco-walls.imgix.net/%s?auto=compress&w=2560&h=1440&crop=entropy&fm=png", key)
}

// ThumbUrl returns the URL a small cropped version hosted by imgix.
func ThumbURL(key string) string {
	return fmt.Sprintf("https://icco-walls.imgix.net/%s?w=600&h=400&fit=crop&auto=compress&fm=png", key)
}

// File is a subset of storage.ObjectAttrs that we need.
type File struct {
	CRC32C       uint32    `json:"crc"`
	Etag         string    `json:"etag"`
	FileURL      string    `json:"image"`
	FullRezURL   string    `json:"cdn"`
	Name         string    `json:"key"`
	Size         int64     `json:"size"`
	ThumbnailURL string    `json:"thumbnail"`
	Updated      time.Time `json:"updatedAt"`
}

// GetAll returns all of the attributes for files in GCS.
func GetAll(ctx context.Context) ([]*File, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	var ret []*File

	it := client.Bucket(Bucket).Objects(ctx, nil)
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error on iterating: %w", err)
		}

		ret = append(ret, &File{
			CRC32C:       objAttrs.CRC32C,
			Etag:         objAttrs.Etag,
			Name:         objAttrs.Name,
			Size:         objAttrs.Size,
			Updated:      objAttrs.Updated,
			ThumbnailURL: ThumbURL(objAttrs.Name),
			FileURL:      objAttrs.MediaLink,
			FullRezURL:   FullRezURL(objAttrs.Name),
		})
	}

	return ret, nil
}

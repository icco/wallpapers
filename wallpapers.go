package wallpapers

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"path/filepath"
	"regexp"
	"slices"
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

func GetGoogleCRC(ctx context.Context, filename string) (uint32, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return 0, err
	}

	attr, err := client.Bucket(Bucket).Object(filename).Attrs(ctx)
	if err != nil {
		if !errors.Is(err, storage.ErrObjectNotExist) {
			return 0, fmt.Errorf("could not get attrs: %w", err)
		}

		if errors.Is(err, storage.ErrObjectNotExist) {
			return 0, nil
		}
	}

	return attr.CRC32C, nil
}

func GetFileCRC(content []byte) uint32 {
	return crc32.Checksum(content, crc32.MakeTable(crc32.Castagnoli))
}

func DeleteFile(ctx context.Context, filename string) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	return client.Bucket(Bucket).Object(filename).Delete(ctx)
}

// UploadFile takes a file name and content and uploads it to GoogleCloud.
func UploadFile(ctx context.Context, filename string, content []byte) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	wc := client.Bucket(Bucket).Object(filename).NewWriter(ctx)
	wc.CRC32C = GetFileCRC(content)
	wc.SendCRC32C = true
	wc.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}

	if _, err := wc.Write(content); err != nil {
		return fmt.Errorf("failed write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("failed close: %w", err)
	}

	return nil
}

// FullRezURL returns the URL a cropped version hosted by imgix.
func FullRezURL(key string) string {
	w := 3840
	h := 2160
	return fmt.Sprintf("https://icco-walls.imgix.net/%s?auto=compress&w=%d&h=%d&crop=entropy&fm=png", key, w, h)
}

// ThumbUrl returns the URL a small cropped version hosted by imgix.
func ThumbURL(key string) string {
	w := 800
	h := 450
	return fmt.Sprintf("https://icco-walls.imgix.net/%s?w=%d&h=%d&fit=crop&auto=compress&auto=format", key, w, h)
}

// File is a subset of storage.ObjectAttrs that we need.
type File struct {
	CRC32C       uint32    `json:"-"`
	Etag         string    `json:"etag"`
	FileURL      string    `json:"-"`
	FullRezURL   string    `json:"cdn"`
	Name         string    `json:"key"`
	Size         int64     `json:"-"`
	ThumbnailURL string    `json:"thumbnail"`
	Created      time.Time `json:"created_at"`
	Updated      time.Time `json:"updated_at"`
}

// GetAll returns all of the attributes for files in GCS.
func GetAll(ctx context.Context) ([]*File, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	var ret []*File

	query := &storage.Query{
		Projection: storage.ProjectionNoACL,
	}

	it := client.Bucket(Bucket).Objects(ctx, query)
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
			Created:      objAttrs.Created,
			Updated:      objAttrs.Updated,
			ThumbnailURL: ThumbURL(objAttrs.Name),
			FileURL:      objAttrs.MediaLink,
			FullRezURL:   FullRezURL(objAttrs.Name),
		})
	}

	// Sort by created date
	slices.SortStableFunc(ret, func(b, a *File) int {
		return cmp.Compare(a.Created.String(), b.Created.String())
	})
	return ret, nil
}

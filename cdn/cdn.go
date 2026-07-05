// Package cdn builds the public URLs for wallpaper assets: imgix-derived
// thumbnails and full-resolution renders, plus the direct GCS object URL.
// It is the single source of truth for the bucket name and imgix host so the
// server and uploader cannot drift apart.
package cdn

import "fmt"

const (
	// Bucket is the GCS bucket name where wallpaper files are stored.
	Bucket = "iccowalls"

	// imgixHost is the imgix source mapped to the GCS bucket.
	imgixHost = "icco-walls.imgix.net"
)

// Thumb returns the URL for a small cropped thumbnail via imgix.
func Thumb(key string) string {
	return fmt.Sprintf("https://%s/%s?w=800&h=450&fit=crop&auto=compress&auto=format", imgixHost, key)
}

// FullRez returns the URL for a desktop-sized (3840x2160) render via imgix.
func FullRez(key string) string {
	return fmt.Sprintf("https://%s/%s?auto=compress&w=3840&h=2160&crop=entropy&fm=png", imgixHost, key)
}

// Raw returns the direct URL to the original object in GCS.
func Raw(key string) string {
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", Bucket, key)
}

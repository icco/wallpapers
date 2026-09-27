// Package cdn builds the public URLs for wallpaper assets and is the single
// source of truth for the bucket name and imgix host.
package cdn

import (
	"fmt"
	"net/url"
)

const (
	// Bucket is the GCS bucket name where wallpaper files are stored.
	Bucket = "iccowalls"

	// imgixHost is the imgix source mapped to the GCS bucket.
	imgixHost = "images.natwelch.com/wallpapers"
)

// Thumb returns the URL for a small cropped thumbnail via imgix.
func Thumb(key string) string {
	return fmt.Sprintf("https://%s/%s?w=800&h=450&fit=crop", imgixHost, (&url.URL{Path: key}).EscapedPath())
}

// FullRez returns the URL for a desktop-sized (3840x2160) render via imgix.
func FullRez(key string) string {
	return fmt.Sprintf("https://%s/%s?w=3840&h=2160&fit=crop&crop=entropy&fm=png", imgixHost, (&url.URL{Path: key}).EscapedPath())
}

// Raw returns the direct URL to the original object in GCS.
func Raw(key string) string {
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", Bucket, key)
}

// Package static embeds the HTML templates, CSS, and other static assets
// served by the wallpapers HTTP server.
package static

import "embed"

// Assets are our static files for sharing.
//
//go:embed *
var Assets embed.FS

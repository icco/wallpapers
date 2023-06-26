package static

import "embed"

// Assets are our static files for sharing.
//
//go:embed *
var Assets embed.FS

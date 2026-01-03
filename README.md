# Wallpapers

A [site](http://walls.natwelch.com/) that displays my wallpaper collection. Images are stored in Google Cloud Storage with metadata in SQLite.

## Features

- Masonry grid gallery with search by keyword, color (`#ff5500`), resolution (`1920x1080`), or format
- Automatic image analysis using Gemini (colors, OCR, content tags)

## Usage

```bash
# Run the server
go run ./cmd/server

# Sync images from Dropbox and analyze new ones
GEMINI_API_KEY=... ./update.sh
```

## Environment Variables

- `GEMINI_API_KEY`: Required for image analysis during upload
- `PORT`: Server port (default: 8080)

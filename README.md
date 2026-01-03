# Wallpapers

A [site](http://walls.natwelch.com/) that lists the photos currently in my Wallpaper rotation. Hosts the images in Google Cloud Storage, and the site on my dev server.

## Features

- **Image Gallery**: Browse wallpapers with a masonry grid layout
- **Search**: Find images by word, color (hex), resolution, or file format
- **Metadata**: Each image is analyzed for colors, text (OCR), and content description

## Architecture

- Images stored in Google Cloud Storage
- SQLite database (`wallpapers.db`) stores metadata for each image
- Uploader syncs from local Dropbox folder and analyzes new images
- Server provides JSON API and static file serving

## Environment Variables

- `GEMINI_API_KEY`: Required for image analysis during upload (extracts text and descriptive words)
- `PORT`: Server port (default: 8080)

## Database Schema

The `wallpapers.db` SQLite database stores:

| Field | Description |
|-------|-------------|
| filename | Image filename |
| date_added | When the image was first added |
| last_modified | Last modification time |
| width, height | Image dimensions in pixels |
| pixel_density | Megapixels |
| file_format | jpeg, png, gif, webp |
| color1, color2, color3 | Top 3 prominent colors (hex) |
| words | JSON array of OCR text and descriptive tags |
| processed_at | When analysis was completed |

## Usage

### Running the Server

```bash
go run ./cmd/server
```

### Syncing Images

Set `GEMINI_API_KEY` and run:

```bash
./update.sh
```

This will:
1. Sync images from Dropbox to Google Cloud Storage
2. Analyze new images for colors and words
3. Store metadata in the SQLite database

### Search Examples

- `mountain` - Find images with "mountain" in words
- `#ff5500` - Find images with a specific color
- `1920x1080` - Find images by resolution
- `jpeg` - Find images by format

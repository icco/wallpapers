# Wallpapers

A [site](http://walls.natwelch.com/) that displays Nat Welch's personal wallpaper collection. Images live in Google Cloud Storage, served via Imgix CDN. Metadata (dimensions, colors, tags) is stored in a local SQLite file (`wallpapers.db`) that ships with the container.

## Architecture

| Component | Path | Purpose |
|---|---|---|
| Web server | `cmd/server/` | Serves the gallery and all HTML pages |
| Uploader | `cmd/uploader/` | Syncs Dropbox → GCS, analyzes new images |
| DB package | `db/` | GORM/SQLite models and queries |
| Analysis package | `analysis/` | Color extraction (k-means) + Gemini AI tagging |

## Pages

| Route | Description |
|---|---|
| `/` | Masonry gallery with search |
| `/image/{filename}` | Image detail (metadata, colors, keywords) |
| `/resolutions` | Browse all unique resolutions by count |
| `/colors` | Browse all extracted colors as swatches |
| `/tags` | Browse all AI-extracted keywords as a tag cloud |
| `/search?q=` | JSON search API |
| `/all.json` | JSON dump of all images |

## Search

- **Keyword** — matches against AI-extracted tags, filename, file format
- **Color** (`#rrggbb`) — fuzzy match using RGB Euclidean distance (threshold ≈80), sorted closest-first
- **Resolution** (`1920x1080`) — fuzzy match within ±20% of each dimension, sorted closest-first

## Image Pipeline (uploader)

1. Walk `~/Dropbox/Photos/Wallpapers/DesktopWallpapers`
2. Upload new/changed files to GCS (CRC32 deduplication)
3. Analyze with Gemini (`gemini-2.5-flash`): extract top-3 colors (k-means), resolution, and content tags
4. Store metadata in `wallpapers.db`
5. Delete GCS files no longer present locally

## Development

```bash
# Run the server (requires wallpapers.db in the working directory)
go run ./cmd/server
# or
task server

# Run tests
task test

# Run uploader (syncs Dropbox → GCS, needs GCP credentials)
task uploader

# Lint / format / vet
task check
```

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `GEMINI_API_KEY` | Uploader only | Google Gemini API key for image analysis |
| `GOOGLE_APPLICATION_CREDENTIALS` | Uploader only | Path to GCP service account JSON |
| `PORT` | No | Server port (default: `8080`) |

## Deployment

Docker image builds both binaries from a single `golang:1.26` stage and copies `wallpapers.db` into the final `debian:bookworm-slim` image. The container runs `server` by default on port 8080.

```bash
docker build -t wallpapers .
docker run -p 8080:8080 wallpapers
```

The uploader is a one-shot CLI run locally via `update.sh` (updates deps, syncs, commits the updated `wallpapers.db`).

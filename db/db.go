// Package db provides SQLite database operations for image metadata.
package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	colorful "github.com/lucasb-eyer/go-colorful"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const (
	bucket   = "iccowalls"
	imgixURL = "icco-walls.imgix.net"
)

// StringSlice is a custom type for storing []string as JSON in the database.
type StringSlice []string

// Scan implements the sql.Scanner interface.
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan type %T into StringSlice", value)
	}

	if len(bytes) == 0 {
		*s = nil
		return nil
	}

	return json.Unmarshal(bytes, s)
}

// Value implements the driver.Valuer interface.
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Image represents metadata for a wallpaper image.
type Image struct {
	ID           int64       `json:"id" gorm:"primaryKey;autoIncrement"`
	Filename     string      `json:"key" gorm:"uniqueIndex;not null"`
	DateAdded    time.Time   `json:"created_at" gorm:"autoCreateTime"`
	LastModified time.Time   `json:"updated_at"`
	Width        int         `json:"width,omitempty"`
	Height       int         `json:"height,omitempty"`
	PixelDensity float64     `json:"pixel_density,omitempty"`
	FileFormat   string      `json:"file_format,omitempty"`
	Colors       StringSlice `json:"colors,omitempty" gorm:"type:text"`
	Words        StringSlice `json:"words,omitempty" gorm:"type:text"`
	ProcessedAt  *time.Time  `json:"-"`
}

// TableName specifies the table name for the Image model.
func (Image) TableName() string {
	return "images"
}

// ThumbnailURL returns the URL for a small cropped thumbnail via imgix.
func (img *Image) ThumbnailURL() string {
	return fmt.Sprintf("https://%s/%s?w=800&h=450&fit=crop&auto=compress&auto=format", imgixURL, img.Filename)
}

// FullRezURL returns the URL for a desktop-sized version via imgix.
func (img *Image) FullRezURL() string {
	return fmt.Sprintf("https://%s/%s?auto=compress&w=3840&h=2160&crop=entropy&fm=png", imgixURL, img.Filename)
}

// RawURL returns the direct URL to the original file in GCS.
func (img *Image) RawURL() string {
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, img.Filename)
}

// MarshalJSON implements custom JSON marshaling to include computed URL fields.
func (img *Image) MarshalJSON() ([]byte, error) {
	type imageAlias Image // avoid recursion
	return json.Marshal(&struct {
		*imageAlias
		ThumbnailURL string `json:"thumbnail,omitempty"`
		FullRezURL   string `json:"cdn,omitempty"`
		RawURL       string `json:"raw,omitempty"`
	}{
		imageAlias:   (*imageAlias)(img),
		ThumbnailURL: img.ThumbnailURL(),
		FullRezURL:   img.FullRezURL(),
		RawURL:       img.RawURL(),
	})
}

// MergeMetadata copies analysis metadata from another image (keeps original filename/dates).
func (img *Image) MergeMetadata(other *Image) {
	if other == nil {
		return
	}
	img.Width = other.Width
	img.Height = other.Height
	img.PixelDensity = other.PixelDensity
	img.FileFormat = other.FileFormat
	img.Colors = other.Colors
	img.Words = other.Words
}

// DB wraps the GORM database connection.
type DB struct {
	conn *gorm.DB
	path string
}

// DefaultDBPath returns the default path to the database file.
func DefaultDBPath() string {
	// Look for DB in current directory, then try to find it relative to executable
	if _, err := os.Stat("wallpapers.db"); err == nil {
		return "wallpapers.db"
	}

	// Try to find it in the module root
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		dbPath := filepath.Join(dir, "wallpapers.db")
		if _, err := os.Stat(dbPath); err == nil {
			return dbPath
		}
	}

	return "wallpapers.db"
}

// Open opens or creates the SQLite database.
func Open(dbPath string) (*DB, error) {
	conn, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn, path: dbPath}

	// Auto-migrate the schema
	if err := conn.AutoMigrate(&Image{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	sqlDB, err := db.conn.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying DB: %w", err)
	}
	return sqlDB.Close()
}

// UpsertImage inserts or updates an image record.
func (db *DB) UpsertImage(img *Image) error {
	return db.conn.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "filename"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"date_added", "last_modified", "width", "height", "pixel_density",
			"file_format", "colors", "words", "processed_at",
		}),
	}).Create(img).Error
}

// GetByFilename retrieves an image by filename.
func (db *DB) GetByFilename(filename string) (*Image, error) {
	var img Image
	err := db.conn.Where("filename = ?", filename).First(&img).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &img, nil
}

// IsProcessed checks if an image has been processed.
func (db *DB) IsProcessed(filename string) (bool, error) {
	var imgs []Image
	err := db.conn.Select("processed_at").Where("filename = ?", filename).Limit(1).Find(&imgs).Error
	if err != nil {
		return false, err
	}
	if len(imgs) == 0 {
		return false, nil
	}
	return imgs[0].ProcessedAt != nil, nil
}

// GetAll retrieves all images.
func (db *DB) GetAll() ([]*Image, error) {
	var images []*Image
	err := db.conn.Order("last_modified DESC").Find(&images).Error
	return images, err
}

var (
	colorQueryRe      = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	resolutionQueryRe = regexp.MustCompile(`^(\d+)x(\d+)$`)
)

// Search searches for images by query string.
// For hex colors, uses RGB color distance for fuzzy matching.
// For resolutions, uses ±20% tolerance.
// Otherwise searches words, colors, filename, and file format.
func (db *DB) Search(query string) ([]*Image, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return db.GetAll()
	}

	if colorQueryRe.MatchString(query) {
		return db.searchByColor(query)
	}

	if m := resolutionQueryRe.FindStringSubmatch(query); m != nil {
		qW, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("invalid width in resolution query: %w", err)
		}
		qH, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("invalid height in resolution query: %w", err)
		}
		return db.searchByResolution(qW, qH)
	}

	searchPattern := "%" + query + "%"
	var images []*Image

	err := db.conn.Where(
		"LOWER(words) LIKE ? OR "+
			"LOWER(filename) LIKE ? OR "+
			"LOWER(file_format) LIKE ?",
		searchPattern, searchPattern, searchPattern,
	).Order("last_modified DESC").Find(&images).Error

	return images, err
}

// searchByColor returns images whose stored colors are within maxColorDist of
// hexQuery, using go-colorful's normalized RGB distance (0–√3), sorted closest-first.
func (db *DB) searchByColor(hexQuery string) ([]*Image, error) {
	const maxColorDist = 0.314 // ≈ 80/255 in normalized [0,1] RGB space

	query, err := colorful.Hex(hexQuery)
	if err != nil {
		return nil, fmt.Errorf("invalid color %q: %w", hexQuery, err)
	}

	var all []*Image
	if err := db.conn.Where("colors IS NOT NULL AND colors != '[]' AND colors != ''").
		Order("last_modified DESC").Find(&all).Error; err != nil {
		return nil, err
	}

	type scored struct {
		img  *Image
		dist float64
	}
	var matches []scored
	for _, img := range all {
		minDist := math.MaxFloat64
		for _, c := range img.Colors {
			col, err := colorful.Hex(c)
			if err != nil {
				continue
			}
			if d := query.DistanceRgb(col); d < minDist {
				minDist = d
			}
		}
		if minDist < maxColorDist {
			matches = append(matches, scored{img, minDist})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].dist < matches[j].dist
	})

	result := make([]*Image, len(matches))
	for i, m := range matches {
		result[i] = m.img
	}
	return result, nil
}

// searchByResolution returns images within ±20% of the given dimensions, sorted closest-first.
func (db *DB) searchByResolution(qW, qH int) ([]*Image, error) {
	tolW := int(math.Round(float64(qW) * 0.20))
	tolH := int(math.Round(float64(qH) * 0.20))

	var images []*Image
	err := db.conn.Where(
		"width > 0 AND height > 0 AND ABS(width - ?) <= ? AND ABS(height - ?) <= ?",
		qW, tolW, qH, tolH,
	).Find(&images).Error
	if err != nil {
		return nil, err
	}

	sort.Slice(images, func(i, j int) bool {
		di := math.Abs(float64(images[i].Width-qW)) + math.Abs(float64(images[i].Height-qH))
		dj := math.Abs(float64(images[j].Width-qW)) + math.Abs(float64(images[j].Height-qH))
		return di < dj
	})

	return images, nil
}

// Delete removes an image by filename.
func (db *DB) Delete(filename string) error {
	return db.conn.Where("filename = ?", filename).Delete(&Image{}).Error
}

// ResolutionEntry holds a unique resolution and its occurrence count.
type ResolutionEntry struct {
	Width  int
	Height int
	Count  int
}

// ColorEntry holds a hex color and its occurrence count across all images.
type ColorEntry struct {
	Hex   string
	Count int
}

// TagEntry holds a word/tag and its occurrence count across all images.
type TagEntry struct {
	Word  string
	Count int
}

// GetResolutions returns all unique resolutions sorted by count descending.
func (db *DB) GetResolutions() ([]ResolutionEntry, error) {
	var result []ResolutionEntry
	err := db.conn.Raw(
		"SELECT width, height, COUNT(*) as count FROM images WHERE width > 0 AND height > 0 GROUP BY width, height ORDER BY count DESC",
	).Scan(&result).Error
	return result, err
}

// GetColors returns all unique colors (from images.colors JSON arrays) sorted by count descending.
func (db *DB) GetColors() ([]ColorEntry, error) {
	var images []*Image
	if err := db.conn.Where("colors IS NOT NULL AND colors != '[]' AND colors != ''").Find(&images).Error; err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, img := range images {
		for _, c := range img.Colors {
			counts[strings.ToLower(c)]++
		}
	}
	result := make([]ColorEntry, 0, len(counts))
	for hex, cnt := range counts {
		result = append(result, ColorEntry{Hex: hex, Count: cnt})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	return result, nil
}

// GetTags returns all unique words/tags sorted by count descending.
func (db *DB) GetTags() ([]TagEntry, error) {
	var images []*Image
	if err := db.conn.Where("words IS NOT NULL AND words != '[]' AND words != ''").Find(&images).Error; err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, img := range images {
		for _, w := range img.Words {
			counts[strings.ToLower(w)]++
		}
	}
	result := make([]TagEntry, 0, len(counts))
	for word, cnt := range counts {
		result = append(result, TagEntry{Word: word, Count: cnt})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Word < result[j].Word
	})
	return result, nil
}

// EnsureImage creates a basic record for an image if it doesn't exist.
// Used to track images that exist in GCS but haven't been processed yet.
func (db *DB) EnsureImage(filename string, created, updated time.Time) error {
	img := &Image{
		Filename:     filename,
		DateAdded:    created,
		LastModified: updated,
	}

	return db.conn.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "filename"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"last_modified": updated,
		}),
		Where: clause.Where{
			Exprs: []clause.Expression{
				clause.Eq{Column: "images.processed_at", Value: nil},
			},
		},
	}).Create(img).Error
}

// RunMigrations runs data migrations on the database.
func (db *DB) RunMigrations() error {
	return db.migrateCleanInvalidWords()
}

// migrateCleanInvalidWords removes invalid words (unicode, meta-phrases) from all images.
func (db *DB) migrateCleanInvalidWords() error {
	var images []*Image
	if err := db.conn.Where("words IS NOT NULL AND words != '[]' AND words != ''").Find(&images).Error; err != nil {
		return fmt.Errorf("failed to fetch images: %w", err)
	}

	// Regex to match only ASCII letters, numbers, spaces, and common punctuation
	asciiOnly := regexp.MustCompile(`^[a-zA-Z0-9\s\-']+$`)

	// Patterns that indicate invalid/meta content
	invalidPatterns := []string{
		"no text", "not visible", "not readable", "cannot read",
		"no visible", "n/a", "text not", "no words", "unreadable",
	}

	for _, img := range images {
		if len(img.Words) == 0 {
			continue
		}

		cleanedWords := make([]string, 0, len(img.Words))
		changed := false

		for _, word := range img.Words {
			word = strings.TrimSpace(word)

			if word == "" {
				changed = true
				continue
			}

			if !asciiOnly.MatchString(word) {
				changed = true
				continue
			}

			lower := strings.ToLower(word)
			skip := false
			for _, pattern := range invalidPatterns {
				if strings.Contains(lower, pattern) {
					skip = true
					break
				}
			}
			if skip {
				changed = true
				continue
			}

			if strings.HasPrefix(word, "(") || strings.HasSuffix(word, ")") {
				changed = true
				continue
			}

			if len(word) == 1 && word != "a" && word != "i" {
				changed = true
				continue
			}

			cleanedWords = append(cleanedWords, word)
		}

		if changed {
			img.Words = cleanedWords
			if err := db.conn.Model(img).Update("words", img.Words).Error; err != nil {
				return fmt.Errorf("failed to update image %s: %w", img.Filename, err)
			}
		}
	}

	return nil
}

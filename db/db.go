// Package db provides SQLite database operations for image metadata.
package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/icco/wallpapers"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
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

	// Computed fields (not stored in database)
	ThumbnailURL string `json:"thumbnail,omitempty" gorm:"-"`
	FullRezURL   string `json:"cdn,omitempty" gorm:"-"`
	RawURL       string `json:"raw,omitempty" gorm:"-"`
}

// TableName specifies the table name for the Image model.
func (Image) TableName() string {
	return "images"
}

// WithURLs populates the computed URL fields based on the filename.
func (img *Image) WithURLs() *Image {
	img.ThumbnailURL = wallpapers.ThumbURL(img.Filename)
	img.FullRezURL = wallpapers.FullRezURL(img.Filename)
	img.RawURL = wallpapers.RawURL(img.Filename)
	return img
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
			"last_modified", "width", "height", "pixel_density",
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
	err := db.conn.Order("date_added DESC").Find(&images).Error
	return images, err
}

// Search searches for images by query string.
// Searches in words (JSON array), colors, filename, and file format.
func (db *DB) Search(query string) ([]*Image, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return db.GetAll()
	}

	searchPattern := "%" + query + "%"
	var images []*Image

	err := db.conn.Where(
		"LOWER(words) LIKE ? OR "+
			"LOWER(colors) LIKE ? OR "+
			"LOWER(filename) LIKE ? OR "+
			"LOWER(file_format) LIKE ? OR "+
			"(width || 'x' || height) LIKE ?",
		searchPattern, searchPattern, searchPattern, searchPattern, searchPattern,
	).Order("date_added DESC").Find(&images).Error

	return images, err
}

// Delete removes an image by filename.
func (db *DB) Delete(filename string) error {
	return db.conn.Where("filename = ?", filename).Delete(&Image{}).Error
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

			// Skip empty
			if word == "" {
				changed = true
				continue
			}

			// Skip non-ASCII (unicode characters from other languages)
			if !asciiOnly.MatchString(word) {
				changed = true
				continue
			}

			// Skip meta-phrases
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

			// Skip parenthetical content
			if strings.HasPrefix(word, "(") || strings.HasSuffix(word, ")") {
				changed = true
				continue
			}

			// Skip single character words (except common ones)
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

// Package db provides SQLite database operations for image metadata.
package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	Color1       string      `json:"color1,omitempty" gorm:"index:idx_colors"`
	Color2       string      `json:"color2,omitempty" gorm:"index:idx_colors"`
	Color3       string      `json:"color3,omitempty" gorm:"index:idx_colors"`
	Words        StringSlice `json:"words,omitempty" gorm:"type:text"`
	ProcessedAt  *time.Time  `json:"-"`

	// Computed fields (not stored in database)
	ThumbnailURL string `json:"thumbnail,omitempty" gorm:"-"`
	FullRezURL   string `json:"cdn,omitempty" gorm:"-"`
}

// TableName specifies the table name for the Image model.
func (Image) TableName() string {
	return "images"
}

// WithURLs populates the computed URL fields based on the filename.
func (img *Image) WithURLs() *Image {
	img.ThumbnailURL = thumbURL(img.Filename)
	img.FullRezURL = fullRezURL(img.Filename)
	return img
}

// SetColors sets Color1, Color2, Color3 from a slice.
func (img *Image) SetColors(colors []string) {
	if len(colors) > 0 {
		img.Color1 = colors[0]
	}
	if len(colors) > 1 {
		img.Color2 = colors[1]
	}
	if len(colors) > 2 {
		img.Color3 = colors[2]
	}
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
	img.Color1 = other.Color1
	img.Color2 = other.Color2
	img.Color3 = other.Color3
	img.Words = other.Words
}

// fullRezURL returns the URL for a cropped version hosted by imgix.
func fullRezURL(key string) string {
	w := 3840
	h := 2160
	return fmt.Sprintf("https://icco-walls.imgix.net/%s?auto=compress&w=%d&h=%d&crop=entropy&fm=png", key, w, h)
}

// thumbURL returns the URL for a small cropped version hosted by imgix.
func thumbURL(key string) string {
	w := 800
	h := 450
	return fmt.Sprintf("https://icco-walls.imgix.net/%s?w=%d&h=%d&fit=crop&auto=compress&auto=format", key, w, h)
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
	conn, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
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
			"file_format", "color1", "color2", "color3", "words", "processed_at",
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
	var img Image
	err := db.conn.Select("processed_at").Where("filename = ?", filename).First(&img).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return img.ProcessedAt != nil, nil
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
			"LOWER(color1) LIKE ? OR "+
			"LOWER(color2) LIKE ? OR "+
			"LOWER(color3) LIKE ? OR "+
			"LOWER(filename) LIKE ? OR "+
			"LOWER(file_format) LIKE ? OR "+
			"(width || 'x' || height) LIKE ?",
		searchPattern, searchPattern, searchPattern,
		searchPattern, searchPattern, searchPattern, searchPattern,
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

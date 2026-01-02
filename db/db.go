// Package db provides SQLite database operations for image metadata.
package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Image represents metadata for a wallpaper image.
type Image struct {
	ID           int64     `json:"id"`
	Filename     string    `json:"filename"`
	DateAdded    time.Time `json:"date_added"`
	LastModified time.Time `json:"last_modified"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	PixelDensity float64   `json:"pixel_density"`
	FileFormat   string    `json:"file_format"`
	Color1       string    `json:"color1"`
	Color2       string    `json:"color2"`
	Color3       string    `json:"color3"`
	Words        []string  `json:"words"`
	ProcessedAt  time.Time `json:"processed_at"`
}

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
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
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn, path: dbPath}

	if err := db.init(); err != nil {
		if cerr := conn.Close(); cerr != nil {
			return nil, fmt.Errorf("failed to initialize database: %w (also failed to close: %v)", err, cerr)
		}
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return db, nil
}

// init creates the schema if it doesn't exist.
func (db *DB) init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS images (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT UNIQUE NOT NULL,
		date_added DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_modified DATETIME,
		width INTEGER,
		height INTEGER,
		pixel_density REAL,
		file_format TEXT,
		color1 TEXT,
		color2 TEXT,
		color3 TEXT,
		words TEXT,
		processed_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_filename ON images(filename);
	CREATE INDEX IF NOT EXISTS idx_colors ON images(color1, color2, color3);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// UpsertImage inserts or updates an image record.
func (db *DB) UpsertImage(img *Image) error {
	wordsJSON, err := json.Marshal(img.Words)
	if err != nil {
		return fmt.Errorf("failed to marshal words: %w", err)
	}

	query := `
	INSERT INTO images (filename, date_added, last_modified, width, height, pixel_density, file_format, color1, color2, color3, words, processed_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(filename) DO UPDATE SET
		last_modified = excluded.last_modified,
		width = excluded.width,
		height = excluded.height,
		pixel_density = excluded.pixel_density,
		file_format = excluded.file_format,
		color1 = excluded.color1,
		color2 = excluded.color2,
		color3 = excluded.color3,
		words = excluded.words,
		processed_at = excluded.processed_at
	`

	_, err = db.conn.Exec(query,
		img.Filename,
		img.DateAdded,
		img.LastModified,
		img.Width,
		img.Height,
		img.PixelDensity,
		img.FileFormat,
		img.Color1,
		img.Color2,
		img.Color3,
		string(wordsJSON),
		img.ProcessedAt,
	)
	return err
}

// GetByFilename retrieves an image by filename.
func (db *DB) GetByFilename(filename string) (*Image, error) {
	query := `SELECT id, filename, date_added, last_modified, width, height, pixel_density, file_format, color1, color2, color3, words, processed_at FROM images WHERE filename = ?`

	row := db.conn.QueryRow(query, filename)
	return db.scanImage(row)
}

// IsProcessed checks if an image has been processed.
func (db *DB) IsProcessed(filename string) (bool, error) {
	query := `SELECT processed_at FROM images WHERE filename = ?`
	var processedAt sql.NullTime
	err := db.conn.QueryRow(query, filename).Scan(&processedAt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return processedAt.Valid, nil
}

// GetAll retrieves all images.
func (db *DB) GetAll() ([]*Image, error) {
	query := `SELECT id, filename, date_added, last_modified, width, height, pixel_density, file_format, color1, color2, color3, words, processed_at FROM images ORDER BY date_added DESC`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var images []*Image
	for rows.Next() {
		img, err := db.scanImageRows(rows)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, rows.Err()
}

// Search searches for images by query string.
// Searches in words (JSON array), colors, and filename.
func (db *DB) Search(query string) ([]*Image, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return db.GetAll()
	}

	// Search in words JSON, colors, filename, and file format
	sqlQuery := `
	SELECT id, filename, date_added, last_modified, width, height, pixel_density, file_format, color1, color2, color3, words, processed_at 
	FROM images 
	WHERE 
		LOWER(words) LIKE ? OR
		LOWER(color1) LIKE ? OR
		LOWER(color2) LIKE ? OR
		LOWER(color3) LIKE ? OR
		LOWER(filename) LIKE ? OR
		LOWER(file_format) LIKE ? OR
		(width || 'x' || height) LIKE ?
	ORDER BY date_added DESC
	`

	searchPattern := "%" + query + "%"
	rows, err := db.conn.Query(sqlQuery,
		searchPattern, searchPattern, searchPattern,
		searchPattern, searchPattern, searchPattern, searchPattern,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var images []*Image
	for rows.Next() {
		img, err := db.scanImageRows(rows)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, rows.Err()
}

// Delete removes an image by filename.
func (db *DB) Delete(filename string) error {
	_, err := db.conn.Exec("DELETE FROM images WHERE filename = ?", filename)
	return err
}

// scanImage scans a single row into an Image.
func (db *DB) scanImage(row *sql.Row) (*Image, error) {
	var img Image
	var wordsJSON string
	var dateAdded, lastModified, processedAt sql.NullTime

	err := row.Scan(
		&img.ID,
		&img.Filename,
		&dateAdded,
		&lastModified,
		&img.Width,
		&img.Height,
		&img.PixelDensity,
		&img.FileFormat,
		&img.Color1,
		&img.Color2,
		&img.Color3,
		&wordsJSON,
		&processedAt,
	)
	if err != nil {
		return nil, err
	}

	if dateAdded.Valid {
		img.DateAdded = dateAdded.Time
	}
	if lastModified.Valid {
		img.LastModified = lastModified.Time
	}
	if processedAt.Valid {
		img.ProcessedAt = processedAt.Time
	}

	if wordsJSON != "" {
		if err := json.Unmarshal([]byte(wordsJSON), &img.Words); err != nil {
			return nil, fmt.Errorf("failed to unmarshal words: %w", err)
		}
	}

	return &img, nil
}

// scanImageRows scans a rows result into an Image.
func (db *DB) scanImageRows(rows *sql.Rows) (*Image, error) {
	var img Image
	var wordsJSON string
	var dateAdded, lastModified, processedAt sql.NullTime

	err := rows.Scan(
		&img.ID,
		&img.Filename,
		&dateAdded,
		&lastModified,
		&img.Width,
		&img.Height,
		&img.PixelDensity,
		&img.FileFormat,
		&img.Color1,
		&img.Color2,
		&img.Color3,
		&wordsJSON,
		&processedAt,
	)
	if err != nil {
		return nil, err
	}

	if dateAdded.Valid {
		img.DateAdded = dateAdded.Time
	}
	if lastModified.Valid {
		img.LastModified = lastModified.Time
	}
	if processedAt.Valid {
		img.ProcessedAt = processedAt.Time
	}

	if wordsJSON != "" {
		if err := json.Unmarshal([]byte(wordsJSON), &img.Words); err != nil {
			return nil, fmt.Errorf("failed to unmarshal words: %w", err)
		}
	}

	return &img, nil
}

// EnsureImage creates a basic record for an image if it doesn't exist.
// Used to track images that exist in GCS but haven't been processed yet.
func (db *DB) EnsureImage(filename string, created, updated time.Time) error {
	query := `
	INSERT INTO images (filename, date_added, last_modified)
	VALUES (?, ?, ?)
	ON CONFLICT(filename) DO UPDATE SET
		last_modified = excluded.last_modified
	WHERE images.processed_at IS NULL
	`
	_, err := db.conn.Exec(query, filename, created, updated)
	return err
}

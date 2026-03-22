package db

import (
	"testing"
	"time"
)

// newTestDB opens an in-memory SQLite database for testing.
func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seed(t *testing.T, db *DB, images ...*Image) {
	t.Helper()
	for _, img := range images {
		if err := db.UpsertImage(img); err != nil {
			t.Fatalf("seed image %s: %v", img.Filename, err)
		}
	}
}

func now() *time.Time { t := time.Now(); return &t }

// --- searchByColor ---

func TestSearchByColor_ExactMatch(t *testing.T) {
	db := newTestDB(t)
	seed(t, db,
		&Image{Filename: "red.jpg", Colors: StringSlice{"#ff0000", "#111111", "#222222"}, ProcessedAt: now()},
		&Image{Filename: "blue.jpg", Colors: StringSlice{"#0000ff", "#333333", "#444444"}, ProcessedAt: now()},
	)

	results, err := db.searchByColor("#ff0000")
	if err != nil {
		t.Fatalf("searchByColor: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result for exact color match")
	}
	if results[0].Filename != "red.jpg" {
		t.Errorf("expected red.jpg first, got %s", results[0].Filename)
	}
}

func TestSearchByColor_NearMatch(t *testing.T) {
	db := newTestDB(t)
	// #fe0101 is very close to #ff0000 (distance ≈ 0.006)
	seed(t, db,
		&Image{Filename: "nearred.jpg", Colors: StringSlice{"#fe0101"}, ProcessedAt: now()},
		&Image{Filename: "blue.jpg", Colors: StringSlice{"#0000ff"}, ProcessedAt: now()},
	)

	results, err := db.searchByColor("#ff0000")
	if err != nil {
		t.Fatalf("searchByColor near: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected near-red to match #ff0000 search")
	}
	if results[0].Filename != "nearred.jpg" {
		t.Errorf("expected nearred.jpg, got %s", results[0].Filename)
	}
}

func TestSearchByColor_NoMatch(t *testing.T) {
	db := newTestDB(t)
	seed(t, db,
		&Image{Filename: "blue.jpg", Colors: StringSlice{"#0000ff"}, ProcessedAt: now()},
	)

	results, err := db.searchByColor("#ff0000")
	if err != nil {
		t.Fatalf("searchByColor no match: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results for distant color, got %d", len(results))
	}
}

func TestSearchByColor_InvalidHex(t *testing.T) {
	db := newTestDB(t)
	_, err := db.searchByColor("#zzz")
	if err == nil {
		t.Fatal("expected error for invalid hex color")
	}
}

func TestSearchByColor_SortedByCloseness(t *testing.T) {
	db := newTestDB(t)
	// #ff1010 (dist≈0.089) is closer to #ff0000 than #ff3030 (dist≈0.266); both within threshold.
	seed(t, db,
		&Image{Filename: "far.jpg", Colors: StringSlice{"#ff3030"}, ProcessedAt: now()},
		&Image{Filename: "close.jpg", Colors: StringSlice{"#ff1010"}, ProcessedAt: now()},
	)

	results, err := db.searchByColor("#ff0000")
	if err != nil {
		t.Fatalf("searchByColor sort: %v", err)
	}
	if len(results) < 2 {
		t.Fatal("expected both images to match")
	}
	if results[0].Filename != "close.jpg" {
		t.Errorf("expected close.jpg first, got %s", results[0].Filename)
	}
}

// --- searchByResolution ---

func TestSearchByResolution_ExactMatch(t *testing.T) {
	db := newTestDB(t)
	seed(t, db,
		&Image{Filename: "hd.jpg", Width: 1920, Height: 1080, ProcessedAt: now()},
		&Image{Filename: "4k.jpg", Width: 3840, Height: 2160, ProcessedAt: now()},
	)

	results, err := db.searchByResolution(1920, 1080)
	if err != nil {
		t.Fatalf("searchByResolution: %v", err)
	}
	if len(results) != 1 || results[0].Filename != "hd.jpg" {
		t.Errorf("expected only hd.jpg, got %v", fileNames(results))
	}
}

func TestSearchByResolution_WithinTolerance(t *testing.T) {
	db := newTestDB(t)
	// 1800x1000 is within ±20% of 1920x1080
	seed(t, db,
		&Image{Filename: "near.jpg", Width: 1800, Height: 1000, ProcessedAt: now()},
		&Image{Filename: "far.jpg", Width: 640, Height: 480, ProcessedAt: now()},
	)

	results, err := db.searchByResolution(1920, 1080)
	if err != nil {
		t.Fatalf("searchByResolution tolerance: %v", err)
	}
	if len(results) != 1 || results[0].Filename != "near.jpg" {
		t.Errorf("expected near.jpg, got %v", fileNames(results))
	}
}

func TestSearchByResolution_SortedByCloseness(t *testing.T) {
	db := newTestDB(t)
	seed(t, db,
		&Image{Filename: "farish.jpg", Width: 1700, Height: 900, ProcessedAt: now()},
		&Image{Filename: "close.jpg", Width: 1900, Height: 1060, ProcessedAt: now()},
	)

	results, err := db.searchByResolution(1920, 1080)
	if err != nil {
		t.Fatalf("searchByResolution sort: %v", err)
	}
	if len(results) < 2 {
		t.Fatal("expected both images to match")
	}
	if results[0].Filename != "close.jpg" {
		t.Errorf("expected close.jpg first, got %s", results[0].Filename)
	}
}

// --- GetResolutions ---

func TestGetResolutions(t *testing.T) {
	db := newTestDB(t)
	seed(t, db,
		&Image{Filename: "a.jpg", Width: 1920, Height: 1080},
		&Image{Filename: "b.jpg", Width: 1920, Height: 1080},
		&Image{Filename: "c.jpg", Width: 3840, Height: 2160},
	)

	resolutions, err := db.GetResolutions()
	if err != nil {
		t.Fatalf("GetResolutions: %v", err)
	}
	if len(resolutions) != 2 {
		t.Fatalf("expected 2 unique resolutions, got %d", len(resolutions))
	}
	// Most common should be first.
	if resolutions[0].Width != 1920 || resolutions[0].Height != 1080 {
		t.Errorf("expected 1920x1080 first, got %dx%d", resolutions[0].Width, resolutions[0].Height)
	}
	if resolutions[0].Count != 2 {
		t.Errorf("expected count 2, got %d", resolutions[0].Count)
	}
}

func TestGetResolutions_Empty(t *testing.T) {
	db := newTestDB(t)
	results, err := db.GetResolutions()
	if err != nil {
		t.Fatalf("GetResolutions empty: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty, got %d", len(results))
	}
}

// --- GetColors ---

func TestGetColors_GridSize(t *testing.T) {
	db := newTestDB(t)
	colors, err := db.GetColors()
	if err != nil {
		t.Fatalf("GetColors: %v", err)
	}
	want := colorGridSize * colorGridSize
	if len(colors) != want {
		t.Fatalf("expected %d grid entries, got %d", want, len(colors))
	}
}

func TestGetColors_CountsImages(t *testing.T) {
	db := newTestDB(t)
	// Seed an image with a vivid red. At least one grid cell should have count >= 1.
	seed(t, db, &Image{Filename: "red.jpg", Colors: StringSlice{"#ff0000"}})

	colors, err := db.GetColors()
	if err != nil {
		t.Fatalf("GetColors: %v", err)
	}
	anyNonZero := false
	for _, c := range colors {
		if c.Count > 0 {
			anyNonZero = true
			break
		}
	}
	if !anyNonZero {
		t.Error("expected at least one grid cell with count > 0 after seeding a red image")
	}
}

func TestGetColors_EmptyDB(t *testing.T) {
	db := newTestDB(t)
	colors, err := db.GetColors()
	if err != nil {
		t.Fatalf("GetColors empty: %v", err)
	}
	// Grid is always returned; all counts should be 0.
	want := colorGridSize * colorGridSize
	if len(colors) != want {
		t.Fatalf("expected %d entries, got %d", want, len(colors))
	}
	for _, c := range colors {
		if c.Count != 0 {
			t.Errorf("expected count 0 for empty db, got %d for %s", c.Count, c.Hex)
		}
	}
}

// --- GetTags ---

func TestGetTags(t *testing.T) {
	db := newTestDB(t)
	seed(t, db,
		&Image{Filename: "a.jpg", Words: StringSlice{"mountain", "sky"}},
		&Image{Filename: "b.jpg", Words: StringSlice{"mountain", "ocean"}},
	)

	tags, err := db.GetTags()
	if err != nil {
		t.Fatalf("GetTags: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("expected 3 unique tags, got %d", len(tags))
	}
	if tags[0].Word != "mountain" {
		t.Errorf("expected mountain first, got %s", tags[0].Word)
	}
	if tags[0].Count != 2 {
		t.Errorf("expected count 2, got %d", tags[0].Count)
	}
}

func TestGetTags_Empty(t *testing.T) {
	db := newTestDB(t)
	results, err := db.GetTags()
	if err != nil {
		t.Fatalf("GetTags empty: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty, got %d", len(results))
	}
}

// --- helpers ---

func fileNames(imgs []*Image) []string {
	names := make([]string, len(imgs))
	for i, img := range imgs {
		names[i] = img.Filename
	}
	return names
}

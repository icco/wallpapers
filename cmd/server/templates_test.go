package main

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"github.com/icco/wallpapers/db"
)

// loadAll loads all five page templates and fails the test on any error.
func loadAll(t *testing.T) {
	t.Helper()
	var err error
	if indexTemplate, err = loadTemplate("index", nil); err != nil {
		t.Fatalf("load index: %v", err)
	}
	if detailTemplate, err = loadTemplate("detail", nil); err != nil {
		t.Fatalf("load detail: %v", err)
	}
	if resolutionsTemplate, err = loadTemplate("resolutions", nil); err != nil {
		t.Fatalf("load resolutions: %v", err)
	}
	if colorsTemplate, err = loadTemplate("colors", nil); err != nil {
		t.Fatalf("load colors: %v", err)
	}
	if tagsTemplate, err = loadTemplate("tags", nil); err != nil {
		t.Fatalf("load tags: %v", err)
	}
}

func TestIndexTemplateRenders(t *testing.T) {
	loadAll(t)

	data := PageData{
		Query: "",
		Images: []*db.Image{
			{Filename: "test.jpg", Width: 1920, Height: 1080},
		},
	}

	var buf bytes.Buffer
	if err := indexTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("execute index: %v", err)
	}
	body := buf.String()
	assertContains(t, body, "Wallpapers")
	assertContains(t, body, "test.jpg")
	assertContains(t, body, "/resolutions")
	assertContains(t, body, "/colors")
	assertContains(t, body, "/tags")
}

func TestIndexTemplateWithQuery(t *testing.T) {
	loadAll(t)

	data := PageData{Query: "mountain", Images: nil}
	var buf bytes.Buffer
	if err := indexTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("execute index with query: %v", err)
	}
	assertContains(t, buf.String(), "mountain")
}

func TestDetailTemplateRenders(t *testing.T) {
	loadAll(t)

	now := time.Now()
	data := DetailPageData{
		Image: &db.Image{
			Filename:     "sunset.jpg",
			Width:        3840,
			Height:       2160,
			FileFormat:   "jpeg",
			PixelDensity: 8.29,
			DateAdded:    now,
			Colors:       db.StringSlice{"#ff5500", "#223344", "#aabbcc"},
			Words:        db.StringSlice{"sunset", "orange", "sky"},
		},
	}

	var buf bytes.Buffer
	if err := detailTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("execute detail: %v", err)
	}
	body := buf.String()
	assertContains(t, body, "sunset.jpg")
	assertContains(t, body, "3840")
	assertContains(t, body, "#ff5500")
	assertContains(t, body, "sunset")
	assertContains(t, body, "&larr; Back")
}

func TestDetailTemplateNoWords(t *testing.T) {
	loadAll(t)

	data := DetailPageData{
		Image: &db.Image{Filename: "blank.png", DateAdded: time.Now()},
	}
	var buf bytes.Buffer
	if err := detailTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("execute detail (no words): %v", err)
	}
	assertContains(t, buf.String(), "No keywords extracted yet")
}

func TestResolutionsTemplateRenders(t *testing.T) {
	loadAll(t)

	data := ResolutionsPageData{
		Resolutions: []db.ResolutionEntry{
			{Width: 3840, Height: 2160, Count: 42},
			{Width: 1920, Height: 1080, Count: 17},
		},
	}

	var buf bytes.Buffer
	if err := resolutionsTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("execute resolutions: %v", err)
	}
	body := buf.String()
	assertContains(t, body, "Resolutions")
	assertContains(t, body, "3840")
	assertContains(t, body, "1920")
	assertContains(t, body, "/?q=3840x2160")
}

func TestColorsTemplateRenders(t *testing.T) {
	loadAll(t)

	data := ColorsPageData{
		Colors: []db.ColorEntry{
			{Hex: "#ff5500", Count: 10},
			{Hex: "#001122", Count: 3},
		},
	}

	var buf bytes.Buffer
	if err := colorsTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("execute colors: %v", err)
	}
	body := buf.String()
	assertContains(t, body, "Colors")
	assertContains(t, body, "#ff5500")
	assertContains(t, body, "background-color: #ff5500")
}

func TestTagsTemplateRenders(t *testing.T) {
	loadAll(t)

	data := TagsPageData{
		Tags: []db.TagEntry{
			{Word: "mountain", Count: 50},
			{Word: "sunset", Count: 20},
			{Word: "ocean", Count: 1},
		},
	}

	var buf bytes.Buffer
	if err := tagsTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("execute tags: %v", err)
	}
	body := buf.String()
	assertContains(t, body, "Tags")
	assertContains(t, body, "mountain")
	assertContains(t, body, "--count:")
	assertContains(t, body, "/?q=mountain")
}

func TestTagsTemplateEmpty(t *testing.T) {
	loadAll(t)

	data := TagsPageData{}
	var buf bytes.Buffer
	if err := tagsTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("execute tags (empty): %v", err)
	}
}

func TestLayoutSharedElements(t *testing.T) {
	loadAll(t)

	now := time.Now()
	// Every page should include the analytics script and tachyons CSS.
	cases := []struct {
		name string
		tmpl *template.Template
		data any
	}{
		{"index", indexTemplate, PageData{}},
		{"detail", detailTemplate, DetailPageData{Image: &db.Image{Filename: "x.jpg", DateAdded: now}}},
		{"resolutions", resolutionsTemplate, ResolutionsPageData{}},
		{"colors", colorsTemplate, ColorsPageData{}},
		{"tags", tagsTemplate, TagsPageData{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tc.tmpl.Execute(&buf, tc.data); err != nil {
				t.Fatalf("execute %s: %v", tc.name, err)
			}
			body := buf.String()
			assertContains(t, body, "tachyons.min.css")
			assertContains(t, body, "web-vitals")
			assertContains(t, body, "reportd.natwelch.com")
		})
	}
}

func assertContains(t *testing.T, body, want string) {
	t.Helper()
	if !bytes.Contains([]byte(body), []byte(want)) {
		t.Errorf("expected output to contain %q", want)
	}
}

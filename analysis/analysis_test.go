package analysis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestParseWords(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"basic comma list", "mountain, sunset, sky", []string{"mountain", "sunset", "sky"}},
		{"lowercases", "Mountain, SUNSET", []string{"mountain", "sunset"}},
		{"dedupes case-insensitively", "sky, sky, Sky", []string{"sky"}},
		{"strips markdown and parens", "*mountain*, `sky`, (no text visible)", []string{"mountain", "sky"}},
		{"drops meta-phrases", "mountain, no text, unreadable, n/a", []string{"mountain"}},
		{"drops non-ascii", "mountain, 山, café", []string{"mountain"}},
		{"newline separated", "mountain\nsunset\nsky", []string{"mountain", "sunset", "sky"}},
		{"empty input", "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseWords(tc.in)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseWords(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// stubGemini points extractWords at a fake Gemini endpoint returning body.
func stubGemini(t *testing.T, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	t.Setenv("GEMINI_API_KEY", "test-key")
	orig := geminiBaseURL
	geminiBaseURL = srv.URL
	t.Cleanup(func() { geminiBaseURL = orig })
}

// geminiBody renders a generateContent response carrying text.
func geminiBody(text string) string {
	b, err := json.Marshal(map[string]any{
		"candidates": []any{map[string]any{
			"content": map[string]any{"role": "model", "parts": []any{map[string]any{"text": text}}},
		}},
	})
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestExtractWords(t *testing.T) {
	stubGemini(t, geminiBody("mountain, sunset, snow peak"))

	got, err := extractWords(t.Context(), []byte{1, 2, 3}, "png")
	if err != nil {
		t.Fatalf("extractWords: %v", err)
	}
	want := []string{"mountain", "sunset", "snow peak"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("extractWords = %v, want %v", got, want)
	}
}

// A model that has nothing to say about a wallpaper is a normal outcome, not a
// failure. Returning an error here would make AnalyzeImage log a warning for
// every blank or abstract image.
func TestExtractWordsEmptyResponseIsNotAnError(t *testing.T) {
	stubGemini(t, geminiBody(""))

	got, err := extractWords(t.Context(), []byte{1, 2, 3}, "jpeg")
	if err != nil {
		t.Fatalf("extractWords on an empty response = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("extractWords = %v, want no keywords", got)
	}
	if got == nil {
		t.Error("extractWords returned a nil slice; want an empty one")
	}
}

func TestExtractWordsSurfacesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	t.Setenv("GEMINI_API_KEY", "test-key")
	orig := geminiBaseURL
	geminiBaseURL = srv.URL
	defer func() { geminiBaseURL = orig }()

	if _, err := extractWords(t.Context(), []byte{1, 2, 3}, "png"); err == nil {
		t.Fatal("extractWords on a 429 = nil, want an error")
	}
}

func TestExtractWordsRequiresAPIKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")

	if _, err := extractWords(t.Context(), []byte{1, 2, 3}, "png"); err == nil {
		t.Fatal("extractWords with no API key = nil, want an error")
	}
}

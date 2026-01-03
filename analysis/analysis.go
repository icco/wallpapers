// Package analysis provides image analysis functionality including
// color extraction, resolution detection, and Gemini API integration.
package analysis

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/EdlinOrg/prominentcolor"
	_ "golang.org/x/image/webp"
	"google.golang.org/genai"
)

// ImageInfo contains analyzed information about an image.
type ImageInfo struct {
	Width        int
	Height       int
	PixelDensity float64
	FileFormat   string
	Colors       []string // Top 3 colors as hex strings
	Words        []string // Words from OCR and description
}

// AnalyzeImage performs full analysis on an image file.
func AnalyzeImage(ctx context.Context, filePath string, fileData []byte) (*ImageInfo, error) {
	info := &ImageInfo{}

	// Get file format from extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		info.FileFormat = "jpeg"
	case ".png":
		info.FileFormat = "png"
	case ".gif":
		info.FileFormat = "gif"
	case ".webp":
		info.FileFormat = "webp"
	default:
		info.FileFormat = strings.TrimPrefix(ext, ".")
	}

	// Get resolution
	width, height, err := getResolution(fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to get resolution: %w", err)
	}
	info.Width = width
	info.Height = height

	// Calculate pixel density (megapixels)
	info.PixelDensity = float64(width*height) / 1000000.0

	// Extract colors
	colors, err := extractColors(fileData)
	if err != nil {
		// Log but don't fail - colors are nice to have
		fmt.Fprintf(os.Stderr, "warning: failed to extract colors: %v\n", err)
		info.Colors = []string{}
	} else {
		info.Colors = colors
	}

	// Extract words using Gemini
	words, err := extractWords(ctx, fileData, info.FileFormat)
	if err != nil {
		// Log but don't fail - we can still index without words
		fmt.Fprintf(os.Stderr, "warning: failed to extract words: %v\n", err)
		info.Words = []string{}
	} else {
		info.Words = words
	}

	return info, nil
}

// getResolution decodes the image to get its dimensions.
func getResolution(data []byte) (width, height int, err error) {
	img, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	return img.Width, img.Height, nil
}

// extractColors extracts the top 3 prominent colors from an image.
func extractColors(data []byte) ([]string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Extract prominent colors
	colors, err := prominentcolor.KmeansWithArgs(prominentcolor.ArgumentNoCropping, img)
	if err != nil {
		return nil, fmt.Errorf("failed to extract colors: %w", err)
	}

	// Get top 3 colors as hex strings
	result := make([]string, 0, 3)
	for i := 0; i < len(colors) && i < 3; i++ {
		hex := fmt.Sprintf("#%02x%02x%02x", colors[i].Color.R, colors[i].Color.G, colors[i].Color.B)
		result = append(result, hex)
	}

	return result, nil
}

// extractWords uses Gemini API to extract words from an image.
// It performs both OCR (text in image) and description (what's in the image).
func extractWords(ctx context.Context, data []byte, format string) ([]string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Set up the prompt for extracting words
	prompt := `Analyze this image and provide keywords describing what's in the image (objects, scenery, mood, style, colors). If there is readable English text in the image, include those words too.

Rules:
- Return ONLY a comma-separated list of single words or short phrases (2-3 words max)
- English words only - no other languages or scripts
- Do not include meta-commentary like "no text visible" or "text not readable"
- Do not include explanations, categories, or sentences
- If you cannot identify anything, return nothing

Example output: mountain, sunset, orange sky, peaceful, landscape, snow peak, clouds

Words:`

	// Determine MIME type
	mimeType := "image/jpeg"
	switch format {
	case "png":
		mimeType = "image/png"
	case "gif":
		mimeType = "image/gif"
	case "webp":
		mimeType = "image/webp"
	}

	parts := []*genai.Part{
		{Text: prompt},
		{InlineData: &genai.Blob{Data: data, MIMEType: mimeType}},
	}

	resp, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash",
		[]*genai.Content{{Parts: parts}}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return []string{}, nil
	}

	// Extract text from response
	var text string
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			text += part.Text
		}
	}

	// Parse the comma-separated words
	words := parseWords(text)
	return words, nil
}

// parseWords parses a comma-separated string of words into a slice.
func parseWords(text string) []string {
	// Clean up the text
	text = strings.TrimSpace(text)

	// Remove any markdown formatting
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "`", "")

	// Remove parenthetical content (e.g., "(no text visible)")
	text = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(text, "")

	// Split by comma or newline
	parts := regexp.MustCompile(`[,\n]+`).Split(text, -1)

	words := make([]string, 0, len(parts))
	seen := make(map[string]bool)

	// Regex to match only ASCII letters, numbers, spaces, and common punctuation
	asciiOnly := regexp.MustCompile(`^[a-zA-Z0-9\s\-']+$`)

	// Meta-phrases to skip
	skipPhrases := []string{
		"no text", "not visible", "not readable", "cannot read",
		"no visible", "none", "n/a", "nothing", "text not",
	}

	for _, part := range parts {
		word := strings.TrimSpace(strings.ToLower(part))
		// Skip empty strings and duplicates
		if word == "" || seen[word] {
			continue
		}
		// Skip if too long (probably a sentence)
		if len(word) > 50 {
			continue
		}
		// Skip non-ASCII content (unicode characters from other languages)
		if !asciiOnly.MatchString(word) {
			continue
		}
		// Skip meta-phrases
		skip := false
		for _, phrase := range skipPhrases {
			if strings.Contains(word, phrase) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		seen[word] = true
		words = append(words, word)
	}

	return words
}

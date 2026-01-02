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
	"github.com/google/generative-ai-go/genai"
	_ "golang.org/x/image/webp"
	"google.golang.org/api/option"
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

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer func() {
		if cerr := client.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close Gemini client: %v\n", cerr)
		}
	}()

	model := client.GenerativeModel("gemini-2.0-flash")

	// Set up the prompt for extracting words
	prompt := `Analyze this image and provide:
1. Any text visible in the image (OCR)
2. Keywords describing what's in the image (objects, scenery, mood, style, colors)

Return ONLY a comma-separated list of single words or short phrases (2-3 words max).
Do not include sentences, explanations, or categories.
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

	resp, err := model.GenerateContent(ctx,
		genai.Blob{MIMEType: mimeType, Data: data},
		genai.Text(prompt),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return []string{}, nil
	}

	// Extract text from response
	var text string
	for _, part := range resp.Candidates[0].Content.Parts {
		if t, ok := part.(genai.Text); ok {
			text += string(t)
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

	// Split by comma or newline
	parts := regexp.MustCompile(`[,\n]+`).Split(text, -1)

	words := make([]string, 0, len(parts))
	seen := make(map[string]bool)

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
		seen[word] = true
		words = append(words, word)
	}

	return words
}

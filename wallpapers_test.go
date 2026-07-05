package wallpapers

import (
	"path/filepath"
	"strings"
	"testing"
)

// Inputs below all normalize to a base of at least 15 characters so FormatName
// never reaches the babble-padding path, which depends on a system dictionary
// (/usr/share/dict/words) that isn't present on all CI runners.

func TestFormatName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"jpeg normalized to jpg", "MountainLandscape.JPEG", "mountainlandscape.jpg"},
		{"lowercase jpeg normalized", "MountainLandscape.jpeg", "mountainlandscape.jpg"},
		{"uppercase png lowered", "MountainLandscape.PNG", "mountainlandscape.png"},
		{"uppercase gif lowered", "MountainLandscape.GIF", "mountainlandscape.gif"},
		{"spaces and punctuation stripped", "My Cool Wallpaper File!.png", "mycoolwallpaperfile.png"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatName(tc.in); got != tc.want {
				t.Errorf("FormatName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormatNameTruncatesTo100(t *testing.T) {
	got := FormatName(strings.Repeat("a", 150) + ".JPG")
	base := strings.TrimSuffix(got, filepath.Ext(got))
	if len(base) != 100 {
		t.Errorf("base length = %d, want 100", len(base))
	}
	if ext := filepath.Ext(got); ext != ".jpg" {
		t.Errorf("ext = %q, want .jpg", ext)
	}
}

package analysis

import (
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

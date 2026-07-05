package words

import "testing"

func TestIsValid(t *testing.T) {
	cases := []struct {
		tag  string
		want bool
	}{
		{"mountain", true},
		{"orange sky", true},
		{"snow-peak", true},
		{"a", true},          // single-letter allowlist
		{"i", true},          // single-letter allowlist
		{"x", false},         // other single letters are noise
		{"", false},          // empty
		{"café", false},      // non-ASCII
		{"山", false},         // non-ASCII
		{"no text", false},   // meta-phrase
		{"unreadable", false}, // meta-phrase
		{"(aside)", false},   // parenthesized
		{"aside)", false},    // trailing paren
	}
	for _, tc := range cases {
		if got := IsValid(tc.tag); got != tc.want {
			t.Errorf("IsValid(%q) = %v, want %v", tc.tag, got, tc.want)
		}
	}
}

func TestIsValidRejectsLongTags(t *testing.T) {
	long := ""
	for range 51 {
		long += "a"
	}
	if IsValid(long) {
		t.Errorf("IsValid(%d-char tag) = true, want false", len(long))
	}
}

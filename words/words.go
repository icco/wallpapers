// Package words provides shared validation for image keyword tags. Both the
// analysis pipeline (parsing model output) and the database cleanup migration
// use it so the two paths cannot disagree on what counts as a usable keyword.
package words

import (
	"regexp"
	"strings"
)

// asciiWord matches only ASCII letters, numbers, spaces, hyphens and
// apostrophes. Keywords with other scripts or punctuation are rejected.
var asciiWord = regexp.MustCompile(`^[a-zA-Z0-9\s\-']+$`)

// invalidPhrases are meta-commentary fragments the model sometimes emits
// instead of real keywords. A tag containing any of these is dropped.
var invalidPhrases = []string{
	"no text", "not visible", "not readable", "cannot read",
	"no visible", "none", "nothing", "n/a", "text not",
	"no words", "unreadable",
}

// IsValid reports whether tag is a usable keyword. The tag should already be
// whitespace-trimmed; matching is case-insensitive.
func IsValid(tag string) bool {
	if tag == "" || len(tag) > 50 {
		return false
	}
	if !asciiWord.MatchString(tag) {
		return false
	}
	if strings.HasPrefix(tag, "(") || strings.HasSuffix(tag, ")") {
		return false
	}

	lower := strings.ToLower(tag)
	if len(lower) == 1 && lower != "a" && lower != "i" {
		return false
	}
	for _, p := range invalidPhrases {
		if strings.Contains(lower, p) {
			return false
		}
	}
	return true
}

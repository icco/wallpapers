// Package words provides shared validation for image keyword tags, used by
// both the analysis pipeline and the database cleanup migration.
package words

import (
	"regexp"
	"strings"
)

// asciiWord matches ASCII letters, numbers, spaces, hyphens and apostrophes.
var asciiWord = regexp.MustCompile(`^[a-zA-Z0-9\s\-']+$`)

// invalidPhrases are meta-commentary fragments to drop, not real keywords.
var invalidPhrases = []string{
	"no text", "not visible", "not readable", "cannot read",
	"no visible", "none", "nothing", "n/a", "text not",
	"no words", "unreadable",
}

// IsValid reports whether tag is a usable keyword (case-insensitive; tag
// should be pre-trimmed).
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

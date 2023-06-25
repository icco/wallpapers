package wallpapers

import (
	"path/filepath"
	"regexp"
	"strings"
)

const NameRegex = regexp.MustCompile("[^a-z0-9]")

func FormatName(in string) string {
	ext := strings.ToLower(filepath.Ext(in))
	if ext == ".jpeg" {
		ext = ".jpg"
	}

	name, _ := strings.CutSuffix(in, filepath.Ext(in))
	name = strings.ToLower(filepath.Base(name))
	name = NameRegex.ReplaceAllString(name, "")

	return name + ext
}

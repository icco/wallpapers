package cdn

import (
	"net/url"
	"testing"
)

func TestImageURLs(t *testing.T) {
	for _, build := range []func(string) string{Thumb, FullRez} {
		u, err := url.Parse(build("a b&c.png"))
		if err != nil {
			t.Fatal(err)
		}
		if u.Host != "images.natwelch.com" || u.Path != "/wallpapers/a b&c.png" {
			t.Fatalf("bad image URL: %s", u)
		}
		if u.Query().Get("fit") != "crop" {
			t.Fatal("must fill wallpaper dimensions")
		}
	}
}

package ui

import (
	"strings"
	"testing"
)

func TestEmbeddedAssetsNonEmpty(t *testing.T) {
	cases := []struct {
		name string
		n    int
		min  int
	}{
		{"indexHTML", len(indexHTML), 500},
		{"appCSS", len(appCSS), 500},
		{"appJS", len(appJS), 500},
		{"forgeJS", len(forgeJS), 500},
		{"threeJS", len(threeJS), 100_000},
		{"afpMarkPNG", len(afpMarkPNG), 100},
	}
	for _, c := range cases {
		if c.n < c.min {
			t.Fatalf("%s: len=%d want >= %d", c.name, c.n, c.min)
		}
	}
}

func TestEmbeddedUIHasNoExternalAssets(t *testing.T) {
	html := string(indexHTML)
	for _, forbidden := range []string{
		"fonts.googleapis.com",
		"fonts.gstatic.com",
		"cdn.",
		"unpkg.com",
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("index.html must not load external assets (%s)", forbidden)
		}
	}
	css := string(appCSS)
	if !strings.Contains(css, `@font-face`) || !strings.Contains(css, `url("/fonts/`) {
		t.Fatal("app.css must self-host fonts via /fonts/")
	}
}

func TestEmbeddedFonts(t *testing.T) {
	for _, name := range []string{
		"static/fonts/ibm-plex-mono-400-latin.woff2",
		"static/fonts/ibm-plex-mono-500-latin.woff2",
		"static/fonts/syne-latin.woff2",
	} {
		data, err := fontFS.ReadFile(name)
		if err != nil {
			t.Fatalf("missing embedded font %s: %v", name, err)
		}
		if len(data) < 1000 {
			t.Fatalf("font %s too small (%d bytes)", name, len(data))
		}
	}
}

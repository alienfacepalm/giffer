package convert

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsArchiveAndStem(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		kind bool
		stem string
	}{
		{"photos.zip", true, "photos"},
		{"Photos.ZIP", true, "Photos"},
		{"shots.tar.gz", true, "shots"},
		{"shots.tgz", true, "shots"},
		{"shots.tar.bz2", true, "shots"},
		{"shots.tbz2", true, "shots"},
		{"shots.tbz", true, "shots"},
		{"shots.tar.xz", true, "shots"},
		{"shots.txz", true, "shots"},
		{"shots.tar", true, "shots"},
		{"shots.7z", true, "shots"},
		{"shots.rar", true, "shots"},
		{"Shots.RAR", true, "Shots"},
		{"shots.gif", false, "shots"},
		{"readme.txt", false, "readme"},
	}
	for _, tc := range cases {
		if got := IsArchive(tc.name); got != tc.kind {
			t.Errorf("IsArchive(%q)=%v want %v", tc.name, got, tc.kind)
		}
		if got := ArchiveStem(tc.name); got != tc.stem {
			t.Errorf("ArchiveStem(%q)=%q want %q", tc.name, got, tc.stem)
		}
	}
}

func TestArchiveToGIFTarGz(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "series.tar.gz")
	if err := writePNGTarGz(archivePath, map[string]image.Image{
		"b.png": solidPNG(12, 8, color.RGBA{0, 0, 255, 255}),
		"a.png": solidPNG(12, 8, color.RGBA{255, 0, 0, 255}),
	}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "series.gif")
	if err := ArchiveToGIF(Options{
		Input:   archivePath,
		Output:  out,
		DelayMS: 100,
		Loop:    0,
	}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Image) != 2 {
		t.Fatalf("frames=%d want 2", len(g.Image))
	}
}

func TestArchiveToGIFRar(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "photos.rar")
	archivePath := filepath.Join(dir, "series.rar")
	in, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath, in, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "series.gif")
	if err := ArchiveToGIF(Options{
		Input:   archivePath,
		Output:  out,
		DelayMS: 100,
		Loop:    0,
	}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Image) != 2 {
		t.Fatalf("frames=%d want 2", len(g.Image))
	}
}

func writePNGTarGz(path string, files map[string]image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	for name, img := range files {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		hdr := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(buf.Len()),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func solidPNG(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestSafeExtractPathRejectsZipSlip(t *testing.T) {
	t.Parallel()
	dest := t.TempDir()
	cases := []string{
		"../outside.png",
		"..\\outside.png",
		"foo/../../outside.png",
		"../../../../../../tmp/evil.png",
	}
	for _, entry := range cases {
		_, err := safeExtractPath(dest, entry)
		if err == nil {
			t.Fatalf("expected reject for %q", entry)
		}
		if !strings.Contains(err.Error(), "unsafe") && !strings.Contains(err.Error(), "invalid") {
			t.Fatalf("want unsafe/invalid for %q, got %v", entry, err)
		}
	}
	// Absolute-looking paths must not escape dest (land inside or reject).
	absCases := []string{"/etc/passwd", `C:\Windows\evil.png`, `C:/Windows/evil.png`}
	for _, entry := range absCases {
		target, err := safeExtractPath(dest, entry)
		if err == nil {
			rel, relErr := filepath.Rel(dest, target)
			if relErr != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
				t.Fatalf("accepted absolute entry %q escaped dest: %q", entry, target)
			}
		}
	}
	ok, err := safeExtractPath(dest, "photos/a.png")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dest, "photos", "a.png")
	if ok != want {
		t.Fatalf("got %q want %q", ok, want)
	}
}

func TestExtractLimiterRejectsOversizedEntry(t *testing.T) {
	prevEntry := MaxEntryBytes
	prevTotal := MaxExtractBytes
	MaxEntryBytes = 32
	MaxExtractBytes = 1 << 20
	t.Cleanup(func() {
		MaxEntryBytes = prevEntry
		MaxExtractBytes = prevTotal
	})

	dir := t.TempDir()
	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	lim := newExtractLimiter()
	err := lim.copyFile(filepath.Join(dest, "big.bin"), strings.NewReader(strings.Repeat("x", 64)))
	if err == nil || !strings.Contains(err.Error(), "entry exceeds size limit") {
		t.Fatalf("want entry size limit error, got %v", err)
	}
}

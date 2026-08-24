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
		{"shots.tar.xz", true, "shots"},
		{"shots.txz", true, "shots"},
		{"shots.tar", true, "shots"},
		{"shots.7z", true, "shots"},
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
